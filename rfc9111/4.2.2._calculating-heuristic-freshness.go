package rfc9111

// §  4.2.2.  Calculating Heuristic Freshness
// §
// §     Since origin servers do not always provide explicit expiration times,
// §     a cache MAY assign a heuristic expiration time when an explicit time
// §     is not specified, employing algorithms that use other field values
// §     (such as the Last-Modified time) to estimate a plausible expiration
// §     time.  This specification does not provide specific algorithms, but
// §     it does impose worst-case constraints on their results.
// §
// §     A cache MUST NOT use heuristics to determine freshness when an
// §     explicit expiration time is present in the stored response.  Because
// §     of the requirements in Section 3, heuristics can only be used on
// §     responses without explicit freshness whose status codes are defined
// §     as "heuristically cacheable" (e.g., see Section 15.1 of [HTTP]) and
// §     on responses without explicit freshness that have been marked as
// §     explicitly cacheable (e.g., with a public response directive).
// §
// §     Note that in previous specifications, heuristically cacheable
// §     response status codes were called "cacheable by default".
// §
// §     If the response has a Last-Modified header field (Section 8.8.2 of
// §     [HTTP]), caches are encouraged to use a heuristic expiration value
// §     that is no more than some fraction of the interval since that time.
// §     A typical setting of this fraction might be 10%.
// §
// §        |  *Note:* A previous version of the HTTP specification
// §        |  (Section 13.9 of [RFC2616]) prohibited caches from calculating
// §        |  heuristic freshness for URIs with query components (i.e., those
// §        |  containing "?").  In practice, this has not been widely
// §        |  implemented.  Therefore, origin servers are encouraged to send
// §        |  explicit directives (e.g., Cache-Control: no-cache) if they
// §        |  wish to prevent caching.
