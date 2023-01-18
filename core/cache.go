package core

import (
	"crypto/tls"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/always-cache/always-cache/rfc9111"
	"github.com/always-cache/always-cache/rfc9211"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Config struct {
	Cache         CacheProvider
	OriginURL     url.URL
	OriginHost    string
	UpdateTimeout time.Duration
	Rules         Rules
}

type AlwaysCache struct {
	cache         CacheProvider
	originURL     url.URL
	originHost    string
	updateTimeout time.Duration
	rules         Rules
	httpClient    http.Client
}

// CreateCache initializes the always-cache instance.
// It starts the needed background processes
// and sets up the needed variables
func CreateCache(config Config) *AlwaysCache {
	a := &AlwaysCache{
		cache:         config.Cache,
		originURL:     config.OriginURL,
		originHost:    config.OriginHost,
		updateTimeout: config.UpdateTimeout,
		rules:         config.Rules,
		httpClient: http.Client{
			// do not follow redirects
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}

	// start a goroutine to update expired entries
	if a.updateTimeout != 0 {
		go a.updateCache()
	}

	// use provided hostname for origin if configured
	if a.originHost != "" {
		a.httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				ServerName: a.originHost,
			},
		}
	}

	return a
}

// ServeHTTP implements the http.Handler interface.
func (a *AlwaysCache) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer a.recover(w, r)
	a.handle(w, r)
}

// recover recovers from panics and sends the response to the escape hatch if needed.
func (a *AlwaysCache) recover(w http.ResponseWriter, r *http.Request) {
	if err := recover(); err != nil {
		a.escapeHatch(w, r)
		log.WithLevel(zerolog.PanicLevel).Interface("error", err).Msg("Panic in cache handler")
	}
}

// escapeHatch is a fallback handler that just proxies the request to the origin.
func (a *AlwaysCache) escapeHatch(w http.ResponseWriter, r *http.Request) {
	originReq := rfc9111.GetForwardRequest(r)
	// TODO use just httpClient.Do here (by creating the request first)
	originRes, err := a.fetch(originReq)
	if err != nil {
		log.Error().Err(err).Msg("Error connecting to origin")
		http.Error(w, "Could not connect to origin", http.StatusBadGateway)
		return
	}
	w.WriteHeader(originRes.response.StatusCode)
	copyHeader(w.Header(), originRes.response.Header)
	defer originRes.response.Body.Close()
	_, err = io.Copy(w, originRes.response.Body)
	if err != nil {
		log.Error().Err(err).Msg("Error writing to client")
	}
}

// hondle is the main entry point for the caching middleware.
func (a *AlwaysCache) handle(w http.ResponseWriter, r *http.Request) {
	// this is a temporary workaround
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
			if res != nil {
				res.Request = r
			}
			if fwdReason == "" {
				cacheStatus.Hit()
				cacheStatus.TimeToLive = rfc9111.TimeToLive(sRes.response, sRes.responseTime, sRes.requestTime)
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
		cacheStatus.Stored = true
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

	if err := a.cache.Put(key, exp, responseBytes); err != nil {
		log.Error().Err(err).Str("key", key).Msg("Could not write to cache")
		panic(err)
	}
	log.Trace().Str("key", key).Time("expiry", exp).Msg("Cache write")
}

// fetch the resource specified in the incoming request from the origin
func (a *AlwaysCache) fetch(r *http.Request) (timedResponse, error) {
	timedRes := timedResponse{requestTime: time.Now()}
	uri := a.originURL.String() + r.URL.RequestURI()
	// need to specifically set body to nil on the outgoing request if content is zero length
	// see https://github.com/golang/go/issues/16036
	body := r.Body
	if r.ContentLength == 0 {
		body = nil
	}
	req, err := http.NewRequest(r.Method, uri, body)
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

	originResponse, err := a.httpClient.Do(req)
	timedRes.responseTime = time.Now()
	// as per https://www.rfc-editor.org/rfc/rfc9110#section-6.6.1-8
	if err == nil && originResponse.Header.Get("Date") == "" {
		originResponse.Header.Set("Date", rfc9111.ToHttpDate(time.Now()))
	}
	timedRes.response = originResponse
	return timedRes, err
}

func send(w http.ResponseWriter, r *http.Response, status rfc9211.CacheStatus) error {
	evt := log.Debug()
	if r.Request == nil {
		log.Warn().Msg("Could not get request for response to client")
	} else {
		evt = evt.Str("url", r.Request.URL.String())
	}
	isHit := 0
	if status.FwdReason == "" {
		isHit = 1
	}
	evt.
		Str("status", string(status.Status)).
		Str("fwd", string(status.FwdReason)).
		Bool("stored", status.Stored).
		Int("ttl", status.TimeToLive).
		Int("hit", isHit).
		Msg("Sending response to client")

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
		req, err := getRequestFromKey(key)
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
	log.Debug().
		Str("key", key).
		Str("url", req.URL.String()).
		Msg("Requesting content from origin")

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

// copyHeadersTo copies the headers from one http.Header to another.
func copyHeadersTo(dst, src http.Header) {
	for name, values := range src {
		for _, value := range values {
			dst.Set(name, value)
		}
	}
}
