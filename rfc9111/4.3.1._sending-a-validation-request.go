package rfc9111

import "net/http"

// §  4.3.1.  Sending a Validation Request
func generateConditionalRequest(req *http.Request, res *http.Response) (*http.Request, error) {
	// §
	// §     When generating a conditional request for validation, a cache either
	// §     starts with a request it is attempting to satisfy or -- if it is
	// §     initiating the request independently -- synthesizes a request using a
	// §     stored response by copying the method, target URI, and request header
	// §     fields identified by the Vary header field (Section 4.1).
	baseRequest := req
	if baseRequest == nil {
		baseRequest = res.Request
	}
	downstreamReq, err := http.NewRequest(baseRequest.Method, baseRequest.URL.String(), baseRequest.Body)
	if err != nil {
		return nil, err
	}
	if req != nil {
		downstreamReq.Header = req.Header
	} else {
		for _, varyField := range GetListHeader(res.Header, "Vary") {
			for _, value := range res.Request.Header.Values(varyField) {
				req.Header.Add(varyField, value)
			}
		}
	}
	// §
	// §     It then updates that request with one or more precondition header
	// §     fields.  These contain validator metadata sourced from a stored
	// §     response(s) that has the same URI.  Typically, this will include only
	// §     the stored response(s) that has the same cache key, although a cache
	// §     is allowed to validate a response that it cannot choose with the
	// §     request header fields it is sending (see Section 4.1).
	// §
	// §     The precondition header fields are then compared by recipients to
	// §     determine whether any stored response is equivalent to a current
	// §     representation of the resource.
	// §
	// §     One such validator is the timestamp given in a Last-Modified header
	// §     field (Section 8.8.2 of [HTTP]), which can be used in an If-Modified-
	// §     Since header field for response validation, or in an If-Unmodified-
	// §     Since or If-Range header field for representation selection (i.e.,
	// §     the client is referring specifically to a previously obtained
	// §     representation with that timestamp).
	if lastModified := res.Header.Get("Last-Modified"); lastModified != "" {
		downstreamReq.Header.Set("If-Modified-Since", lastModified)
	}
	// §
	// §     Another validator is the entity tag given in an ETag field
	// §     (Section 8.8.3 of [HTTP]).  One or more entity tags, indicating one
	// §     or more stored responses, can be used in an If-None-Match header
	// §     field for response validation, or in an If-Match or If-Range header
	// §     field for representation selection (i.e., the client is referring
	// §     specifically to one or more previously obtained representations with
	// §     the listed entity tags).
	if etag := res.Header.Get("ETag"); etag != "" {
		downstreamReq.Header.Set("If-None-Match", etag)
	}
	// §
	// §     When generating a conditional request for validation, a cache:
	// §
	// §     *  MUST send the relevant entity tags (using If-Match, If-None-Match,
	// §        or If-Range) if the entity tags were provided in the stored
	// §        response(s) being validated.
	// §
	// §     *  SHOULD send the Last-Modified value (using If-Modified-Since) if
	// §        the request is not for a subrange, a single stored response is
	// §        being validated, and that response contains a Last-Modified value.
	// §
	// §     *  MAY send the Last-Modified value (using If-Unmodified-Since or If-
	// §        Range) if the request is for a subrange, a single stored response
	// §        is being validated, and that response contains only a Last-
	// §        Modified value (not an entity tag).
	// §
	// §     In most cases, both validators are generated in cache validation
	// §     requests, even when entity tags are clearly superior, to allow old
	// §     intermediaries that do not understand entity tag preconditions to
	// §     respond appropriately.
	return downstreamReq, nil
}
