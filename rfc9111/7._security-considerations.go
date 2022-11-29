package rfc9111

// §  7.  Security Considerations
// §
// §     This section is meant to inform developers, information providers,
// §     and users of known security concerns specific to HTTP caching.  More
// §     general security considerations are addressed in "HTTP/1.1"
// §     (Section 11 of [HTTP/1.1]) and "HTTP Semantics" (Section 17 of
// §     [HTTP]).
// §
// §     Caches expose an additional attack surface because the contents of
// §     the cache represent an attractive target for malicious exploitation.
// §     Since cache contents persist after an HTTP request is complete, an
// §     attack on the cache can reveal information long after a user believes
// §     that the information has been removed from the network.  Therefore,
// §     cache contents need to be protected as sensitive information.
// §
// §     In particular, because private caches are restricted to a single
// §     user, they can be used to reconstruct a user's activity.  As a
// §     result, it is important for user agents to allow end users to control
// §     them, for example, by allowing stored responses to be removed for
// §     some or all origin servers.
// §
// §  7.1.  Cache Poisoning
// §
// §     Storing malicious content in a cache can extend the reach of an
// §     attacker to affect multiple users.  Such "cache poisoning" attacks
// §     happen when an attacker uses implementation flaws, elevated
// §     privileges, or other techniques to insert a response into a cache.
// §     This is especially effective when shared caches are used to
// §     distribute malicious content to many clients.
// §
// §     One common attack vector for cache poisoning is to exploit
// §     differences in message parsing on proxies and in user agents; see
// §     Section 6.3 of [HTTP/1.1] for the relevant requirements regarding
// §     HTTP/1.1.
// §
// §  7.2.  Timing Attacks
// §
// §     Because one of the primary uses of a cache is to optimize
// §     performance, its use can "leak" information about which resources
// §     have been previously requested.
// §
// §     For example, if a user visits a site and their browser caches some of
// §     its responses and then navigates to a second site, that site can
// §     attempt to load responses it knows exist on the first site.  If they
// §     load quickly, it can be assumed that the user has visited that site,
// §     or even a specific page on it.
// §
// §     Such "timing attacks" can be mitigated by adding more information to
// §     the cache key, such as the identity of the referring site (to prevent
// §     the attack described above).  This is sometimes called "double
// §     keying".
// §
// §  7.3.  Caching of Sensitive Information
// §
// §     Implementation and deployment flaws (often led to by the
// §     misunderstanding of cache operation) might lead to the caching of
// §     sensitive information (e.g., authentication credentials) that is
// §     thought to be private, exposing it to unauthorized parties.
// §
// §     Note that the Set-Cookie response header field [COOKIE] does not
// §     inhibit caching; a cacheable response with a Set-Cookie header field
// §     can be (and often is) used to satisfy subsequent requests to caches.
// §     Servers that wish to control caching of these responses are
// §     encouraged to emit appropriate Cache-Control response header fields.