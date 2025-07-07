package main

import (
	"database/sql"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"

	"github.com/jad21/ki"
	"github.com/jad21/ki/session"
)

func main() {
	app := ki.New()

	app.Provide(func() *sql.DB {
		d := sql.DB{}
		return &d
	})

	app.Use(func(ctx *ki.Context, r *http.Request) {
		log.Printf("----------------------\n")
		log.Printf("%s - %s (%s)", r.Method, r.URL.Path, r.RemoteAddr)
		log.Printf("----------------------\n")
		// Logging middleware
		ctx.Next()
	})

	app.Static("/static/", "example/static")
	// or
	// app.Static("/static/", "static")

	app.StaticHandler("/static/-", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uPath := r.URL.Path
		io.WriteString(w, uPath)
	}))

	app.Get("/cookies", func(db *sql.DB, s session.Service, ctx *ki.Context) {
		ctx.JSON(http.StatusOK, ki.M{
			"Cookies": ctx.Request.Cookies(),
			// "db": db.Stats(),
		})
	})
	app.Get("/cookie/set/{key}/{value}", func(db *sql.DB, s session.Service, ctx *ki.Context) {
		key := ctx.Vars()["key"]
		err := s.Set(key, ctx.Vars()["value"])
		ctx.JSON(http.StatusOK, ki.M{
			"key": key,
			"err": func() string {
				if err != nil {
					return err.Error()
				}
				return ""
			}(),
		})
	})
	app.Get("/cookie/get/{key}", func(db *sql.DB, s session.Service, ctx *ki.Context) {
		key := ctx.Vars()["key"]
		v, ok := s.Get(key)
		if ok {
			ctx.JSON(http.StatusOK, ki.M{
				"key":   key,
				"value": v.(string),
				"ok":    ok,
			})
			return
		}
		ctx.JSON(http.StatusOK, ki.M{
			"key":   key,
			"value": v,
			"ok":    ok,
		})
	})
	app.Get("/cookie/del/{key}", func(db *sql.DB, s session.Service, ctx *ki.Context) {
		key := ctx.Vars()["key"]
		err := s.Delete(key)
		ctx.JSON(http.StatusOK, ki.M{
			"key": key,
			"err": err,
		})
	})

	app.Get("/flash/{msg}", func(db *sql.DB, s session.Service, ctx *ki.Context) {
		msg := ctx.Vars()["msg"]
		err := ctx.Flash(msg)
		ctx.JSON(http.StatusOK, ki.M{
			"msg": msg,
			"err": err,
		})
	})
	app.Get("/flashes", func(db *sql.DB, s session.Service, ctx *ki.Context) {
		flashes, err := ctx.Flashes()
		ctx.JSON(http.StatusOK, ki.M{
			"flashes": flashes,
			"err":     err,
		})
	})

	app.Get("/kame", func(ctx *ki.Context) {
		ctx.Text(http.StatusOK, "hameha")
	})
	app.Get("/kaio", func(ctx *ki.Context) {
		ctx.JSON(http.StatusOK, ki.M{"body": "ken"})
	})
	app.Get("/great", func(ctx *ki.Context) {
		ctx.Render(http.StatusOK, ki.M{"Name": "Goku"}, "great.html")
	})
	app.Get("/planet/{name}", func(ctx *ki.Context) {
		err := ctx.Render(http.StatusOK, nil, fmt.Sprintf("planet/%s.html", ctx.Vars()["name"]))
		if err != nil {
			ctx.Text(404, err.Error())
		}
	})

	app.Get("/logged", func(ctx *ki.Context) {
		ctx.JSON(http.StatusOK, ki.M{"logged": "1"})
	}, authMiddlewares)

	app.Group("/group", func(g ki.Router) {
		g.Get("/", func(ctx *ki.Context) {
			ctx.Text(http.StatusOK, "g index")
		})
		g.Get("/2", func(ctx *ki.Context) {
			ctx.Text(http.StatusOK, "g 2")
		})
	})

	app.Get("/csv", func(ctx *ki.Context) {
		err := ctx.CSV(200, [][]string{
			{"id", "name"},
			{"1", "Goku"},
			{"2", "Vegeta"},
		})
		if err != nil {
			ctx.Text(500, "Error generando CSV: "+err.Error())
		}
	})

	fs.WalkDir(app.Templates, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			fmt.Println(path)
		}
		return nil
	})

	app.ListenAndServe()
}

func authMiddlewares(ctx *ki.Context) {
	// read basic auth information
	_, _, ok := ctx.Request.BasicAuth()
	if ok {
		ctx.Next() // no responde nada por el método DELETE
		return
	}
	ctx.Text(http.StatusProxyAuthRequired, "no login")
}
