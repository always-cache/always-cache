package alwayscache

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/always-cache/always-cache/cache"
	cachekey "github.com/always-cache/always-cache/pkg/cache-key"
	serializer "github.com/always-cache/always-cache/pkg/response-serializer"
	tee "github.com/always-cache/always-cache/pkg/response-writer-tee"
	"github.com/always-cache/always-cache/rfc9111"
	"github.com/always-cache/always-cache/rfc9211"

	"github.com/rs/zerolog"
)

type Config struct {
	// Storage for cache entries.
	Cache cache.CacheProvider
	// URL of the origin server.
	// Origins with paths are not supparted.
	OriginURL url.URL
	// Hostname to use for HTTP requests and TLS negotiation.
	// Use if needed if e.g. the origin URL is just an IP address.
	OriginHost string
	// Logger to use. The global zerolog logger is used if nil.
	Logger *zerolog.Logger
	// Optional function for mutating the incoming request.
	// Use it e.g. for setting the request `Cache-Key` header when needed.
	RequestModifier func(*http.Request)
	// Optional function for transforming the origin response.
	// Use it e.g. for adding Cache-Control or other headers.
	ResponseModifier func(*http.Response) error
	// Disable automatic updates of expiring content, i.e. legacy mode.
	DisableUpdates bool
}

type AlwaysCache struct {
	cache          cache.CacheProvider
	keyer          cachekey.CacheKeyer
	log            zerolog.Logger
	updateTimeout  time.Duration
	reverseproxy   httputil.ReverseProxy
	modifyResponse func(*http.Request)
}

// CreateCache initializes the always-cache instance.
// It starts the needed background processes
// and sets up the needed variables
func CreateCache(config Config) *AlwaysCache {
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
		cache:          config.Cache,
		keyer:          cachekey.NewCacheKeyer(config.OriginURL.String()),
		log:            logger,
		modifyResponse: config.RequestModifier,
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
		ModifyResponse: config.ResponseModifier,
	}

	// start a goroutine to update expired entries
	if !config.DisableUpdates {
		a.updateTimeout = time.Second
	}
	if a.updateTimeout != 0 {
		go a.updateCache()
	}

	return a
}

type request struct {
	r           *http.Request
	cacheStatus rfc9211.CacheStatus
	log         zerolog.Logger
}

// ServeHTTP implements the http.Handler interface.
func (a *AlwaysCache) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if a.modifyResponse != nil {
		a.modifyResponse(r)
	}
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
	a.log.Trace().Str("key", keyUriPrefix).Msg("Getting cached entries")
	cacheEntries, err := a.cache.All(keyUriPrefix)
	if err != nil {
		a.log.Error().Err(err).Msg("Could not retrieve from cache")
		return nil
	}
	a.log.Trace().Str("key", keyUriPrefix).Msgf("Found %v cache entries", len(cacheEntries))
	return cacheEntries
}

func (a *AlwaysCache) updateResposeAndLocations(rw *tee.ResponseSaver, r *http.Request) {
	a.writeCache(rw, r)
	a.updateIfNeeded(r, &http.Response{
		StatusCode: rw.StatusCode(),
		Header:     rw.Header(),
		Request:    r,
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
	// if it's a redirect, though, the redirect is likely to be to the updated content,
	// in which case we update synchronously
	if isRedirect(rwtee.StatusCode()) {
		a.updateResposeAndLocations(rwtee, r)
	} else {
		go a.updateResposeAndLocations(rwtee, r)
	}
}

func isRedirect(statusCode int) bool {
	if statusCode == 301 ||
		statusCode == 302 ||
		statusCode == 303 ||
		statusCode == 307 ||
		statusCode == 308 {
		return true
	}
	return false
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

// copyHeadersTo copies the headers from one http.Header to another.
func copyHeadersTo(dst, src http.Header) {
	for name, values := range src {
		for _, value := range values {
			dst.Set(name, value)
		}
	}
}
