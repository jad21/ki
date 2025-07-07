package ki

import (
	"context"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"reflect"
	"sync"
	"time"

	"git.jdev.run/jad21/di"
	"github.com/jad21/ki/env"
)

// IMPORTANTE: los siguientes imports serán válidos cuando crees estos archivos y paquetes.
// Puedes comenzar como package main en los archivos nuevos para pruebas iniciales, luego ajustar el import path.

type Handler interface {
	Expose(*App)
}
type M map[string]any

// Deprecated: Use M instead. H will be removed in a future version.
type H map[string]any

type FileReader interface {
	Open(string) (fs.File, error)
}

type App struct {
	context.Context
	pool           sync.Pool
	Router         *router
	WriteTimeout   time.Duration
	ReadTimeout    time.Duration
	Handlers       []Handler
	Templates      FileReader
	TemplatesFuncs template.FuncMap
	DI             di.Injector

	// Nuevos handlers globales
	onError  func(ctx *Context, err error)
	notFound func(ctx *Context)
	before   func(ctx *Context)
	after    func(ctx *Context)
}

type options struct {
	WriteTimeout   time.Duration
	ReadTimeout    time.Duration
	Templates      FileReader
	TemplatesFuncs template.FuncMap
}
type Option func(o *options)

var defaultOptions = options{
	Templates:    os.DirFS(env.GetEnvVar("TEMPLATES", "templates")),
	WriteTimeout: 60 * time.Second,
	ReadTimeout:  60 * time.Second,
}

func New(opt ...Option) *App {
	opts := defaultOptions
	for _, o := range opt {
		o(&opts)
	}

	app := &App{
		DI:             di.New(),
		Context:        context.Background(),
		WriteTimeout:   opts.WriteTimeout,
		ReadTimeout:    opts.ReadTimeout,
		Templates:      opts.Templates,
		TemplatesFuncs: opts.TemplatesFuncs,
	}
	app.Router = NewRoute(app)
	app.pool.New = func() interface{} {
		return NewContext(app.Context, app, nil, nil)
	}
	app.DI.Map(app.DI, di.WithInterface((*di.Injector)(nil)))
	app.Inject(app.Context)
	app.Inject(app.Router)
	return app
}

func SetTemplates(reader FileReader) Option {
	return func(o *options) {
		o.Templates = reader
	}
}
func SetTemplatesFuncs(fn template.FuncMap) Option {
	return func(o *options) {
		o.TemplatesFuncs = fn
	}
}
func SetWriteTimeout(t time.Duration) Option {
	return func(o *options) {
		o.WriteTimeout = t
	}
}
func SetReadTimeout(t time.Duration) Option {
	return func(o *options) {
		o.ReadTimeout = t
	}
}

func (s *App) acquireContext() *Context {
	return s.pool.Get().(*Context)
}
func (e *App) releaseContext(c *Context) {
	c.next = nil
	c.injector = nil
	c.Request = nil
	c.Writer = nil
	e.pool.Put(c)
}

// Inyectar variable inicializada
func (s *App) Inject(v interface{}, o ...di.Option) reflect.Type {
	return s.DI.Map(v, o...)
}

// Inyectar variable lazy
func (s *App) Provide(v interface{}, o ...di.Option) []reflect.Type {
	return s.DI.Provide(v, o...)
}

// Obtener variable inyectada
func (s *App) Resolve(v ...di.Pointer) error {
	return s.DI.Resolve(v)
}

// Obtener variable inyectada
func (s *App) Invoke(v interface{}) ([]reflect.Value, error) {
	return s.DI.Invoke(v)
}

// ----------------------------------------------
// NUEVOS MÉTODOS PIPELINE EN APP
// ----------------------------------------------
func (app *App) Path(path string) *RouteBuilder {
	return NewRouteBuilder(app, app.Router).Path(path)
}

func (app *App) PathPrefix(prefix string) *RouteBuilder {
	return NewRouteBuilder(app, app.Router).PathPrefix(prefix)
}

func (app *App) Domain(domain string) *RouteBuilder {
	return NewRouteBuilder(app, app.Router).Domain(domain)
}

func (app *App) Group(path string, fn ...func(r Router)) *GroupRouter {
	gr := NewGroupRouter(app, app.Router, path, nil)
	if len(fn) > 0 {
		fn[0](gr)
	}
	return gr
}

// ----------------------------------------------
// API TRADICIONAL (RETROCOMPATIBLE, AHORA PIPELINE)
// ----------------------------------------------

func (s *App) Use(middlewares ...Middleware) *RouteBuilder {
	return s.Router.Use(middlewares...)
}

func (s *App) Get(path string, fn HandlerFunc, middlewares ...Middleware) *RouteBuilder {
	return s.Router.Get(path, fn, middlewares...)
}

func (s *App) Post(path string, fn HandlerFunc, middlewares ...Middleware) *RouteBuilder {
	return s.Router.Post(path, fn, middlewares...)
}

func (s *App) Put(path string, fn HandlerFunc, middlewares ...Middleware) *RouteBuilder {
	return s.Router.Put(path, fn, middlewares...)
}

func (s *App) Delete(path string, fn HandlerFunc, middlewares ...Middleware) *RouteBuilder {
	return s.Router.Delete(path, fn, middlewares...)
}

func (s *App) Options(path string, fn HandlerFunc, middlewares ...Middleware) *RouteBuilder {
	return s.Router.Options(path, fn, middlewares...)
}

func (s *App) Head(path string, fn HandlerFunc, middlewares ...Middleware) *RouteBuilder {
	return s.Router.Head(path, fn, middlewares...)
}

// Set handler
func (s *App) AddHandler(h Handler) {
	s.Handlers = append(s.Handlers, h)
	h.Expose(s)
}

// Emula PathPrefix usando tu propio router, ejecutando un HandlerFunc para todo lo que comienza con tpl
func (s *App) PathPrefixOld(tpl string, fn HandlerFunc) {
	s.Router.AddPathPrefix(tpl, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, err := UseContext(s, w, r)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		// Invoca usando DI, así soportas cualquier firma
		if err := ctx.injector.InvokeWithErrorOnly(fn); err != nil {
			http.Error(w, err.Error(), 500)
		}
	}))
}

// Para servir estáticos, igual: asocia un handler nativo para todo lo que empiece con path
func (s *App) StaticHandler(path string, h http.Handler) {
	s.Router.AddPathPrefix(path, h)
}

// Set static path and dir string
func (s *App) Static(path string, dir string) {
	staticDir := http.Dir(dir)
	// StripPrefix para que FileServer busque correctamente
	s.StaticHandler(path, http.StripPrefix(path, http.FileServer(staticDir)))
}

// Set static path and dir http.FS
func (s *App) StaticFS(path string, fs fs.FS) {
	s.StaticHandler(path, http.FileServer(http.FS(fs)))
}

// ----------------------------------------------
// Server
// ----------------------------------------------
func (s *App) ListenAndServe() {
	port := env.GetEnvVar("PORT", "5000")
	log.Printf("go to http://localhost:%s", port)

	handler := LoggingHandler(s.Router)
	handler = ProxyHeaders(handler)

	srv := &http.Server{
		Handler:      handler,
		Addr:         ":" + port,
		WriteTimeout: s.WriteTimeout,
		ReadTimeout:  s.ReadTimeout,
	}
	log.Fatal(srv.ListenAndServe())
}
