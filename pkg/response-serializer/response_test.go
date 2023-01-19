package serializer

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
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

func TestTimedResponseSerialization(t *testing.T) {
	res := http.Response{
		Status:           "",
		StatusCode:       201,
		Proto:            "",
		ProtoMajor:       0,
		ProtoMinor:       0,
		Header:           map[string][]string{},
		Body:             nil,
		ContentLength:    0,
		TransferEncoding: []string{},
		Close:            false,
		Uncompressed:     false,
		Trailer:          map[string][]string{},
		Request:          &http.Request{},
		TLS:              &tls.ConnectionState{},
	}
	res.Header.Add("Test", "-ing")
	// create times now and now + 1s
	reqTime := time.Now()
	resTime := reqTime.Add(time.Second)
	bts, err := StoredResponseToBytes(TimedResponse{
		Response:     &res,
		ResponseTime: resTime,
		RequestTime:  reqTime,
	})
	if err != nil {
		t.Fatalf("Error creating bytes: %+v", err)
	}
	// deserialize
	res2, err := BytesToStoredResponse(bts)
	if err != nil {
		t.Fatalf("Error creating response: %+v", err)
	}
	// check header, times
	if res2.Response.Header.Get("Test") != "-ing" {
		t.Fatalf("Test header wrong %+v", res2.Response.Header)
	}
	if res2.Response.Header.Get("Response-Time") != "" || res2.Response.Header.Get("Response-Time") != "" {
		t.Fatalf("Wrong amount of headers %+v", res2.Response.Header)
	}
}
