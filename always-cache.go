package alwayscache

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/always-cache/always-cache/cache"
	cachekey "github.com/always-cache/always-cache/pkg/cache-key"
	serializer "github.com/always-cache/always-cache/pkg/response-serializer"
	responsetransformer "github.com/always-cache/always-cache/pkg/response-transformer"
	tee "github.com/always-cache/always-cache/pkg/response-writer-tee"
	"github.com/always-cache/always-cache/rfc9111"
	"github.com/always-cache/always-cache/rfc9211"

	"github.com/rs/zerolog"
)

type Config struct {
	Cache      cache.CacheProvider
	OriginURL  url.URL
	OriginHost string
	Logger     *zerolog.Logger
	// Unique cache key identifier.
	// By default OriginURL will be used.
	CacheKey string
	// DEPRECATED: will be changed before v1
	UpdateTimeout time.Duration
	// DEPRECATED: will be changed before v1
	Rules responsetransformer.Rules
}

type AlwaysCache struct {
	cache         cache.CacheProvider
	keyer         cachekey.CacheKeyer
	log           zerolog.Logger
	updateTimeout time.Duration
	reverseproxy  httputil.ReverseProxy
}

// CreateCache initializes the always-cache instance.
// It starts the needed background processes
// and sets up the needed variables
func CreateCache(config Config) *AlwaysCache {
	// cache key is origin url if not set in config
	cacheKey := config.CacheKey
	if cacheKey == "" {
		cacheKey = config.OriginURL.String()
	}

	// use console logger if not specified in config
	var logger zerolog.Logger
	if config.Logger == nil {
		logger = zerolog.New(zerolog.NewConsoleWriter())
	} else {
		logger = *config.Logger
	}

	// create a child logger and add defaults
	logger = logger.With().
		Str("origin", config.OriginURL.String()).
		Logger()

	a := &AlwaysCache{
		cache:         config.Cache,
		keyer:         cachekey.NewCacheKeyer(cacheKey),
		log:           logger,
		updateTimeout: config.UpdateTimeout,
	}

	host := config.OriginURL.Host
	hostHeader := host
	transport := http.DefaultTransport
	if config.OriginHost != "" {
		hostHeader = config.OriginHost
		transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				ServerName: config.OriginHost,
			},
		}
	}

	a.reverseproxy = httputil.ReverseProxy{
		Director:       createDirector(config.OriginURL.Scheme, host, hostHeader),
		Transport:      transport,
		ModifyResponse: config.Rules.Apply,
	}

	// start a goroutine to update expired entries
	if a.updateTimeout != 0 {
		go a.updateCache()
	}

	return a
}

type requestModifier func(*http.Request)
type responseModifier func(*http.Response) error

func createRequestHelloWorlder(next requestModifier) requestModifier {
	return func(r *http.Request) {
		r.GetBody = func() (io.ReadCloser, error) {
			fmt.Printf("request %p url %p\n", r, r.URL)
			return io.NopCloser(strings.NewReader("hello world")), nil
		}
		r.Header.Add("Cache-Key", "hello world")
		if next != nil {
			next(r)
		}
	}
}

func createResponseHelloWorlder(next responseModifier) responseModifier {
	return func(res *http.Response) error {
		body, err := res.Request.GetBody()
		if err != nil {
			return err
		}
		buf := new(strings.Builder)
		_, err = io.Copy(buf, body)
		if err != nil {
			return err
		}
		str := buf.String()
		fmt.Println("request body", str)
		return next(res)
	}
}

type request struct {
	r           *http.Request
	cacheStatus rfc9211.CacheStatus
	log         zerolog.Logger
}

// ServeHTTP implements the http.Handler interface.
func (a *AlwaysCache) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	createRequestHelloWorlder(nil)(r)
	for _, ce := range a.getResponsesForUri(r) {
		if a.reuseOrValidate(w, r, ce) == "" {
			return
		}
	}
	a.proxy(w, r)
}

func (a *AlwaysCache) reuseOrValidate(w http.ResponseWriter, r *http.Request, ce cache.CacheEntry) rfc9211.FwdReason {
	res := a.createStoredResponse(ce)
	if fwdReason, validationReq, err := rfc9111.MustNotReuse(r, res, ce.RequestedAt, ce.ReceivedAt); err != nil {
		a.log.Error().Err(err).Msg("Could not determine reusability")
		return rfc9211.FwdReasonMiss
	} else if validationReq != nil {
		if a.sendToClientIfValidationFailed(w, r, validationReq) {
			return ""
		}
	} else if fwdReason != "" {
		return fwdReason
	}
	// if we get here, the response is ok to use
	cs := rfc9211.CacheStatus{}
	cs.Hit()
	a.sendStoredResponse(w, r, res, ce, cs)
	return cs.FwdReason
}

