package serializer

import (
	"bufio"
	"bytes"
	"net/http"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	responseTimeHeaderName = "Acache-Response-Time"
	requestTimeHeaderName  = "Acache-Request-Time"
)

type TimedResponse struct {
	Response *http.Response
	// The value of the clock at the time of the request that resulted in the stored response.
	// Needed for age calculation.
	RequestTime time.Time
	// The value of the clock at the time the response was received.
	// Needed for age calculation.
	ResponseTime time.Time
}

func BytesToStoredResponse(b []byte) (TimedResponse, error) {
	sRes := TimedResponse{}
	res, err := bytesToResponse(b)
	if err != nil {
		return sRes, err
	}
	sRes.Response = res
	resTimeInt, err := strconv.ParseInt(res.Header.Get(responseTimeHeaderName), 10, 64)
	if err != nil {
		return sRes, err
	}
	reqTimeInt, err := strconv.ParseInt(res.Header.Get(requestTimeHeaderName), 10, 64)
	if err != nil {
		return sRes, err
	}
	sRes.ResponseTime = time.Unix(resTimeInt, 0)
	sRes.RequestTime = time.Unix(reqTimeInt, 0)
	// delete extra headers
	sRes.Response.Header.Del(responseTimeHeaderName)
	sRes.Response.Header.Del(requestTimeHeaderName)
	return sRes, nil
}

var delim = []byte("\r\n\r\n----\r\n\r\n")

func StoredResponseToBytes(sRes TimedResponse) ([]byte, error) {
	res := sRes.Response
	req := sRes.Response.Request
	buf := &bytes.Buffer{}

	if req != nil {
		err := req.Write(buf)
		if err != nil {
			log.Warn().Err(err).Msg("Could not write request to bytes")
		}
	} else {
		log.Warn().Msg("Request not set")
	}
	buf.Write(delim)

	res.Header.Set(responseTimeHeaderName, strconv.FormatInt(sRes.ResponseTime.Unix(), 10))
	res.Header.Set(requestTimeHeaderName, strconv.FormatInt(sRes.RequestTime.Unix(), 10))
	bts, err := responseToBytes(sRes.Response)
	// remove the extra headers just in case
	res.Header.Del(responseTimeHeaderName)
	res.Header.Del(requestTimeHeaderName)

	buf.Write(bts)

	return buf.Bytes(), err
}

// bytesToResponse converts a byte slice to a http.Response.
func bytesToResponse(b []byte) (*http.Response, error) {
	bParts := bytes.Split(b, delim)
	reqBytes := bParts[0]
	resBytes := bParts[1]
	req, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(reqBytes)))
	if err != nil {
		log.Warn().Err(err).Bytes("bytes", reqBytes).Msg("Could not read request from stored response")
	}
	return http.ReadResponse(bufio.NewReader(bytes.NewReader(resBytes)), req)
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
