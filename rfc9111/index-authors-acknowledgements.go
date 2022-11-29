package rfc9111

// §  Acknowledgements
// §
// §     See Appendix "Acknowledgements" of [HTTP], which applies to this
// §     document as well.
// §
// §  Index
// §
// §     A C E F G H M N O P S V W
// §
// §        A
// §
// §           age  Section 4.2
// §           Age header field  *_Section 5.1_*
// §
// §        C
// §
// §           cache  Section 1
// §           cache key  Section 2; Section 2
// §           Cache-Control header field  *_Section 5.2_*
// §           collapsed requests  Section 4
// §
// §        E
// §
// §           Expires header field  *_Section 5.3_*
// §           explicit expiration time  Section 4.2
// §
// §        F
// §
// §           Fields
// §              Age  *_Section 5.1_*; *_Section 5.1_*
// §              Cache-Control  *_Section 5.2_*
// §              Expires  *_Section 5.3_*; *_Section 5.3_*
// §              Pragma  *_Section 5.4_*; *_Section 5.4_*
// §              Warning  *_Section 5.5_*
// §           fresh  Section 4.2
// §           freshness lifetime  Section 4.2
// §
// §        G
// §
// §           Grammar
// §              Age  *_Section 5.1_*
// §              Cache-Control  *_Section 5.2_*
// §              DIGIT  *_Section 1.2_*
// §              Expires  *_Section 5.3_*
// §              cache-directive  *_Section 5.2_*
// §              delta-seconds  *_Section 1.2.2_*
// §
// §        H
// §
// §           Header Fields
// §              Age  *_Section 5.1_*; *_Section 5.1_*
// §              Cache-Control  *_Section 5.2_*
// §              Expires  *_Section 5.3_*; *_Section 5.3_*
// §              Pragma  *_Section 5.4_*; *_Section 5.4_*
// §              Warning  *_Section 5.5_*
// §           heuristic expiration time  Section 4.2
// §           heuristically cacheable  Section 4.2.2
// §
// §        M
// §
// §           max-age (cache directive)  *_Section 5.2.1.1_*;
// §              *_Section 5.2.2.1_*
// §           max-stale (cache directive)  *_Section 5.2.1.2_*
// §           min-fresh (cache directive)  *_Section 5.2.1.3_*
// §           must-revalidate (cache directive)  *_Section 5.2.2.2_*
// §           must-understand (cache directive)  *_Section 5.2.2.3_*
// §
// §        N
// §
// §           no-cache (cache directive)  *_Section 5.2.1.4_*;
// §              *_Section 5.2.2.4_*
// §           no-store (cache directive)  *_Section 5.2.1.5_*;
// §              *_Section 5.2.2.5_*
// §           no-transform (cache directive)  *_Section 5.2.1.6_*;
// §              *_Section 5.2.2.6_*
// §
// §        O
// §
// §           only-if-cached (cache directive)  *_Section 5.2.1.7_*
// §
// §        P
// §
// §           Pragma header field  *_Section 5.4_*
// §           private (cache directive)  *_Section 5.2.2.7_*
// §           private cache  Section 1
// §           proxy-revalidate (cache directive)  *_Section 5.2.2.8_*
// §           public (cache directive)  *_Section 5.2.2.9_*
// §
// §        S
// §
// §           s-maxage (cache directive)  *_Section 5.2.2.10_*
// §           shared cache  Section 1
// §           stale  Section 4.2
// §
// §        V
// §
// §           validator  Section 4.3.1
// §
// §        W
// §
// §           Warning header field  *_Section 5.5_*
// §
// §  Authors' Addresses
// §
// §     Roy T. Fielding (editor)
// §     Adobe
// §     345 Park Ave
// §     San Jose, CA 95110
// §     United States of America
// §     Email: fielding@gbiv.com
// §     URI:   https://roy.gbiv.com/
// §
// §     Mark Nottingham (editor)
// §     Fastly
// §     Prahran
// §     Australia
// §     Email: mnot@mnot.net
// §     URI:   https://www.mnot.net/
// §
// §     Julian Reschke (editor)
// §     greenbytes GmbH
// §     Hafenweg 16
// §     48155 Münster
// §     Germany
// §     Email: julian.reschke@greenbytes.de
// §     URI:   https://greenbytes.de/tech/webdav/