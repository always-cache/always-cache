package rfc9111

import (
	"strings"
	"time"
)

// CacheControl implements parsing of the "Cache-Control" header (/field).
//
// §  5.2. Cache-Control
// §
// §  The "Cache-Control" header field is used to list directives for caches along
// §  the request/response chain. Cache directives are unidirectional, in that the
// §  presence of a directive in a request does not imply that the same directive is
// §  present or copied in the response.
// §
// §  See Section 5.2.3 for information about how Cache-Control directives
// §  defined elsewhere are handled. A proxy, whether or not it implements a cache,
// §  MUST pass cache directives through in forwarded messages, regardless of their
// §  significance to that application, since the directives might apply to all
// §  recipients along the request/response chain. It is not possible to target a
// §  directive to a specific cache. Cache directives are identified by a token, to
// §  be compared case-insensitively, and have an optional argument that can use both
// §  token and quoted-string syntax. For the directives defined below that define
// §  arguments, recipients ought to accept both forms, even if a specific form is
// §  required for generation.
// §
// §    Cache-Control   = #cache-directive
// §
// §    cache-directive = token [ "=" ( token / quoted-string ) ]
// §
// §  For the cache directives defined below, no argument is defined (nor allowed) unless stated otherwise.
type CacheControl struct {
	directives map[string]string
}

// Get returns the value (/argument) of the specified directive,
// along with a boolean indicating whether this directive is present
func (c CacheControl) Get(directive string) (string, bool) {
	val, ok := c.directives[directive]
	return val, ok
}

// HasDirective returns whether the specified directive is present
func (c CacheControl) HasDirective(directive string) bool {
	_, ok := c.Get(directive)
	return ok
}

// ParseCacheControl takes Cache-Control headers as a slice of strings
// and returns an instance of `CacheControl`.
func ParseCacheControl(headers []string) CacheControl {
	m := make(map[string]string)
	// process all headers
	// note setting map values like this means last defined directive wins
	for _, header := range headers {
		// process directives "#" means comma-separated list
		for _, directive := range strings.Split(header, ", ") {
			parts := strings.SplitN(directive, "=", 2)
			name := getCacheControlDirectiveName(parts[0])
			var arg string
			if len(parts) > 1 {
				arg = getCacheControlDirectiveArgument(parts[1])
			}
			m[name] = arg
		}
	}
	return CacheControl{m}
}

// getCacheControlDirectiveName returns a normalized name for the given directive.
func getCacheControlDirectiveName(token string) string {
	// §  [...] to be compared case-insensitively [...]
	return strings.ToLower(token)
}

// getCacheControlDirectiveArgument returns the directive argument in token form,
// i.e. it converts the argument from "quoted-string" to "token" form if needed.
func getCacheControlDirectiveArgument(arg string) string {
	// §  [...] argument that can use both token and quoted-string syntax. [...]
	return strings.Trim(arg, "\"")
}

// §  5.2.1. Request Directives
// §  This section defines cache request directives. They are advisory; caches
// §  MAY implement them, but are not required to.
//
// Request directives are not implemented at this time.

// §  5.2.2. Response Directives
// §
// §  This section defines cache response directives. A cache MUST obey the Cache-
// §  Control directives defined in this section.

// MaxAge returns "max-age" as a duration, along with a boolean indicating
// whether the "max-age" directive was present.
//
// §  5.2.2.1. max-age
// §
// §  Argument syntax:
// §
// §      delta-seconds (see Section 1.2.2)
// §
// §  The max-age response directive indicates that the response is to be considered
// §  stale after its age is greater than the specified number of seconds. This
// §  directive uses the token form of the argument syntax: e.g., 'max-age=5' not
// §  'max-age="5"'. A sender MUST NOT generate the quoted-string form.
func (c CacheControl) MaxAge() (time.Duration, bool) {
	return c.getDeltaSeconds("max-age")
}

// §  5.2.2.10.  s-maxage
// §
// §     Argument syntax:
// §
// §        delta-seconds (see Section 1.2.2)
// §
// §     The s-maxage response directive indicates that, for a shared cache,
// §     the maximum age specified by this directive overrides the maximum age
// §     specified by either the max-age directive or the Expires header
// §     field.
// §
// §     The s-maxage directive incorporates the semantics of the
// §     proxy-revalidate response directive (Section 5.2.2.8) for a shared
// §     cache.  A shared cache MUST NOT reuse a stale response with s-maxage
// §     to satisfy another request until it has been successfully validated
// §     by the origin, as defined by Section 4.3.  This directive also
// §     permits a shared cache to reuse a response to a request containing an
// §     Authorization header field, subject to the above requirements on
// §     maximum age and revalidation (Section 3.5).
// §
// §     This directive uses the token form of the argument syntax: e.g.,
// §     's-maxage=10' not 's-maxage="10"'.  A sender MUST NOT generate the
// §     quoted-string form.
func (c CacheControl) SMaxAge() (time.Duration, bool) {
	return c.getDeltaSeconds("s-maxage")
}

// TODO implement other response directives

// getDeltaSeconds returns the "delta-seconds" as `time.Duration`,
// as well as a boolean indicating whether the directive was set.
//
// Examples:
// directive    -> 0,  false
// directive=0  -> 0,  true
// directive=60 -> 60, true
func (c CacheControl) getDeltaSeconds(directive string) (time.Duration, bool) {
	if secondsStr, ok := c.Get(directive); ok && secondsStr != "" {
		return deltaSeconds(secondsStr), true
	}
	return 0, false
}
