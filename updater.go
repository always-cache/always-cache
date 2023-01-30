package alwayscache

import (
	"net/http"
	"time"

	cachekey "github.com/always-cache/always-cache/pkg/cache-key"
	"github.com/always-cache/always-cache/pkg/cache-update"
	tee "github.com/always-cache/always-cache/pkg/response-writer-tee"
	"github.com/always-cache/always-cache/rfc9111"
)

func (a *AlwaysCache) updateIfNeeded(downReq *http.Request, upRes *http.Response) {
	if a.updateTimeout == 0 {
		a.invalidateUris(
			rfc9111.GetInvalidateURIs(downReq, upRes))
	} else {
		a.revalidateUris(
			rfc9111.GetInvalidateURIs(downReq, upRes))
	}
	a.saveUpdates(
		cacheupdate.GetCacheUpdates(downReq, upRes))
}

func (a *AlwaysCache) saveUpdates(updates []cacheupdate.CacheUpdate) {
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

// updateCache runs an infinite loop to update the cache,
// one entry at a time.
// It assumes that the cache key equals the request URL.
// It will query the cache for entries expiring within the update timeout.
// If it finds one, it will update the cache for that entry.
// If it does not find any, it will sleep for the duration of the update timeout.
func (a *AlwaysCache) updateCache() {
	a.log.Info().Msgf("Starting cache update loop with timeout %s", a.updateTimeout)
	for {
		key, expiry, err := a.cache.Oldest(a.keyer.MethodPrefix("GET"))
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
