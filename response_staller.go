package turbolinks

import (
	"bytes"
	"net/http"
)

type responseStaller struct {
	w    http.ResponseWriter
	code int
	buf  *bytes.Buffer
}

// stallHTTP executes the given http.Handler, saving the response data in the
// returned responseStaller. The data can be used to inspect and modify the
// outgoing HTTP response before it's written.
func stallHTTP(h http.Handler, w http.ResponseWriter, r *http.Request) *responseStaller {
	rs := &responseStaller{
		w:    w,
		code: 0,
		buf:  &bytes.Buffer{},
	}

	h.ServeHTTP(rs, r)

	return rs
}

// Write is a wrapper that calls the underlying response writer's Write
// method, but write the response to a buffer instead.
func (rs *responseStaller) Write(b []byte) (int, error) {
	return rs.buf.Write(b)
}

// WriteHeader saves the status code, to be sent later during the SendReponse
// call.
func (rs *responseStaller) WriteHeader(code int) {
	rs.code = code
}

// Header wraps the underlying response writers Header method.
func (rs *responseStaller) Header() http.Header {
	return rs.w.Header()
}

// sendResponse writes the header to the underlying response writer, and
// writes the response.
func (rs *responseStaller) sendResponse() {
	if rs.code != 0 {
		rs.w.WriteHeader(rs.code)
	}
	rs.buf.WriteTo(rs.w)
}
