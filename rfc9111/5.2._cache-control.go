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
// WARNING Request directives are not implemented at this time.

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

// §  5.2.2.2.  must-revalidate
// §
// §     The must-revalidate response directive indicates that once the
// §     response has become stale, a cache MUST NOT reuse that response to
// §     satisfy another request until it has been successfully validated by
// §     the origin, as defined by Section 4.3.
// §
// §     The must-revalidate directive is necessary to support reliable
// §     operation for certain protocol features.  In all circumstances, a
// §     cache MUST NOT ignore the must-revalidate directive; in particular,
// §     if a cache is disconnected, the cache MUST generate an error response
// §     rather than reuse the stale response.  The generated status code
// §     SHOULD be 504 (Gateway Timeout) unless another error status code is
// §     more applicable.
// §
// §     The must-revalidate directive ought to be used by servers if and only
// §     if failure to validate a request could cause incorrect operation,
// §     such as a silently unexecuted financial transaction.
// §
// §     The must-revalidate directive also permits a shared cache to reuse a
// §     response to a request containing an Authorization header field
// §     (Section 11.6.2 of [HTTP]), subject to the above requirement on
// §     revalidation (Section 3.5).
// §
// §  5.2.2.3.  must-understand
// §
// §     The must-understand response directive limits caching of the response
// §     to a cache that understands and conforms to the requirements for that
// §     response's status code.
// §
// §     A response that contains the must-understand directive SHOULD also
// §     contain the no-store directive.  When a cache that implements the
// §     must-understand directive receives a response that includes it, the
// §     cache SHOULD ignore the no-store directive if it understands and
// §     implements the status code's caching requirements.
// §
// §  5.2.2.4.  no-cache
// §
// §     Argument syntax:
// §
// §        #field-name
// §
// §     The no-cache response directive, in its unqualified form (without an
// §     argument), indicates that the response MUST NOT be used to satisfy
// §     any other request without forwarding it for validation and receiving
// §     a successful response; see Section 4.3.
// §
// §     This allows an origin server to prevent a cache from using the
// §     response to satisfy a request without contacting it, even by caches
// §     that have been configured to send stale responses.
// §
// §     The qualified form of the no-cache response directive, with an
// §     argument that lists one or more field names, indicates that a cache
// §     MAY use the response to satisfy a subsequent request, subject to any
// §     other restrictions on caching, if the listed header fields are
// §     excluded from the subsequent response or the subsequent response has
// §     been successfully revalidated with the origin server (updating or
// §     removing those fields).  This allows an origin server to prevent the
// §     reuse of certain header fields in a response, while still allowing
// §     caching of the rest of the response.
// §
// §     The field names given are not limited to the set of header fields
// §     defined by this specification.  Field names are case-insensitive.
// §
// §     This directive uses the quoted-string form of the argument syntax.  A
// §     sender SHOULD NOT generate the token form (even if quoting appears
// §     not to be needed for single-entry lists).
// §
// §        |  *Note:* The qualified form of the directive is often handled by
// §        |  caches as if an unqualified no-cache directive was received;
// §        |  that is, the special handling for the qualified form is not
// §        |  widely implemented.
// §
// §  5.2.2.5.  no-store
// §
// §     The no-store response directive indicates that a cache MUST NOT store
// §     any part of either the immediate request or the response and MUST NOT
// §     use the response to satisfy any other request.
// §
// §     This directive applies to both private and shared caches.  "MUST NOT
// §     store" in this context means that the cache MUST NOT intentionally
// §     store the information in non-volatile storage and MUST make a best-
// §     effort attempt to remove the information from volatile storage as
// §     promptly as possible after forwarding it.
// §
// §     This directive is not a reliable or sufficient mechanism for ensuring
// §     privacy.  In particular, malicious or compromised caches might not
// §     recognize or obey this directive, and communications networks might
// §     be vulnerable to eavesdropping.
// §
// §     Note that the must-understand cache directive overrides no-store in
// §     certain circumstances; see Section 5.2.2.3.
// §
// §  5.2.2.6.  no-transform
// §
// §     The no-transform response directive indicates that an intermediary
// §     (regardless of whether it implements a cache) MUST NOT transform the
// §     content, as defined in Section 7.7 of [HTTP].
// §
// §  5.2.2.7.  private
// §
// §     Argument syntax:
// §
// §        #field-name
// §
// §     The unqualified private response directive indicates that a shared
// §     cache MUST NOT store the response (i.e., the response is intended for
// §     a single user).  It also indicates that a private cache MAY store the
// §     response, subject to the constraints defined in Section 3, even if
// §     the response would not otherwise be heuristically cacheable by a
// §     private cache.
// §
// §     If a qualified private response directive is present, with an
// §     argument that lists one or more field names, then only the listed
// §     header fields are limited to a single user: a shared cache MUST NOT
// §     store the listed header fields if they are present in the original
// §     response but MAY store the remainder of the response message without
// §     those header fields, subject the constraints defined in Section 3.
// §
// §     The field names given are not limited to the set of header fields
// §     defined by this specification.  Field names are case-insensitive.
// §
// §     This directive uses the quoted-string form of the argument syntax.  A
// §     sender SHOULD NOT generate the token form (even if quoting appears
// §     not to be needed for single-entry lists).
// §
// §        |  *Note:* This usage of the word "private" only controls where
// §        |  the response can be stored; it cannot ensure the privacy of the
// §        |  message content.  Also, the qualified form of the directive is
// §        |  often handled by caches as if an unqualified private directive
// §        |  was received; that is, the special handling for the qualified
// §        |  form is not widely implemented.
// §
// §  5.2.2.8.  proxy-revalidate
// §
// §     The proxy-revalidate response directive indicates that once the
// §     response has become stale, a shared cache MUST NOT reuse that
// §     response to satisfy another request until it has been successfully
// §     validated by the origin, as defined by Section 4.3.  This is
// §     analogous to must-revalidate (Section 5.2.2.2), except that proxy-
// §     revalidate does not apply to private caches.
// §
// §     Note that proxy-revalidate on its own does not imply that a response
// §     is cacheable.  For example, it might be combined with the public
// §     directive (Section 5.2.2.9), allowing the response to be cached while
// §     requiring only a shared cache to revalidate when stale.
// §
// §  5.2.2.9.  public
// §
// §     The public response directive indicates that a cache MAY store the
// §     response even if it would otherwise be prohibited, subject to the
// §     constraints defined in Section 3.  In other words, public explicitly
// §     marks the response as cacheable.  For example, public permits a
// §     shared cache to reuse a response to a request containing an
// §     Authorization header field (Section 3.5).
// §
// §     Note that it is unnecessary to add the public directive to a response
// §     that is already cacheable according to Section 3.
// §
// §     If a response with the public directive has no explicit freshness
// §     information, it is heuristically cacheable (Section 4.2.2).

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

