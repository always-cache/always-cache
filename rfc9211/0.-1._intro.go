package rfc9211

// §  Internet Engineering Task Force (IETF)                     M. Nottingham
// §  Request for Comments: 9211                                        Fastly
// §  Category: Standards Track                                      June 2022
// §  ISSN: 2070-1721
// §
// §                The Cache-Status HTTP Response Header Field
// §
// §  Abstract
// §
// §     To aid debugging, HTTP caches often append header fields to a
// §     response, explaining how they handled the request in an ad hoc
// §     manner.  This specification defines a standard mechanism to do so
// §     that is aligned with HTTP's caching model.
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
// §     https://www.rfc-editor.org/info/rfc9211.
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
// §  Table of Contents
// §
// §     1.  Introduction
// §       1.1.  Notational Conventions
// §     2.  The Cache-Status HTTP Response Header Field
// §       2.1.  The hit Parameter
// §       2.2.  The fwd Parameter
// §       2.3.  The fwd-status Parameter
// §       2.4.  The ttl Parameter
// §       2.5.  The stored Parameter
// §       2.6.  The collapsed Parameter
// §       2.7.  The key Parameter
// §       2.8.  The detail Parameter
// §     3.  Examples
// §     4.  Defining New Cache-Status Parameters
// §     5.  IANA Considerations
// §     6.  Security Considerations
// §     7.  References
// §       7.1.  Normative References
// §       7.2.  Informative References
// §     Author's Address
// §
// §  1.  Introduction
// §
// §     To aid debugging (both by humans and automated tools), HTTP caches
// §     often append header fields to a response explaining how they handled
// §     the request.  Unfortunately, the semantics of these header fields are
// §     often unclear, and both the semantics and syntax used vary between
// §     implementations.
// §
// §     This specification defines a new HTTP response header field, "Cache-
// §     Status", for this purpose with standardized syntax and semantics.
// §
// §  1.1.  Notational Conventions
// §
// §     The key words "MUST", "MUST NOT", "REQUIRED", "SHALL", "SHALL NOT",
// §     "SHOULD", "SHOULD NOT", "RECOMMENDED", "NOT RECOMMENDED", "MAY", and
// §     "OPTIONAL" in this document are to be interpreted as described in
// §     BCP 14 [RFC2119] [RFC8174] when, and only when, they appear in all
// §     capitals, as shown here.
// §
// §     This document uses the following terminology from Section 3 of
// §     [STRUCTURED-FIELDS] to specify syntax and parsing: List, String,
// §     Token, Integer, and Boolean.
// §
// §     This document also uses terminology from [HTTP] and [HTTP-CACHING].
