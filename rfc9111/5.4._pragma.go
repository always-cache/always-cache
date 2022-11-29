package rfc9111

// §  5.4.  Pragma
// §
// §     The "Pragma" request header field was defined for HTTP/1.0 caches, so
// §     that clients could specify a "no-cache" request (as Cache-Control was
// §     not defined until HTTP/1.1).
// §
// §     However, support for Cache-Control is now widespread.  As a result,
// §     this specification deprecates Pragma.
// §
// §        |  *Note:* Because the meaning of "Pragma: no-cache" in responses
// §        |  was never specified, it does not provide a reliable replacement
// §        |  for "Cache-Control: no-cache" in them.