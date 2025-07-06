package session

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/gob"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type SessionData map[string]interface{}

type UserSession struct {
	ID           string
	Username     string
	AccessToken  string
	RefreshToken string
	Data         []byte
}

var (
	SessionKey           = []byte(getEnv("SESSION_KEY", "supersecretkey"))
	SessionName          = getEnv("STORAGE_SESSION_NAME", "ki_session")
	SessionMaxAgeMin     = mustInt(getEnv("SESSION_MAX_AGE_MIN", "60")) // minutos
	SessionMaxAge        = SessionMaxAgeMin * 60                        // segundos
	ErrSessionNotStarted = errors.New("session not started")
	ErrNotLogin          = errors.New("not login")
)

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
func mustInt(s string) int {
	i, _ := strconv.Atoi(s)
	if i == 0 {
		return 60
	}
	return i
}

func init() {
	gob.Register(UserSession{})
}

type Service interface {
	Start(ctx context.Context, w http.ResponseWriter, r *http.Request) error
	User() (*UserSession, error)
	SetUser(userSession *UserSession) error
	ClearUser() error
	Set(key string, value interface{}) error
	Delete(key string) error
	Get(key string) (interface{}, bool)
	Flash(message string) error
	Flashes() ([]string, error)
	Commit() error
	Destroy() error
	Refresh() error
	Flush() error
	Clone(ctx context.Context, w http.ResponseWriter, r *http.Request) *service
}

type service struct {
	ctx     context.Context
	session SessionData
	changed bool
	r       *http.Request
	w       http.ResponseWriter
}

func New() Service {
	return &service{
		ctx:     context.Background(),
		session: make(SessionData),
	}
}

func (s *service) Start(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	s.ctx = ctx
	s.r, s.w = r, w
	cookie, err := r.Cookie(SessionName)
	if err == nil && cookie.Value != "" {
		parts := strings.SplitN(cookie.Value, "|", 2)
		if len(parts) == 2 {
			dataB64, sig := parts[0], parts[1]
			data, _ := base64.RawURLEncoding.DecodeString(dataB64)
			if verify(data, sig, SessionKey) {
				if m, err := deserializeGob(data); err == nil {
					s.session = m
				}
			}
		}
	}
	if s.session == nil {
		s.session = make(SessionData)
	}
	return nil
}

func (s *service) Commit() error {
	data, err := serializeGob(s.session)
	if err != nil {
		return err
	}
	dataB64 := base64.RawURLEncoding.EncodeToString(data)
	sig := sign(data, SessionKey)
	cookieVal := dataB64 + "|" + sig
	cookie := &http.Cookie{
		Name:     SessionName,
		Value:    cookieVal,
		Path:     "/",
		MaxAge:   SessionMaxAge,
		HttpOnly: true,
		Secure:   true, // Ajusta si quieres http
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(s.w, cookie)
	return nil
}

func (s *service) Set(key string, value interface{}) error {
	s.session[key] = value
	s.changed = true
	return s.Commit()
}

func (s *service) Get(key string) (interface{}, bool) {
	val, ok := s.session[key]
	return val, ok
}

func (s *service) Delete(key string) error {
	delete(s.session, key)
	s.changed = true
	return s.Commit()
}

func (s *service) Flush() error {
	s.session = make(SessionData)
	s.changed = true
	return s.Commit()
}
func (s *service) Destroy() error {
	cookie := &http.Cookie{
		Name:     SessionName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(s.w, cookie)
	return nil
}

// Mensajes flash (one-shot)
func (s *service) Flash(message string) error {
	flashes, _ := s.session["__flashes"].([]string)
	flashes = append(flashes, message)
	s.session["__flashes"] = flashes
	s.changed = true
	return s.Commit()
}

func (s *service) Flashes() ([]string, error) {
	flashes, _ := s.session["__flashes"].([]string)
	delete(s.session, "__flashes") // One-shot
	s.changed = true
	s.Commit()
	return flashes, nil
}

// UserSession helpers
func (s *service) SetUser(user *UserSession) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(user); err != nil {
		return err
	}
	s.session["__user"] = buf.Bytes()
	s.changed = true
	return s.Commit()
}

func (s *service) User() (*UserSession, error) {
	raw, ok := s.session["__user"]
	if !ok {
		return nil, ErrNotLogin
	}
	b, ok := raw.([]byte)
	if !ok {
		return nil, ErrNotLogin
	}
	var user UserSession
	if err := gob.NewDecoder(bytes.NewReader(b)).Decode(&user); err != nil {
		// Deserialización fallida, limpiar sesión de usuario (logout)
		s.ClearUser() // Borra el usuario de la sesión y commitea
		return nil, ErrNotLogin
	}
	return &user, nil
}

func (s *service) ClearUser() error {
	delete(s.session, "__user")
	s.changed = true
	return s.Commit()
}

func (s *service) Clone(ctx context.Context, w http.ResponseWriter, r *http.Request) *service {
	clone := &service{
		ctx:     ctx,
		session: make(SessionData, len(s.session)),
		r:       r,
		w:       w,
	}
	for k, v := range s.session {
		clone.session[k] = v
	}
	return clone
}

func (s *service) Refresh() error {
	// Simplemente vuelve a hacer Commit para renovar la cookie
	return s.Commit()
}

// --- Helpers de serialización y seguridad ---

func serializeGob(m SessionData) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(m)
	return buf.Bytes(), err
}
func deserializeGob(data []byte) (SessionData, error) {
	var m SessionData
	dec := gob.NewDecoder(bytes.NewReader(data))
	err := dec.Decode(&m)
	return m, err
}

// Firma y verificación HMAC
func sign(data, key []byte) string {
	mac := hmac.New(sha256.New, key)
	mac.Write(data)
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
func verify(data []byte, sig string, key []byte) bool {
	expected := sign(data, key)
	return hmac.Equal([]byte(expected), []byte(sig))
}
