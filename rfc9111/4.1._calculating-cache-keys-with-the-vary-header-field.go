package rfc9111

import (
	"net/http"
	"strings"

	"github.com/rs/zerolog/log"
)

// §  4.1.  Calculating Cache Keys with the Vary Header Field
// §
// §     When a cache receives a request that can be satisfied by a stored
// §     response and that stored response contains a Vary header field
// §     (Section 12.5.5 of [HTTP]), the cache MUST NOT use that stored
// §     response without revalidation unless all the presented request header
// §     fields nominated by that Vary field value match those fields in the
// §     original request (i.e., the request that caused the cached response
// §     to be stored).
func headerFieldsMatch(req *http.Request, res *http.Response) bool {
	// TODO
	ceMatch := false
	for _, item := range res.Header.Values("Vary") {
		log.Trace().Msgf("Checking Vary header %s", item)
		// if vary is only for content-encoding, and the stored header matches
		// that of the request, we are good to go
		if strings.ToLower(item) == "accept-encoding" {
			for _, accepted := range strings.Split(req.Header.Get("Accept-Encoding"), ", ") {
				if accepted == res.Header.Get("Content-Encoding") {
					ceMatch = true
				}
			}
		} else if item != "" {
			return false
		}
	}
	return ceMatch
}

// §     The header fields from two requests are defined to match if and only
// §     if those in the first request can be transformed to those in the
// §     second request by applying any of the following:
// §
// §     *  adding or removing whitespace, where allowed in the header field's
// §        syntax
// §
// §     *  combining multiple header field lines with the same field name
// §        (see Section 5.2 of [HTTP])
// §
// §     *  normalizing both header field values in a way that is known to
// §        have identical semantics, according to the header field's
// §        specification (e.g., reordering field values when order is not
// §        significant; case-normalization, where values are defined to be
// §        case-insensitive)
// §
// §     If (after any normalization that might take place) a header field is
// §     absent from a request, it can only match another request if it is
// §     also absent there.
// §
// §     A stored response with a Vary header field value containing a member
// §     "*" always fails to match.
// §
// §     If multiple stored responses match, the cache will need to choose one
// §     to use.  When a nominated request header field has a known mechanism
// §     for ranking preference (e.g., qvalues on Accept and similar request
// §     header fields), that mechanism MAY be used to choose a preferred
// §     response.  If such a mechanism is not available, or leads to equally
// §     preferred responses, the most recent response (as determined by the
// §     Date header field) is chosen, as per Section 4.
// §
// §     Some resources mistakenly omit the Vary header field from their
// §     default response (i.e., the one sent when the request does not
// §     express any preferences), with the effect of choosing it for
// §     subsequent requests to that resource even when more preferable
// §     responses are available.  When a cache has multiple stored responses
// §     for a target URI and one or more omits the Vary header field, the
// §     cache SHOULD choose the most recent (see Section 4.2.3) stored
// §     response with a valid Vary field value.
// §
// §     If no stored response matches, the cache cannot satisfy the presented
// §     request.  Typically, the request is forwarded to the origin server,
// §     potentially with preconditions added to describe what responses the
// §     cache has already stored (Section 4.3).
