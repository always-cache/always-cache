package rfc9111

// §  3.1.  Storing Header and Trailer Fields
// §
// §     Caches MUST include all received response header fields -- including
// §     unrecognized ones -- when storing a response; this assures that new
// §     HTTP header fields can be successfully deployed.  However, the
// §     following exceptions are made:
// §
// §     *  The Connection header field and fields whose names are listed in
// §        it are required by Section 7.6.1 of [HTTP] to be removed before
// §        forwarding the message.  This MAY be implemented by doing so
// §        before storage.
// §
// §     *  Likewise, some fields' semantics require them to be removed before
// §        forwarding the message, and this MAY be implemented by doing so
// §        before storage; see Section 7.6.1 of [HTTP] for some examples.
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
// §
// §     Caches MAY either store trailer fields separate from header fields or
// §     discard them.  Caches MUST NOT combine trailer fields with header
// §     fields.
