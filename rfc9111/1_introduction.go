package rfc9111

// §  1.  Introduction
// §
// §     The Hypertext Transfer Protocol (HTTP) is a stateless application-
// §     level request/response protocol that uses extensible semantics and
// §     self-descriptive messages for flexible interaction with network-based
// §     hypertext information systems.  It is typically used for distributed
// §     information systems, where the use of response caches can improve
// §     performance.  This document defines aspects of HTTP related to
// §     caching and reusing response messages.
// §
// §     An HTTP "cache" is a local store of response messages and the
// §     subsystem that controls storage, retrieval, and deletion of messages
// §     in it.  A cache stores cacheable responses to reduce the response
// §     time and network bandwidth consumption on future equivalent requests.
// §     Any client or server MAY use a cache, though not when acting as a
// §     tunnel (Section 3.7 of [HTTP]).
// §
// §     A "shared cache" is a cache that stores responses for reuse by more
// §     than one user; shared caches are usually (but not always) deployed as
// §     a part of an intermediary.  A "private cache", in contrast, is
// §     dedicated to a single user; often, they are deployed as a component
// §     of a user agent.
// §
// §     The goal of HTTP caching is significantly improving performance by
// §     reusing a prior response message to satisfy a current request.  A
// §     cache considers a stored response "fresh", as defined in Section 4.2,
// §     if it can be reused without "validation" (checking with the origin
// §     server to see if the cached response remains valid for this request).
// §     A fresh response can therefore reduce both latency and network
// §     overhead each time the cache reuses it.  When a cached response is
// §     not fresh, it might still be reusable if validation can freshen it
// §     (Section 4.3) or if the origin is unavailable (Section 4.2.4).
// §
// §     This document obsoletes RFC 7234, with the changes being summarized
// §     in Appendix B.