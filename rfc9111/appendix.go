package rfc9111

// §  Appendix A.  Collected ABNF
// §
// §     In the collected ABNF below, list rules are expanded per
// §     Section 5.6.1 of [HTTP].
// §
// §     Age = delta-seconds
// §
// §     Cache-Control = [ cache-directive *( OWS "," OWS cache-directive ) ]
// §
// §     Expires = HTTP-date
// §
// §     HTTP-date = <HTTP-date, see [HTTP], Section 5.6.7>
// §
// §     OWS = <OWS, see [HTTP], Section 5.6.3>
// §
// §     cache-directive = token [ "=" ( token / quoted-string ) ]
// §
// §     delta-seconds = 1*DIGIT
// §
// §     field-name = <field-name, see [HTTP], Section 5.1>
// §
// §     quoted-string = <quoted-string, see [HTTP], Section 5.6.4>
// §
// §     token = <token, see [HTTP], Section 5.6.2>
// §
// §  Appendix B.  Changes from RFC 7234
// §
// §     Handling of duplicate and conflicting cache directives has been
// §     clarified.  (Section 4.2.1)
// §
// §     Cache invalidation of the URIs in the Location and Content-Location
// §     header fields is no longer required but is still allowed.
// §     (Section 4.4)
// §
// §     Cache invalidation of the URIs in the Location and Content-Location
// §     header fields is disallowed when the origin is different; previously,
// §     it was the host.  (Section 4.4)
// §
// §     Handling invalid and multiple Age header field values has been
// §     clarified.  (Section 5.1)
// §
// §     Some cache directives defined by this specification now have stronger
// §     prohibitions against generating the quoted form of their values,
// §     since this has been found to create interoperability problems.
// §     Consumers of extension cache directives are no longer required to
// §     accept both token and quoted-string forms, but they still need to
// §     parse them properly for unknown extensions.  (Section 5.2)
// §
// §     The public and private cache directives were clarified, so that they
// §     do not make responses reusable under any condition.  (Section 5.2.2)
// §
// §     The must-understand cache directive was introduced; caches are no
// §     longer required to understand the semantics of new response status
// §     codes unless it is present.  (Section 5.2.2.3)
// §
// §     The Warning response header was obsoleted.  Much of the information
// §     supported by Warning could be gleaned by examining the response, and
// §     the remaining information -- although potentially useful -- was
// §     entirely advisory.  In practice, Warning was not added by caches or
// §     intermediaries.  (Section 5.5)