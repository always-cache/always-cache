package rfc9111

// §  8.  IANA Considerations
// §
// §     The change controller for the following registrations is: "IETF
// §     (iesg@ietf.org) - Internet Engineering Task Force".
// §
// §  8.1.  Field Name Registration
// §
// §     IANA has updated the "Hypertext Transfer Protocol (HTTP) Field Name
// §     Registry" at <https://www.iana.org/assignments/http-fields>, as
// §     described in Section 18.4 of [HTTP], with the field names listed in
// §     the table below:
// §
// §     +===============+============+=========+==========+
// §     | Field Name    | Status     | Section | Comments |
// §     +===============+============+=========+==========+
// §     | Age           | permanent  | 5.1     |          |
// §     +---------------+------------+---------+----------+
// §     | Cache-Control | permanent  | 5.2     |          |
// §     +---------------+------------+---------+----------+
// §     | Expires       | permanent  | 5.3     |          |
// §     +---------------+------------+---------+----------+
// §     | Pragma        | deprecated | 5.4     |          |
// §     +---------------+------------+---------+----------+
// §     | Warning       | obsoleted  | 5.5     |          |
// §     +---------------+------------+---------+----------+
// §
// §                           Table 1
// §
// §  8.2.  Cache Directive Registration
// §
// §     IANA has updated the "Hypertext Transfer Protocol (HTTP) Cache
// §     Directive Registry" at <https://www.iana.org/assignments/http-cache-
// §     directives> with the registration procedure per Section 5.2.4 and the
// §     cache directive names summarized in the table below.
// §
// §     +==================+==================+
// §     | Cache Directive  | Section          |
// §     +==================+==================+
// §     | max-age          | 5.2.1.1, 5.2.2.1 |
// §     +------------------+------------------+
// §     | max-stale        | 5.2.1.2          |
// §     +------------------+------------------+
// §     | min-fresh        | 5.2.1.3          |
// §     +------------------+------------------+
// §     | must-revalidate  | 5.2.2.2          |
// §     +------------------+------------------+
// §     | must-understand  | 5.2.2.3          |
// §     +------------------+------------------+
// §     | no-cache         | 5.2.1.4, 5.2.2.4 |
// §     +------------------+------------------+
// §     | no-store         | 5.2.1.5, 5.2.2.5 |
// §     +------------------+------------------+
// §     | no-transform     | 5.2.1.6, 5.2.2.6 |
// §     +------------------+------------------+
// §     | only-if-cached   | 5.2.1.7          |
// §     +------------------+------------------+
// §     | private          | 5.2.2.7          |
// §     +------------------+------------------+
// §     | proxy-revalidate | 5.2.2.8          |
// §     +------------------+------------------+
// §     | public           | 5.2.2.9          |
// §     +------------------+------------------+
// §     | s-maxage         | 5.2.2.10         |
// §     +------------------+------------------+
// §
// §                     Table 2
// §
// §  8.3.  Warn Code Registry
// §
// §     IANA has added the following note to the "Hypertext Transfer Protocol
// §     (HTTP) Warn Codes" registry at <https://www.iana.org/assignments/
// §     http-warn-codes> stating that "Warning" has been obsoleted:
// §
// §     |  The Warning header field (and the warn codes that it uses) has
// §     |  been obsoleted for HTTP per [RFC9111].