package rfc9111

import "net/http"

// §  4.2.  Freshness
// §
// §     A "fresh" response is one whose age has not yet exceeded its
// §     freshness lifetime.  Conversely, a "stale" response is one where it
// §     has.
// §
// §     A response's "freshness lifetime" is the length of time between its
// §     generation by the origin server and its expiration time.  An
// §     "explicit expiration time" is the time at which the origin server
// §     intends that a stored response can no longer be used by a cache
// §     without further validation, whereas a "heuristic expiration time" is
// §     assigned by a cache when no explicit expiration time is available.
// §
// §     A response's "age" is the time that has passed since it was generated
// §     by, or successfully validated with, the origin server.
// §
// §     When a response is fresh, it can be used to satisfy subsequent
// §     requests without contacting the origin server, thereby improving
// §     efficiency.
// §
// §     The primary mechanism for determining freshness is for an origin
// §     server to provide an explicit expiration time in the future, using
// §     either the Expires header field (Section 5.3) or the max-age response
// §     directive (Section 5.2.2.1).  Generally, origin servers will assign
// §     future explicit expiration times to responses in the belief that the
// §     representation is not likely to change in a semantically significant
// §     way before the expiration time is reached.
// §
// §     If an origin server wishes to force a cache to validate every
// §     request, it can assign an explicit expiration time in the past to
// §     indicate that the response is already stale.  Compliant caches will
// §     normally validate a stale cached response before reusing it for
// §     subsequent requests (see Section 4.2.4).
// §
// §     Since origin servers do not always provide explicit expiration times,
// §     caches are also allowed to use a heuristic to determine an expiration
// §     time under certain circumstances (see Section 4.2.2).

// §     The calculation to determine if a response is fresh is:
// §
// §        response_is_fresh = (freshness_lifetime > current_age)
// §
// §     freshness_lifetime is defined in Section 4.2.1; current_age is
// §     defined in Section 4.2.3.
func isFresh(res *http.Response) bool {
	return freshness_lifetime(res) > current_age(res)
}

// §     Clients can send the max-age or min-fresh request directives
// §     (Section 5.2.1) to suggest limits on the freshness calculations for
// §     the corresponding response.  However, caches are not required to
// §     honor them.
// §
// §     When calculating freshness, to avoid common problems in date parsing:
// §
// §     *  Although all date formats are specified to be case-sensitive, a
// §        cache recipient SHOULD match the field value case-insensitively.
// §
// §     *  If a cache recipient's internal implementation of time has less
// §        resolution than the value of an HTTP-date, the recipient MUST
// §        internally represent a parsed Expires date as the nearest time
// §        equal to or earlier than the received value.
// §
// §     *  A cache recipient MUST NOT allow local time zones to influence the
// §        calculation or comparison of an age or expiration time.
// §
// §     *  A cache recipient SHOULD consider a date with a zone abbreviation
// §        other than "GMT" to be invalid for calculating expiration.
// §
// §     Note that freshness applies only to cache operation; it cannot be
// §     used to force a user agent to refresh its display or reload a
// §     resource.  See Section 6 for an explanation of the difference between
// §     caches and history mechanisms.