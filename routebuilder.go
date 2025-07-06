package ki

import (
	"io/fs"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// RouteBuilder permite crear rutas avanzadas, pipelines y grupos anidados.

type RouteBuilder struct {
	app       *App
	router    *router
	parent    *RouteBuilder
	method    string
	path      string
	prefix    string
	domain    string
	headers   map[string]string
	mws       []Middleware
	regexVars map[string]*regexp.Regexp

	cacheConf *cachePolicy

	// Hooks y handlers avanzados
	onError    func(ctx *Context, err error)
	notFound   func(ctx *Context)
	beforeEach func(ctx *Context)
	afterEach  func(ctx *Context)
}

// ========== CONSTRUCTOR PRINCIPAL ==========

func NewRouteBuilder(app *App, r *router) *RouteBuilder {
	return &RouteBuilder{
		app:       app,
		router:    r,
		headers:   make(map[string]string),
		regexVars: make(map[string]*regexp.Regexp),
	}
}

// ========== PIPELINE FLUIDO DE DEFINICIÓN ==========

func (rb *RouteBuilder) Path(path string) *RouteBuilder {
	rb.path = path
	return rb
}
func (rb *RouteBuilder) PathPrefix(prefix string) *RouteBuilder {
	rb.prefix = prefix
	return rb
}
func (rb *RouteBuilder) Method(method string) *RouteBuilder {
	rb.method = strings.ToUpper(method)
	return rb
}
func (rb *RouteBuilder) Domain(domain string) *RouteBuilder {
	rb.domain = domain
	return rb
}
func (rb *RouteBuilder) Headers(kvs ...string) *RouteBuilder {
	if rb.headers == nil {
		rb.headers = make(map[string]string)
	}
	for i := 0; i+1 < len(kvs); i += 2 {
		rb.headers[kvs[i]] = kvs[i+1]
	}
	return rb
}
func (rb *RouteBuilder) Use(mws ...Middleware) *RouteBuilder {
	rb.mws = append(rb.mws, mws...)
	return rb
}
func (rb *RouteBuilder) RegexVar(varName, pattern string) *RouteBuilder {
	if rb.regexVars == nil {
		rb.regexVars = make(map[string]*regexp.Regexp)
	}
	rb.regexVars[varName] = regexp.MustCompile(pattern)
	return rb
}
func (rb *RouteBuilder) Cache(duration time.Duration) *RouteBuilder {
	rb.cacheConf = &cachePolicy{duration: duration}
	return rb
}

// ========== HOOKS Y HANDLERS DE ERROR/NOTFOUND ==========

func (rb *RouteBuilder) OnError(fn func(ctx *Context, err error)) *RouteBuilder {
	rb.onError = fn
	return rb
}
func (rb *RouteBuilder) NotFound(fn func(ctx *Context)) *RouteBuilder {
	rb.notFound = fn
	return rb
}
func (rb *RouteBuilder) BeforeEach(fn func(ctx *Context)) *RouteBuilder {
	rb.beforeEach = fn
	return rb
}
func (rb *RouteBuilder) AfterEach(fn func(ctx *Context)) *RouteBuilder {
	rb.afterEach = fn
	return rb
}

// ========== REGISTRO DE HANDLER PRINCIPAL ==========

func (rb *RouteBuilder) Handle(fn HandlerFunc) *RouteBuilder {
	// Si hay política de caché, inserta el middleware al inicio
	if rb.cacheConf != nil {
		rb.mws = append([]Middleware{cacheMiddleware(rb.cacheConf.duration)}, rb.mws...)
	}
	rb.router.addRouteAdvanced(rb, fn)
	return rb
}

// Atajos para verbos HTTP comunes
func (g *RouteBuilder) Get(path string, fn HandlerFunc, middlewares ...Middleware) *RouteBuilder {
	return g.Path(path).Method("GET").Use(middlewares...).Handle(fn)
}
func (g *RouteBuilder) Post(path string, fn HandlerFunc, middlewares ...Middleware) *RouteBuilder {
	return g.Path(path).Method("POST").Use(middlewares...).Handle(fn)
}
func (g *RouteBuilder) Put(path string, fn HandlerFunc, middlewares ...Middleware) *RouteBuilder {
	return g.Path(path).Method("PUT").Use(middlewares...).Handle(fn)
}
func (g *RouteBuilder) Delete(path string, fn HandlerFunc, middlewares ...Middleware) *RouteBuilder {
	return g.Path(path).Method("DELETE").Use(middlewares...).Handle(fn)
}
func (g *RouteBuilder) Options(path string, fn HandlerFunc, middlewares ...Middleware) *RouteBuilder {
	return g.Path(path).Method("OPTIONS").Use(middlewares...).Handle(fn)
}
func (g *RouteBuilder) Head(path string, fn HandlerFunc, middlewares ...Middleware) *RouteBuilder {
	return g.Path(path).Method("HEAD").Use(middlewares...).Handle(fn)
}

// Para estáticos
// Para estáticos con string dir
func (rb *RouteBuilder) Static(dir string) *RouteBuilder {
	if rb.path == "" && rb.prefix == "" {
		panic("Debes usar Path o PathPrefix antes de Static")
	}
	fileServer := http.FileServer(http.Dir(dir))
	handler := http.Handler(fileServer)
	// Si se usa PathPrefix, aplica StripPrefix automáticamente
	if rb.prefix != "" {
		handler = http.StripPrefix(rb.prefix, fileServer)
	}
	rb.router.addRouteAdvanced(rb, handler)
	return rb
}

// Para estáticos con fs.FS
func (rb *RouteBuilder) StaticFS(fs fs.FS) *RouteBuilder {
	if rb.path == "" && rb.prefix == "" {
		panic("Debes usar Path o PathPrefix antes de StaticFS")
	}
	fileServer := http.FileServer(http.FS(fs))
	handler := http.Handler(fileServer)
	if rb.prefix != "" {
		handler = http.StripPrefix(rb.prefix, fileServer)
	}
	rb.router.addRouteAdvanced(rb, handler)
	return rb
}

// ========== ANIDAMIENTO DE GRUPOS Y PREFIJOS ==========

func (rb *RouteBuilder) Group(path string, fn ...func(r Router)) *GroupRouter {
	child := &GroupRouter{
		RouteBuilder: &RouteBuilder{
			app:        rb.app,
			router:     rb.router,
			parent:     rb,
			path:       rb.path + path,
			domain:     rb.domain,
			mws:        append([]Middleware{}, rb.mws...),
			headers:    copyMap(rb.headers),
			regexVars:  copyRegex(rb.regexVars),
			cacheConf:  rb.cacheConf,
			onError:    rb.onError,
			notFound:   rb.notFound,
			beforeEach: rb.beforeEach,
			afterEach:  rb.afterEach,
		},
		basePath: rb.path + path,
	}
	if len(fn) > 0 {
		fn[0](child)
	}
	return child
}
func (rb *RouteBuilder) PathPrefixGroup(prefix string, fn ...func(r Router)) *GroupRouter {
	child := &GroupRouter{
		RouteBuilder: &RouteBuilder{
			app:        rb.app,
			router:     rb.router,
			parent:     rb,
			prefix:     rb.prefix + prefix,
			domain:     rb.domain,
			mws:        append([]Middleware{}, rb.mws...),
			headers:    copyMap(rb.headers),
			regexVars:  copyRegex(rb.regexVars),
			cacheConf:  rb.cacheConf,
			onError:    rb.onError,
			notFound:   rb.notFound,
			beforeEach: rb.beforeEach,
			afterEach:  rb.afterEach,
		},
		basePath: rb.prefix + prefix,
	}
	if len(fn) > 0 {
		fn[0](child)
	}
	return child
}

// ========== HELPERS INTERNOS ==========

func copyMap(m map[string]string) map[string]string {
	out := make(map[string]string)
	for k, v := range m {
		out[k] = v
	}
	return out
}
func copyRegex(m map[string]*regexp.Regexp) map[string]*regexp.Regexp {
	out := make(map[string]*regexp.Regexp)
	for k, v := range m {
		out[k] = v
	}
	return out
}
