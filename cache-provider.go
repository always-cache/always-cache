package cache

import (
	"sync"
	"time"
)

// CacheProvider is an interface for a cache provider.
// It stores and retrieves []byte values, which represent HTTP responses.
// It also keeps track of expiration times of cache entries.
//
// Implementations must be thread-safe!
type CacheProvider interface {
	// Get returns the cached response for the given key, if it exists.
	// It also returns a boolean indicating whether retrieval was successful.
	// If the cache entry has expired, the boolean should be false.
	// (In this case, the cache provider should also purge the entry.)
	Get(key string) ([]byte, bool, error)
	// Put stores the given response in the cache under the given key.
	// It also sets an expiration time for the entry.
	Put(key string, expires time.Time, bytes []byte) error
	// Oldest returns the key and expiration time of the oldest entry in the cache.
	// The oldest entry is the one with the earliest expiration time.
	Oldest() (string, time.Time, error)
	// Purge removes the cache entry for the given key.
	// It is a utility method that is not used by the cache middleware.
	Purge(key string)
}

type memCacheEntry struct {
	expires time.Time
	bytes   []byte
}

type MemCache struct {
	mutex *sync.RWMutex
	db    map[string]memCacheEntry
}

func NewMemCache() MemCache {
	return MemCache{
		mutex: &sync.RWMutex{},
		db:    make(map[string]memCacheEntry),
	}
}

func (m MemCache) Get(key string) ([]byte, bool, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	entry, ok := m.db[key]
	if !ok {
		return nil, false, nil
	}
	if time.Now().After(entry.expires) {
		return nil, false, nil
	}
	return entry.bytes, true, nil
}

func (m MemCache) Put(key string, expires time.Time, bytes []byte) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.db[key] = memCacheEntry{expires, bytes}
	return nil
}

func (m MemCache) Oldest() (string, time.Time, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	var oldestKey string
	var oldestTime time.Time
	for key, entry := range m.db {
		if oldestKey == "" || entry.expires.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.expires
		}
	}
	return oldestKey, oldestTime, nil
}

func (m MemCache) Purge(key string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	delete(m.db, key)
}
