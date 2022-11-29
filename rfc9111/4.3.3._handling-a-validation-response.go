package rfc9111

// §  4.3.3.  Handling a Validation Response
// §
// §     Cache handling of a response to a conditional request depends upon
// §     its status code:
// §
// §     *  A 304 (Not Modified) response status code indicates that the
// §        stored response can be updated and reused; see Section 4.3.4.
// §
// §     *  A full response (i.e., one containing content) indicates that none
// §        of the stored responses nominated in the conditional request are
// §        suitable.  Instead, the cache MUST use the full response to
// §        satisfy the request.  The cache MAY store such a full response,
// §        subject to its constraints (see Section 3).
// §
// §     *  However, if a cache receives a 5xx (Server Error) response while
// §        attempting to validate a response, it can either forward this
// §        response to the requesting client or act as if the server failed
// §        to respond.  In the latter case, the cache can send a previously
// §        stored response, subject to its constraints on doing so (see
// §        Section 4.2.4), or retry the validation request.