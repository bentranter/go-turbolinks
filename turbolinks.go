package turbolinks

import (
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
		referrer, ok := isTurbolinks(w, r)
		if !ok {
			h.ServeHTTP(w, r)
			return
		}

		if sessionContainsTurbolinksLocation(s, w, r) {
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
		rs := stallHTTP(h, w, r)

		// Check if a redirect was performed. If there was, then we need a way
		// to tell the next request to set the special Turbolinks header that
		// will force Turbolinks to update the URL (as push state history) for
		// that redirect. We do this by setting a session on this request that
		// we can check on the next request.
		location := rs.Header().Get("Location")
		if location == "" {
			rs.sendResponse()
			return
		}

		if r.Method == http.MethodPost {
			redirectAfterPostRequest(rs, location)
		} else {
			saveTurbolinksLocation(s, w, r, referrer, location)
		}

		rs.sendResponse()
	})
}

// isTurbolinks returns the TurbolinksReferrer and true if Turbolinks is
// enabled for this request.
func isTurbolinks(w http.ResponseWriter, r *http.Request) (string, bool) {
	if referrer := r.Header.Get(TurbolinksReferrer); referrer != "" {
		return referrer, true
	}

	// For "remote" form submissions, we still need to run the request
	// through the Turbolinks handler, so we treat those as valid Turbolinks
	// requests.
	if requestedWith := r.Header.Get("X-Requested-With"); requestedWith == "XMLHttpRequest" {
		return "", true
	}

	return "", false
}

// sessionContainsTurbolinksLocation checks if the Turbolinks location value
// is present. If it is, this indicates that,
//
//  1. An HTTP redirect occured in the previous request, so we set the
//     Turbolinks-Location header in order to tell the browser to update the
//     URL to that value.
//  2. No further Turbolinks request handling needs to occur.
//
// If the session does not contain a Turbolinks location value, we return
// false which indicates that Turbolinks processing should continue.
func sessionContainsTurbolinksLocation(s *session, w http.ResponseWriter, r *http.Request) bool {
	if location := s.get(r, TurbolinksLocation); location != "" {
		w.Header().Set("Turbolinks-Location", location)
		s.delete(w, r, TurbolinksLocation)
		return true
	}
	return false
}

// redirectAfterPostRequest handles the case where a POST request was
// submitted, and the handler responded with a redirect.
//
// In order for Turbolinks to correctly handle the redirection in the browser
// in this case, we respond with a JavaScript that performs a client side
// Turbolinks redirection.
//
// TODO Handle the "replace" directive like Rails does.
func redirectAfterPostRequest(rs *responseStaller, location string) {
	rs.Header().Set("Content-Type", "text/javascript")
	rs.Header().Set("X-Content-Type-Options", "nosniff")
	rs.WriteHeader(http.StatusOK)

	// Remove the Location header since we're returning a 200
	// response along with our JavaScript.
	rs.Header().Del("Location")

	// Create the JavaScript to send to the frontend for
	// redirection after a form submission.
	//
	// Also, escape the location value so that it can't be used
	// for frontend JavaScript injection.
	js := []byte(`Turbolinks.clearCache();Turbolinks.visit("` +
		template.JSEscapeString(location) + `", {action: "advance"});`)
	rs.Header().Set("X-Xhr-Redirect", location)
	rs.Write(js)
}

// saveTurbolinksLocation tells the next request to set the special Turbolinks
// header that  will force Turbolinks to update the URL (as push state
// history) for that redirect. We do this by setting a session on this request
// that we check on the next request.
//
// The session is only saved if it's an internal redirect.
func saveTurbolinksLocation(s *session, w http.ResponseWriter, r *http.Request, referrer, location string) {
	origin, err := url.Parse(referrer)
	if err != nil {
		return
	}

	destination, err := url.Parse(location)
	if err != nil {
		return
	}

	// If the destination is a relative link, we can assume it's an
	// internal redirect and therefor set the location session value.
	if origin.Host != destination.Host && !strings.HasPrefix(destination.Path, "/") {
		return
	}

	s.set(w, r, TurbolinksLocation, location)
}
