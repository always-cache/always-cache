package cache

import (
	"bufio"
	"bytes"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
)

type AlwaysCache struct {
	cache         CacheProvider
	next          http.Handler
	defaultMaxAge time.Duration
	updateTimeout time.Duration
}

type Config struct {
	Cache         CacheProvider
	DefaultMaxAge time.Duration
	UpdateTimeout time.Duration
}

// NewAlwaysCache instantiates an AlwaysCache with the given config.
// The returned `AlwaysCache` is a `http.Handler` and can be used as a middleware.
func New(c Config) *AlwaysCache {
	acache := AlwaysCache{
		cache:         c.Cache,
		defaultMaxAge: c.DefaultMaxAge,
		updateTimeout: c.UpdateTimeout,
	}
	if acache.cache == nil {
		acache.cache = NewMemCache()
	}
	if acache.defaultMaxAge == 0 {
		acache.defaultMaxAge = time.Hour
	}
	if acache.updateTimeout == 0 {
		acache.updateTimeout = time.Minute
	}
	return &acache
}

// Middleware returns a new instance of AlwaysCache.
// AlwaysCache itself is a http.Handler, so it can be used as a middleware.
func (a *AlwaysCache) Middleware(next http.Handler) http.Handler {
	// set downstream handler
	a.next = next
	// TODO start a goroutine to warm up cache
	// start a goroutine to update expired entries
	go a.updateCache()
	return a
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
			url := getURL(r, update)
			delay := getDelay(update)
			logger.Trace().Str("key", key).Dur("delay", delay).Msgf("Updating cache for %s based on header", url.Path)
			req, _ := http.NewRequest("GET", url.Path, nil)
			if delay > 0 {
				go func() {
					time.Sleep(delay)
					a.saveToCache(nil, req, logger)
				}()
			} else {
				a.saveToCache(nil, req, logger)
			}
		}
	}
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

// saveToCache saves the response to a particular request `r` to the cache, if the response is cachable.
// The response is also tee'd to the `w` ResponseWriter.
// It returns a boolean indicating if the response was cached, along with a possible error.
// It uses the underlying `next` handler to get the response.
// `ResponseSaver` is used to save the response to the cache and to tee the response to the `w` ResponseWriter.
func (a *AlwaysCache) saveToCache(w http.ResponseWriter, r *http.Request, logger *zerolog.Logger) (bool, error) {
	rw := NewResponseSaver(w)
	a.next.ServeHTTP(rw, r)
	key := getKey(r)
	if doCache, expiry := a.shouldCache(rw); doCache {
		if err := a.cache.Put(key, expiry, rw.Response()); err != nil {
			logger.Error().Err(err).Str("key", key).Msg("Could not write to cache")
			return false, err
		}
		logger.Trace().Str("key", key).Time("expiry", expiry).Msg("Cache write")
		return true, nil
	}
	logger.Trace().Str("key", key).Int("http-status", rw.StatusCode()).Msg("Non-cacheable response")
	return false, nil
}

// shouldCache checks if the response should be cached.
// If the response is cachable, it will return true, along with the expiration time.
func (a *AlwaysCache) shouldCache(rw *ResponseSaver) (bool, time.Time) {
	if rw.StatusCode() != http.StatusOK {
		return false, time.Time{}
	}
	cacheControl := rw.Header().Get("Cache-Control")
	maxAge := a.defaultMaxAge
	if matches := regexp.MustCompile(`(?i)\bmax-age=(\d+)`).FindStringSubmatch(cacheControl); matches != nil {
		if duration, err := time.ParseDuration(matches[1] + "s"); err == nil {
			maxAge = duration
		}
	}
	return true, time.Now().Add(maxAge)
}

// updateCache runs an infinite loop to update the cache,
// one entry at a time.
// It assumes that the cache key equals the request URL.
// It will query the cache for entries expiring within the update timeout.
// If it finds one, it will update the cache for that entry.
// If it does not find any, it will sleep for the duration of the update timeout.
func (a *AlwaysCache) updateCache() {
	log.Info().Msgf("Starting cache update loop with timeout %s", a.updateTimeout)
	for {
		key, expiry, err := a.cache.Oldest()
		// if error, try again in 1 minute
		if err != nil {
			log.Error().Err(err).Msg("Could not get oldest entry")
			time.Sleep(a.updateTimeout)
			continue
		}
		// if expiring within 1 minute, update
		// else sleep for 1 minute
		if key != "" && expiry.Sub(time.Now()) <= a.updateTimeout {
			log.Trace().Str("key", key).Time("expiry", expiry).Msg("Updating cache")
			req, _ := http.NewRequest("GET", key, nil)
			a.saveToCache(nil, req, &log.Logger)
		} else {
			log.Trace().Msg("No entries expiring, pausing update")
			time.Sleep(a.updateTimeout)
		}
	}
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
