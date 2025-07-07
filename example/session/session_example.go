package main

import (
	"os"

	"github.com/jad21/ki"
	"github.com/jad21/ki/session"
)

// Simulación de usuarios
var mockDB = map[string]string{
	"goku":   "1234",
	"vegeta": "9000",
}

func main() {
	app := ki.New(
		ki.SetTemplates(os.DirFS("templates")),
	)

	// Login (GET)
	app.Get("/login", func(s session.Service, ctx *ki.Context) {
		flashes, _ := s.Flashes()
		ctx.Render(200, ki.M{
			"Flashes": flashes,
		}, "login.html")
	})

	// Login (POST)
	app.Post("/login", func(s session.Service, ctx *ki.Context) {
		user := ctx.FormValue("user")
		pass := ctx.FormValue("pass")
		if realPass, ok := mockDB[user]; !ok || pass != realPass {
			s.Flash("Credenciales inválidas")
			ctx.Redirect("/login", 302)
			return
		}
		// Autenticado
		s.SetUser(&session.UserSession{
			ID:       user,
			Username: user,
		})
		s.Flash("¡Bienvenido, " + user + "!")
		ctx.Redirect("/me", 302)
	})

	// Ruta protegida
	app.Get("/me", func(s session.Service, ctx *ki.Context) {
		user, err := s.User()
		if err != nil {
			s.Flash("Inicia sesión primero")
			ctx.Redirect("/login", 302)
			return
		}
		// ctx.Text(200, "Hola, "+user.Username+" | <a href='/logout'>Salir</a>")
		ctx.Render(200, ki.M{
			"User": user,
		}, "me.html")

	})

	// Logout
	app.Get("/logout", func(s session.Service, ctx *ki.Context) {
		s.ClearUser()
		s.Flash("Sesión cerrada correctamente")
		ctx.Redirect("/login", 302)
	})

	app.ListenAndServe()
}
