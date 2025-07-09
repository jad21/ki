package ki

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// type Middleware interface{}

func chainMiddlewares(final HandlerFunc, mws []Middleware) HandlerFunc {
	if len(mws) == 0 {
		return final
	}
	// Encadena recursivamente
	mw := mws[0]
	next := chainMiddlewares(final, mws[1:])
	return func(ctx *Context) {
		ctx.next = func() error {
			return dispatch(ctx, next)
		}
		dispatch(ctx, mw)
	}
}

// LoggingHandler middleware
func LoggingHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		duration := time.Since(start)
		fmt.Fprintf(os.Stdout, "%s - %s (%s) [%s]\n", r.Method, r.URL.Path, r.RemoteAddr, duration)
	})
}

func LoggingHandlerWithOutput(out io.Writer, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		duration := time.Since(start)
		fmt.Fprintf(out, "%s - %s (%s) [%s]\n", r.Method, r.URL.Path, r.RemoteAddr, duration)
	})
}

// ProxyHeaders middleware
func ProxyHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set RemoteAddr from X-Forwarded-For, if exists
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			r.RemoteAddr = xff
		}
		// Set scheme from X-Forwarded-Proto, if exists
		if xfproto := r.Header.Get("X-Forwarded-Proto"); xfproto != "" {
			r.URL.Scheme = xfproto
		}
		next.ServeHTTP(w, r)
	})
}

// Middleware automático para refrescar la sesión
func RefreshSessionMiddleware(ctx *Context) {
	// app.Use(RefreshSessionMiddleware)
	_ = ctx.RefreshSession()
	ctx.Next()
}
