package ki

import (
	"database/sql"
	"embed"
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Servicio simulado para DI
type mockService struct {
	Value string
}

func TestRouter_Advanced(t *testing.T) {
	app := New()

	// Inyecta dependencias
	app.Provide(func() *mockService {
		return &mockService{Value: "injected"}
	})
	app.Provide(func() *sql.DB {
		return &sql.DB{}
	})

	// Middlewares (contar ejecuciones)
	mwCounter := 0
	testMiddleware := func(ctx *Context) {
		// fmt.Println(">> testMiddleware ejecutado")

		mwCounter++
		ctx.Next()
	}

	app.Use(testMiddleware)

	// Rutas simples
	app.Get("/ping", func(ctx *Context) {
		ctx.Text(200, "pong")
	})

	// Rutas con parámetros
	app.Get("/user/:id", func(ctx *Context) {
		id := ctx.Vars()["id"]
		ctx.JSON(200, H{"user_id": id})
	})

	// Inyección de dependencias en handler
	app.Get("/inject", func(s *mockService, ctx *Context) {
		ctx.JSON(200, H{"val": s.Value})
	})

	// Grupo de rutas
	app.Group("/admin", func(r Router) {
		r.Get("/status", func(ctx *Context) {
			ctx.Text(200, "admin-ok")
		})
	})

	// PathPrefix
	prefixHit := false
	app.PathPrefix("/api/").Handle(func(ctx *Context) {
		prefixHit = true
		ctx.Text(200, "api-prefix")
	})

	// StaticHandler (handler simulado)
	app.StaticHandler("/static/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "static-file")
	}))

	// Lanzar servidor de prueba
	server := httptest.NewServer(app.Router)
	defer server.Close()

	// Subtests

	t.Run("Get /ping", func(t *testing.T) {
		resp, body := httpGet(t, server.URL+"/ping")
		assertStatus(t, resp, 200)
		assertBody(t, body, "pong")
	})

	t.Run("Middleware executed", func(t *testing.T) {
		// El middleware se ejecuta en todas las rutas
		mwCounter = 0
		// fmt.Printf("Antes de /ping, mwCounter: %d\n", mwCounter)
		httpGet(t, server.URL+"/ping")
		// fmt.Printf("Después de /ping, mwCounter: %d\n", mwCounter)
		if mwCounter == 0 {
			t.Error("Middleware was not executed")
		}
	})

	t.Run("Get /user/:id", func(t *testing.T) {
		resp, body := httpGet(t, server.URL+"/user/42")
		assertStatus(t, resp, 200)
		var data map[string]string
		json.Unmarshal([]byte(body), &data)
		if data["user_id"] != "42" {
			t.Errorf("Expected user_id=42, got %v", data)
		}
	})

	t.Run("Get /inject (DI)", func(t *testing.T) {
		resp, body := httpGet(t, server.URL+"/inject")
		assertStatus(t, resp, 200)
		var data map[string]string
		json.Unmarshal([]byte(body), &data)
		if data["val"] != "injected" {
			t.Errorf("Expected val=injected, got %v", data)
		}
	})

	t.Run("Get /admin/status (Group)", func(t *testing.T) {
		resp, body := httpGet(t, server.URL+"/admin/status")
		assertStatus(t, resp, 200)
		assertBody(t, body, "admin-ok")
	})

	t.Run("PathPrefix /api/", func(t *testing.T) {
		prefixHit = false
		resp, body := httpGet(t, server.URL+"/api/any/path")
		assertStatus(t, resp, 200)
		assertBody(t, body, "api-prefix")
		if !prefixHit {
			t.Error("PathPrefix handler was not hit")
		}
	})

	t.Run("StaticHandler /static/", func(t *testing.T) {
		resp, body := httpGet(t, server.URL+"/static/test.txt")
		assertStatus(t, resp, 200)
		assertBody(t, body, "static-file")
	})

	t.Run("404 NotFound", func(t *testing.T) {
		resp, _ := httpGet(t, server.URL+"/nope")
		if resp.StatusCode != 404 {
			t.Errorf("Expected 404, got %d", resp.StatusCode)
		}
	})
}

// =========== Helpers ============

func httpGet(t *testing.T, url string) (*http.Response, string) {
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("http.Get failed: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	return resp, string(body)
}

func assertStatus(t *testing.T, resp *http.Response, want int) {
	t.Helper()
	if resp.StatusCode != want {
		t.Fatalf("Expected status %d, got %d", want, resp.StatusCode)
	}
}

func assertBody(t *testing.T, got string, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("Expected body %q, got %q", want, got)
	}
}

