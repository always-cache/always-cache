package cachekey

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/always-cache/always-cache/rfc9111"
)

var ErrorMethodNotSupported = fmt.Errorf("Method not supported")

const (
	originSeparator = ":"
	methodSeparator = ":"
	varySeparator   = "\t"
)

type CacheKeyer struct {
	// Unique identifier for the origin.
	// Usually this should be the origin - well - origin.
	OriginId string
	// Cache key prefix for this origin
	OriginPrefix string
}

func NewCacheKeyer(originId string) CacheKeyer {
	return CacheKeyer{
		OriginId:     originId,
		OriginPrefix: originId + originSeparator,
	}
}

// MethodPrefix gets the key prefix for the origin with the given method.
// E.g. prefix for all GET requests in the coche.
func (c CacheKeyer) MethodPrefix(method string) string {
	return c.OriginId + originSeparator + method + methodSeparator
}

// getKeyPrefix returns the cache key for a request without the vary headers (i.e. a key prefix).
// The returned key is suitable for finding all stored response variants for a porticular request.
// If the request has a `Cache-Key` header, that value is included in the key prefix.
func (c CacheKeyer) GetKeyPrefix(r *http.Request) string {
	key := c.OriginId + originSeparator + r.Method + methodSeparator + r.URL.RequestURI() + varySeparator
	if ck := r.Header.Get("Cache-Key"); ck != "" {
		key += ck
	}
	return key
}

// addVaryKeys returns the full cache key (including vary headers) based on a previously generated
// cache key prefix and the request and response involved.
func (c CacheKeyer) AddVaryKeys(prefix string, req *http.Request, res *http.Response) string {
	key := prefix
	for _, name := range rfc9111.GetListHeader(res.Header, "Vary") {
		if !rfc9111.FieldAbsent(req.Header, name) {
			key = key + "\n" + strings.ToLower(name) + ": " + req.Header.Get(name)
		}
	}
	return key
}

// getRequestFromKey generates a caching-wise equal request than the request that resulted in the
// provided key. This means it takes vary headers into account.
// It returns an error if the request cannot for some reason be deducted.
func (c CacheKeyer) GetRequestFromKey(key string) (*http.Request, error) {
	if !strings.HasPrefix(key, c.OriginPrefix) {
		return nil, fmt.Errorf("Key and origin do not match")
	}
	keyNoOrigin := strings.TrimPrefix(key, c.OriginPrefix)
	keyNoVary, _, found := strings.Cut(keyNoOrigin, varySeparator)
	if !found {
		return nil, fmt.Errorf("Malformed key: %s", key)
	}
	method, uri, found := strings.Cut(keyNoVary, methodSeparator)
	if !found {
		return nil, fmt.Errorf("Malformed key: %s", key)
	}
	req, err := http.NewRequest(method, uri, nil)
	if err != nil {
		return req, err
	}
	req.Header = c.GetVaryHeaders(key)
	return req, nil
}

// getVaryHeaders creates a http.Header instance containing all the vary keys included in a key.
func (c CacheKeyer) GetVaryHeaders(key string) http.Header {
	header := make(http.Header)
	lines := strings.Split(key, "\n")
	for i := 1; i < len(lines); i++ {
		entry := strings.SplitN(lines[i], ": ", 2)
		header.Add(entry[0], entry[1])
	}
	return header
}
