package rfc9111

// §  Internet Engineering Task Force (IETF)                  R. Fielding, Ed.
// §  Request for Comments: 9111                                         Adobe
// §  STD: 98                                               M. Nottingham, Ed.
// §  Obsoletes: 7234                                                   Fastly
// §  Category: Standards Track                                J. Reschke, Ed.
// §  ISSN: 2070-1721                                               greenbytes
// §                                                                 June 2022
// §
// §                                HTTP Caching
// §
// §  Abstract
// §
// §     The Hypertext Transfer Protocol (HTTP) is a stateless application-
// §     level protocol for distributed, collaborative, hypertext information
// §     systems.  This document defines HTTP caches and the associated header
// §     fields that control cache behavior or indicate cacheable response
// §     messages.
// §
// §     This document obsoletes RFC 7234.
// §
// §  Status of This Memo
// §
// §     This is an Internet Standards Track document.
// §
// §     This document is a product of the Internet Engineering Task Force
// §     (IETF).  It represents the consensus of the IETF community.  It has
// §     received public review and has been approved for publication by the
// §     Internet Engineering Steering Group (IESG).  Further information on
// §     Internet Standards is available in Section 2 of RFC 7841.
// §
// §     Information about the current status of this document, any errata,
// §     and how to provide feedback on it may be obtained at
// §     https://www.rfc-editor.org/info/rfc9111.
// §
// §  Copyright Notice
// §
// §     Copyright (c) 2022 IETF Trust and the persons identified as the
// §     document authors.  All rights reserved.
// §
// §     This document is subject to BCP 78 and the IETF Trust's Legal
// §     Provisions Relating to IETF Documents
// §     (https://trustee.ietf.org/license-info) in effect on the date of
// §     publication of this document.  Please review these documents
// §     carefully, as they describe your rights and restrictions with respect
// §     to this document.  Code Components extracted from this document must
// §     include Revised BSD License text as described in Section 4.e of the
// §     Trust Legal Provisions and are provided without warranty as described
// §     in the Revised BSD License.
// §
// §     This document may contain material from IETF Documents or IETF
// §     Contributions published or made publicly available before November
// §     10, 2008.  The person(s) controlling the copyright in some of this
// §     material may not have granted the IETF Trust the right to allow
// §     modifications of such material outside the IETF Standards Process.
// §     Without obtaining an adequate license from the person(s) controlling
// §     the copyright in such materials, this document may not be modified
// §     outside the IETF Standards Process, and derivative works of it may
// §     not be created outside the IETF Standards Process, except to format
// §     it for publication as an RFC or to translate it into languages other
// §     than English.
// §
// §  Table of Contents
// §
// §     1.  Introduction
// §       1.1.  Requirements Notation
// §       1.2.  Syntax Notation
// §         1.2.1.  Imported Rules
// §         1.2.2.  Delta Seconds
// §     2.  Overview of Cache Operation
// §     3.  Storing Responses in Caches
// §       3.1.  Storing Header and Trailer Fields
// §       3.2.  Updating Stored Header Fields
// §       3.3.  Storing Incomplete Responses
// §       3.4.  Combining Partial Content
// §       3.5.  Storing Responses to Authenticated Requests
// §     4.  Constructing Responses from Caches
// §       4.1.  Calculating Cache Keys with the Vary Header Field
// §       4.2.  Freshness
// §         4.2.1.  Calculating Freshness Lifetime
// §         4.2.2.  Calculating Heuristic Freshness
// §         4.2.3.  Calculating Age
// §         4.2.4.  Serving Stale Responses
// §       4.3.  Validation
// §         4.3.1.  Sending a Validation Request
// §         4.3.2.  Handling a Received Validation Request
// §         4.3.3.  Handling a Validation Response
// §         4.3.4.  Freshening Stored Responses upon Validation
// §         4.3.5.  Freshening Responses with HEAD
// §       4.4.  Invalidating Stored Responses
// §     5.  Field Definitions
// §       5.1.  Age
// §       5.2.  Cache-Control
// §         5.2.1.  Request Directives
// §           5.2.1.1.  max-age
// §           5.2.1.2.  max-stale
// §           5.2.1.3.  min-fresh
// §           5.2.1.4.  no-cache
// §           5.2.1.5.  no-store
// §           5.2.1.6.  no-transform
// §           5.2.1.7.  only-if-cached
// §         5.2.2.  Response Directives
// §           5.2.2.1.  max-age
// §           5.2.2.2.  must-revalidate
// §           5.2.2.3.  must-understand
// §           5.2.2.4.  no-cache
// §           5.2.2.5.  no-store
// §           5.2.2.6.  no-transform
// §           5.2.2.7.  private
// §           5.2.2.8.  proxy-revalidate
// §           5.2.2.9.  public
// §           5.2.2.10. s-maxage
// §         5.2.3.  Extension Directives
// §         5.2.4.  Cache Directive Registry
// §       5.3.  Expires
// §       5.4.  Pragma
// §       5.5.  Warning
// §     6.  Relationship to Applications and Other Caches
// §     7.  Security Considerations
// §       7.1.  Cache Poisoning
// §       7.2.  Timing Attacks
// §       7.3.  Caching of Sensitive Information
// §     8.  IANA Considerations
// §       8.1.  Field Name Registration
// §       8.2.  Cache Directive Registration
// §       8.3.  Warn Code Registry
// §     9.  References
// §       9.1.  Normative References
// §       9.2.  Informative References
// §     Appendix A.  Collected ABNF
// §     Appendix B.  Changes from RFC 7234
// §     Acknowledgements
// §     Index
// §     Authors' Addresses