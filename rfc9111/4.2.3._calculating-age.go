package rfc9111

import (
	"net/http"
	"time"
)

// §  4.2.3.  Calculating Age
// §
// §     The Age header field is used to convey an estimated age of the
// §     response message when obtained from a cache.  The Age field value is
// §     the cache's estimate of the number of seconds since the origin server
// §     generated or validated the response.  The Age value is therefore the
// §     sum of the time that the response has been resident in each of the
// §     caches along the path from the origin server, plus the time it has
// §     been in transit along network paths.
// §
// §     Age calculation uses the following data:
// §
// §     "age_value"
// §        The term "age_value" denotes the value of the Age header field
// §        (Section 5.1), in a form appropriate for arithmetic operation; or
// §        0, if not available.
func age_value(res *http.Response) time.Duration {
	if age, present := getAge(res); present {
		return age
	}
	return 0
}

// §
// §     "date_value"
// §        The term "date_value" denotes the value of the Date header field,
// §        in a form appropriate for arithmetic operations.  See
// §        Section 6.6.1 of [HTTP] for the definition of the Date header
// §        field and for requirements regarding responses without it.
func date_value(res *http.Response) time.Time {
	if dateHeader := res.Header.Get("Date"); dateHeader != "" {
		if date, err := httpDate(dateHeader); err == nil {
			return date
		}
	}
	// we should never get here (stored responses shauld have the field added)
	return time.Time{}
}

// §
// §     "now"
// §        The term "now" means the current value of this implementation's
// §        clock (Section 5.6.7 of [HTTP]).
func now() time.Time {
	return time.Now()
}

// §
// §     "request_time"
// §        The value of the clock at the time of the request that resulted in
// §        the stored response.
func request_time(res *http.Response) time.Time {
	// WARNING assuming no network latency
	return date_value(res)
}

// §
// §     "response_time"
// §        The value of the clock at the time the response was received.
func response_time(res *http.Response) time.Time {
	// WARNING assuming no network latency
	return date_value(res)
}

// §
// §     A response's age can be calculated in two entirely independent ways:
// §
// §     1.  the "apparent_age": response_time minus date_value, if the
// §         implementation's clock is reasonably well synchronized to the
// §         origin server's clock.  If the result is negative, the result is
// §         replaced by zero.
// §
// §     2.  the "corrected_age_value", if all of the caches along the
// §         response path implement HTTP/1.1 or greater.  A cache MUST
// §         interpret this value relative to the time the request was
// §         initiated, not the time that the response was received.
// §
// §       apparent_age = max(0, response_time - date_value);
func apparent_age(res *http.Response) time.Duration {
	return durationMax(0, response_time(res).Sub(date_value(res)))
}

// §       response_delay = response_time - request_time;

func response_delay(res *http.Response) time.Duration {
	return 0
}

// §       corrected_age_value = age_value + response_delay;
func corrected_age_value(res *http.Response) time.Duration {
	return age_value(res) + response_delay(res)
}

// §
// §     The corrected_age_value MAY be used as the corrected_initial_age.  In
// §     circumstances where very old cache implementations that might not
// §     correctly insert Age are present, corrected_initial_age can be
// §     calculated more conservatively as
// §
// §       corrected_initial_age = max(apparent_age, corrected_age_value);
func corrected_initial_age(res *http.Response) time.Duration {
	return durationMax(apparent_age(res), corrected_age_value(res))
}

// §
// §     The current_age of a stored response can then be calculated by adding
// §     the time (in seconds) since the stored response was last validated by
// §     the origin server to the corrected_initial_age.
// §
// §       resident_time = now - response_time;
func resident_time(res *http.Response) time.Duration {
	return now().Sub(response_time(res))
}

// §       current_age = corrected_initial_age + resident_time;
func current_age(res *http.Response) time.Duration {
	return corrected_initial_age(res) + resident_time(res)
}

func durationMax(d1, d2 time.Duration) time.Duration {
	if d1 > d2 {
		return d1
	}
	return d2
}