func (a *AlwaysCache) sendStoredResponse(w http.ResponseWriter, r *http.Request, res *http.Response, ce cache.CacheEntry, cacheStatus rfc9211.CacheStatus) {
	rfc9111.AddAgeHeader(res, ce.ReceivedAt, ce.RequestedAt)
	if res.Body != nil {
		defer res.Body.Close()
	}
	copyHeader(w.Header(), res.Header)
	w.Header().Add("Cache-Status", cacheStatus.String())
	w.WriteHeader(res.StatusCode)
	bytesWritten, err := io.Copy(w, res.Body)
	if err != nil {
		a.log.Error().Err(err).Msg("Could not write response body to client")
	}
	a.logRequest(r, cacheStatus)
	a.log.Trace().Msgf("Wrote body (%d bytes)", bytesWritten)
}

func (a *AlwaysCache) sendToClientIfValidationFailed(w http.ResponseWriter, clientRequest, validationReq *http.Request) bool {
	rwtee := tee.NewResponseSaver(w, http.StatusNotModified)
	a.reverseproxy.ServeHTTP(rwtee, validationReq)
	// all status codes other than 304 means the response was written to the client
	// the response will need to be saved as well
	if rwtee.StatusCode() != http.StatusNotModified {
		go a.updateResposeAndLocations(rwtee, clientRequest)
		return true
	}
	return false
}

func (a *AlwaysCache) createStoredResponse(ce cache.CacheEntry) *http.Response {
	originalReq, err := a.keyer.GetRequestFromKey(ce.Key)
	if err != nil {
		a.log.Error().Err(err).Msg("Could not get request from key")
		return nil
	}
	res, err := http.ReadResponse(bufio.NewReader(bytes.NewReader(ce.Bytes)), originalReq)
	if err != nil {
		a.log.Error().Err(err).Msg("Could not create response")
		return nil
	}
	return res
}

func (a *AlwaysCache) getResponsesForUri(r *http.Request) []cache.CacheEntry {
	keyUriPrefix := a.keyer.GetKeyPrefix(r)
	cacheEntries, err := a.cache.All(keyUriPrefix)
	if err != nil {
		a.log.Error().Err(err).Msg("Could not retrieve from cache")
		return nil
	}
	return cacheEntries
}

func (a *AlwaysCache) updateResposeAndLocations(rw *tee.ResponseSaver, r *http.Request) {
	a.writeCache(rw, r)
	a.updateIfNeeded(r, &http.Response{
		StatusCode: rw.StatusCode(),
		Header:     rw.Header(),
	})
}

func (a *AlwaysCache) proxy(w http.ResponseWriter, r *http.Request) {
	a.log.Trace().Msgf("proxying %s", r.URL.String())
	// set cache-status on underlying rw only (i.e. do not save to cache)
	cs := rfc9211.CacheStatus{}
	cs.Forward(rfc9211.FwdReasonUriMiss)
	w.Header().Add("Cache-Status", cs.String())

	rwtee := tee.NewResponseSaver(w)
	a.reverseproxy.ServeHTTP(rwtee, r)

	// log request
	go a.logRequest(r, cs)
	// save to cache in goroutine (do not slow down response)
	go a.updateResposeAndLocations(rwtee, r)
}

func (a *AlwaysCache) writeCache(rw *tee.ResponseSaver, r *http.Request) (bool, error) {
	res := &http.Response{
		Header:     rw.Header(),
		StatusCode: rw.StatusCode(),
		Request:    r,
	}
	if noStore, err := rfc9111.MustNotStore(res); err != nil {
		return false, err
	} else if noStore {
		return false, nil
	}
	keyPrefix := a.keyer.GetKeyPrefix(r)
	key := a.keyer.AddVaryKeys(keyPrefix, r, &http.Response{
		Header: rw.Header(),
	})
	exp := rfc9111.GetExpiration(res)
	ce := cache.CacheEntry{
		Key:         key,
		Expires:     exp,
		RequestedAt: rw.CreatedAt,
		ReceivedAt:  time.Now(),
		Bytes:       rw.Response(),
	}
	a.log.Trace().Msgf("Writing to cache: %v %v", key, exp)
	err := a.cache.PutCE(ce)
	return err == nil, err
}

func createDirector(scheme, host, hostHeader string) func(req *http.Request) {
	return func(req *http.Request) {
		req.URL.Scheme = scheme
		req.URL.Host = host
		if hostHeader != "" {
			req.Host = hostHeader
		}
	}
}

func (a *AlwaysCache) getResponses(r *http.Request) ([]serializer.TimedResponse, error) {
	prefix := a.keyer.GetKeyPrefix(r)
	if entries, err := a.cache.All(prefix); err == nil && len(entries) > 0 {
		a.log.Trace().Str("key", prefix).Msg("Found cached response(s)")
		responses := make([]serializer.TimedResponse, 0, len(entries))
		for _, e := range entries {
			if res, err := serializer.BytesToStoredResponse(e.Bytes); err == nil {
				responses = append(responses, res)
			}
		}
		return responses, nil
	} else {
		return []serializer.TimedResponse{}, err
	}
}

