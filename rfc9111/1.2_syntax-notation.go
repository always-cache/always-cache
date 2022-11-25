package rfc9111

import (
	"fmt"
	"strconv"
	"time"
)

// §  1.2.  Syntax Notation
// §
// §     This specification uses the Augmented Backus-Naur Form (ABNF)
// §     notation of [RFC5234], extended with the notation for case-
// §     sensitivity in strings defined in [RFC7405].
// §
// §     It also uses a list extension, defined in Section 5.6.1 of [HTTP],
// §     that allows for compact definition of comma-separated lists using a
// §     "#" operator (similar to how the "*" operator indicates repetition).
// §     Appendix A shows the collected grammar with all list operators
// §     expanded to standard ABNF notation.
// §
// §  1.2.1.  Imported Rules
// §
// §     The following core rule is included by reference, as defined in
// §     [RFC5234], Appendix B.1: DIGIT (decimal 0-9).
// §
// §     [HTTP] defines the following rules:
// §
// §       HTTP-date     = <HTTP-date, see [HTTP], Section 5.6.7>
// §       OWS           = <OWS, see [HTTP], Section 5.6.3>
// §       field-name    = <field-name, see [HTTP], Section 5.1>
// §       quoted-string = <quoted-string, see [HTTP], Section 5.6.4>
// §       token         = <token, see [HTTP], Section 5.6.2>

// TODO move to rfc9110 module
func httpDate(dateStr string) (time.Time, error) {
	return time.Parse(time.RFC1123, dateStr)
}

// §  1.2.2. Delta Seconds
// §
// §  The delta-seconds rule specifies a non-negative integer, representing time
// §  in seconds.
// §
// §      delta-seconds  = 1*DIGIT
// §
// §  A recipient parsing a delta-seconds value and converting it to binary form
// §  ought to use an arithmetic type of at least 31 bits of non-negative integer
// §  range. If a cache receives a delta-seconds value greater than the greatest
// §  integer it can represent, or if any of its subsequent calculations overflows,
// §  the cache MUST consider the value to be 2147483648 (231) or the greatest
// §  positive integer it can conveniently represent.
// §
// §        |  *Note:* The value 2147483648 is here for historical reasons,
// §        |  represents infinity (over 68 years), and does not need to be
// §        |  stored in binary form; an implementation could produce it as a
// §        |  string if any overflow occurs, even if the calculations are
// §        |  performed with an arithmetic type incapable of directly
// §        |  representing that number.  What matters here is that an
// §        |  overflow be detected and not treated as a negative value in
// §        |  later calculations.
func deltaSeconds(secondsStr string) time.Duration {
	if seconds, err := strconv.ParseUint(secondsStr, 10, 64); err == nil {
		return time.Second * time.Duration(seconds)
	}
	return 0
}

func toDeltaSeconds(duration time.Duration) string {
	return fmt.Sprintf("%d", duration.Seconds())
}
