package rfc9111

// §  9.  References
// §
// §  9.1.  Normative References
// §
// §     [HTTP]     Fielding, R., Ed., Nottingham, M., Ed., and J. Reschke,
// §                Ed., "HTTP Semantics", STD 97, RFC 9110,
// §                DOI 10.17487/RFC9110, June 2022,
// §                <https://www.rfc-editor.org/info/rfc9110>.
// §
// §     [RFC2119]  Bradner, S., "Key words for use in RFCs to Indicate
// §                Requirement Levels", BCP 14, RFC 2119,
// §                DOI 10.17487/RFC2119, March 1997,
// §                <https://www.rfc-editor.org/info/rfc2119>.
// §
// §     [RFC5234]  Crocker, D., Ed. and P. Overell, "Augmented BNF for Syntax
// §                Specifications: ABNF", STD 68, RFC 5234,
// §                DOI 10.17487/RFC5234, January 2008,
// §                <https://www.rfc-editor.org/info/rfc5234>.
// §
// §     [RFC7405]  Kyzivat, P., "Case-Sensitive String Support in ABNF",
// §                RFC 7405, DOI 10.17487/RFC7405, December 2014,
// §                <https://www.rfc-editor.org/info/rfc7405>.
// §
// §     [RFC8174]  Leiba, B., "Ambiguity of Uppercase vs Lowercase in RFC
// §                2119 Key Words", BCP 14, RFC 8174, DOI 10.17487/RFC8174,
// §                May 2017, <https://www.rfc-editor.org/info/rfc8174>.
// §
// §  9.2.  Informative References
// §
// §     [COOKIE]   Barth, A., "HTTP State Management Mechanism", RFC 6265,
// §                DOI 10.17487/RFC6265, April 2011,
// §                <https://www.rfc-editor.org/info/rfc6265>.
// §
// §     [HTTP/1.1] Fielding, R., Ed., Nottingham, M., Ed., and J. Reschke,
// §                Ed., "HTTP/1.1", STD 99, RFC 9112, DOI 10.17487/RFC9112,
// §                June 2022, <https://www.rfc-editor.org/info/rfc9112>.
// §
// §     [RFC2616]  Fielding, R., Gettys, J., Mogul, J., Frystyk, H.,
// §                Masinter, L., Leach, P., and T. Berners-Lee, "Hypertext
// §                Transfer Protocol -- HTTP/1.1", RFC 2616,
// §                DOI 10.17487/RFC2616, June 1999,
// §                <https://www.rfc-editor.org/info/rfc2616>.
// §
// §     [RFC5861]  Nottingham, M., "HTTP Cache-Control Extensions for Stale
// §                Content", RFC 5861, DOI 10.17487/RFC5861, May 2010,
// §                <https://www.rfc-editor.org/info/rfc5861>.
// §
// §     [RFC7234]  Fielding, R., Ed., Nottingham, M., Ed., and J. Reschke,
// §                Ed., "Hypertext Transfer Protocol (HTTP/1.1): Caching",
// §                RFC 7234, DOI 10.17487/RFC7234, June 2014,
// §                <https://www.rfc-editor.org/info/rfc7234>.
// §
// §     [RFC8126]  Cotton, M., Leiba, B., and T. Narten, "Guidelines for
// §                Writing an IANA Considerations Section in RFCs", BCP 26,
// §                RFC 8126, DOI 10.17487/RFC8126, June 2017,
// §                <https://www.rfc-editor.org/info/rfc8126>.