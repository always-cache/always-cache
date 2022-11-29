package rfc9111

import (
	"net/http"
	"time"
)

func GetExpiration(res *http.Response) time.Time {
	if ttl := freshness_lifetime(res); ttl != 0 {
		return time.Now().Add(ttl)
	}
	return time.Time{}
}

// §  4.2.1.  Calculating Freshness Lifetime
// §
func freshness_lifetime(res *http.Response) time.Duration {
	resCacheControl := ParseCacheControl(res.Header.Values("Cache-Control"))
	// §     A cache can calculate the freshness lifetime (denoted as
	// §     freshness_lifetime) of a response by evaluating the following rules
	// §     and using the first match:
	// §
	// §     *  If the cache is shared and the s-maxage response directive
	// §        (Section 5.2.2.10) is present, use its value, or
	if val, ok := resCacheControl.SMaxAge(); ok {
		return val
	}
	// §
	// §     *  If the max-age response directive (Section 5.2.2.1) is present,
	// §        use its value, or
	if val, ok := resCacheControl.MaxAge(); ok {
		return val
	}
	// §
	// §     *  If the Expires response header field (Section 5.3) is present, use
	// §        its value minus the value of the Date response header field (using
	// §        the time the message was received if it is not present, as per
	// §        Section 6.6.1 of [HTTP]), or
	if expires, err := getExpires(res); err == nil {
		// WARNING assuming date header is stored as current date if missing
		if date, err := HttpDate(res.Header.Get("Date")); err == nil {
			return expires.Sub(date)
		}
	}
	// §
	// §     *  Otherwise, no explicit expiration time is present in the response.
	// §        A heuristic freshness lifetime might be applicable; see
	// §        Section 4.2.2.
	return 0
}

// §
// §     Note that this calculation is intended to reduce clock skew by using
// §     the clock information provided by the origin server whenever
// §     possible.
// §
// §     When there is more than one value present for a given directive
// §     (e.g., two Expires header field lines or multiple Cache-Control: max-
// §     age directives), either the first occurrence should be used or the
// §     response should be considered stale.  If directives conflict (e.g.,
// §     both max-age and no-cache are present), the most restrictive
// §     directive should be honored.  Caches are encouraged to consider
// §     responses that have invalid freshness information (e.g., a max-age
// §     directive with non-integer content) to be stale.
