package turbolinks

import (
	"net/http"
	"strings"

	"github.com/gorilla/securecookie"
)

type session struct {
	sc *securecookie.SecureCookie
}

// isTLS is a helper to check if a request was performed over HTTPS.
func isTLS(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	if strings.ToLower(r.Header.Get("X-Forwarded-Proto")) == "https" {
		return true
	}
	return false
}

// set sets a session value using a securecookie.
func (s *session) set(w http.ResponseWriter, r *http.Request, key string, value interface{}) {
	values := make(map[string]interface{})

	// Atempt to decode the existing session key value pairs before creating
	// new ones.
	cookie, err := r.Cookie(TurbolinksLocation)
	if err == nil {
		s.sc.Decode(TurbolinksLocation, cookie.Value, &values)
	}

	values[key] = value

	encoded, err := s.sc.Encode(TurbolinksLocation, values)
	if err != nil {
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     TurbolinksLocation,
		Value:    encoded,
		Path:     "/",
		Secure:   isTLS(r),
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
}

// get retrieves the value with the given key from a session.
func (s *session) get(r *http.Request, key string) string {
	cookie, err := r.Cookie(TurbolinksLocation)
	if err != nil {
		return ""
	}

	values := make(map[string]interface{})
	if err := s.sc.Decode(TurbolinksLocation, cookie.Value, &values); err != nil {
		return ""
	}

	value, ok := values[key]
	if !ok {
		return ""
	}

	str, ok := value.(string)
	if !ok {
		return ""
	}

	return str
}

// delete deletes the key value pair from the session if it exists.
func (s *session) delete(w http.ResponseWriter, r *http.Request, key string) {
	cookie, err := r.Cookie(TurbolinksLocation)
	if err != nil {
		return
	}

	values := make(map[string]interface{})
	if err := s.sc.Decode(TurbolinksLocation, cookie.Value, &values); err != nil {
		return
	}

	delete(values, key)

	encoded, err := s.sc.Encode(TurbolinksLocation, values)
	if err != nil {
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     TurbolinksLocation,
		Value:    encoded,
		Path:     "/",
		Secure:   isTLS(r),
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
}
