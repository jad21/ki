package ki

import (
	"io/fs"
	"regexp"
	"time"
)

// GroupRouter representa un grupo de rutas con base path y settings heredados.
type GroupRouter struct {
	*RouteBuilder
	basePath string
}

// ========== CONSTRUCTOR PRINCIPAL ==========

func NewGroupRouter(app *App, router *router, base string, parent *RouteBuilder) *GroupRouter {
	rb := &RouteBuilder{
		app:    app,
		router: router,
		path:   base,
	}
	if parent != nil {
		rb.parent = parent
		rb.domain = parent.domain
		rb.mws = append([]Middleware{}, parent.mws...)
		rb.headers = copyMap(parent.headers)
		rb.regexVars = copyRegex(parent.regexVars)
		rb.cacheConf = parent.cacheConf
		rb.onError = parent.onError
		rb.notFound = parent.notFound
		rb.beforeEach = parent.beforeEach
		rb.afterEach = parent.afterEach
	} else {
		rb.headers = make(map[string]string)
		rb.regexVars = make(map[string]*regexp.Regexp)
	}
	return &GroupRouter{
		RouteBuilder: rb,
		basePath:     base,
	}
}

// ========== ATJOS DE GRUPO ==========

func (g *GroupRouter) Path(path string) *RouteBuilder {
	return &RouteBuilder{
		app:        g.app,
		router:     g.router,
		parent:     g.RouteBuilder,
		path:       g.basePath + path,
		domain:     g.domain,
		mws:        append([]Middleware{}, g.mws...),
		headers:    copyMap(g.headers),
		regexVars:  copyRegex(g.regexVars),
		cacheConf:  g.cacheConf,
		onError:    g.onError,
		notFound:   g.notFound,
		beforeEach: g.beforeEach,
		afterEach:  g.afterEach,
	}
}

func (g *GroupRouter) PathPrefix(prefix string) *RouteBuilder {
	return &RouteBuilder{
		app:        g.app,
		router:     g.router,
		parent:     g.RouteBuilder,
		prefix:     g.basePath + prefix,
		domain:     g.domain,
		mws:        append([]Middleware{}, g.mws...),
		headers:    copyMap(g.headers),
		regexVars:  copyRegex(g.regexVars),
		cacheConf:  g.cacheConf,
		onError:    g.onError,
		notFound:   g.notFound,
		beforeEach: g.beforeEach,
		afterEach:  g.afterEach,
	}
}

// ========== ANIDAMIENTO DE GRUPOS ==========

func (g *GroupRouter) Group(path string, fn ...func(r Router)) *GroupRouter {
	child := &GroupRouter{
		RouteBuilder: &RouteBuilder{
			app:        g.app,
			router:     g.router,
			parent:     g.RouteBuilder,
			path:       g.basePath + path,
			domain:     g.domain,
			mws:        append([]Middleware{}, g.mws...),
			headers:    copyMap(g.headers),
			regexVars:  copyRegex(g.regexVars),
			cacheConf:  g.cacheConf,
			onError:    g.onError,
			notFound:   g.notFound,
			beforeEach: g.beforeEach,
			afterEach:  g.afterEach,
		},
		basePath: g.basePath + path,
	}
	if len(fn) > 0 {
		fn[0](child)
	}
	return child
}

func (g *GroupRouter) PathPrefixGroup(prefix string, fn ...func(r Router)) *GroupRouter {
	child := &GroupRouter{
		RouteBuilder: &RouteBuilder{
			app:        g.app,
			router:     g.router,
			parent:     g.RouteBuilder,
			prefix:     g.basePath + prefix,
			domain:     g.domain,
			mws:        append([]Middleware{}, g.mws...),
			headers:    copyMap(g.headers),
			regexVars:  copyRegex(g.regexVars),
			cacheConf:  g.cacheConf,
			onError:    g.onError,
			notFound:   g.notFound,
			beforeEach: g.beforeEach,
			afterEach:  g.afterEach,
		},
		basePath: g.basePath + prefix,
	}
	if len(fn) > 0 {
		fn[0](child)
	}
	return child
}

// ========== ATAJOS PARA VERBOS (necesarios) HTTP ==========

func (g *GroupRouter) Get(path string, fn HandlerFunc, middlewares ...Middleware) *RouteBuilder {
	return g.Path(path).Method("GET").Use(middlewares...).Handle(fn)
}
func (g *GroupRouter) Post(path string, fn HandlerFunc, middlewares ...Middleware) *RouteBuilder {
	return g.Path(path).Method("POST").Use(middlewares...).Handle(fn)
}
func (g *GroupRouter) Put(path string, fn HandlerFunc, middlewares ...Middleware) *RouteBuilder {
	return g.Path(path).Method("PUT").Use(middlewares...).Handle(fn)
}
func (g *GroupRouter) Delete(path string, fn HandlerFunc, middlewares ...Middleware) *RouteBuilder {
	return g.Path(path).Method("DELETE").Use(middlewares...).Handle(fn)
}
func (g *GroupRouter) Options(path string, fn HandlerFunc, middlewares ...Middleware) *RouteBuilder {
	return g.Path(path).Method("OPTIONS").Use(middlewares...).Handle(fn)
}
func (g *GroupRouter) Head(path string, fn HandlerFunc, middlewares ...Middleware) *RouteBuilder {
	return g.Path(path).Method("HEAD").Use(middlewares...).Handle(fn)
}

// ========== PARA ARCHIVOS ESTÁTICOS ==========

// Para estáticos en GroupRouter con un directorio
func (g *GroupRouter) Static(path string, dir string) *RouteBuilder {
	return g.PathPrefix(path).Static(dir)
}
func (g *GroupRouter) StaticFS(path string, fsys fs.FS) *RouteBuilder {
	return g.PathPrefix(path).StaticFS(fsys)
}

func (g *GroupRouter) Cache(duration time.Duration) *GroupRouter {
	g.cacheConf = &cachePolicy{duration: duration}
	return g
}

// ========== HOOKS Y HANDLERS DE ERROR/NOTFOUND ==========

func (g *GroupRouter) OnError(fn func(ctx *Context, err error)) *GroupRouter {
	g.onError = fn
	return g
}
func (g *GroupRouter) NotFound(fn func(ctx *Context)) *GroupRouter {
	g.notFound = fn
	return g
}
func (g *GroupRouter) BeforeEach(fn func(ctx *Context)) *GroupRouter {
	g.beforeEach = fn
	return g
}
func (g *GroupRouter) AfterEach(fn func(ctx *Context)) *GroupRouter {
	g.afterEach = fn
	return g
}
