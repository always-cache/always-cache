package rfc9111

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestConstructResponse(t *testing.T) {
	r := http.Response{
		Status:           "",
		StatusCode:       200,
		Proto:            "",
		ProtoMajor:       0,
		ProtoMinor:       0,
		Header:           map[string][]string{},
		Body:             io.NopCloser(strings.NewReader("Hello, world")),
		ContentLength:    0,
		TransferEncoding: []string{},
		Close:            false,
		Uncompressed:     false,
		Trailer:          map[string][]string{},
		Request:          &http.Request{},
		TLS:              &tls.ConnectionState{},
	}
	r.Header.Add("test", "header")
	res := ConstructResponse(&r)

	if res.StatusCode != 200 {
		t.Fatalf("Status code is %d", res.StatusCode)
	}
	if body, err := io.ReadAll(res.Body); err != nil {
		t.Fatalf("Error reading body %v", err)
	} else if fmt.Sprintf("%s", body) != "Hello, world" {
		t.Fatalf("Body is %s", body)
	}
	if res.Header.Get("Test") != "header" {
		t.Fatalf("Test header is %s", res.Header.Get("Test"))
	}
}
