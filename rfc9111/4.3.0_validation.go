package rfc9111

// §  4.3.  Validation
// §
// §     When a cache has one or more stored responses for a requested URI,
// §     but cannot serve any of them (e.g., because they are not fresh, or
// §     one cannot be chosen; see Section 4.1), it can use the conditional
// §     request mechanism (Section 13 of [HTTP]) in the forwarded request to
// §     give the next inbound server an opportunity to choose a valid stored
// §     response to use, updating the stored metadata in the process, or to
// §     replace the stored response(s) with a new response.  This process is
// §     known as "validating" or "revalidating" the stored response.