func TestRouteBuilder_StaticHandler(t *testing.T) {
	// Crea archivo temporal de prueba
	dir := t.TempDir()
	err := os.WriteFile(dir+"/hello.txt", []byte("world!"), 0644)
	if err != nil {
		t.Fatalf("No se pudo crear archivo: %v", err)
	}

	app := New()
	called := false
	app.Use(func(ctx *Context) {
		called = true
		ctx.Next()
	})

	// 1. Modo helper tradicional (App.Static)
	app.Static("/static/", dir)

	// 2. Pipeline builder (PathPrefix().Static)
	app.PathPrefix("/assets/").Static(dir)

	server := httptest.NewServer(app.Router)
	defer server.Close()

	// Prueba para App.Static
	resp, err := http.Get(server.URL + "/static/hello.txt")
	if err != nil {
		t.Fatalf("http.Get failed: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if string(body) != "world!" {
		t.Fatalf("[App.Static] Expected file content 'world!', got %q", string(body))
	}
	if !called {
		t.Fatalf("[App.Static] Middleware was not executed for static file")
	}

	// Reinicia el flag para la siguiente prueba
	called = false

	// Prueba para pipeline builder
	resp, err = http.Get(server.URL + "/assets/hello.txt")
	if err != nil {
		t.Fatalf("http.Get failed: %v", err)
	}
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()

	if string(body) != "world!" {
		t.Fatalf("[PathPrefix.Static] Expected file content 'world!', got %q", string(body))
	}
	if !called {
		t.Fatalf("[PathPrefix.Static] Middleware was not executed for static file")
	}
}

func TestRouter_Features_Advanced(t *testing.T) {
	// 1. Regex en rutas
	t.Run("Route param regex match", func(t *testing.T) {
		app := New()
		app.Get("/onlynum/:id([0-9]+)", func(ctx *Context) {
			ctx.Text(200, "ok:"+ctx.Vars()["id"])
		})

		server := httptest.NewServer(app.Router)
		defer server.Close()

		// Coincide con número
		resp, body := httpGet(t, server.URL+"/onlynum/42")
		assertStatus(t, resp, 200)
		assertBody(t, body, "ok:42")

		// No coincide con letras
		resp, _ = httpGet(t, server.URL+"/onlynum/abc")
		if resp.StatusCode == 200 {
			t.Errorf("Expected 404 for invalid param")
		}
	})

	// 1.2. Parámetros y grupos sin superponer
	t.Run("Route param overwritten match", func(t *testing.T) {
		app := New()
		app.Get("/api/list", func(ctx *Context) {
			ctx.Text(200, "list")
		})
		app.Get("/api/{id}", func(ctx *Context) {
			ctx.Text(200, "ok:"+ctx.Vars()["id"])
		})

		server := httptest.NewServer(app.Router)
		defer server.Close()

		// Coincide con número
		resp, body := httpGet(t, server.URL+"/api/42")
		assertStatus(t, resp, 200)
		assertBody(t, body, "ok:42")

		// Coincide con path
		resp, body = httpGet(t, server.URL+"/api/list")
		assertStatus(t, resp, 200)
		assertBody(t, body, "list")

	})

	// 2. GroupRouter anidado
	t.Run("Nested group", func(t *testing.T) {
		app := New()
		admin := app.Group("/admin")
		users := admin.Group("/users")
		users.Get("/me", func(ctx *Context) {
			ctx.Text(200, "admin-user-me")
		})

		server := httptest.NewServer(app.Router)
		defer server.Close()
		resp, body := httpGet(t, server.URL+"/admin/users/me")
		assertStatus(t, resp, 200)
		assertBody(t, body, "admin-user-me")
	})

	// 3. StaticFS (archivos embed/fs.FS)
	t.Run("StaticFS", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(dir+"/file.txt", []byte("embed!"), 0644)

		app := New()
		app.PathPrefix("/files/").StaticFS(os.DirFS(dir))

		server := httptest.NewServer(app.Router)
		defer server.Close()
		resp, body := httpGet(t, server.URL+"/files/file.txt")
		assertStatus(t, resp, 200)
		assertBody(t, body, "embed!")
	})

	// 4. NotFound y OnError Hooks
	t.Run("Custom NotFound and OnError", func(t *testing.T) {
		app := New()
		app.NotFound(func(ctx *Context) {
			ctx.Text(404, "not-found-custom")
		})
		app.OnError(func(ctx *Context, err error) {
			if err != nil {
				ctx.Text(500, "err:"+err.Error())
				return
			}
			ctx.Text(500, "err")
		})
		app.Get("/fail", func(ctx *Context) {
			panic("forced error")
		})
		server := httptest.NewServer(app.Router)
		defer server.Close()
		resp, body := httpGet(t, server.URL+"/fail")
		assertStatus(t, resp, 500)
		assertBody(t, body, "err:forced error")

		resp, body = httpGet(t, server.URL+"/nope404")
		assertStatus(t, resp, 404)
		assertBody(t, body, "not-found-custom")
	})

	// 5. Middleware en Grupo
	t.Run("Group middleware", func(t *testing.T) {
		app := New()
		called := false
		admin := app.Group("/admintest")
		admin.Use(func(ctx *Context) {
			called = true
			ctx.Next()
		})
		admin.Get("/ping", func(ctx *Context) {
			ctx.Text(200, "pong")
		})

		server := httptest.NewServer(app.Router)
		defer server.Close()
		called = false
		resp, body := httpGet(t, server.URL+"/admintest/ping")
		assertStatus(t, resp, 200)
		assertBody(t, body, "pong")
		if !called {
			t.Fatal("Group middleware was not executed")
		}
	})

	// 6. Path + .Handle y .Method
	t.Run("Path + Method + Handle", func(t *testing.T) {
		app := New()
		app.Path("/methodtest").Method("GET").Handle(func(ctx *Context) {
			ctx.Text(200, "hello-method")
		})

		server := httptest.NewServer(app.Router)
		defer server.Close()
		resp, body := httpGet(t, server.URL+"/methodtest")
		assertStatus(t, resp, 200)
		assertBody(t, body, "hello-method")
	})
}

