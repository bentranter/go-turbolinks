package turbolinks

import (
	"bytes"
	"html/template"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/securecookie"
)

const (
	// TurbolinksReferrer is the header sent by the Turbolinks frontend on any
	// XHR requests powered by Turbolinks. We use this header to detect if the
	// current request was sent from Turbolinks.
	TurbolinksReferrer = "Turbolinks-Referrer"

	// TurbolinksLocation is the name of the session key and cookie that we
	// use to handle redirect requests correctly.
	//
	// We name it `_turbolinks_location` to be consistent with the name Rails
	// gives to their session key that serves the same purpose.
	TurbolinksLocation = "_turbolinks_location"
)

// Middleware wraps an HTTP handler to support the behaviour required by the
// Turbolinks JavaScript library.
func Middleware(h http.Handler) http.Handler {
	hashKey := securecookie.GenerateRandomKey(64)
	sc := securecookie.New(hashKey, nil)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		referer := r.Header.Get(TurbolinksReferrer)
		if referer == "" {
			// Turbolinks isn't enabled, so don't do anything.
			h.ServeHTTP(w, r)
			return
		}

		// Check for a POST request. If we do encounter a POST request,
		// execute the HTTP handler, but then tell the client to redirect
		// accordingly.
		if r.Method == http.MethodPost {
			rs := &responseStaller{
				w:    w,
				code: 0,
				buf:  &bytes.Buffer{},
			}
			h.ServeHTTP(rs, r)

			if location := rs.Header().Get("Location"); location != "" {
				rs.Header().Set("Content-Type", "text/javascript")
				rs.Header().Set("X-Content-Type-Options", "nosniff")
				rs.WriteHeader(http.StatusOK)

				// Remove the Location header since we're returning a 200
				// response.
				rs.Header().Del("Location")

				// Create the JavaScript to send to the frontend for
				// redirection after a form submission.
				//
				// Also, escape the location value so that it can't be used
				// for frontend JavaScript injection.
				js := []byte(`Turbolinks.clearCache();Turbolinks.visit("` +
					template.JSEscapeString(location) + `", {action: "advance"});`)
				rs.Write(js)
			}

			rs.SendResponse()
			return
		}

		// If the Turbolinks session is found, then redirect to the location
		// specified by it.
		location := get(sc, r, TurbolinksLocation)
		if location != "" {
			w.Header().Set("Turbolinks-Location", location)
			del(sc, w, r, TurbolinksLocation)
		}

		// Handle the request. We use a "response staller" here so that,
		//
		//	* The request isn't sent when the underlying http.ResponseWriter
		//	  calls write.
		//	* We can still write to the header after the request is handled.
		//
		// This is done in order to append the `_turbolinks_location` session
		// for the requests that need it.
		rs := &responseStaller{
			w:    w,
			code: 0,
			buf:  &bytes.Buffer{},
		}
		h.ServeHTTP(rs, r)

		// Check if a redirect was performed. Is there was, then we need a way
		// to tell the next request to set the special Turbolinks header that
		// will force Turbolinks to update the URL (as push state history) for
		// that redirect. We do this by setting a session on this request that
		// we can check on the next request.
		//
		// However, if the location of the redirect is different than the
		// referrer, it's an external redirect, so we don't do anything an
		// just serve the request normally.
		if location := rs.Header().Get("Location"); location != "" {
			origin, err := url.Parse(referer)
			if err != nil {
				rs.SendResponse()
				return
			}

			destination, err := url.Parse(location)
			if err != nil {
				rs.SendResponse()
				return
			}

			if origin.Host != destination.Host {
				rs.SendResponse()
				return
			}

			set(sc, w, r, TurbolinksLocation, location)
		}

		rs.SendResponse()
	})
}

type responseStaller struct {
	w    http.ResponseWriter
	code int
	buf  *bytes.Buffer
}

// Write is a wrapper that calls the underlying response writer's Write
// method, but write the response to a buffer instead.
func (rw *responseStaller) Write(b []byte) (int, error) {
	return rw.buf.Write(b)
}

// WriteHeader saves the status code, to be sent later during the SendReponse
// call.
func (rw *responseStaller) WriteHeader(code int) {
	rw.code = code
}

// Header wraps the underlying response writers Header method.
func (rw *responseStaller) Header() http.Header {
	return rw.w.Header()
}

// SendResponse writes the header to the underlying response writer, and
// writes the response.
func (rw *responseStaller) SendResponse() {
	if rw.code != 0 {
		rw.w.WriteHeader(rw.code)
	}
	rw.buf.WriteTo(rw.w)
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
func set(sc *securecookie.SecureCookie, w http.ResponseWriter, r *http.Request, key string, value interface{}) {
	values := make(map[string]interface{})

	// Atempt to decode the existing session key value pairs before creating
	// new ones.
	cookie, err := r.Cookie(TurbolinksLocation)
	if err == nil {
		sc.Decode(TurbolinksLocation, cookie.Value, &values)
	}

	values[key] = value

	encoded, err := sc.Encode(TurbolinksLocation, values)
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
func get(sc *securecookie.SecureCookie, r *http.Request, key string) string {
	cookie, err := r.Cookie(TurbolinksLocation)
	if err != nil {
		return ""
	}

	values := make(map[string]interface{})
	if err := sc.Decode(TurbolinksLocation, cookie.Value, &values); err != nil {
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

// del deletes the key value pair from the session if it exists.
func del(sc *securecookie.SecureCookie, w http.ResponseWriter, r *http.Request, key string) {
	cookie, err := r.Cookie(TurbolinksLocation)
	if err != nil {
		return
	}

	values := make(map[string]interface{})
	if err := sc.Decode(TurbolinksLocation, cookie.Value, &values); err != nil {
		return
	}

	delete(values, key)

	encoded, err := sc.Encode(TurbolinksLocation, values)
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
