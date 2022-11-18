package main

import (
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
	initialized   bool
	port          int
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
func (a *AlwaysCache) init() {
	if a.initialized {
		return
	}
	a.initialized = true

	log.Trace().Msgf("Defaults: %+v", a.defaults)
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

func (a *AlwaysCache) Run() error {
	// initialize
	a.init()
	// start the server
	log.Info().Msgf("Proxying port %v to %s", a.port, a.originURL)
	return http.ListenAndServe(fmt.Sprintf(":%d", a.port), a)
}

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

// ServeHTTP implements the http.Handler interface.
// It is the main entry point for the caching middleware.
func (a *AlwaysCache) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// TODO defaults from path matches (that we get here) are not used yet!
	// defaults := a.getDefaults(r)

	log.Trace().Interface("headers", r.Header).Msgf("Incoming request: %s %s", r.Method, r.URL.Path)

	key := getKey(r)

	log := log.With().Str("key", key).Logger()
	var cacheStatus CacheStatus

	if testId := r.Header.Get("test-id"); testId != "" {
		log.Debug().Str("testId", testId).Msg("Request for test")
	}

	// see if this request is cacheble, per configuration
	// (do not send cached results even if we have a cached entry if the config changed)
	if a.isCacheable(r) {
		// check if we have a cached version
		if cachedBytes, ok, _ := a.cache.Get(key); ok {
			log.Trace().Str("key", key).Msg("Cache hit and serving")
			if cachedResponse, err := bytesToResponse(cachedBytes); err == nil {
				cacheStatus.Hit()
				send(w, cachedResponse, cacheStatus)
				return
			} else {
				cacheStatus.Forward(CacheStatusFwdBypass)
				cacheStatus.Detail("error")
				log.Error().Err(err).Str("key", key).Msg("Could not read from cache")
				a.cache.Purge(key)
			}
		} else {
			cacheStatus.Forward(CacheStatusFwdMiss)
		}
	} else {
		cacheStatus.Forward(CacheStatusFwdRequest)
	}

	log.Trace().Msg("Forwarding to origin")

	originResponse, err := a.fetch(r)
	if err != nil {
		panic(err)
	}

	log.Trace().Msg("Got response from origin")

	// remove connection header from the response
	originResponse.Header.Del("Connection")

	if cacheable, expiration := a.shouldCache(originResponse); cacheable {
		a.save(key, originResponse, expiration)
	}

	send(w, originResponse, cacheStatus)

	if updates := getUpdates(originResponse); len(updates) > 0 {
		a.saveUpdates(updates)
	}
}

type CacheUpdate struct {
	Path  string
	Delay time.Duration
}

func getUpdates(res *http.Response) []CacheUpdate {
	if res.Request.Method == http.MethodGet {
		return nil
	}
	updates := make([]CacheUpdate, 0)
	// only auto-add self if success
	if res.StatusCode == http.StatusOK {
		updates = append(updates, CacheUpdate{Path: res.Request.RequestURI})
	}
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

func (a *AlwaysCache) saveUpdates(updates []CacheUpdate) {
	for _, update := range updates {
		log.Trace().Str("update", update.Path).Msgf("Updating cache based on header")
		req, _ := http.NewRequest("GET", update.Path, nil)
		if update.Delay > 0 {
			go func() {
				time.Sleep(update.Delay)
				updatedResponse, err := a.fetch(req)
				if err != nil {
					panic(err)
				}
				if cacheable, expiration := a.shouldCache(updatedResponse); cacheable {
					a.save(getKey(req), updatedResponse, expiration)
				}
			}()
		} else {
			updatedResponse, err := a.fetch(req)
			if err != nil {
				panic(err)
			}
			if cacheable, expiration := a.shouldCache(updatedResponse); cacheable {
				a.save(getKey(req), updatedResponse, expiration)
			}
		}
	}
}

func (a *AlwaysCache) save(key string, res *http.Response, exp time.Time) {
	responseBytes, err := responseToBytes(res)
	if err != nil {
		panic(err)
	}
	if err := a.cache.Put(key, exp, responseBytes); err != nil {
		log.Error().Err(err).Str("key", key).Msg("Could not write to cache")
		panic(err)
	}
	log.Trace().Str("key", key).Time("expiry", exp).Msg("Cache write")
}

// fetch the resource specified in the incoming request from the origin
func (a *AlwaysCache) fetch(r *http.Request) (*http.Response, error) {
	req, err := http.NewRequest(r.Method, a.originURL.String()+r.URL.RequestURI(), r.Body)
	copyHeader(req.Header, r.Header)
	req.Header.Set("Host", a.originURL.Host)
	// do not forward connection header, this causes trouble
	// bug surfaced it cache-tests headers-store-Connection test
	req.Header.Del("Connection")
	if err != nil {
		panic(err)
	}
	log.Trace().Msgf("Executing request %+v", *req)
	return a.client.Do(req)
}

func send(w http.ResponseWriter, r *http.Response, status CacheStatus) error {
	log.Trace().Msg("Sending response")
	defer r.Body.Close()
	copyHeader(w.Header(), r.Header)
	w.Header().Add("Cache-Status", status.String())
	w.WriteHeader(r.StatusCode)
	bytesWritten, err := io.Copy(w, r.Body)
	log.Trace().Msgf("Wrote body (%d bytes)", bytesWritten)
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

// shouldCache checks if the response should be cached.
// If the response is cachable, it will return true, along with the expiration time.
func (a *AlwaysCache) shouldCache(res *http.Response) (bool, time.Time) {
	// cache only success (HTTP 200)
	if res.StatusCode != http.StatusOK {
		return false, time.Time{}
	}

	cacheControl := res.Header.Get("Cache-Control")
	if cacheControl == "" {
		cacheControl = a.defaults.CacheControl
	}
	cc := ParseCacheControl(cacheControl)

	// should not cache if no-cache set
	if _, ok := cc.Get("no-cache"); ok {
		return false, time.Time{}
	}

	var expires time.Time

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

	// if we got a max-age, set expiry as appropriate
	if maxAge != 0 {
		expires = time.Now().Add(maxAge)
	}

	// if no max age specified, see if we have expires header
	if maxAge == 0 {
		if expiresHeader := res.Header.Get("Expires"); expiresHeader != "" {
			if expTime, err := time.Parse(time.RFC1123, expiresHeader); err == nil {
				expires = expTime
			} else {
				log.Trace().Err(err).Msg("Error parsing expires header")
			}
		}
	}

	// do not cache if expiry not set
	if expires.IsZero() {
		return false, time.Time{}
	}

	// do not cache if expiry happens within the update timeout
	if expires.Before(time.Now().Add(a.updateTimeout)) {
		log.Trace().Msgf("Max age %s less than update timeout %s", maxAge, a.updateTimeout)
		return false, time.Time{}
	}

	return true, expires
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
			cached, err := a.saveRequest(req, key)
			// if there was an error, sleep and retry
			if !cached || err != nil {
				time.Sleep(time.Second)
				cached, err = a.saveRequest(req, key)
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

func (a *AlwaysCache) saveRequest(req *http.Request, key string) (bool, error) {
	res, err := a.fetch(req)
	if err != nil {
		return false, err
	}
	if cacheable, exp := a.shouldCache(res); cacheable {
		a.save(key, res, exp)
		return true, nil
	}
	return false, nil
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
	defaults := a.getDefaults(r)
	if defaults.SafeMethods.Has(r.Method) {
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
		} else if bodyHash := bodyHash(r); bodyHash != "" {
			return r.URL.RequestURI() + ":" + bodyHash
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
	if r.Body == nil {
		return ""
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	r.Body = ioutil.NopCloser(bytes.NewBuffer(body))
	return fmt.Sprintf("%x", sha256.Sum256(body))
}

// copyHeadersTo copies the headers from one http.Header to another.
func copyHeadersTo(dst, src http.Header) {
	for name, values := range src {
		for _, value := range values {
			dst.Set(name, value)
		}
	}
}
