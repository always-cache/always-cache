package cacheupdate

import (
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/always-cache/always-cache/rfc9111"
)

// CacheUpdate represents a single `Cache-Update` entry.
type CacheUpdate struct {
	// Fully resolved relative path to the resource.
	// Equivalent to `url.URL.Path`.
	Path string
	// Update delay, i.e. delay update by this duration.
	Delay time.Duration
}

// GetCacheUpdates gets the updates specified by the response.
// The incoming request is used in order to resolve potentially relative update paths.
func GetCacheUpdates(req *http.Request, res *http.Response) []CacheUpdate {
	if !rfc9111.UnsafeRequest(req) {
		return nil
	}
	updates := make([]CacheUpdate, 0)
	for _, update := range res.Header.Values("Cache-Update") {
		cu := CacheUpdate{}
		// path is the first element
		path := strings.Split(update, ";")[0]
		cu.Path = getURL(res.Request, path).Path
		cu.Delay = getDelay(update)

		updates = append(updates, cu)
	}
	return updates
}

// getURL returns the URL to update the cache for from the `Cache-Update` header parameter.
// The URL is the first parameter in the header value (separated by a semicolon).
func getURL(r *http.Request, update string) *url.URL {
	possiblyRelativeURL := update
	if i := strings.Index(update, ";"); i != -1 {
		possiblyRelativeURL = update[:i]
	}
	return r.URL.ResolveReference(&url.URL{Path: possiblyRelativeURL})
}

// getDelay returns the delay to wait before updating the cache for from the `Cache-Update` header parameter.
// The delay directive syntax is `delay=N`, where N is the number of seconds to wait.
// Directives are separated by a semicolon.
// If no delay directive is found, it returns 0.
func getDelay(update string) time.Duration {
	// get the delay directive based on regular expression
	if matches := regexp.MustCompile(`(?i)\bdelay=(\d+)`).FindStringSubmatch(update); matches != nil {
		if delay, err := strconv.Atoi(matches[1]); err == nil {
			return time.Duration(delay) * time.Second
		}
	}
	return 0
}
