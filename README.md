# Ki Web Framework

<img align="right" width="280" src="./docs/supergopher.jpeg">

Ki (気, "Energía") es un framework web moderno, rápido y minimalista para Go, inspirado en la simplicidad de la librería estándar pero con características avanzadas como inyección de dependencias, middlewares, enrutamiento con parámetros, grupos, prefijos, pipelines avanzados y más. ¡Ahora con rutas con expresiones regulares, caching, grupos anidados, pipelines encadenables, estáticos automáticos, hooks, y más!

---

## Tabla de Contenidos

* [Características](#características)
* [Instalación](#instalación)
* [Primer Ejemplo](#primer-ejemplo)
* [Enrutamiento](#enrutamiento)
* [Middlewares](#middlewares)
* [Inyección de Dependencias](#inyección-de-dependencias)
* [Grupos, Prefijos y Pipeline](#grupos-prefijos-y-pipeline)
* [Archivos Estáticos](#archivos-estáticos)
* [Respuestas](#respuestas)
* [Sesiones y Cookies](#sesiones-y-cookies)
* [Manejo de Errores y Hooks](#manejo-de-errores-y-hooks)
* [Caching](#caching)
* [Ejemplo Avanzado](#ejemplo-avanzado)
* [Logs y Proxy](#logs-y-proxy)
* [Contribuciones](#contribuciones)
* [Licencia](#licencia)

---

## Características

* **100% Go puro:** Router propio, sin dependencias externas.
* **Rutas expresivas:** Soporte para parámetros y expresiones regulares (`/user/:id([0-9]+)`).
* **Grupos, prefijos y pipeline encadenable:** Agrupa rutas, anida grupos, encadena middlewares y configuraciones.
* **Middlewares globales, por grupo o por ruta.**
* **Inyección de dependencias:** Handlers y middlewares reciben lo que necesitan automáticamente.
* **Caché por ruta:** Simple y eficiente, con `.Cache(duration)`.
* **Gestión avanzada de sesiones y cookies.**
* **Archivos estáticos directos y por FS embed.**
* **Respuestas JSON, texto, templates o personalizadas.**
* **Hooks de ciclo de vida y manejo de errores/not-found personalizados.**
* **Manejo de logs y cabeceras de proxy nativo.**

---

## Instalación

```sh
go get -u git.jdev.run/pkg/ki
```

---

## Primer Ejemplo

```go
package main

import (
	"net/http"
	"git.jdev.run/pkg/ki"
)

func main() {
	app := ki.Default()

	app.Get("/kame", func(ctx *ki.Context) {
		ctx.Text(http.StatusOK, "hame-ha")
	})
	app.Get("/kaio", func(ctx *ki.Context) {
		ctx.Json(http.StatusOK, ki.M{"body": "ken"})
	})
	app.ListenAndServe()
}
```

---

## Enrutamiento

* **Parámetros y regex en rutas:**

  ```go
  app.Get("/hello/:name", func(ctx *ki.Context) {
      ctx.Text(200, "Hola, "+ctx.Vars()["name"])
  })
  // Solo números válidos
  app.Get("/user/:id([0-9]+)", func(ctx *ki.Context) {
      ctx.Text(200, "ID numérico: "+ctx.Vars()["id"])
  })
  ```

* **Métodos soportados:** GET, POST, PUT, DELETE, OPTIONS, HEAD

* **API tipo pipeline:**

  ```go
  app.Path("/custom").Method("GET").Handle(func(ctx *ki.Context) {
      ctx.Text(200, "pipeline!")
  })
  ```

---

## Middlewares

Puedes registrar middlewares globales, por grupo, prefijo, o ruta.

```go
app.Use(func(ctx *ki.Context) {
    // Antes de cualquier handler
    ctx.Next()
})

// Por grupo:
admin := app.Group("/admin")
admin.Use(AdminAuthMiddleware)
```

---

## Inyección de Dependencias

Ki resuelve y entrega automáticamente las dependencias en los handlers o middlewares.

```go
type Service struct { /* ... */ }

app.Provide(func() *Service {
    return &Service{}
})

app.Get("/info", func(s *Service, ctx *ki.Context) {
    ctx.Json(200, ki.M{"info": s})
})
```

---

## Grupos, Prefijos y Pipeline

Organiza rutas bajo un mismo prefijo, comparte middlewares y anida grupos.

```go
// Tradicional
app.Group("/admin", func(g ki.Router) {
    g.Get("/dashboard", func(ctx *ki.Context) {
        ctx.Text(200, "Panel de Admin")
    })
})

// Pipeline encadenable:
auth := app.Group("/auth")
auth.Get("/login", handler)
auth.Use(AuthMiddleware).Get("/me", handler)

// Prefijos por pipeline
app.PathPrefix("/api/").Handle(func(ctx *ki.Context) {
    ctx.Text(200, "API Route!")
})
```

* **Anidamiento de grupos:**

  ```go
  admin := app.Group("/admin")
  users := admin.Group("/users")
  users.Get("/me", func(ctx *ki.Context) { ... })
  ```

---

## Archivos Estáticos

Sirve archivos estáticos de manera fácil y segura.

```go
// Tradicional, con strip automático del prefijo
app.Static("/static/", "./static")

// Embed o fs.FS
app.StaticFS("/assets/", os.DirFS("./assets"))

// Pipeline builder:
app.PathPrefix("/docs/").Static("./docs")
app.PathPrefix("/public/").StaticFS(os.DirFS("./public"))
```

---

## Respuestas

* **Texto:**
  `ctx.Text(200, "Hola mundo")`
* **JSON:**
  `ctx.Json(200, ki.M{"key": "value"})`
* **Renderizar templates:**
  `ctx.Render(200, ki.M{"Name": "Ki"}, "template.html")`
* **Redirecciones:**
  `ctx.Redirect("/login", 302)`

---

## Sesiones y Cookies

* **Gestión de sesiones:**
  Usa `session.Service` en tus handlers para acceder a datos de usuario, flashes y cookies.

```go
app.Get("/cookies", func(s session.Service, ctx *ki.Context) {
    cookies := s.Cookies()
    data := make(map[string]string)
    for _, c := range cookies {
        data[c.Name] = c.Value
    }
    ctx.Json(200, ki.M{"cookies": data})
})
```

---

## Manejo de Errores y Hooks

* **NotFound personalizado:**

  ```go
  app.NotFound(func(ctx *ki.Context) {
      ctx.Text(404, "Página no encontrada personalizada")
  })
  ```

* **OnError personalizado:**

  ```go
  app.OnError(func(ctx *ki.Context, err error) {
      ctx.Text(500, "Error interno: "+err.Error())
  })
  ```

* **Hooks de ciclo de vida por app o por grupo/ruta:**

  ```go
  app.BeforeEach(func(ctx *ki.Context) { ... })
  app.AfterEach(func(ctx *ki.Context) { ... })
  ```

---

## Caching

Agrega cache por ruta fácilmente:

```go
import "time"

// Guarda y reutiliza la respuesta durante 30s
app.Path("/cached").Cache(30 * time.Second).Handle(func(ctx *ki.Context) {
    ctx.Text(200, "Esta respuesta está cacheada")
})
```

---

## Ejemplo Avanzado

```go
app.Provide(func() *sql.DB { /* ... */ })

app.Get("/user/:id", func(db *sql.DB, ctx *ki.Context) {
    id := ctx.Vars()["id"]
    // consulta y responde usando db e id
    ctx.Json(200, ki.M{"user": id})
})

// Grupo avanzado y pipeline:
admin := app.Group("/admin")
admin.Use(AdminAuth)
admin.Get("/status", func(ctx *ki.Context) {
    ctx.Text(200, "admin-ok")
})

// Estáticos por FS embebido
app.StaticFS("/static/", os.DirFS("./static"))
```

---

## Logs y Proxy

Ki incluye middlewares nativos de logging y soporte de cabeceras de proxy:

```go
handler := ki.LoggingHandler(app.Router)
handler = ki.ProxyHeaders(handler)

srv := &http.Server{
    Handler: handler,
    Addr:    ":5000",
}
srv.ListenAndServe()
```

---

## Contribuciones

Las contribuciones son bienvenidas.
Abre un issue o PR en el repositorio.

---

## Licencia

MIT © JAD21, Jose A Delgado.

---
