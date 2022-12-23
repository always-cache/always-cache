package rfc9111

import (
	"net/http"
	"strings"
)

// §  3.1.  Storing Header and Trailer Fields
func storableHeader(header http.Header) http.Header {
	if header == nil {
		return nil
	}
	// §     Caches MUST include all received response header fields -- including
	// §     unrecognized ones -- when storing a response; this assures that new
	// §     HTTP header fields can be successfully deployed.  However, the
	// §     following exceptions are made:
	h := header.Clone()
	// §
	// §     *  The Connection header field and fields whose names are listed in
	// §        it are required by Section 7.6.1 of [HTTP] to be removed before
	// §        forwarding the message.  This MAY be implemented by doing so
	// §        before storage.
	for _, header := range GetListHeader(header, "Connection") {
		h.Del(header)
	}
	h.Del("Connection")
	// §
	// §     *  Likewise, some fields' semantics require them to be removed before
	// §        forwarding the message, and this MAY be implemented by doing so
	// §        before storage; see Section 7.6.1 of [HTTP] for some examples.
	h.Del("Proxy-Connection")
	h.Del("Keep-Alive")
	h.Del("TE")
	h.Del("Transfer-Encoding")
	h.Del("Upgrade")
	// §
	// §     *  The no-cache (Section 5.2.2.4) and private (Section 5.2.2.7) cache
	// §        directives can have arguments that prevent storage of header
	// §        fields by all caches and shared caches, respectively.
	// §
	// §     *  Header fields that are specific to the proxy that a cache uses
	// §        when forwarding a request MUST NOT be stored, unless the cache
	// §        incorporates the identity of the proxy into the cache key.
	// §        Effectively, this is limited to Proxy-Authenticate (Section 11.7.1
	// §        of [HTTP]), Proxy-Authentication-Info (Section 11.7.3 of [HTTP]),
	// §        and Proxy-Authorization (Section 11.7.2 of [HTTP]).
	return h
}

// §     Caches MAY either store trailer fields separate from header fields or
// §     discard them.  Caches MUST NOT combine trailer fields with header
// §     fields.
func storableTrailer(trailer http.Header) http.Header {
	return make(http.Header)
}

// TODO move to http rfc
func GetListHeader(header http.Header, field string) []string {
	list := make([]string, 0)
	for _, hdr := range header.Values(field) {
		for _, item := range strings.Split(hdr, ",") {
			list = append(list, strings.TrimSpace(item))
		}
	}
	return list
}

// TODO move to http rfc
func GetForwardRequest(req *http.Request) *http.Request {
	r := req.Clone(req.Context())

	for _, header := range GetListHeader(r.Header, "Connection") {
		r.Header.Del(header)
	}
	r.Header.Del("Connection")
	r.Header.Del("Proxy-Connection")
	r.Header.Del("Keep-Alive")
	r.Header.Del("TE")
	r.Header.Del("Transfer-Encoding")
	r.Header.Del("Upgrade")

	return r
}