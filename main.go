package cache

import (
	"bufio"
	"bytes"
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
)

type AlwaysCache struct {
	cache CacheProvider
	next  http.Handler
}

// Middleware returns a new instance of AlwaysCache.
// AlwaysCache itself is a http.Handler, so it can be used as a middleware.
func Middleware(next http.Handler) http.Handler {
	return &AlwaysCache{cache: NewMemCache(), next: next}
}

// ServeHTTP implements the http.Handler interface.
// It is the main entry point for the caching middleware.
func (a *AlwaysCache) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := getLogger(r)
	key := getKey(r)

	if isCacheable(r) {
		// check if we have a cached version
		if cachedResponse, ok, _ := a.cache.Get(key); ok {
			logger.Trace().Str("key", key).Msg("Cache hit and serving")
			resp, err := bytesToResponse(cachedResponse)
			if err != nil {
				http.Error(w, "Cannot read response", http.StatusInternalServerError)
				return
			}
			copyHeadersTo(w.Header(), resp.Header)
			io.Copy(w, resp.Body)
		} else {
			a.saveToCache(w, r, logger)
		}
	} else {
		rw := NewResponseSaver(w)
		a.next.ServeHTTP(rw, r)
		// update cache for the GET of the same URL
		logger.Trace().Str("key", key).Msg("Updating cache for self")
		req, _ := http.NewRequest("GET", r.URL.RequestURI(), nil)
		a.saveToCache(nil, req, logger)
		// update cache based on cache-update header
		for _, update := range rw.Updates() {
			logger.Trace().Str("key", key).Msgf("Updating cache for %s based on header", update)
			req, _ := http.NewRequest("GET", update, nil)
			a.saveToCache(nil, req, logger)
		}
	}
}

// saveToCache saves the response to a particular request `r` to the cache, if the response is cachable.
// The response is also tee'd to the `w` ResponseWriter.
// It returns a boolean indicating if the response was cached, along with a possible error.
// It uses the underlying `next` handler to get the response.
// `ResponseSaver` is used to save the response to the cache and to tee the response to the `w` ResponseWriter.
func (a *AlwaysCache) saveToCache(w http.ResponseWriter, r *http.Request, logger *zerolog.Logger) (bool, error) {
	rw := NewResponseSaver(w)
	a.next.ServeHTTP(rw, r)
	key := getKey(r)
	if doCache, expiry := shouldCache(rw); doCache {
		if err := a.cache.Put(key, expiry, rw.Response()); err != nil {
			logger.Error().Err(err).Str("key", key).Msg("Could not write to cache")
			return false, err
		}
		logger.Trace().Str("key", key).Time("expiry", expiry).Msg("Cache write")
		return true, nil
	}
	logger.Trace().Str("key", key).Msg("Non-cacheable response")
	return false, nil
}

// getLogger returns the logger from the request context.
// If no logger is found, it will return the default logger.
func getLogger(r *http.Request) *zerolog.Logger {
	logger := hlog.FromRequest(r)
	if logger.GetLevel() == zerolog.Disabled {
		logger = &log.Logger
	}
	return logger
}

// getKey returns the cache key for a request.
func getKey(r *http.Request) string {
	return r.URL.RequestURI()
}

// isCacheable checks if the request is cachable.
func isCacheable(r *http.Request) bool {
	return r.Method == "GET"
}

// shouldCache checks if the response should be cached.
// If the response is cachable, it will return true, along with the expiration time.
func shouldCache(rw *ResponseSaver) (bool, time.Time) {
	if rw.StatusCode() != http.StatusOK {
		return false, time.Time{}
	}
	cacheControl := rw.Header().Get("Cache-Control")
	maxAge := time.Hour
	if matches := regexp.MustCompile(`(?i)\bmax-age=(\d+)`).FindStringSubmatch(cacheControl); matches != nil {
		if duration, err := time.ParseDuration(matches[1] + "s"); err == nil {
			maxAge = duration
		}
	}
	return true, time.Now().Add(maxAge)
}

// bytesToResponse converts a byte slice to a http.Response.
func bytesToResponse(b []byte) (*http.Response, error) {
	return http.ReadResponse(bufio.NewReader(bytes.NewReader(b)), nil)
}

// copyHeadersTo copies the headers from one http.Header to another.
func copyHeadersTo(dst, src http.Header) {
	for name, values := range src {
		for _, value := range values {
			dst.Set(name, value)
		}
	}
}
