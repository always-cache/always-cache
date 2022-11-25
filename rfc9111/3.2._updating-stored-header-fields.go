package rfc9111

// §  3.2.  Updating Stored Header Fields
// §
// §     Caches are required to update a stored response's header fields from
// §     another (typically newer) response in several situations; for
// §     example, see Sections 3.4, 4.3.4, and 4.3.5.
// §
// §     When doing so, the cache MUST add each header field in the provided
// §     response to the stored response, replacing field values that are
// §     already present, with the following exceptions:
// §
// §     *  Header fields excepted from storage in Section 3.1,
// §
// §     *  Header fields that the cache's stored response depends upon, as
// §        described below,
// §
// §     *  Header fields that are automatically processed and removed by the
// §        recipient, as described below, and
// §
// §     *  The Content-Length header field.
// §
// §     In some cases, caches (especially in user agents) store the results
// §     of processing the received response, rather than the response itself,
// §     and updating header fields that affect that processing can result in
// §     inconsistent behavior and security issues.  Caches in this situation
// §     MAY omit these header fields from updating stored responses on an
// §     exceptional basis but SHOULD limit such omission to those fields
// §     necessary to assure integrity of the stored response.
// §
// §     For example, a browser might decode the content coding of a response
// §     while it is being received, creating a disconnect between the data it
// §     has stored and the response's original metadata.  Updating that
// §     stored metadata with a different Content-Encoding header field would
// §     be problematic.  Likewise, a browser might store a post-parse HTML
// §     tree rather than the content received in the response; updating the
// §     Content-Type header field would not be workable in this case because
// §     any assumptions about the format made in parsing would now be
// §     invalid.
// §
// §     Furthermore, some fields are automatically processed and removed by
// §     the HTTP implementation, such as the Content-Range header field.
// §     Implementations MAY automatically omit such header fields from
// §     updates, even when the processing does not actually occur.
// §
// §     Note that the Content-* prefix is not a signal that a header field is
// §     omitted from update; it is a convention for MIME header fields, not
// §     HTTP.
