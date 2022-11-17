package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestResponseToBytesBodyIntact(t *testing.T) {
	response := `HTTP/1.1 200 OK
Server: Test

This is the body`

	res, err := http.ReadResponse(bufio.NewReader(strings.NewReader(response)), nil)
	if err != nil {
		panic(err)
	}

	_, err = responseToBytes(res)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	if fmt.Sprintf("%s", body) != "This is the body" {
		t.Fatalf("Body: %s", body)
	}
}
