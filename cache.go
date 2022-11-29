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

	"github.com/ericselin/always-cache/rfc9111"

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

// ServeHTTP implements the http.Handler interface.
// It is the main entry point for the caching middleware.
//
// - get matching response(s)
// - select suitable response, if none goto store
// - construct response
// - store: check we may store
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

	if responses, err := a.getResponses(r); err == nil {
		for _, sRes := range responses {
			if res := rfc9111.ConstructReusableResponse(r, sRes.response, sRes.requestTime, sRes.responseTime); res != nil {
				cacheStatus.Hit()
				send(w, res, cacheStatus)
				return
			}
		}
	} else {
		log.Warn().Err(err).Msg("Error getting responses")
	}

	log.Trace().Msg("Forwarding to origin")

	res, err := a.fetch(r)
	originResponse := res.response
	if err != nil {
		panic(err)
	}

	// set default cache-control header if nothing specified by origin
	if a.defaults.CacheControl != "" && originResponse.Header.Get("Cache-Control") == "" {
		originResponse.Header.Set("Cache-Control", a.defaults.CacheControl)
	}

	// if MUST NOT store according to spec, just forward the response
	if noStore, err := rfc9111.MustNotStore(r, originResponse); noStore || err != nil {
		send(w, originResponse, cacheStatus)
		// log possible error
		if err != nil {
			log.Error().Err(err).Msg("Failed to determine if response may be stored")
		}
		return
	}

	log.Trace().Msg("Got response from origin")

	// remove connection header from the response
	originResponse.Header.Del("Connection")

	// we already checked that we may store the response
	a.save(key, res)

	send(w, originResponse, cacheStatus)

	if updates := getUpdates(originResponse); len(updates) > 0 {
		a.saveUpdates(updates)
	}
}

func (a *AlwaysCache) getResponses(r *http.Request) ([]timedResponse, error) {
	key := getKey(r)
	if cachedBytes, ok, _ := a.cache.Get(key); ok {
		log.Trace().Str("key", key).Msg("Found cached response")
		res, err := bytesToStoredResponse(cachedBytes)
		return []timedResponse{res}, err
	}
	return []timedResponse{}, nil
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
		updateCache := func() {
			req, _ := http.NewRequest("GET", update.Path, nil)
			_, err := a.saveRequest(req, getKey(req))
			if err != nil {
				panic(err)
			}
		}
		if update.Delay > 0 {
			go func() {
				time.Sleep(update.Delay)
				updateCache()
			}()
		} else {
			updateCache()
		}
	}
}

func (a *AlwaysCache) save(key string, sRes timedResponse) {
	responseBytes, err := storedResponseToBytes(sRes)
	if err != nil {
		panic(err)
	}
	exp := rfc9111.GetExpiration(sRes.response)
	if !exp.IsZero() {
		if err := a.cache.Put(key, exp, responseBytes); err != nil {
			log.Error().Err(err).Str("key", key).Msg("Could not write to cache")
			panic(err)
		}
		log.Trace().Str("key", key).Time("expiry", exp).Msg("Cache write")
	}
}

// fetch the resource specified in the incoming request from the origin
func (a *AlwaysCache) fetch(r *http.Request) (timedResponse, error) {
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

	timedRes := timedResponse{requestTime: time.Now()}
	originResponse, err := a.client.Do(req)
	timedRes.responseTime = time.Now()
	// as per https://www.rfc-editor.org/rfc/rfc9110#section-6.6.1-8
	if err == nil && originResponse.Header.Get("Date") == "" {
		originResponse.Header.Set("Date", rfc9111.ToHttpDate(time.Now()))
	}
	timedRes.response = originResponse
	return timedRes, err
}

func send(w http.ResponseWriter, r *http.Response, status CacheStatus) error {
	log.Trace().Msg("Sending response")
	if r.Body != nil {
		defer r.Body.Close()
	}
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
	if noStore, err := rfc9111.MustNotStore(req, res.response); !noStore && err == nil {
		a.save(key, res)
		return true, nil
	} else if err != nil {
		return false, err
	}
	return false, nil
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
	key := r.Method + ":" + r.URL.RequestURI()
	if r.Method == "POST" {
		if multipartHash := multipartHash(r); multipartHash != "" {
			return key + ":" + multipartHash
		} else if bodyHash := bodyHash(r); bodyHash != "" {
			return key + ":" + bodyHash
		}
	}
	return key
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
