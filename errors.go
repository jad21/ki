package ki

// ErrorHandler es la función para capturar errores globales.
type ErrorHandler func(ctx *Context, err error)

// NotFoundHandler es la función para manejar rutas no encontradas.
type NotFoundHandler func(ctx *Context)

// Helpers para App (globales)
func (app *App) OnError(fn ErrorHandler) {
	app.onError = fn
}
func (app *App) NotFound(fn NotFoundHandler) {
	app.notFound = fn
}

// En RouteBuilder y GroupRouter ya están los setters por scope.
