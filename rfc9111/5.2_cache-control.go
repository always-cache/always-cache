package rfc9111

import "strings"

// §  3.2. Cache-Control
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

func (c CacheControl) Get(directive string) (string, bool) {
	val, ok := c.directives[directive]
	return val, ok
}

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

func getCacheControlDirectiveName(token string) string {
	// §  [...] to be compared case-insensitively [...]
	return strings.ToLower(token)
}

func getCacheControlDirectiveArgument(arg string) string {
	// §  [...] argument that can use both token and quoted-string syntax. [...]
	return strings.Trim(arg, "\"")
}
