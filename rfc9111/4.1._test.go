package rfc9111

import (
	"net/http"
	"testing"
)

func TestVaryAcceptEncoding(t *testing.T) {
	req := &http.Request{
		Header: map[string][]string{},
	}
	res := &http.Response{
		Header: map[string][]string{
			"Vary": {"Accept-Encoding"},
		},
	}
	if !headerFieldsMatch(req, res) {
		t.Fatal("Request with no accept encoding should match response with no encoding")
	}
}