func (a *AlwaysCache) updateIfNeeded(downReq *http.Request, upRes *http.Response) {
	if a.updateTimeout == 0 {
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
		a.log.Trace().Str("update", update.Path).Msgf("Updating cache based on header")
		updateCache := func() {
			req, err := http.NewRequest("GET", update.Path, nil)
			if err != nil {
				a.log.Error().Err(err).Str("path", update.Path).Msg("Could not create request for updates")
				return
			}
			_, err = a.saveRequest(req, a.keyer.GetKeyPrefix(req))
			if err != nil {
				a.log.Error().Err(err).Str("path", update.Path).Msg("Could not save updates")
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
		a.log.Trace().Str("uri", uri).Msgf("Revalidating possibly stored response")
		req, err := http.NewRequest("GET", uri, nil)
		if err != nil {
			a.log.Error().Err(err).Str("uri", uri).Msg("Could not create request for revalidation")
			continue
		}
		key := a.keyer.GetKeyPrefix(req)
		if a.cache.Has(key) {
			_, err := a.saveRequest(req, key)
			if err != nil {
				a.log.Error().Err(err).Str("key", key).Msg("Error revalidating stored request")
			}
		}
	}
}

func (a *AlwaysCache) invalidateUris(uris []string) {
	for _, uri := range uris {
		a.log.Trace().Str("uri", uri).Msgf("Invalidating stored response")
		req, err := http.NewRequest("GET", uri, nil)
		if err != nil {
			a.log.Error().Err(err).Str("uri", uri).Msg("Could not create request for invalidation")
			continue
		}
		a.cache.Purge(a.keyer.GetKeyPrefix(req))
	}
}

func (a *AlwaysCache) logRequest(r *http.Request, cs rfc9211.CacheStatus) {
	isHit := 0
	if cs.FwdReason == "" {
		isHit = 1
	}
	a.log.Debug().
		Str("method", r.Method).
		Str("url", r.URL.String()).
		Str("sourceIp", getRequestSourceIp(r)).
		Str("status", string(cs.Status)).
		Str("fwd", string(cs.FwdReason)).
		Bool("stored", cs.Stored).
		Int("ttl", cs.TimeToLive).
		Int("hit", isHit).
		Msg("Sending response to client")
}

func getRequestSourceIp(r *http.Request) string {
	// RemoteAddr is in the format:
	// 1.2.3.4:10000 for ipv4
	// [1:2:3]:10000 for ipv6
	ipAndPort := r.RemoteAddr
	portSepIdx := strings.LastIndex(ipAndPort, ":")
	// if not found, return
	if portSepIdx < 0 {
		return ipAndPort
	}
	ip := ipAndPort[:portSepIdx]
	return ip
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
	a.log.Info().Msgf("Starting cache update loop with timeout %s", a.updateTimeout)
	for {
		key, expiry, err := a.cache.Oldest(a.keyer.OriginPrefix)
		// if error, try again in 1 minute
		if err != nil {
			a.log.Error().Err(err).Msg("Could not get oldest entry")
			time.Sleep(a.updateTimeout)
			continue
		}
		// if expiring within 1 minute, update
		// else sleep for 1 minute
		if key != "" && expiry.Sub(time.Now()) <= a.updateTimeout {
			a.updateEntry(key)
		} else {
			a.log.Trace().Msg("No entries expiring, pausing update")
			time.Sleep(a.updateTimeout)
		}
	}
}

func (a *AlwaysCache) updateAll() {
	a.cache.AllKeys(a.keyer.OriginPrefix, func(key string) {
		a.updateEntry(key)
	})
}

// updateKey will update the stored response identified by the given key.
// It is assumed that the key exists in the cache, if not (and the key is still valid),
// a new entry identified by the key is created.
// If there is an error while updating, the key will be purged from the cache.
func (a *AlwaysCache) updateEntry(key string) {
	var (
		err    error
		cached bool
	)
	// log error by default (see below)
	logError := true

	// get request based on key and save response to cache
	var req *http.Request
	if req, err = a.keyer.GetRequestFromKey(key); err == cachekey.ErrorMethodNotSupported {
		logError = false
	} else if err == nil {
		a.log.Trace().Str("key", key).Str("req.path", req.URL.Path).Msg("Updating cache")
		cached, err = a.saveRequest(req, key)
		// if there was an error, sleep and retry
		if !cached || err != nil {
			time.Sleep(time.Second)
			cached, err = a.saveRequest(req, key)
		}
	}

	// log error if not explicitly disabled
	if err != nil && logError {
		a.log.Error().Err(err).Str("key", key).Msg("Could not update cache entry")
	}
	// if there was an error, it should most definitely be purged
	// if the response was not cached, it means it should be purged
	if err != nil || !cached {
		a.cache.Purge(key)
	}
}

func (a *AlwaysCache) saveRequest(req *http.Request, key string) (bool, error) {
	a.log.Debug().
		Str("method", req.Method).
		Str("url", req.URL.String()).
		Str("key", key).
		Msg("Requesting content from origin")

	rw := tee.NewResponseSaver(nil)
	a.reverseproxy.ServeHTTP(rw, req)

	return a.writeCache(rw, req)
}

// copyHeadersTo copies the headers from one http.Header to another.
func copyHeadersTo(dst, src http.Header) {
	for name, values := range src {
		for _, value := range values {
			dst.Set(name, value)
		}
	}
}
