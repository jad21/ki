go-init(){
    go mod init $1
    mkdir -p cmd/web/
	mkdir -p internal/$2
	echo "package $2" >> internal/$2/handler.go
	echo "package $2" >> internal/$2/service.go
    cat >> cmd/web/web.go <<EOF
package main

import (
	"fmt"

	"git.jdev.run/pkg/config"
	"git.jdev.run/pkg/ki"
	"$1/internal/$2"
)

func main() {
	if config.IsDevelopment() {
		fmt.Println("IsDevelopment")
	}
	app := ki.Default()
	app.Static("/static/", "./static")
	app.AddHandler($2.NewHandler())
	app.ListenAndServe()
}
EOF
	cat >> Makefile <<EOF

install:
	go mod tidy 
run:
	go run $1/cmd/web

EOF
	cat >> internal/$2/handler.go <<EOF

import (
	"git.jdev.run/pkg/ki"
)

type Handler struct { }

func NewHandler() *Handler {
	return &Handler{}
}

func (s *Handler) Expose(app *ki.App) {
	app.GET("/echo", s.handlerEcho)
}

func (s *Handler) handlerEcho(ctx *ki.Context) {
	ctx.STRING(200, "echo")
}

EOF
	cat >> .env <<EOF
STORAGE_SESSION_NAME=`uuid`
SESSION_KEY=`uuid`
ENV=development
PORT=5000
EOF

go get git.jdev.run/pkg/ki
go get git.jdev.run/pkg/config


}