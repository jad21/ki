package ki

import (
	"net/http"
	"sync"
	"time"
)

// cachePolicy para futuras implementaciones de cache por ruta
type cachePolicy struct {
	duration time.Duration
}

// Estructura interna de caché en memoria para una ruta
type cacheEntry struct {
	content   []byte
	status    int
	header    http.Header
	expiresAt time.Time
}

func cacheMiddleware(duration time.Duration) Middleware {
	var mu sync.Mutex
	var cache *cacheEntry

	return func(ctx *Context) {
		mu.Lock()
		defer mu.Unlock()
		now := time.Now()
		if cache != nil && now.Before(cache.expiresAt) {
			// Sirve respuesta cacheada
			for k, vals := range cache.header {
				for _, v := range vals {
					ctx.Writer.Header().Add(k, v)
				}
			}
			ctx.Writer.WriteHeader(cache.status)
			ctx.Writer.Write(cache.content)
			return
		}
		// Captura respuesta del handler
		rec := &responseRecorder{ResponseWriter: ctx.Writer, header: make(http.Header)}
		ctx.Writer = rec
		ctx.Next() // Ejecuta el resto del pipeline (handler)
		cache = &cacheEntry{
			content:   append([]byte(nil), rec.body...), // copia defensiva
			status:    rec.status,
			header:    rec.header.Clone(),
			expiresAt: time.Now().Add(duration),
		}
	}
}

// Minimal recorder para respuestas HTTP
type responseRecorder struct {
	http.ResponseWriter
	header http.Header
	body   []byte
	status int
}

func (r *responseRecorder) Header() http.Header {
	return r.header
}
func (r *responseRecorder) Write(b []byte) (int, error) {
	r.body = append(r.body, b...)
	return r.ResponseWriter.Write(b)
}
func (r *responseRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}
