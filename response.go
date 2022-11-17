package main

import (
	"bufio"
	"bytes"
	"net/http"
)

// bytesToResponse converts a byte slice to a http.Response.
func bytesToResponse(b []byte) (*http.Response, error) {
	return http.ReadResponse(bufio.NewReader(bytes.NewReader(b)), nil)
}

// responseToBytes converts a response to a byte slice.
// It returns the HTTP/1.1 representation of the response
func responseToBytes(res *http.Response) ([]byte, error) {
	// write response to buffer
	buf := &bytes.Buffer{}
	res.Write(buf)
	// set response body back
	bts := buf.Bytes()
	clonedRes, err := http.ReadResponse(bufio.NewReader(bytes.NewReader(bts)), res.Request)
	if err != nil {
		panic(err)
	}
	res.Body = clonedRes.Body
	// return buffer bytes
	return bts, nil
}
