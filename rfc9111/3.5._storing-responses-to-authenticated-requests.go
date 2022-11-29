package rfc9111

// §  3.5.  Storing Responses to Authenticated Requests
// §
// §     A shared cache MUST NOT use a cached response to a request with an
// §     Authorization header field (Section 11.6.2 of [HTTP]) to satisfy any
// §     subsequent request unless the response contains a Cache-Control field
// §     with a response directive (Section 5.2.2) that allows it to be stored
// §     by a shared cache, and the cache conforms to the requirements of that
// §     directive for that response.
// §
// §     In this specification, the following response directives have such an
// §     effect: must-revalidate (Section 5.2.2.2), public (Section 5.2.2.9),
// §     and s-maxage (Section 5.2.2.10).
func mayUseResponseForAuthenticatedRequest(resCacheControl CacheControl) bool {
	// TODO must-revalidate is not yet implemented
	return resCacheControl.HasDirective("public") || resCacheControl.HasDirective("s-maxage")
}
