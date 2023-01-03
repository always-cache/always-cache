package main

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
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
	"github.com/ericselin/always-cache/rfc9211"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
)

type AlwaysCache struct {
	initialized   bool
	port          int
	cache         CacheProvider
	originURL     *url.URL
	originHost    string
	updateTimeout time.Duration
	rules         Rules
	client        http.Client
	// LEGACY MODE
	// Only invalidate cache, i.e. do not update cache on invalidation
	invalidateOnly bool
}

// Init initializes the always-cache instance.
// It starts the needed background processes
// and sets up the needed variables
func (a *AlwaysCache) init() {
	if a.initialized {
		return
	}
	a.initialized = true

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

	// use provided hostname for origin if configured
	if a.originHost != "" {
		a.client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				ServerName: a.originHost,
			},
		}
	}
}

func (a *AlwaysCache) Run() error {
	// initialize
	a.init()
	// start the server
	log.Info().Msgf("Proxying port %v to %s with hostname %s", a.port, a.originURL, a.originHost)
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
	if r.URL.Path == "/.acache-update" {
		go a.updateAll()
		w.WriteHeader(http.StatusAccepted)
		io.WriteString(w, "Updating all content...")
		return
	}

	log.Trace().Interface("headers", r.Header).Msgf("Incoming request: %s %s", r.Method, r.URL.Path)

	keyPrefix := getKeyPrefix(r)

	log := log.With().Str("key", keyPrefix).Logger()
	var cacheStatus rfc9211.CacheStatus
	var responseIfValidated *http.Response

	if responses, err := a.getResponses(r); err != nil {
		log.Warn().Err(err).Msg("Error getting responses")
	} else if len(responses) > 0 {
		for _, sRes := range responses {
			res, validationReq, fwdReason := rfc9111.ConstructReusableResponse(r, sRes.response, sRes.requestTime, sRes.responseTime)
			if fwdReason == "" {
				cacheStatus.Hit()
				send(w, res, cacheStatus)
				return
			}
			cacheStatus.Forward(fwdReason)
			if validationReq != nil {
				log.Trace().Msgf("Response is ok as long as it is validated with %+v", validationReq)
				responseIfValidated = res
				r = validationReq
			}
		}
	} else {
		cacheStatus.Forward(rfc9211.FwdReasonUriMiss)
	}

	upstreamRequest := rfc9111.GetForwardRequest(r)

	log.Trace().Msg("Forwarding to origin")
	res, err := a.fetch(upstreamRequest)
	if err != nil {
		http.Error(w, "Error contacting origin", http.StatusBadGateway)
		log.Error().Err(err).Msg("Could not fetch response from server")
		return
	}
	log.Trace().Msg("Got response from origin")

	if responseIfValidated != nil && res.response.StatusCode == http.StatusNotModified {
		send(w, responseIfValidated, cacheStatus)
		return
	}

	a.rules.Apply(res.response)

	downstreamResponse, mayStore := rfc9111.ConstructDownstreamResponse(r, res.response)
	res.response = downstreamResponse

	if mayStore {
		key := addVaryKeys(keyPrefix, r, res.response)
		a.save(key, res)
	}

	send(w, downstreamResponse, cacheStatus)

	a.updateIfNeeded(r, res.response)
}

func (a *AlwaysCache) getResponses(r *http.Request) ([]timedResponse, error) {
	prefix := getKeyPrefix(r)
	if entries, err := a.cache.All(prefix); err == nil && len(entries) > 0 {
		log.Trace().Str("key", prefix).Msg("Found cached response(s)")
		responses := make([]timedResponse, 0, len(entries))
		for _, e := range entries {
			if res, err := bytesToStoredResponse(e.Bytes); err == nil {
				responses = append(responses, res)
			}
		}
		return responses, nil
	} else {
		return []timedResponse{}, err
	}
}

func (a *AlwaysCache) updateIfNeeded(downReq *http.Request, upRes *http.Response) {
	if a.invalidateOnly {
		a.invalidateUris(
			rfc9111.GetInvalidateURIs(downReq, upRes))
	} else {

		a.revalidateUris(
			rfc9111.GetInvalidateURIs(downReq, upRes))
	}
	a.saveUpdates(
		getUpdateHeaderUpdates(downReq, upRes))
}

type CacheUpdate struct {
	Path  string
	Delay time.Duration
}

