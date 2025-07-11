package ki

import (
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
)

type HandlerFunc interface{}
type Middleware interface{}

// Define tipos nominales sólo para facilitar los type-switch
type (
	fnEmpty  = func()
	ctxOnly  = func(*Context)
	wrReq    = func(http.ResponseWriter, *http.Request)
	reqWr    = func(*http.Request, http.ResponseWriter)
	ctxWrReq = func(*Context, http.ResponseWriter, *http.Request)
	ctxReqWr = func(*Context, *http.Request, http.ResponseWriter)
)

// ---------- NUEVA INTERFAZ PIPELINE PARA ROUTER ----------

type Router interface {
	Use(...Middleware) *RouteBuilder
	Get(string, HandlerFunc, ...Middleware) *RouteBuilder
	Post(string, HandlerFunc, ...Middleware) *RouteBuilder
	Put(string, HandlerFunc, ...Middleware) *RouteBuilder
	Delete(string, HandlerFunc, ...Middleware) *RouteBuilder
	Options(string, HandlerFunc, ...Middleware) *RouteBuilder
	Head(string, HandlerFunc, ...Middleware) *RouteBuilder
	Handle(HandlerFunc) *RouteBuilder
	Group(string, ...func(Router)) *GroupRouter
}

// ------------- ESTRUCTURA INTERNA DE LA RUTA --------------

type route struct {
	method      string
	pattern     string
	segments    []string
	prefix      string
	isPrefix    bool
	handler     HandlerFunc
	middlewares []Middleware

	// Avanzado
	domain    string
	headers   map[string]string
	regexVars map[string]*regexp.Regexp

	// Hooks y handlers
	onError    func(ctx *Context, err error)
	notFound   func(ctx *Context)
	beforeEach func(ctx *Context)
	afterEach  func(ctx *Context)
}

// Router principal
type router struct {
	routes      []*route
	middlewares []Middleware
	app         *App
}

// ---------- CONSTRUCTOR ----------

func NewRoute(app *App) *router {
	return &router{app: app}
}

// ----------- PIPELINE ----------

// Usa middlewares globales
func (r *router) Use(mws ...Middleware) *RouteBuilder {
	r.middlewares = append(r.middlewares, mws...)
	return NewRouteBuilder(r.app, r)
}

// Métodos HTTP
func (r *router) Get(path string, fn HandlerFunc, mws ...Middleware) *RouteBuilder {
	return NewRouteBuilder(r.app, r).Path(path).Method("GET").Use(mws...).Handle(fn)
}
func (r *router) Post(path string, fn HandlerFunc, mws ...Middleware) *RouteBuilder {
	return NewRouteBuilder(r.app, r).Path(path).Method("POST").Use(mws...).Handle(fn)
}
func (r *router) Put(path string, fn HandlerFunc, mws ...Middleware) *RouteBuilder {
	return NewRouteBuilder(r.app, r).Path(path).Method("PUT").Use(mws...).Handle(fn)
}
func (r *router) Delete(path string, fn HandlerFunc, mws ...Middleware) *RouteBuilder {
	return NewRouteBuilder(r.app, r).Path(path).Method("DELETE").Use(mws...).Handle(fn)
}
func (r *router) Options(path string, fn HandlerFunc, mws ...Middleware) *RouteBuilder {
	return NewRouteBuilder(r.app, r).Path(path).Method("OPTIONS").Use(mws...).Handle(fn)
}
func (r *router) Head(path string, fn HandlerFunc, mws ...Middleware) *RouteBuilder {
	return NewRouteBuilder(r.app, r).Path(path).Method("HEAD").Use(mws...).Handle(fn)
}

// Genérico: permite método arbitrario
func (r *router) Handle(method string, path string, fn HandlerFunc, mws ...Middleware) *RouteBuilder {
	return NewRouteBuilder(r.app, r).Path(path).Method(method).Use(mws...).Handle(fn)
}

// Grupos pipeline: devuelve RouteBuilder para continuar pipeline
func (r *router) Group(prefix string, fn ...func(Router)) *RouteBuilder {
	rb := NewRouteBuilder(r.app, r).Path(prefix)
	gr := rb.Group(prefix)
	if len(fn) > 0 {
		fn[0](gr)
	}
	return rb
}

// ----------- CORE: REGISTRO DE LA RUTA EN EL ROUTER ----------

func (r *router) addRouteAdvanced(rb *RouteBuilder, handler HandlerFunc) {
	isPrefix := rb.prefix != ""
	pattern := rb.path
	if isPrefix {
		pattern = rb.prefix
	}
	segments := splitPattern(pattern)

	// Middlewares: globales primero, luego builder
	var mws []Middleware
	// globalCount := 0
	// builderCount := 0
	if r.middlewares != nil {
		mws = append(mws, r.middlewares...)
		// globalCount = len(r.middlewares)
	}
	if rb.mws != nil {
		mws = append(mws, rb.mws...)
		// builderCount = len(rb.mws)
	}
	// ⬇️ Aquí añade este bloque para debug:
	// if len(mws) == 0 {
	// 	panic("Sin middlewares en la ruta registrada: debe tener al menos los globales si .Use() fue llamado")
	// }
	// totalCount := len(mws)
	// DEPURACIÓN
	// fmt.Printf("[DEBUG] Registrando ruta %-20s | middlewares globales: %-2d | builder: %-2d | total: %-2d\n", pattern, globalCount, builderCount, totalCount)

	rt := &route{
		method:      rb.method,
		pattern:     pattern,
		segments:    segments,
		prefix:      rb.prefix,
		isPrefix:    isPrefix,
		handler:     handler,
		middlewares: mws,
		domain:      rb.domain,
		headers:     copyMap(rb.headers),
		regexVars:   copyRegex(rb.regexVars),
		onError:     rb.onError,
		notFound:    rb.notFound,
		beforeEach:  rb.beforeEach,
		afterEach:   rb.afterEach,
	}
	r.routes = append(r.routes, rt)

	// Sort alphabetically by path: fix: las que tienen parámetros, este de ultimo
	sort.Slice(r.routes, func(i, j int) bool {
		left := strings.Join(r.routes[i].segments, "/")
		right := strings.Join(r.routes[j].segments, "/")
		return left < right
	})
}

