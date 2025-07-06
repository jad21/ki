package session

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Requiere tu Service (la nueva implementación)

func TestSession_BasicFlow(t *testing.T) {
	// Crea un request y response fake
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	ctx := context.Background()

	// Inicia la sesión
	sess := New()
	err := sess.Start(ctx, w, req)
	if err != nil {
		t.Fatalf("No pudo iniciar sesión: %v", err)
	}

	// Set/Get
	if err := sess.Set("foo", "bar"); err != nil {
		t.Fatalf("No pudo Set: %v", err)
	}
	val, ok := sess.Get("foo")
	if !ok || val != "bar" {
		t.Fatalf("Get falló, esperado 'bar', obtenido: %v", val)
	}

	// Commit y leer de cookie
	resp := w.Result()
	cookie := resp.Cookies()[0]

	// Simula próxima request con cookie presente
	req2 := httptest.NewRequest("GET", "/", nil)
	req2.AddCookie(cookie)
	w2 := httptest.NewRecorder()
	sess2 := New()
	err = sess2.Start(ctx, w2, req2)
	if err != nil {
		t.Fatalf("No pudo leer sesión desde cookie: %v", err)
	}
	val2, ok2 := sess2.Get("foo")
	if !ok2 || val2 != "bar" {
		t.Fatalf("Get después de recargar sesión falló, esperado 'bar', obtenido: %v", val2)
	}

	// Flash y Flashes
	_ = sess2.Flash("hola!")
	flashes, _ := sess2.Flashes()
	if len(flashes) != 1 || flashes[0] != "hola!" {
		t.Fatalf("Flash/Flashes falló, obtenido: %v", flashes)
	}
	flashes2, _ := sess2.Flashes()
	if len(flashes2) != 0 {
		t.Fatalf("Flashes debería estar vacío después de leer: %v", flashes2)
	}

	// Usuario
	user := &UserSession{ID: "1", Username: "test"}
	if err := sess2.SetUser(user); err != nil {
		t.Fatalf("SetUser falló: %v", err)
	}
	user2, err := sess2.User()
	if err != nil || user2.Username != "test" {
		t.Fatalf("User() falló, obtenido: %+v, err: %v", user2, err)
	}
	_ = sess2.ClearUser()
	_, err = sess2.User()
	if err == nil {
		t.Fatalf("ClearUser debería eliminar el usuario de sesión")
	}

	// Flush
	_ = sess2.Set("otra", 123)
	_ = sess2.Flush()
	if _, ok := sess2.Get("otra"); ok {
		t.Fatalf("Flush no borró los datos de sesión")
	}

	// Destroy (cookie vacía)
	_ = sess2.Destroy()
	resp2 := w2.Result()
	found := false
	for _, c := range resp2.Cookies() {
		if c.Name == SessionName && c.Value == "" && c.MaxAge < 0 {
			found = true
		}
	}
	if !found {
		t.Fatalf("Destroy no eliminó la cookie de sesión: %+v", resp2.Cookies())
	}
}

func TestSession_FirmaInvalida(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	ctx := context.Background()

	// Crea sesión y setea un dato
	sess := New()
	_ = sess.Start(ctx, w, req)
	_ = sess.Set("x", 42)
	resp := w.Result()
	cookie := resp.Cookies()[0]

	// Modifica la cookie simulando ataque (altera el valor pero deja la firma)
	parts := strings.SplitN(cookie.Value, "|", 2)
	if len(parts) != 2 {
		t.Fatalf("Cookie mal formada para prueba de integridad")
	}
	// Cambia el valor codificado
	corrupt := base64.RawURLEncoding.EncodeToString([]byte("hack")) + "|" + parts[1]

	req2 := httptest.NewRequest("GET", "/", nil)
	req2.AddCookie(&http.Cookie{Name: cookie.Name, Value: corrupt})
	w2 := httptest.NewRecorder()
	sess2 := New()
	_ = sess2.Start(ctx, w2, req2)

	val, ok := sess2.Get("x")
	if ok || val != nil {
		t.Fatalf("Integridad rota: la sesión debería estar vacía si la firma es inválida, obtenido: %v", val)
	}
}

func TestSession_ExpiracionCookie(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	ctx := context.Background()

	sess := New()
	_ = sess.Start(ctx, w, req)
	_ = sess.Set("y", "z")
	resp := w.Result()
	cookie := resp.Cookies()[0]

	// Simula cookie expirada: MaxAge negativo y valor vacío
	expired := &http.Cookie{
		Name:   cookie.Name,
		Value:  "",
		MaxAge: -1,
	}
	req2 := httptest.NewRequest("GET", "/", nil)
	req2.AddCookie(expired)
	w2 := httptest.NewRecorder()
	sess2 := New()
	_ = sess2.Start(ctx, w2, req2)
	val, ok := sess2.Get("y")
	if ok || val != nil {
		t.Fatalf("Cookie expirada: la sesión debería estar vacía, obtenido: %v", val)
	}
}