// §  5.2.3.  Extension Directives
// §
// §     The Cache-Control header field can be extended through the use of one
// §     or more extension cache directives.  A cache MUST ignore unrecognized
// §     cache directives.
// §
// §     Informational extensions (those that do not require a change in cache
// §     behavior) can be added without changing the semantics of other
// §     directives.
// §
// §     Behavioral extensions are designed to work by acting as modifiers to
// §     the existing base of cache directives.  Both the new directive and
// §     the old directive are supplied, such that applications that do not
// §     understand the new directive will default to the behavior specified
// §     by the old directive, and those that understand the new directive
// §     will recognize it as modifying the requirements associated with the
// §     old directive.  In this way, extensions to the existing cache
// §     directives can be made without breaking deployed caches.
// §
// §     For example, consider a hypothetical new response directive called
// §     "community" that acts as a modifier to the private directive: in
// §     addition to private caches, only a cache that is shared by members of
// §     the named community is allowed to cache the response.  An origin
// §     server wishing to allow the UCI community to use an otherwise private
// §     response in their shared cache(s) could do so by including
// §
// §     Cache-Control: private, community="UCI"
// §
// §     A cache that recognizes such a community cache directive could
// §     broaden its behavior in accordance with that extension.  A cache that
// §     does not recognize the community cache directive would ignore it and
// §     adhere to the private directive.
// §
// §     New extension directives ought to consider defining:
// §
// §     *  What it means for a directive to be specified multiple times,
// §
// §     *  When the directive does not take an argument, what it means when
// §        an argument is present,
// §
// §     *  When the directive requires an argument, what it means when it is
// §        missing, and
// §
// §     *  Whether the directive is specific to requests, specific to
// §        responses, or able to be used in either.
// §
// §  5.2.4.  Cache Directive Registry
// §
// §     The "Hypertext Transfer Protocol (HTTP) Cache Directive Registry"
// §     defines the namespace for the cache directives.  It has been created
// §     and is now maintained at <https://www.iana.org/assignments/http-
// §     cache-directives>.
// §
// §     A registration MUST include the following fields:
// §
// §     *  Cache Directive Name
// §
// §     *  Pointer to specification text
// §
// §     Values to be added to this namespace require IETF Review (see
// §     [RFC8126], Section 4.8).
