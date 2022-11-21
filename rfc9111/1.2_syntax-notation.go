package rfc9111

import (
	"strconv"
	"time"
)

// TODO rest of 1.2.

// §  1.2.2. Delta Seconds
// §
// §  The delta-seconds rule specifies a non-negative integer, representing time
// §  in seconds.
// §
// §    delta-seconds  = 1*DIGIT
// §
// §  A recipient parsing a delta-seconds value and converting it to binary form
// §  ought to use an arithmetic type of at least 31 bits of non-negative integer
// §  range. If a cache receives a delta-seconds value greater than the greatest
// §  integer it can represent, or if any of its subsequent calculations overflows,
// §  the cache MUST consider the value to be 2147483648 (231) or the greatest
// §  positive integer it can conveniently represent.
// §
// §  Note: The value 2147483648 is here for historical reasons, represents
// §  infinity (over 68 years), and does not need to be stored in binary form; an
// §  implementation could produce it as a string if any overflow occurs, even if
// §  the calculations are performed with an arithmetic type incapable of directly
// §  representing that number. What matters here is that an overflow be detected and
// §  not treated as a negative value in later calculations.

// deltaSeconds parses a string into "delta-seconds" as `time.Duration`.
// If the string cannot be parsed, it returns the zero value, which is 0.
func deltaSeconds(secondsStr string) time.Duration {
	if seconds, err := strconv.ParseUint(secondsStr, 10, 64); err == nil {
		return time.Second * time.Duration(seconds)
	}
	return 0
}
