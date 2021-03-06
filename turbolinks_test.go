package turbolinks_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bentranter/go-turbolinks"
)

func TestTurbolinks(t *testing.T) {
	t.Run("turbolinks redirect", func(t *testing.T) {
		handler := turbolinks.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "http://localhost:3000/redirect", http.StatusFound)
		}))

		res := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/", nil)

		// Set the header to make sure we hit the Turbolinks handler.
		req.Header.Set("Turbolinks-Referrer", "http://localhost:3000/redirect")
		handler.ServeHTTP(res, req)

		if res.Code != http.StatusFound {
			t.Fatalf("expected HTTP status %d but got %d", http.StatusFound, res.Code)
		}

		cookieReq := &http.Request{Header: http.Header{"Cookie": res.HeaderMap["Set-Cookie"]}}
		if _, err := cookieReq.Cookie(turbolinks.TurbolinksLocation); err != nil {
			t.Fatalf("expected session cookie to be set but got error %v", err.Error())
		}
	})

	t.Run("turbolinks external redirect", func(t *testing.T) {
		handler := turbolinks.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "https://example.com", http.StatusFound)
		}))

		res := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/", nil)

		// Set the header to make sure we hit the Turbolinks handler.
		req.Header.Set("Turbolinks-Referrer", "http://localhost:3000/redirect")
		handler.ServeHTTP(res, req)

		if res.Code != http.StatusFound {
			t.Fatalf("expected HTTP status %d but got %d", http.StatusFound, res.Code)
		}

		// The redirection cookie should not be set for external redirects.
		cookieReq := &http.Request{Header: http.Header{"Cookie": res.HeaderMap["Set-Cookie"]}}
		cookie, err := cookieReq.Cookie(turbolinks.TurbolinksLocation)
		if err != http.ErrNoCookie {
			t.Fatalf("expected http: named cookie not present but got %s", err.Error())
		}
		if cookie != nil {
			t.Fatalf("expected session cookie to be nil for external redirect but got %#v", cookie)
		}
	})

	t.Run("turbolinks form submission", func(t *testing.T) {
		h := turbolinks.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/", http.StatusFound)
		}))

		res := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/", nil)

		// Set the header to make sure we hit the Turbolinks handler.
		req.Header.Set("Turbolinks-Referrer", "http://localhost:3000/redirect")
		h.ServeHTTP(res, req)

		if res.Code != http.StatusOK {
			t.Fatalf("expected HTTP status %d but got %d", http.StatusOK, res.Code)
		}
		contentType := res.Header().Get("Content-Type")
		if contentType != "text/javascript" {
			t.Fatalf("expected Content-Type to be text/javascript but got %s", contentType)
		}
		expectedJS := `Turbolinks.clearCache();Turbolinks.visit("/", {action: "advance"});`
		actualJS := res.Body.String()
		if actualJS != expectedJS {
			t.Fatalf("expected response to be %s but got %s", expectedJS, actualJS)
		}
	})
}
