package cache

import (
	"bytes"
	"fmt"
	"net/http"
)

// ResponseSaver is a wrapper around http.ResponseWriter that saves the response to a buffer.
// It optionally writes the response to the underlying http.ResponseWriter.
type ResponseSaver struct {
	rw           http.ResponseWriter
	b            *bytes.Buffer
	header       http.Header
	status       int
	wroteHeaders bool
}

// Implementation of http.ResponseWriter
func (t *ResponseSaver) Header() http.Header {
	return t.header
}

// Implementation of http.ResponseWriter
func (t *ResponseSaver) WriteHeader(statusCode int) {
	// remember that we wrote the headers
	t.wroteHeaders = true
	// set the status code so we can return it later
	t.status = statusCode
	// write http status, headers, and separator to buffer
	// this uses HTTP 1.1 format only
	t.b.WriteString(fmt.Sprintf("HTTP/1.1 %d %s\n", statusCode, http.StatusText(statusCode)))
	t.header.Write(t.b)
	t.b.WriteString("\n")
	// write to underlying http.ResponseWriter if not nil
	if t.rw != nil {
		// if t.rw is not nil, then t.header is the same as t.rw.Header()
		// so we don't need to write the headers again
		t.rw.WriteHeader(statusCode)
	}
}

// Implementation of http.ResponseWriter
func (t *ResponseSaver) Write(b []byte) (int, error) {
	// write headers if not already written
	if !t.wroteHeaders {
		t.WriteHeader(http.StatusOK)
	}
	// write to underlying http.ResponseWriter if not nil
	if t.rw != nil {
		t.rw.Write(b)
	}
	// write to buffer and return written bytes
	return t.b.Write(b)
}

// Response returns the recorded response as a byte slice.
func (t *ResponseSaver) Response() []byte {
	return t.b.Bytes()
}

// Updates returns a slice of the urls that should be updated as a result of the (write) request.
func (t *ResponseSaver) Updates() []string {
	return t.header.Values("cache-update")
}

// StatusCode returns the status code of the response.
func (t *ResponseSaver) StatusCode() int {
	return t.status
}

// NewResponseSaver returns a new ResponseSaver.
// If rw is not nil, the response will be written (tee'd) to it in addition to saving to buffer.
func NewResponseSaver(w http.ResponseWriter) *ResponseSaver {
	rs := &ResponseSaver{
		rw: w,
		b:  &bytes.Buffer{},
	}
	if w == nil {
		rs.header = http.Header{}
	} else {
		rs.header = w.Header()
	}
	return rs
}
