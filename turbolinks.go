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
	// gives to their session key and cookie that serves the same purpose.
	TurbolinksLocation = "_turbolinks_location"
)

//
// Middleware wraps an HTTP handler to support the behaviour required by the
// Turbolinks JavaScript library.
//
func Middleware(h http.Handler) http.Handler {
	s := &session{securecookie.New(securecookie.GenerateRandomKey(64), nil)}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		referer := r.Header.Get(TurbolinksReferrer)
		if referer == "" {
			// Check if this a remote submission.
			requestedWith := r.Header.Get("X-Requested-With")
			if requestedWith != "XMLHttpRequest" {
				// Turbolinks isn't enabled, so don't do anything.
				h.ServeHTTP(w, r)
				return
			}
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
				//
				// TODO Handle the "replace" directive like Rails does.
				js := []byte(`Turbolinks.clearCache();Turbolinks.visit("` +
					template.JSEscapeString(location) + `", {action: "advance"});`)
				rs.Header().Set("X-Xhr-Redirect", location)
				rs.Write(js)
			}

			rs.SendResponse()
			return
		}

		// If the Turbolinks session is found, then redirect to the location
		// specified by it.
		location := s.get(r, TurbolinksLocation)
		if location != "" {
			w.Header().Set("Turbolinks-Location", location)
			s.delete(w, r, TurbolinksLocation)
			h.ServeHTTP(w, r)
			return
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

		location = rs.Header().Get("Location")
		if location != "" {
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

			// If the destination is a relative link, we can assume it's an
			// internal redirect and therefor set the location session value.
			if origin.Host != destination.Host && !strings.HasPrefix(destination.Path, "/") {
				rs.SendResponse()
				return
			}

			s.set(w, r, TurbolinksLocation, location)
		}

		rs.SendResponse()
	})
}
