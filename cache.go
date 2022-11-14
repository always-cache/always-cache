package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
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
	originURL     *url.URL
	updateTimeout time.Duration
	defaults      Defaults
	paths         []Path
	client        http.Client
}

type Path struct {
	Prefix   string   `yaml:"prefix"`
	Defaults Defaults `yaml:"defaults"`
}

type Defaults struct {
	CacheControl string      `yaml:"cacheControl"`
	SafeMethods  SafeMethods `yaml:"safeMethods"`
}

type SafeMethods struct {
	m map[string]struct{}
}

func (m SafeMethods) Has(method string) bool {
	_, ok := m.m[method]
	return ok
}

// Init initializes the always-cache instance.
// It starts the needed background processes
// and sets up the needed variables
func (a *AlwaysCache) Init() {
	// start a goroutine to update expired entries
	if a.updateTimeout != 0 {
		go a.updateCache()
	}

	// create client instance to use for origin requests
	a.client = http.Client{
		// do not follow redirects
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// ServeHTTP implements the http.Handler interface.
// It is the main entry point for the caching middleware.
func (a *AlwaysCache) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := getLogger(r)

	if fwdToken := a.shouldBypass(r); fwdToken != "" {
		// TODO this header should be added after potential other cache-status headers
		w.Header().Add("Cache-Status", fmt.Sprintf("Always-Cache; fwd=%s", fwdToken))
		err := a.bypass(w, r)
		if err != nil {
			http.Error(w, "Could not get response", http.StatusBadGateway)
		}
		return
	}

	// TODO defaults from path matches (that we get here) are not used yet!
	// defaults := a.getDefaults(r)

	if a.isCacheable(r) {
		key := getKey(r)
		// check if we have a cached version
		if cachedResponse, ok, _ := a.cache.Get(key); ok {
			logger.Trace().Str("key", key).Msg("Cache hit and serving")
			resp, err := bytesToResponse(cachedResponse)
			if err != nil {
				// in case we have a corrupted cache entry, we delete it and serve the request
				logger.Error().Err(err).Str("key", key).Msg("Could not read from cache")
				a.bypass(w, r)
				a.cache.Purge(key)
				return
			}
			copyHeadersTo(w.Header(), resp.Header)
			w.Header().Add("Cache-Status", "Always-Cache; hit")
			io.Copy(w, resp.Body)
		} else {
			// TODO for cachable POSTs, this call will trigger another read of the body, which is not ideal
			a.saveToCache(w, r, logger)
		}
	} else {
		rw := NewResponseSaver(w)
		a.bypass(rw, r)
		// update cache for the GET of the same URL
		logger.Trace().Str("path", r.URL.Path).Msg("Updating cache for self")
		req, _ := http.NewRequest("GET", r.URL.RequestURI(), nil)
		a.saveToCache(nil, req, logger)
		// update cache based on cache-update header
		for _, update := range rw.Updates() {
			url := getURL(r, update)
			delay := getDelay(update)
			logger.Trace().Str("path", r.URL.Path).Dur("delay", delay).Msgf("Updating cache for %s based on header", url.Path)
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

// fetch the resource specified in the incoming request from the origin
func (a *AlwaysCache) fetch(r *http.Request) (*http.Response, error) {
	req, err := http.NewRequest(r.Method, a.originURL.String()+r.URL.RequestURI(), r.Body)
	copyHeader(req.Header, r.Header)
	req.Header.Set("Host", a.originURL.Host)
	if err != nil {
		panic(err)
	}
	return a.client.Do(req)
}

// bypass just pipes the original request through to the origin and immediately responds to the client
func (a *AlwaysCache) bypass(w http.ResponseWriter, r *http.Request) error {
	res, err := a.fetch(r)
	if err != nil {
		return err
	}
	return send(w, res)
}

func send(w http.ResponseWriter, r *http.Response) error {
	defer r.Body.Close()
	copyHeader(w.Header(), r.Header)
	w.WriteHeader(r.StatusCode)
	_, err := io.Copy(w, r.Body)
	return err
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
	// need to get key before calling next, because next might change the request, and will definitely read the body
	key := getKey(r)

	a.bypass(rw, r)

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
	// cache only success (HTTP 200)
	if rw.StatusCode() != http.StatusOK {
		return false, time.Time{}
	}

	cacheControl := rw.Header().Get("Cache-Control")
	if cacheControl == "" {
		cacheControl = a.defaults.CacheControl
	}
	cc := ParseCacheControl(cacheControl)

	// should not cache if no-cache set
	if _, ok := cc.Get("no-cache"); ok {
		return false, time.Time{}
	}

	// get max age in order: s-maxage, max-age, DEFAULT
	var maxAgeStr string
	if val, ok := cc.Get("s-maxage"); ok {
		maxAgeStr = val
	} else if val, ok := cc.Get("max-age"); ok {
		maxAgeStr = val
	}

	var maxAge time.Duration
	if maxAgeStr != "" {
		if duration, err := time.ParseDuration(maxAgeStr + "s"); err == nil {
			maxAge = duration
		}
	}

	// do not cache if max-age not set
	if maxAge == 0 {
		return false, time.Time{}
	}
	// do not cache if max age less than update interval
	if maxAge < a.updateTimeout {
		log.Trace().Msgf("Max age %s less than update timeout %s", maxAge, a.updateTimeout)
		return false, time.Time{}
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
			cached, err := a.saveToCache(nil, req, &log.Logger)
			// if there was an error, sleep and retry
			if !cached || err != nil {
				time.Sleep(time.Second)
				cached, err = a.saveToCache(nil, req, &log.Logger)
			}
			if !cached {
				a.cache.Purge(key)
			}
			if err != nil {
				log.Warn().Err(err).Str("key", key).Msg("Could not update cache entry")
			}
		} else {
			log.Trace().Msg("No entries expiring, pausing update")
			time.Sleep(a.updateTimeout)
		}
	}
}

// shouldBypass provides a very early hint that the request should be completely
// disregarded by the cache, and should just be passed along without any processing
// I.e. bypass cache completely.
// Returns a non-empty string containing the forward reason if cache should be bypassed.
func (a *AlwaysCache) shouldBypass(r *http.Request) string {
	if r.Header.Get("Authorization") != "" {
		return "method"
	}

	return ""
}

// isCacheable checks if the request is cachable.
func (a *AlwaysCache) isCacheable(r *http.Request) bool {
	if a.defaults.SafeMethods.Has(r.Method) {
		return true
	}

	return r.Method == "GET"
}

// getDefaults gets the configuration for the requested path,
// falling back to the global defaults if no paths match
func (a *AlwaysCache) getDefaults(r *http.Request) Defaults {
	for _, path := range a.paths {
		if strings.HasPrefix(r.URL.Path, path.Prefix) {
			return path.Defaults
		}
	}
	return a.defaults
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
// If it is a GET request, it will return the URL.
// If it is a POST request, it will return the URL combined with a hash of the body.
func getKey(r *http.Request) string {
	if r.Method == "POST" {
		if multipartHash := multipartHash(r); multipartHash != "" {
			return r.URL.RequestURI() + ":" + multipartHash
		} else {
			return r.URL.RequestURI() + ":" + bodyHash(r)
		}
	}
	return r.URL.RequestURI()
}

// multipartHash returns the hash of a multipart request body.
// It returns an empty string if the request is not multipart.
// When it returns, the request body will be rewound to the beginning.
func multipartHash(r *http.Request) string {
	mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		return ""
	}
	if strings.HasPrefix(mediaType, "multipart/") {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			panic(err)
		}
		r.Body = ioutil.NopCloser(bytes.NewBuffer(body))

		mr := multipart.NewReader(bytes.NewBuffer(body), params["boundary"])
		p, err := mr.NextPart()
		if err != nil {
			return ""
		}
		slurp, err := io.ReadAll(p)
		if err != nil {
			panic(err)
		}

		return fmt.Sprintf("%x", sha256.Sum256(slurp))
	}
	return ""
}

// bodyHash returns the hash of a request body.
// When it returns, the request body will be rewound to the beginning.
func bodyHash(r *http.Request) string {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	r.Body = ioutil.NopCloser(bytes.NewBuffer(body))
	return fmt.Sprintf("%x", sha256.Sum256(body))
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
