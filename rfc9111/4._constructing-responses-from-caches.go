package rfc9111

import "net/http"

// §  4.  Constructing Responses from Caches

func MustNotReuse(req *http.Request, res *http.Response) bool {
	resCacheControl := ParseCacheControl(res.Header.Values("Cache-Control"))
	// §     When presented with a request, a cache MUST NOT reuse a stored
	// §     response unless:
	// §
	// §     *  the presented target URI (Section 7.1 of [HTTP]) and that of the
	// §        stored response match, and
	// §
	// §     *  the request method associated with the stored response allows it
	// §        to be used for the presented request, and
	// TODO implement in some meaningful way - although we know it matches because of the key
	mayReuse := true &&
		// §
		// §     *  request header fields nominated by the stored response (if any)
		// §        match those presented (see Section 4.1), and
		headerFieldsMatch(req, res) &&
		// §
		// §     *  the stored response does not contain the no-cache directive
		// §        (Section 5.2.2.4), unless it is successfully validated
		// §        (Section 4.3), and
		!resCacheControl.HasDirective("no-cache") || validate(req, res) &&
		// §
		// §     *  the stored response is one of the following:
		// §
		// §        -  fresh (see Section 4.2), or
		(isFresh(res) ||
			// §
			// §    -  allowed to be served stale (see Section 4.2.4), or
			// TODO
			// staleAllowed(res) ||
			// §
			// §    -  successfully validated (see Section 4.3).
			validate(req, res))
		// §

	// TODO validate potentially called twice

	// §     Note that a cache extension can override any of the requirements
	// §     listed; see Section 5.2.3.

	return !mayReuse
}

func ConstructResponse(storedResponse *http.Response) *http.Response {
	res := &http.Response{
		StatusCode: storedResponse.StatusCode,
		Header:     storedResponse.Header,
		Body:       storedResponse.Body,
	}

	// §     When a stored response is used to satisfy a request without
	// §     validation, a cache MUST generate an Age header field (Section 5.1),
	// §     replacing any present in the response with a value equal to the
	// §     stored response's current_age; see Section 4.2.3.
	age := current_age(storedResponse)
	res.Header.Add("Age", toDeltaSeconds(age))

	return res
}

// §     A cache MUST write through requests with methods that are unsafe
// §     (Section 9.2.1 of [HTTP]) to the origin server; i.e., a cache is not
// §     allowed to generate a reply to such a request before having forwarded
// §     the request and having received a corresponding response.
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

// TODO real implementation
func validate(req *http.Request, res *http.Response) bool {
	return false
}