func TestRouteBuilder_Cache(t *testing.T) {
	app := New()
	counter := 0

	app.Path("/cacheme").Cache(100 * time.Millisecond).Handle(func(ctx *Context) {
		counter++
		ctx.Text(200, "cached value")
	})

	server := httptest.NewServer(app.Router)
	defer server.Close()

	// Primera llamada: handler ejecutado
	resp, body := httpGet(t, server.URL+"/cacheme")
	assertStatus(t, resp, 200)
	assertBody(t, body, "cached value")
	if counter != 1 {
		t.Fatalf("Handler should have been called once, got %d", counter)
	}

	// Segunda llamada (dentro del periodo): no debe volver a ejecutar handler
	resp, body = httpGet(t, server.URL+"/cacheme")
	assertStatus(t, resp, 200)
	assertBody(t, body, "cached value")
	if counter != 1 {
		t.Fatalf("Handler should NOT have been called again within cache window, got %d", counter)
	}

	// Espera a que expire el cache
	time.Sleep(120 * time.Millisecond)

	// Tercera llamada: handler ejecutado de nuevo
	resp, body = httpGet(t, server.URL+"/cacheme")
	assertStatus(t, resp, 200)
	assertBody(t, body, "cached value")
	if counter != 2 {
		t.Fatalf("Handler should have been called again after cache expired, got %d", counter)
	}
}

func TestGroup_Static(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(dir+"/gfile.txt", []byte("grupo!"), 0644)
	if err != nil {
		t.Fatalf("No se pudo crear archivo: %v", err)
	}

	app := New()
	called := false
	group := app.Group("/files")
	group.Use(func(ctx *Context) {
		called = true
		ctx.Next()
	})
	group.Static("/static/", dir)

	server := httptest.NewServer(app.Router)
	defer server.Close()

	resp, err := http.Get(server.URL + "/files/static/gfile.txt")
	if err != nil {
		t.Fatalf("http.Get failed: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if string(body) != "grupo!" {
		t.Fatalf("Expected group file content 'grupo!', got %q", string(body))
	}
	if !called {
		t.Fatalf("Group middleware was not executed for group static file")
	}
}

func TestApp_StaticFS(t *testing.T) {
	// 1. Prepara un FS embebido en un directorio temporal
	dir := t.TempDir()
	filename := "recurso.txt"
	contenido := []byte("contenido estático FS")
	if err := os.WriteFile(filepath.Join(dir, filename), contenido, 0644); err != nil {
		t.Fatalf("No se pudo crear archivo de prueba: %v", err)
	}

	// 2. Configura la app y un middleware para contar ejecuciones
	app := New()
	mwEjecutado := false
	app.Use(func(ctx *Context) {
		mwEjecutado = true
		ctx.Next()
	})

	// 3. Registra el FS en la ruta /static/
	app.StaticFS("/static/", os.DirFS(dir))

	// 4. Arranca servidor de prueba
	server := httptest.NewServer(app.Router)
	defer server.Close()

	// 5. Hace la petición al recurso estático
	url := server.URL + "/static/" + filename
	resp, body := httpGet(t, url)

	// 6. Validaciones
	assertStatus(t, resp, 200)
	if body != string(contenido) {
		t.Errorf("Esperaba cuerpo %q, pero obtuvo %q", contenido, body)
	}
	if !mwEjecutado {
		t.Error("El middleware global no se ejecutó al servir el archivo estático")
	}
}

//go:embed docs/res.txt
var testEmbedFS embed.FS

func TestApp_StaticFS_Embed(t *testing.T) {
	// 1. Crea un sub-FS que sitúe la carpeta "docs" como raíz
	subFS, err := fs.Sub(testEmbedFS, "docs")
	if err != nil {
		t.Fatalf("fs.Sub falló: %v", err)
	}

	// 2. Configura la app con un middleware para verificar ejecución
	app := New()
	mwEjecutado := false
	app.Use(func(ctx *Context) {
		mwEjecutado = true
		ctx.Next()
	})

	// 3. Registra el FS embebido en la ruta /static/
	app.StaticFS("/static/", subFS)

	// 4. Monta servidor de prueba
	server := httptest.NewServer(app.Router)
	defer server.Close()

	// 5. Realiza petición al archivo embebido
	url := server.URL + "/static/" + filepath.Base("res.txt")
	resp, body := httpGet(t, url)
	// 6. Validaciones
	assertStatus(t, resp, 200)
	if body != "kame" {
		t.Errorf("Esperaba cuerpo %q, pero obtuvo %q", "kame", body)
	}
	if !mwEjecutado {
		t.Error("El middleware global no se ejecutó al servir el archivo embebido")
	}
}
