package main

import "fmt"

type CacheStatusStatus string

const (
	CacheStatusHit = "hit"
	CacheStatusFwd = "fwd"
)

type CacheStatusFwdReason string

const (
	// The cache was configured to not handle this request.
	CacheStatusFwdBypass = "bypass"

	// The request method's semantics require the request to be
	// forwarded.
	CacheStatusFwdMethod = "method"

	// The cache did not contain any responses that matched the
	// request URI.
	CacheStatusFwdUriMiss = "uri-miss"

	// The cache contained a response that matched the request
	// URI, but it could not select a response based upon this request's
	// header fields and stored Vary header fields.
	CacheStatusFwdVaryMiss = "vary-miss"

	// The cache did not contain any responses that could be used to
	// satisfy this request (to be used when an implementation cannot
	// distinguish between uri-miss and vary-miss).
	CacheStatusFwdMiss = "miss"

	// The cache was able to select a fresh response for the
	// request, but the request's semantics (e.g., Cache-Control request
	// directives) did not allow its use.
	CacheStatusFwdRequest = "request"

	// The cache was able to select a response for the request, but
	// it was stale.
	CacheStatusFwdStale = "stale"

	// The cache was able to select a partial response for the
	// request, but it did not contain all of the requested ranges (or
	// the request was for the complete response).
	CacheStatusFwdPartial = "partial"
)

type CacheStatus struct {
	status    CacheStatusStatus
	detail    string
	fwdReason CacheStatusFwdReason
}

func (cs *CacheStatus) Hit() {
	cs.status = CacheStatusHit
}

func (cs *CacheStatus) Forward(reason CacheStatusFwdReason) {
	cs.status = CacheStatusFwd
	cs.fwdReason = reason
}

func (cs *CacheStatus) Detail(detail string) {
	cs.detail = detail
}

func (cs *CacheStatus) String() string {
	status := fmt.Sprintf("Always-Cache; %s", cs.status)
	if cs.status == "fwd" && cs.fwdReason != "" {
		status = fmt.Sprintf("%s=%s", status, cs.fwdReason)
	}
	if cs.detail != "" {
		status = status + "; detail=" + cs.detail
	}
	return status
}
