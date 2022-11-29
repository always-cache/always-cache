package rfc9111

import "net/http"

// § 3.  Storing Responses in Caches
func MustNotStore(req *http.Request, res *http.Response) (bool, error) {
	resCacheControl := ParseCacheControl(res.Header.Values("Cache-Control"))
	// §    A cache MUST NOT store a response to a request unless:
	// §      *  the request method is understood by the cache;
	if requestMethodIsUnderstood(req.Method) &&
		// §  *  the response status code is final (see Section 15 of [HTTP]);
		responseStatusCodeIsFinal(res.StatusCode) &&
		// §  *  if the response status code is 206 or 304, or the must-understand
		// §     cache directive (see Section 5.2.2.3) is present: the cache
		// §     understands the response status code;
		statusCodeUnderstoodIfNeeded(res, resCacheControl) &&
		// §  *  the no-store cache directive is not present in the response (see
		// §     Section 5.2.2.5);
		!resCacheControl.HasDirective("no-store") &&
		// §  *  if the cache is shared: the private response directive is either
		// §     not present or allows a shared cache to store a modified response;
		// §     see Section 5.2.2.7);
		//
		// the second part of the or is a "MAY" - we don't do that
		!resCacheControl.HasDirective("private") &&
		// §  *  if the cache is shared: the Authorization header field is not
		// §     present in the request (see Section 11.6.2 of [HTTP]) or a
		// §     response directive is present that explicitly allows shared
		// §     caching (see Section 3.5); and
		req.Header.Get("Authorization") == "" || mayUseResponseForAuthenticatedRequest(resCacheControl) &&
		// §  *  the response contains at least one of the following:
		// §      -  a public response directive (see Section 5.2.2.9);
		(resCacheControl.HasDirective("public") ||
			// §  -  a private response directive, if the cache is not shared (see
			// §     Section 5.2.2.7);
			// §  -  an Expires header field (see Section 5.3);
			res.Header.Get("Expires") != "" ||
			// §  -  a max-age response directive (see Section 5.2.2.1);
			resCacheControl.HasDirective("max-age") ||
			// §  -  if the cache is shared: an s-maxage response directive (see
			// §     Section 5.2.2.10);
			resCacheControl.HasDirective("s-maxage")) {
		// the below "response contains" characteristics are not used
		//
		// §      -  a cache extension that allows it to be cached (see
		// §         Section 5.2.3); or
		// §      -  a status code that is defined as heuristically cacheable (see
		// §         Section 4.2.2).
		return false, nil
	}
	// §  Note that a cache extension can override any of the requirements
	// §  listed; see Section 5.2.3.
	//
	// not used at the moment

	return true, nil
}

// statusCodeUnderstoodIfNeeded checks if the response status code needs to be understood and is.
// It returns false if the response status code needs to be understood but isn't.
// It returns true if the response status code needs to be understood and is.
// It returns true if understanding response status code is not needed.
//
// It implements the following spec requirement (from section 3.):
// §  *  if the response status code is 206 or 304, or the must-understand
// §     cache directive (see Section 5.2.2.3) is present: the cache
// §     understands the response status code;
func statusCodeUnderstoodIfNeeded(res *http.Response, resCacheControl CacheControl) bool {
	if (res.StatusCode == 206 || res.StatusCode == 304) || resCacheControl.HasDirective("must-understand") {
		return responseStatusCodeIsUnderstood(res.StatusCode)
	}
	return true
}

// §  In this context, a cache has "understood" a request method or a
// §  response status code if it recognizes it and implements all specified
// §  caching-related behavior.

func requestMethodIsUnderstood(method string) bool {
	switch method {
	case "GET":
		return true
	case "POST":
		return true
	}
	return false
}

func responseStatusCodeIsUnderstood(statusCode int) bool {
	switch statusCode {
	case 200:
		return true
	}
	return false
}

func responseStatusCodeIsFinal(statusCode int) bool {
	return statusCode >= 200 && statusCode <= 599
}

// §  Note that, in normal operation, some caches will not store a response
// §  that has neither a cache validator nor an explicit expiration time,
// §  as such responses are not usually useful to store.  However, caches
// §  are not prohibited from storing such responses.
