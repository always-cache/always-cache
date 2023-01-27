package tee

import (
	"bytes"
	"fmt"
	"net/http"
	"time"
)

// ResponseSaver is a wrapper around http.ResponseWriter that saves the response to a buffer.
// It optionally writes the response to the underlying http.ResponseWriter.
type ResponseSaver struct {
	rw           http.ResponseWriter
	b            *bytes.Buffer
	header       http.Header
	status       int
	wroteHeaders bool
	statusFilter int
	CreatedAt    time.Time
}

// Implementation of http.ResponseWriter
func (t *ResponseSaver) Header() http.Header {
	return t.header
}

// Implementation of http.ResponseWriter
func (t *ResponseSaver) WriteHeader(statusCode int) {
	// do not write to underlying rw if status code equals filter
	if statusCode == t.statusFilter {
		t.rw = nil
	}
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
		copyHeader(t.rw.Header(), t.header)
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
func NewResponseSaver(w http.ResponseWriter, statusFilter ...int) *ResponseSaver {
	rs := &ResponseSaver{
		CreatedAt: time.Now(),
		rw:        w,
		b:         &bytes.Buffer{},
		header:    http.Header{},
	}
	if len(statusFilter) == 1 {
		rs.statusFilter = statusFilter[0]
	}
	return rs
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}
