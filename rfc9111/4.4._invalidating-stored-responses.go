package rfc9111

// §  4.4.  Invalidating Stored Responses
// §
// §     Because unsafe request methods (Section 9.2.1 of [HTTP]) such as PUT,
// §     POST, or DELETE have the potential for changing state on the origin
// §     server, intervening caches are required to invalidate stored
// §     responses to keep their contents up to date.
// §
// §     A cache MUST invalidate the target URI (Section 7.1 of [HTTP]) when
// §     it receives a non-error status code in response to an unsafe request
// §     method (including methods whose safety is unknown).
// §
// §     A cache MAY invalidate other URIs when it receives a non-error status
// §     code in response to an unsafe request method (including methods whose
// §     safety is unknown).  In particular, the URI(s) in the Location and
// §     Content-Location response header fields (if present) are candidates
// §     for invalidation; other URIs might be discovered through mechanisms
// §     not specified in this document.  However, a cache MUST NOT trigger an
// §     invalidation under these conditions if the origin (Section 4.3.1 of
// §     [HTTP]) of the URI to be invalidated differs from that of the target
// §     URI (Section 7.1 of [HTTP]).  This helps prevent denial-of-service
// §     attacks.
// §
// §     "Invalidate" means that the cache will either remove all stored
// §     responses whose target URI matches the given URI or mark them as
// §     "invalid" and in need of a mandatory validation before they can be
// §     sent in response to a subsequent request.
// §
// §     A "non-error response" is one with a 2xx (Successful) or 3xx
// §     (Redirection) status code.
// §
// §     Note that this does not guarantee that all appropriate responses are
// §     invalidated globally; a state-changing request would only invalidate
// §     responses in the caches it travels through.