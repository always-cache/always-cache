package rfc9111

// §  4.3.4.  Freshening Stored Responses upon Validation
// §
// §     When a cache receives a 304 (Not Modified) response, it needs to
// §     identify stored responses that are suitable for updating with the new
// §     information provided, and then do so.
// §
// §     The initial set of stored responses to update are those that could
// §     have been chosen for that request -- i.e., those that meet the
// §     requirements in Section 4, except the last requirement to be fresh,
// §     able to be served stale, or just validated.
// §
// §     Then, that initial set of stored responses is further filtered by the
// §     first match of:
// §
// §     *  If the new response contains one or more "strong validators" (see
// §        Section 8.8.1 of [HTTP]), then each of those strong validators
// §        identifies a selected representation for update.  All the stored
// §        responses in the initial set with one of those same strong
// §        validators are identified for update.  If none of the initial set
// §        contains at least one of the same strong validators, then the
// §        cache MUST NOT use the new response to update any stored
// §        responses.
// §
// §     *  If the new response contains no strong validators but does contain
// §        one or more "weak validators", and those validators correspond to
// §        one of the initial set's stored responses, then the most recent of
// §        those matching stored responses is identified for update.
// §
// §     *  If the new response does not include any form of validator (such
// §        as where a client generates an If-Modified-Since request from a
// §        source other than the Last-Modified response header field), and
// §        there is only one stored response in the initial set, and that
// §        stored response also lacks a validator, then that stored response
// §        is identified for update.
// §
// §     For each stored response identified, the cache MUST update its header
// §     fields with the header fields provided in the 304 (Not Modified)
// §     response, as per Section 3.2.