package rfc9111

import (
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

// §  4.  Constructing Responses from Caches

// ConstructReusableResponse returns a response that can be sent downstream,
// if the given response may be used to satisfy the given request.
// It returns nil if the response must not be reused.
// If the response may be used after validation, it returns a request for that validation.
// WARNING: The validation request does not include scheme and host!
//
// The response is only safe to use if it is not nil, AND the validation request is nil.
func ConstructReusableResponse(req *http.Request, res *http.Response, requestTime time.Time, responseTime time.Time) (*http.Response, *http.Request) {
	if mustWriteThrough(req, res) {
		return nil, nil
	}
	noReuse, validationRequest := mustNotReuse(req, res, requestTime, responseTime)
	if noReuse {
		return nil, nil
	}
	return constructResponse(res, responseTime, requestTime), validationRequest
}

// mustNotReuse checks to see whether a response MUST NOT be used to satisfy a request.
// The resposte MUST NOT be used if either the returned boolean is true OR the returned validation request is non-nil.
// However, the response may be used if the returned (non-nil) validation request is executed and returns a 304 Not Modified.
func mustNotReuse(req *http.Request, res *http.Response, requestTime time.Time, responseTime time.Time) (bool, *http.Request) {
	resCacheControl := ParseCacheControl(res.Header.Values("Cache-Control"))
	var validationRequest *http.Request
	// §     When presented with a request, a cache MUST NOT reuse a stored
	// §     response unless:
	// §
	// §     *  the presented target URI (Section 7.1 of [HTTP]) and that of the
	// §        stored response match, and
	if req.URL.String() != res.Request.URL.String() {
		log.Debug().Msg("uri-miss")
		return true, nil
	}
	// §
	// §     *  the request method associated with the stored response allows it
	// §        to be used for the presented request, and
	// §
	// §     *  request header fields nominated by the stored response (if any)
	// §        match those presented (see Section 4.1), and
	if !headerFieldsMatch(req, res.Request, res) {
		log.Debug().Msg("vary-miss")
		return true, nil
	}
	// §
	// §     *  the stored response does not contain the no-cache directive
	// §        (Section 5.2.2.4), unless it is successfully validated
	// §        (Section 4.3), and
	if resCacheControl.HasDirective("no-cache") {
		log.Debug().Msg("no-cache")
		var err error
		validationRequest, err = generateConditionalRequest(req, res)
		if err != nil {
			log.Warn().Err(err).Msg("Could not create validation request")
			return true, nil
		}
	}
	// §
	// §     *  the stored response is one of the following:
	// §
	// §        -  fresh (see Section 4.2), or
	// §
	// §        -  allowed to be served stale (see Section 4.2.4), or
	// §
	// §        -  successfully validated (see Section 4.3).
	if !isFresh(res, responseTime, requestTime) {
		log.Debug().Msg("stale")
		if validationRequest == nil {
			var err error
			validationRequest, err = generateConditionalRequest(req, res)
			if err != nil {
				log.Warn().Err(err).Msg("Could not create validation request")
				return true, nil
			}
		}
	}
	// §
	// §     Note that a cache extension can override any of the requirements
	// §     listed; see Section 5.2.3.

	return false, validationRequest
}

func constructResponse(storedResponse *http.Response, responseTime, requestTime time.Time) *http.Response {
	res := &http.Response{
		StatusCode: storedResponse.StatusCode,
		Header:     storedResponse.Header,
		Body:       storedResponse.Body,
	}

	// §     When a stored response is used to satisfy a request without
	// §     validation, a cache MUST generate an Age header field (Section 5.1),
	// §     replacing any present in the response with a value equal to the
	// §     stored response's current_age; see Section 4.2.3.
	age := current_age(storedResponse, responseTime, requestTime)
	res.Header.Set("Age", toDeltaSeconds(age))

	return res
}

// §     A cache MUST write through requests with methods that are unsafe
// §     (Section 9.2.1 of [HTTP]) to the origin server; i.e., a cache is not
// §     allowed to generate a reply to such a request before having forwarded
// §     the request and having received a corresponding response.
func mustWriteThrough(req *http.Request, res *http.Response) bool {
	if UnsafeRequest(req) {
		return true
	}
	return false
}

// §
// §     Also, note that unsafe requests might invalidate already-stored
// §     responses; see Section 4.4.
// §
// §     A cache can use a response that is stored or storable to satisfy
// §     multiple requests, provided that it is allowed to reuse that response
// §     for the requests in question.  This enables a cache to "collapse
// §     requests" -- or combine multiple incoming requests into a single
// §     forward request upon a cache miss -- thereby reducing load on the
// §     origin server and network.  Note, however, that if the cache cannot
// §     use the returned response for some or all of the collapsed requests,
// §     it will need to forward the requests in order to satisfy them,
// §     potentially introducing additional latency.
// §
// §     When more than one suitable response is stored, a cache MUST use the
// §     most recent one (as determined by the Date header field).  It can
// §     also forward the request with "Cache-Control: max-age=0" or "Cache-
// §     Control: no-cache" to disambiguate which response to use.
// §
// §     A cache without a clock (Section 5.6.7 of [HTTP]) MUST revalidate
// §     stored responses upon every use.
