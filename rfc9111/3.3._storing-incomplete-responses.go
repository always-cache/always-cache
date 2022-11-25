package rfc9111

// §  3.3.  Storing Incomplete Responses
// §
// §     If the request method is GET, the response status code is 200 (OK),
// §     and the entire response header section has been received, a cache MAY
// §     store a response that is not complete (Section 6.1 of [HTTP])
// §     provided that the stored response is recorded as being incomplete.
// §     Likewise, a 206 (Partial Content) response MAY be stored as if it
// §     were an incomplete 200 (OK) response.  However, a cache MUST NOT
// §     store incomplete or partial-content responses if it does not support
// §     the Range and Content-Range header fields or if it does not
// §     understand the range units used in those fields.
// §
// §     A cache MAY complete a stored incomplete response by making a
// §     subsequent range request (Section 14.2 of [HTTP]) and combining the
// §     successful response with the stored response, as defined in
// §     Section 3.4.  A cache MUST NOT use an incomplete response to answer
// §     requests unless the response has been made complete, or the request
// §     is partial and specifies a range wholly within the incomplete
// §     response.  A cache MUST NOT send a partial response to a client
// §     without explicitly marking it using the 206 (Partial Content) status
// §     code.
