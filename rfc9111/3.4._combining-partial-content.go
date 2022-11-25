package rfc9111

// §  3.4.  Combining Partial Content
// §
// §     A response might transfer only a partial representation if the
// §     connection closed prematurely or if the request used one or more
// §     Range specifiers (Section 14.2 of [HTTP]).  After several such
// §     transfers, a cache might have received several ranges of the same
// §     representation.  A cache MAY combine these ranges into a single
// §     stored response, and reuse that response to satisfy later requests,
// §     if they all share the same strong validator and the cache complies
// §     with the client requirements in Section 15.3.7.3 of [HTTP].
// §
// §     When combining the new response with one or more stored responses, a
// §     cache MUST update the stored response header fields using the header
// §     fields provided in the new response, as per Section 3.2.
