package ki

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"

	"github.com/jad21/di"
	"github.com/jad21/ki/session"
)

type KeyCtx string

var (
	KeyContextPtr  KeyCtx = "ki-context-ptr"
	KeyContextInit KeyCtx = "ki-context-init"
)

// type HandlerFunc func(ctx *Context)
// type HandlerFunc interface{}
type Context struct {
	context.Context
	Writer   http.ResponseWriter
	Request  *http.Request
	Session  session.Service
	injector di.Injector
	App      *App
	next     func() error
	params   map[string]string
}

func NewContext(ctx context.Context, app *App, w http.ResponseWriter, r *http.Request) *Context {
	var session session.Service = session.New()
	injector := di.New(app.DI)
	c := &Context{
		Context:  ctx,
		Session:  session,
		Writer:   w,
		Request:  r,
		App:      app,
		injector: injector,
	}
	injector.Map(c)

	return c
}

func UseContext(app *App, w http.ResponseWriter, r *http.Request) (*Context, error) {
	if c, ok := r.Context().Value(KeyContextPtr).(*Context); ok {
		return c, nil
	}
	c := &Context{
		Context:  r.Context(),
		Session:  session.New(),
		Writer:   w,
		Request:  r,
		App:      app,
		injector: di.New(app.DI),
	}
	c.injector.Maps(c, r, w, c.Session)

	// guardamos en el request
	*r = *r.WithContext(
		context.WithValue(
			r.Context(),
			KeyContextPtr,
			c,
		),
	)

	r.ParseForm()
	err := c.Session.Start(r.Context(), w, r)

	return c, err
}

// response JSON
func (s *Context) JSON(code int, body any) error {
	s.Writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	s.Writer.WriteHeader(code)
	return json.NewEncoder(s.Writer).Encode(body)
}

// response XML
func (s *Context) XML(code int, body any) error {
	s.Writer.Header().Set("Content-Type", "application/xml; charset=utf-8")
	s.Writer.WriteHeader(code)
	return xml.NewEncoder(s.Writer).Encode(body)
}

// response CSV (con error)
func (s *Context) CSV(code int, records [][]string) error {
	s.Writer.Header().Set("Content-Type", "text/csv; charset=utf-8")
	s.Writer.WriteHeader(code)
	writer := csv.NewWriter(s.Writer)
	defer writer.Flush()
	for _, record := range records {
		if err := writer.Write(record); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}

func (s *Context) Write(body []byte) {
	s.Writer.Write(body)
}

// response text string
func (s *Context) Text(code int, body string) {
	s.Writer.WriteHeader(code)
	s.Writer.Write([]byte(body))
}

// response template
func (s *Context) Render(code int, name string, data any) error {
	s.Writer.WriteHeader(code)
	return s.App.TemplateEngine.ExecuteTemplate(s.Writer, name, data)
}

// response standard for results like success
func (s *Context) Success(message string, args ...any) {
	var body any
	if len(args) == 1 {
		body = args[0]
	}

	s.JSON(http.StatusOK, H{
		"meta": H{
			"success": true,
			"message": message,
		},
		"body": body,
	})
}

// response standard for results like fail
func (s *Context) Fail(code int, err error, args ...any) {
	var body any
	if len(args) == 1 {
		body = args[0]
	}
	s.JSON(code, H{
		"meta": H{
			"success": false,
			"message": err.Error(),
		},
		"body": body,
	})
}

func (s *Context) Redirect(url string, code int) {
	http.Redirect(s.Writer, s.Request, url, code)
}

func (s *Context) RedirectHTML(url string, code int) {
	s.Text(http.StatusFound, fmt.Sprintf(`
		<!DOCTYPE HTML>
		<html lang="en-US">
			<head>
				<meta charset="UTF-8">
				<meta http-equiv="refresh" content="0; url=%[1]s">
				<script type="text/javascript">
					window.location.href = "%[1]s"
				</script>
				<title>Page Redirection</title>
			</head>
			<body>
				<a href="%[1]s">Redirection</a>.
			</body>
		</html>
	`, url))
}

func (c *Context) Next() error {
	if c.next != nil {
		return c.next()
	}
	return nil
}

// Obtener variable inyectada en el contexto
func (s *Context) Resolve(v ...di.Pointer) error {
	return s.injector.Resolve(v)
}

func (s *Context) Flash(message string) error {
	return s.Session.Flash(message)
}

func (s *Context) Flashes() ([]string, error) {
	return s.Session.Flashes()
}

func (s *Context) Set(key any, value any) {
	ctx := context.WithValue(s.Request.Context(), key, value)
	*s.Request = *s.Request.WithContext(ctx)
}

func (s *Context) Get(key string) any {
	return s.Request.Context().Value(key)
}

func (s *Context) Vars() map[string]string {
	if s.params == nil {
		return map[string]string{}
	}
	return s.params
}

// Setea un header de respuesta
func (s *Context) SetHeader(key, value string) {
	s.Writer.Header().Set(key, value)
}

// Obtiene un header de request (si hay varios, devuelve el primero)
func (s *Context) GetHeader(key string) string {
	return s.Request.Header.Get(key)
}

// Añade un header (no reemplaza, permite múltiples valores)
func (s *Context) AddHeader(key, value string) {
	s.Writer.Header().Add(key, value)
}

// Establece una cookie de respuesta
func (s *Context) SetCookie(cookie *http.Cookie) {
	http.SetCookie(s.Writer, cookie)
}

// Recupera una cookie por nombre del request
func (s *Context) GetCookie(name string) (*http.Cookie, error) {
	return s.Request.Cookie(name)
}

// Borra una cookie (expirándola)
func (s *Context) DeleteCookie(name string) {
	http.SetCookie(s.Writer, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
}

func (s *Context) RefreshSession() error {
	if s.Session != nil {
		return s.Session.Refresh()
	}
	return nil
}
