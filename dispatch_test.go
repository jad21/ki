// probar go test -bench=.

package ki

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Handlers simulados
func emptyHandler()                                                        {}
func ctxOnlyHandler(ctx *Context)                                          {}
func wrReqHandler(w http.ResponseWriter, r *http.Request)                  {}
func reqWrHandler(r *http.Request, w http.ResponseWriter)                  {}
func ctxWrReqHandler(ctx *Context, w http.ResponseWriter, r *http.Request) {}
func ctxReqWrHandler(ctx *Context, r *http.Request, w http.ResponseWriter) {}

// Handler con dependencia inyectada
func handlerWithInjection(db *sql.DB, ctx *Context) {}

func BenchmarkDispatch(b *testing.B) {
	// Inicialización desde la aplicación ki real
	app := New()
	app.Provide(func() *sql.DB {
		return &sql.DB{}
	})

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	ctx := &Context{
		Request:  req,
		Writer:   w,
		injector: app.DI,
	}

	app.DI.Maps(ctx)

	handlers := []struct {
		name string
		fn   HandlerFunc
	}{
		{"fnEmpty", emptyHandler},
		{"ctxOnly", ctxOnlyHandler},
		{"wrReq", wrReqHandler},
		{"reqWr", reqWrHandler},
		{"ctxWrReq", ctxWrReqHandler},
		{"ctxReqWr", ctxReqWrHandler},
		{"withInjection", handlerWithInjection},
	}

	for _, h := range handlers {
		b.Run(h.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if err := dispatch(ctx, h.fn); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

/**

 go test -bench=.
goos: linux
goarch: amd64
pkg: github.com/jad21/ki
cpu: Intel(R) Core(TM) i7-6600U CPU @ 2.60GHz
BenchmarkDispatch/fnEmpty-4             97123980                14.76 ns/op
BenchmarkDispatch/ctxOnly-4             100000000               12.57 ns/op
BenchmarkDispatch/wrReq-4               98338012                15.62 ns/op
BenchmarkDispatch/reqWr-4               90342417                14.98 ns/op
BenchmarkDispatch/ctxWrReq-4            100000000               14.59 ns/op
BenchmarkDispatch/ctxReqWr-4            100000000               14.72 ns/op
BenchmarkDispatch/withInjection-4        1731536               664.7 ns/op
PASS
ok      github.com/jad21/ki     10.616s
*/
