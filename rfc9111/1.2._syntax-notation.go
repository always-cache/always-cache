package rfc9111

import (
	"fmt"
	"strconv"
	"strings"
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
	return fmt.Sprintf("%.f", duration.Seconds())
}

// This section is from the HTTP specification (RFC9110), not the cache specification
//
// §  5.6.7.  Date/Time Formats
// §
// §     Prior to 1995, there were three different formats commonly used by
// §     servers to communicate timestamps.  For compatibility with old
// §     implementations, all three are defined here.  The preferred format is
// §     a fixed-length and single-zone subset of the date and time
// §     specification used by the Internet Message Format [RFC5322].
// §
// §       HTTP-date    = IMF-fixdate / obs-date
// §
// §     An example of the preferred format is
// §
// §       Sun, 06 Nov 1994 08:49:37 GMT    ; IMF-fixdate
// §
// §     Examples of the two obsolete formats are
// §
// §       Sunday, 06-Nov-94 08:49:37 GMT   ; obsolete RFC 850 format
// §       Sun Nov  6 08:49:37 1994         ; ANSI C's asctime() format
// §
// §     A recipient that parses a timestamp value in an HTTP field MUST
// §     accept all three HTTP-date formats.  When a sender generates a field
// §     that contains one or more timestamps defined as HTTP-date, the sender
// §     MUST generate those timestamps in the IMF-fixdate format.
//
// TODO make private
func HttpDate(dateStr string) (time.Time, error) {
	if date, err := imfDate(dateStr); err == nil {
		return date, err
	} else {
		// try to parse as obsolete date
		if date, err := obsDate(dateStr); err == nil {
			return date, err
		}
		// return original error if unsuccessful
		return date, err
	}
}

// §     An HTTP-date value represents time as an instance of Coordinated
// §     Universal Time (UTC).  The first two formats indicate UTC by the
// §     three-letter abbreviation for Greenwich Mean Time, "GMT", a
// §     predecessor of the UTC name; values in the asctime format are assumed
// §     to be in UTC.
// §
// §     A "clock" is an implementation capable of providing a reasonable
// §     approximation of the current instant in UTC.  A clock implementation
// §     ought to use NTP ([RFC5905]), or some similar protocol, to
// §     synchronize with UTC.

// §     Preferred format:
// §
// §       IMF-fixdate  = day-name "," SP date1 SP time-of-day SP GMT
// §       ; fixed length/zone/capitalization subset of the format
// §       ; see Section 3.3 of [RFC5322]
// §
// §       day-name     = %s"Mon" / %s"Tue" / %s"Wed"
// §                    / %s"Thu" / %s"Fri" / %s"Sat" / %s"Sun"
// §
// §       date1        = day SP month SP year
// §                    ; e.g., 02 Jun 1982
// §
// §       day          = 2DIGIT
// §       month        = %s"Jan" / %s"Feb" / %s"Mar" / %s"Apr"
// §                    / %s"May" / %s"Jun" / %s"Jul" / %s"Aug"
// §                    / %s"Sep" / %s"Oct" / %s"Nov" / %s"Dec"
// §       year         = 4DIGIT
// §
// §       GMT          = %s"GMT"
// §
// §       time-of-day  = hour ":" minute ":" second
// §                    ; 00:00:00 - 23:59:60 (leap second)
// §
// §       hour         = 2DIGIT
// §       minute       = 2DIGIT
// §       second       = 2DIGIT
const imfDateLayout = "Mon, 02 Jan 2006 15:04:05 MST"

func imfDate(dateStr string) (time.Time, error) {
	date, err := time.Parse(imfDateLayout, normalizeDateStr(dateStr))
	if err != nil {
		return date, err
	}
	if date.Location().String() != "GMT" {
		return date, fmt.Errorf("Date %s is not in GMT time, but %s", date, date.Location())
	}
	return date, err
}

// §     Obsolete formats:
// §
// §       obs-date     = rfc850-date / asctime-date
// §
// §       rfc850-date  = day-name-l "," SP date2 SP time-of-day SP GMT
// §       date2        = day "-" month "-" 2DIGIT
// §                    ; e.g., 02-Jun-82
// §
// §       day-name-l   = %s"Monday" / %s"Tuesday" / %s"Wednesday"
// §                    / %s"Thursday" / %s"Friday" / %s"Saturday"
// §                    / %s"Sunday"
// §
// §       asctime-date = day-name SP date3 SP time-of-day SP year
// §       date3        = month SP ( 2DIGIT / ( SP 1DIGIT ))
// §                    ; e.g., Jun  2
func obsDate(dateStr string) (time.Time, error) {
	str := normalizeDateStr(dateStr)
	if date, err := time.Parse(time.RFC850, str); err == nil {
		return date, err
	}
	return time.Parse(time.ANSIC, str)
}

// §     HTTP-date is case sensitive.  Note that Section 4.2 of [CACHING]
// §     relaxes this for cache recipients.
func normalizeDateStr(dateStr string) string {
	return strings.ToUpper(dateStr)
}

// §
// §     A sender MUST NOT generate additional whitespace in an HTTP-date
// §     beyond that specifically included as SP in the grammar.  The
// §     semantics of day-name, day, month, year, and time-of-day are the same
// §     as those defined for the Internet Message Format constructs with the
// §     corresponding name ([RFC5322], Section 3.3).
// §
// §     Recipients of a timestamp value in rfc850-date format, which uses a
// §     two-digit year, MUST interpret a timestamp that appears to be more
// §     than 50 years in the future as representing the most recent year in
// §     the past that had the same last two digits.
// §
// §     Recipients of timestamp values are encouraged to be robust in parsing
// §     timestamps unless otherwise restricted by the field definition.  For
// §     example, messages are occasionally forwarded over HTTP from a non-
// §     HTTP source that might generate any of the date and time
// §     specifications defined by the Internet Message Format.
// §
// §        |  *Note:* HTTP requirements for timestamp formats apply only to
// §        |  their usage within the protocol stream.  Implementations are
// §        |  not required to use these formats for user presentation,
// §        |  request logging, etc.