// ----------- LEGACY PATHPREFIX -----------

func (r *router) AddPathPrefix(prefix string, handler http.Handler) {
	r.addRouteAdvanced(
		&RouteBuilder{prefix: prefix, mws: nil},
		handler,
	)
}

// ----------- MATCHING Y PIPELINE -----------
func splitPattern(p string) []string {
	p = strings.TrimPrefix(p, "/")
	if p == "" {
		return []string{}
	}
	return strings.Split(p, "/")
}

func (r *router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	path := req.URL.Path
	method := req.Method
	host := req.Host

	var matched *route
	var params map[string]string

	// 1. Matching exacto
	for _, rt := range r.routes {
		if rt.isPrefix {
			continue
		}
		if rt.domain != "" && rt.domain != host {
			continue
		}
		if rt.method != "" && method != rt.method {
			continue
		}
		vars, ok := matchSegmentsWithRegex(rt.segments, splitPattern(path), rt.regexVars)
		if !ok {
			continue
		}
		if !matchHeaders(rt.headers, req.Header) {
			continue
		}
		matched = rt
		params = vars
		break
	}
	// 2. Matching por prefijo
	if matched == nil {
		for _, rt := range r.routes {
			if !rt.isPrefix {
				continue
			}
			if rt.domain != "" && rt.domain != host {
				continue
			}
			if rt.method != "" && method != rt.method {
				continue
			}
			if !strings.HasPrefix(path, rt.prefix) {
				continue
			}
			if !matchHeaders(rt.headers, req.Header) {
				continue
			}
			vars, ok := matchSegmentsWithRegex(rt.segments, splitPattern(path), rt.regexVars)
			if !ok {
				vars = map[string]string{}
			}
			matched = rt
			params = vars
			break
		}
	}
	// 3. Not found/Handler de error
	if matched == nil {
		app := r.app
		ctx, err := UseContext(app, w, req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if app.notFound != nil {
			app.notFound(ctx)
			return
		}
		http.NotFound(w, req)
		return
	}
	// 4. Ejecuta pipeline
	ctx, err := UseContext(r.app, w, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ctx.params = params
	if matched.beforeEach != nil {
		matched.beforeEach(ctx)
	} else if r.app.before != nil {
		r.app.before(ctx)
	}
	handler := chainMiddlewares(matched.handler, matched.middlewares)
	defer func() {
		if matched.afterEach != nil {
			matched.afterEach(ctx)
		} else if r.app.after != nil {
			r.app.after(ctx)
		}
		if rec := recover(); rec != nil {
			var err error
			switch v := rec.(type) {
			case error:
				err = v
			default:
				err = fmt.Errorf("%v", rec)
			}
			if matched.onError != nil {
				matched.onError(ctx, err)
			} else if r.app.onError != nil {
				r.app.onError(ctx, err)
			} else {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}
	}()
	if err := dispatch(ctx, handler); err != nil {
		if matched.onError != nil {
			matched.onError(ctx, err)
		} else if r.app.onError != nil {
			r.app.onError(ctx, err)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// ----------- MATCHING AVANZADO -----------

func matchSegmentsWithRegex(patternSeg, pathSeg []string, regexVars map[string]*regexp.Regexp) (map[string]string, bool) {
	if len(patternSeg) != len(pathSeg) {
		return nil, false
	}
	params := map[string]string{}
	for i := 0; i < len(patternSeg); i++ {
		p := patternSeg[i]
		s := pathSeg[i]
		if len(p) > 0 && (p[0] == ':' || (len(p) > 1 && p[0] == '{' && p[len(p)-1] == '}')) {
			varName, varRegex := parseVarAndRegex(p)
			if varRegex != "" {
				re := regexVars[varName]
				if re == nil {
					re = regexp.MustCompile(varRegex)
				}
				if !re.MatchString(s) {
					return nil, false
				}
			}
			params[varName] = s
		} else if p != s {
			return nil, false
		}
	}
	return params, true
}

func parseVarAndRegex(segment string) (string, string) {
	if strings.Contains(segment, "(") && strings.HasSuffix(segment, ")") {
		start := strings.Index(segment, "(")
		varName := strings.Trim(segment[:start], ":{}")
		regex := segment[start+1 : len(segment)-1]
		return varName, regex
	}
	return strings.Trim(segment, ":{}"), ""
}

func matchHeaders(expected map[string]string, headers http.Header) bool {
	if len(expected) == 0 {
		return true
	}
	for k, v := range expected {
		if headers.Get(k) != v {
			return false
		}
	}
	return true
}

func dispatch(ctx *Context, h HandlerFunc) error {
	w, r := ctx.Writer, ctx.Request
	switch fn := h.(type) {

	case fnEmpty:
		fn()
		return nil
	case ctxOnly:
		fn(ctx)
		return nil

	case wrReq:
		fn(w, r)
		return nil

	case reqWr:
		fn(r, w)
		return nil

	case ctxWrReq:
		fn(ctx, w, r)
		return nil

	case ctxReqWr:
		fn(ctx, r, w)
		return nil

	default:
		// Recurre al inyector sólo cuando hace falta reflexión real
		return ctx.injector.InvokeWithErrorOnly(h)
	}
}
