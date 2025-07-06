package ki

// HookFunc es la función a ejecutar en los hooks de ciclo de vida.
type HookFunc func(ctx *Context)

// Helpers para registrar hooks globales en la App

func (app *App) BeforeEach(fn HookFunc) {
	app.before = fn
}
func (app *App) AfterEach(fn HookFunc) {
	app.after = fn
}

// Helpers para RouteBuilder y GroupRouter (internos, ya implementados en esos structs)

//
// func (rb *RouteBuilder) BeforeEach(fn HookFunc) *RouteBuilder {
//     rb.beforeEach = fn
//     return rb
// }
// func (rb *RouteBuilder) AfterEach(fn HookFunc) *RouteBuilder {
//     rb.afterEach = fn
//     return rb
// }
