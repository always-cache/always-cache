package main

import (
	"bufio"
	"bytes"
	"net/http"
	"strconv"
	"time"
)

type timedResponse struct {
	response *http.Response
	// The value of the clock at the time of the request that resulted in the stored response.
	// Needed for age calculation.
	requestTime time.Time
	// The value of the clock at the time the response was received.
	// Needed for age calculation.
	responseTime time.Time
}

func bytesToStoredResponse(b []byte) (timedResponse, error) {
	sRes := timedResponse{}
	res, err := bytesToResponse(b)
	if err != nil {
		return sRes, err
	}
	sRes.response = res
	resTimeInt, err := strconv.ParseInt(res.Header.Get("Response-Time"), 10, 64)
	if err != nil {
		return sRes, err
	}
	reqTimeInt, err := strconv.ParseInt(res.Header.Get("Request-Time"), 10, 64)
	if err != nil {
		return sRes, err
	}
	sRes.responseTime = time.Unix(resTimeInt, 0)
	sRes.requestTime = time.Unix(reqTimeInt, 0)
	// delete extra headers
	sRes.response.Header.Del("Response-Time")
	sRes.response.Header.Del("Request-Time")
	return sRes, nil
}

func storedResponseToBytes(sRes timedResponse) ([]byte, error) {
	sRes.response.Header.Add("Response-Time", strconv.FormatInt(sRes.responseTime.Unix(), 10))
	sRes.response.Header.Add("Request-Time", strconv.FormatInt(sRes.requestTime.Unix(), 10))
	bts, err := responseToBytes(sRes.response)
	// remove the extra headers just in case
	sRes.response.Header.Del("Response-Time")
	sRes.response.Header.Del("Request-Time")
	return bts, err
}

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
