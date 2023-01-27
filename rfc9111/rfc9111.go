package rfc9111

import (
	"fmt"
	"net/http"
	"time"

	"github.com/always-cache/always-cache/rfc9211"
)

// MustNotStore returns a boolean indicating if a particular origin response
// MUST NOT be stored in the cache.
//
// The response may be a "real" response from e.g. HttpClient.Do(), OR a Response
// struct with the following fields set:
//
// - Header
// - StatusCode
// - Request with at least .Method set
//
// All of the above are strictly needed as defined by the standard.
// An error will be returned if any of these fields are not present.
// Note that an error is also thrown if the headers are empty, since servers send headers.
func MustNotStore(originResponse *http.Response) (bool, error) {
	if originResponse.Header == nil || len(originResponse.Header) == 0 {
		return true, fmt.Errorf("Response headers empty")
	}
	if originResponse.StatusCode == 0 {
		return true, fmt.Errorf("Response status code empty")
	}
	if originResponse.Request == nil {
		return true, fmt.Errorf("Response request object empty")
	}
	if originResponse.Request.Method == "" {
		return true, fmt.Errorf("Response request method empty")
	}

	return mustNotStore(originResponse.Request, originResponse), nil
}

// MustNotReuse returns a forward reason (RFC 9211) if a response MUST NOT be used in order to
// satisfy a particular client request.
// It will also return a validation request to send to the origin IF the response MAY be used
// after successful validation.
//
// The response is most likely not a "real" response, but must nonetheless include the
// following fields:
//
// - Header
// - StatusCode
// - Request - the original request that resulted in the response, with at least:
//   - Method
//   - Header
//   - URL
//
// All of the above are strictly needed as defined by the standard.
// An error will be returned if any of these fields are not present.
// Note that an error is also thrown if the headers are empty, since servers send headers.
func MustNotReuse(
	clientRequest *http.Request, storedResponse *http.Response,
	requestTime time.Time, responseTime time.Time,
) (rfc9211.FwdReason, *http.Request, error) {
	if storedResponse.Header == nil || len(storedResponse.Header) == 0 {
		return "error", nil, fmt.Errorf("Response headers empty")
	}
	if storedResponse.StatusCode == 0 {
		return "error", nil, fmt.Errorf("Response status code empty")
	}
	if storedResponse.Request == nil {
		return "error", nil, fmt.Errorf("Response request object empty")
	}
	if storedResponse.Request.Method == "" {
		return "error", nil, fmt.Errorf("Response request method empty")
	}
	fmt.Printf("req: %+v\n", storedResponse.Request)

	if mustWriteThrough(clientRequest, storedResponse) {
		return rfc9211.FwdReasonMethod, nil, nil
	}
	fwdReason, validationRequest :=
		mustNotReuse(clientRequest, storedResponse, requestTime, responseTime)
	return fwdReason, validationRequest, nil
}

// AddAgeHeader adds the Age header to the response, as mandated by the standard.
// It directly mutates the response headers.
// It is based on the `current_age` calculation.
func AddAgeHeader(storedResponse *http.Response, responseTime, requestTime time.Time) {
	age := current_age(storedResponse, responseTime, requestTime)
	storedResponse.Header.Set("Age", toDeltaSeconds(age))
}
