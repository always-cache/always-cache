package rfc9111

// §  4.2.4.  Serving Stale Responses
// §
// §     A "stale" response is one that either has explicit expiry information
// §     or is allowed to have heuristic expiry calculated, but is not fresh
// §     according to the calculations in Section 4.2.
// §
// §     A cache MUST NOT generate a stale response if it is prohibited by an
// §     explicit in-protocol directive (e.g., by a no-cache response
// §     directive, a must-revalidate response directive, or an applicable
// §     s-maxage or proxy-revalidate response directive; see Section 5.2.2).
// §
// §     A cache MUST NOT generate a stale response unless it is disconnected
// §     or doing so is explicitly permitted by the client or origin server
// §     (e.g., by the max-stale request directive in Section 5.2.1, extension
// §     directives such as those defined in [RFC5861], or configuration in
// §     accordance with an out-of-band contract).