func getUpdateHeaderUpdates(clientRequest *http.Request, res *http.Response) []CacheUpdate {
	if !rfc9111.UnsafeRequest(clientRequest) {
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

func (a *AlwaysCache) saveUpdates(updates []CacheUpdate) {
	for _, update := range updates {
		log.Trace().Str("update", update.Path).Msgf("Updating cache based on header")
		updateCache := func() {
			req, err := http.NewRequest("GET", update.Path, nil)
			if err != nil {
				log.Error().Err(err).Str("path", update.Path).Msg("Could not create request for updates")
				return
			}
			_, err = a.saveRequest(req, getKeyPrefix(req))
			if err != nil {
				log.Error().Err(err).Str("path", update.Path).Msg("Could not save updates")
				return
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

func (a *AlwaysCache) revalidateUris(uris []string) {
	for _, uri := range uris {
		log.Trace().Str("uri", uri).Msgf("Revalidating possibly stored response")
		req, err := http.NewRequest("GET", uri, nil)
		if err != nil {
			log.Error().Err(err).Str("uri", uri).Msg("Could not create request for revalidation")
			continue
		}
		key := getKeyPrefix(req)
		if a.cache.Has(key) {
			_, err := a.saveRequest(req, key)
			if err != nil {
				log.Error().Err(err).Str("key", key).Msg("Error revalidating stored request")
			}
		}
	}
}

func (a *AlwaysCache) invalidateUris(uris []string) {
	for _, uri := range uris {
		log.Trace().Str("uri", uri).Msgf("Invalidating stored response")
		req, err := http.NewRequest("GET", uri, nil)
		if err != nil {
			log.Error().Err(err).Str("uri", uri).Msg("Could not create request for invalidation")
			continue
		}
		a.cache.Purge(getKeyPrefix(req))
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
	timedRes := timedResponse{requestTime: time.Now()}
	uri := a.originURL.String() + r.URL.RequestURI()
	req, err := http.NewRequest(r.Method, uri, r.Body)
	if err != nil {
		log.Error().Err(err).Str("uri", uri).Msg("Could not create request for fetching")
		return timedRes, err
	}
	req.Host = a.originHost
	copyHeader(req.Header, r.Header)
	// do not forward connection header, this causes trouble
	// bug surfaced it cache-tests headers-store-Connection test
	req.Header.Del("Connection")
	log.Trace().Msgf("Executing request %+v", *req)

	originResponse, err := a.client.Do(req)
	timedRes.responseTime = time.Now()
	// as per https://www.rfc-editor.org/rfc/rfc9110#section-6.6.1-8
	if err == nil && originResponse.Header.Get("Date") == "" {
		originResponse.Header.Set("Date", rfc9111.ToHttpDate(time.Now()))
	}
	timedRes.response = originResponse
	return timedRes, err
}

func send(w http.ResponseWriter, r *http.Response, status rfc9211.CacheStatus) error {
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

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		// this is a warkaround to remove default headers sent by an upstream proxy
		// some servers do not like the presence of these headers in the downstream request
		if k != "X-Forwarded-For" && k != "X-Forwarded-Proto" && k != "X-Forwarded-Host" {
			for _, v := range vv {
				dst.Add(k, v)
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
			req, err := getRequestFromKey(key)
			if err == nil {
				log.Trace().Str("key", key).Str("req.path", req.URL.Path).Time("expiry", expiry).Msg("Updating cache")
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
			} else if err == errorMethodNotSupported {
				log.Trace().Err(err).Str("key", key).Msg("Method not supported")
			} else {
				log.Error().Err(err).Str("key", key).Msg("Could not create request from key")
			}
			if err != nil {
				a.cache.Purge(key)
			}
		} else {
			log.Trace().Msg("No entries expiring, pausing update")
			time.Sleep(a.updateTimeout)
		}
	}
}

func (a *AlwaysCache) updateAll() {
	a.cache.Keys(func(key string) {
		log.Debug().Msgf("Updating key %s", key)
		req, err := http.NewRequest("GET", key, nil)
		if err != nil {
			log.Warn().Err(err).Str("key", key).Msg("Could not create request for updating all")
			return
		}
		cached, err := a.saveRequest(req, key)
		if err != nil {
			log.Warn().Err(err).Str("key", key).Msg("Could not save request")
		} else if !cached {
			log.Debug().Str("key", key).Msg("Update not cached")
		}
	})
}

func (a *AlwaysCache) saveRequest(req *http.Request, key string) (bool, error) {
	res, err := a.fetch(req)
	if err != nil {
		return false, err
	}
	if dRes, mayStore := rfc9111.ConstructDownstreamResponse(req, res.response); mayStore {
		res.response = dRes
		a.rules.Apply(res.response)
		a.save(key, res)
		return true, nil
	} else if err != nil {
		return false, err
	}
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

// getKeyPrefix returns the cache key for a request.
// If it is a GET request, it will return the URL.
// If it is a POST request, it will return the URL combined with a hash of the body.
func getKeyPrefix(r *http.Request) string {
	key := r.Method + ":" + r.URL.RequestURI() + "\t"
	if r.Method == "POST" {
		if multipartHash := multipartHash(r); multipartHash != "" {
			return key + multipartHash
		} else if bodyHash := bodyHash(r); bodyHash != "" {
			return key + bodyHash
		}
	}
	return key
}

func addVaryKeys(prefix string, req *http.Request, res *http.Response) string {
	key := prefix
	for _, name := range rfc9111.GetListHeader(res.Header, "Vary") {
		if !rfc9111.FieldAbsent(req.Header, name) {
			key = key + "\n" + strings.ToLower(name) + ": " + req.Header.Get(name)
		}
	}
	return key
}

var errorMethodNotSupported = fmt.Errorf("Method not supported")

func getRequestFromKey(key string) (*http.Request, error) {
	if !strings.HasPrefix(key, "GET:") {
		return nil, errorMethodNotSupported
	}
	uri := strings.TrimSpace(strings.TrimLeft(key, "GET:"))
	return http.NewRequest("GET", uri, nil)
}

func getVaryHeaders(key string) http.Header {
	header := make(http.Header)
	lines := strings.Split(key, "\n")
	for i := 1; i < len(lines); i++ {
		entry := strings.SplitN(lines[i], ": ", 2)
		header.Add(entry[0], entry[1])
	}
	return header
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
