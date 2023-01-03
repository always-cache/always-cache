package rfc9211

// §  3.  Examples
// §
// §     The following is an example of a minimal cache hit:
// §
// §     Cache-Status: ExampleCache; hit
// §
// §     However, a polite cache will give some more information, e.g.:
// §
// §     Cache-Status: ExampleCache; hit; ttl=376
// §
// §     A stale hit just has negative freshness, as in this example:
// §
// §     Cache-Status: ExampleCache; hit; ttl=-412
// §
// §     Whereas this is an example of a complete miss:
// §
// §     Cache-Status: ExampleCache; fwd=uri-miss
// §
// §     This is an example of a miss that successfully validated on the
// §     backend server:
// §
// §     Cache-Status: ExampleCache; fwd=stale; fwd-status=304
// §
// §     This is an example of a miss that was collapsed with another request:
// §
// §     Cache-Status: ExampleCache; fwd=uri-miss; collapsed
// §
// §     This is an example of a miss that the cache attempted to collapse,
// §     but couldn't:
// §
// §     Cache-Status: ExampleCache; fwd=uri-miss; collapsed=?0
// §
// §     The following is an example of going through two separate layers of
// §     caching, where the cache closest to the origin responded to an
// §     earlier request with a stored response, and a second cache stored
// §     that response and later reused it to satisfy the current request:
// §
// §     Cache-Status: OriginCache; hit; ttl=1100,
// §                   "CDN Company Here"; hit; ttl=545
// §
// §     The following is an example of going through a three-layer caching
// §     system, where the closest to the origin is a reverse proxy (where the
// §     response was served from cache); the next is a forward proxy
// §     interposed by the network (where the request was forwarded because
// §     there wasn't any response cached with its URI, the request was
// §     collapsed with others, and the resulting response was stored); and
// §     the closest to the user is a browser cache (where there wasn't any
// §     response cached with the request's URI):
// §
// §     Cache-Status: ReverseProxyCache; hit
// §     Cache-Status: ForwardProxyCache; fwd=uri-miss; collapsed; stored
// §     Cache-Status: BrowserCache; fwd=uri-miss
// §
// §  4.  Defining New Cache-Status Parameters
// §
// §     New Cache-Status parameters can be defined by registering them in the
// §     "HTTP Cache-Status" registry.
// §
// §     Registration requests are reviewed and approved by a designated
// §     expert, per [RFC8126], Section 4.5.  A specification document is
// §     appreciated but not required.
// §
// §     The expert(s) should consider the following factors when evaluating
// §     requests:
// §
// §     *  Community feedback
// §
// §     *  If the value is sufficiently well defined
// §
// §     *  Generic parameters are preferred over vendor-specific,
// §        application-specific, or deployment-specific values.  If a generic
// §        value cannot be agreed upon in the community, the parameter's name
// §        should be correspondingly specific (e.g., with a prefix that
// §        identifies the vendor, application, or deployment).
// §
// §     Registration requests should use the following template:
// §
// §     Name:  [a name for the Cache-Status parameter's key; see
// §        Section 3.1.2 of [STRUCTURED-FIELDS] for syntactic requirements]
// §
// §     Type:  [the Structured Type of the parameter's value; see
// §        Section 3.1.2 of [STRUCTURED-FIELDS]]
// §
// §     Description:  [a description of the parameter's semantics]
// §
// §     Reference:  [to a specification defining this parameter, if
// §        available]
// §
// §     See the registry at <https://www.iana.org/assignments/http-cache-
// §     status> for details on where to send registration requests.
// §
// §  5.  IANA Considerations
// §
// §     IANA has created the "HTTP Cache-Status" registry at
// §     <https://www.iana.org/assignments/http-cache-status> and populated it
// §     with the types defined in Section 2; see Section 4 for its associated
// §     procedures.
// §
// §     IANA has added the following entry in the "Hypertext Transfer
// §     Protocol (HTTP) Field Name Registry" defined in [HTTP], Section 18.4:
// §
// §     Field name:  Cache-Status
// §     Status:  permanent
// §     Reference:  RFC 9211
// §
// §  6.  Security Considerations
// §
// §     Attackers can use the information in Cache-Status to probe the
// §     behavior of the cache (and other components) and infer the activity
// §     of those using the cache.  The Cache-Status header field may not
// §     create these risks on its own, but it can assist attackers in
// §     exploiting them.
// §
// §     For example, knowing if a cache has stored a response can help an
// §     attacker execute a timing attack on sensitive data.
// §
// §     Additionally, exposing the cache key can help an attacker understand
// §     modifications to the cache key, which may assist cache poisoning
// §     attacks.  See [ENTANGLE] for details.
// §
// §     The underlying risks can be mitigated with a variety of techniques
// §     (e.g., using encryption and authentication and avoiding the inclusion
// §     of attacker-controlled data in the cache key), depending on their
// §     exact nature.  Note that merely obfuscating the key does not mitigate
// §     this risk.
// §
// §     To avoid assisting such attacks, the Cache-Status header field can be
// §     omitted, only sent when the client is authorized to receive it, or
// §     sent with sensitive information (e.g., the key parameter) only when
// §     the client is authorized.
// §
// §  7.  References
// §
// §  7.1.  Normative References
// §
// §     [HTTP]     Fielding, R., Ed., Nottingham, M., Ed., and J. Reschke,
// §                Ed., "HTTP Semantics", STD 97, RFC 9110,
// §                DOI 10.17487/RFC9110, June 2022,
// §                <https://www.rfc-editor.org/info/rfc9110>.
// §
// §     [HTTP-CACHING]
// §                Fielding, R., Ed., Nottingham, M., Ed., and J. Reschke,
// §                Ed., "HTTP Caching", STD 98, RFC 9111,
// §                DOI 10.17487/RFC9111, June 2022,
// §                <https://www.rfc-editor.org/info/rfc9111>.
// §
// §     [RFC2119]  Bradner, S., "Key words for use in RFCs to Indicate
// §                Requirement Levels", BCP 14, RFC 2119,
// §                DOI 10.17487/RFC2119, March 1997,
// §                <https://www.rfc-editor.org/info/rfc2119>.
// §
// §     [RFC8126]  Cotton, M., Leiba, B., and T. Narten, "Guidelines for
// §                Writing an IANA Considerations Section in RFCs", BCP 26,
// §                RFC 8126, DOI 10.17487/RFC8126, June 2017,
// §                <https://www.rfc-editor.org/info/rfc8126>.
// §
// §     [RFC8174]  Leiba, B., "Ambiguity of Uppercase vs Lowercase in RFC
// §                2119 Key Words", BCP 14, RFC 8174, DOI 10.17487/RFC8174,
// §                May 2017, <https://www.rfc-editor.org/info/rfc8174>.
// §
// §     [STRUCTURED-FIELDS]
// §                Nottingham, M. and P-H. Kamp, "Structured Field Values for
// §                HTTP", RFC 8941, DOI 10.17487/RFC8941, February 2021,
// §                <https://www.rfc-editor.org/info/rfc8941>.
// §
// §  7.2.  Informative References
// §
// §     [ENTANGLE] Kettle, J., "Web Cache Entanglement: Novel Pathways to
// §                Poisoning", September 2020,
// §                <https://portswigger.net/research/web-cache-entanglement>.
// §
// §  Author's Address
// §
// §     Mark Nottingham
// §     Fastly
// §     Prahran
// §     Australia
// §     Email: mnot@mnot.net
// §     URI:   https://www.mnot.net/
