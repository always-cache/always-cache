package rfc9111

import (
	"net/http"
	"time"
)

// §  5.1.  Age
// §
// §     The "Age" response header field conveys the sender's estimate of the
// §     time since the response was generated or successfully validated at
// §     the origin server.  Age values are calculated as specified in
// §     Section 4.2.3.
// §
// §       Age = delta-seconds
// §
// §     The Age field value is a non-negative integer, representing time in
// §     seconds (see Section 1.2.2).
// §
// §     Although it is defined as a singleton header field, a cache
// §     encountering a message with a list-based Age field value SHOULD use
// §     the first member of the field value, discarding subsequent ones.
// §
// §     If the field value (after discarding additional members, as per
// §     above) is invalid (e.g., it contains something other than a non-
// §     negative integer), a cache SHOULD ignore the field.
// §
// §     The presence of an Age header field implies that the response was not
// §     generated or validated by the origin server for this request.
// §     However, lack of an Age header field does not imply the origin was
// §     contacted.
func getAge(res *http.Response) (time.Duration, bool) {
	if secondsStr := res.Header.Get("Age"); secondsStr != "" {
		return deltaSeconds(secondsStr), true
	}
	return 0, false
}