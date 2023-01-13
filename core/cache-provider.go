package core

import (
	"database/sql"
	"strings"
	"sync"
	"time"

	_ "github.com/glebarez/go-sqlite"
)

// CacheProvider is an interface for a cache provider.
// It stores and retrieves []byte values, which represent HTTP responses.
// It also keeps track of expiration times of cache entries.
//
// Implementations must be thread-safe!
type CacheProvider interface {
	// GetAll returns all cache entries that have the specific key prefix
	All(prefix string) ([]CacheEntry, error)
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
	// It should not return items where the expiry is zero
	Oldest() (string, time.Time, error)
	// Purge removes the cache entry for the given key.
	// It is a utility method that is not used by the cache middleware.
	Purge(key string)
	// Has checks if the specified key exists in the cache.
	Has(key string) bool
	// Keys calls the given callback for each key
	Keys(cb func(string))
}

type CacheEntry struct {
	Key     string
	Expires time.Time
	Bytes   []byte
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

func (m MemCache) All(prefix string) ([]CacheEntry, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	entries := make([]CacheEntry, 0)
	for key, val := range m.db {
		if strings.HasPrefix(key, prefix) {
			entries = append(entries, CacheEntry{
				Key:     key,
				Bytes:   val.bytes,
				Expires: val.expires,
			})
		}
	}
	return entries, nil
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
		if !entry.expires.IsZero() && (oldestKey == "" || entry.expires.Before(oldestTime)) {
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

func (m MemCache) Has(key string) bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	_, ok := m.db[key]
	return ok
}

func (m MemCache) Keys(cb func(string)) {
	for key := range m.db {
		cb(key)
	}
}

type SQLiteCache struct {
	db         *sql.DB
	writeMutex *sync.Mutex
}

func NewSQLiteCache() SQLiteCache {
	db, err := sql.Open("sqlite", "./cache.db")
	if err != nil {
		panic(err)
	}
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS cache (key TEXT PRIMARY KEY, expires INTEGER, bytes BLOB)")
	if err != nil {
		panic(err)
	}
	_, err = db.Exec("CREATE INDEX IF NOT EXISTS expires_idx ON cache (expires)")
	if err != nil {
		panic(err)
	}
	_, err = db.Exec("PRAGMA journal_mode=WAL")
	if err != nil {
		panic(err)
	}
	return SQLiteCache{
		db:         db,
		writeMutex: &sync.Mutex{},
	}
}

func (s SQLiteCache) All(prefix string) ([]CacheEntry, error) {
	s.writeMutex.Lock()
	defer s.writeMutex.Unlock()
	entries := make([]CacheEntry, 0)
	rows, err := s.db.Query("SELECT key, expires, bytes FROM cache WHERE key LIKE ?", prefix+"%")
	if err != nil {
		return entries, err
	}
	for rows.Next() {
		var entry CacheEntry
		var exp int64
		if err := rows.Scan(&entry.Key, &exp, &entry.Bytes); err != nil {
			return entries, err
		}
		entry.Expires = time.Unix(exp, 0)
		entries = append(entries, entry)
	}
	return entries, nil
}

func (s SQLiteCache) Get(key string) ([]byte, bool, error) {
	var expires int64
	var bytes []byte
	err := s.db.QueryRow("SELECT expires, bytes FROM cache WHERE key = ?", key).Scan(&expires, &bytes)
	if err != nil {
		return nil, false, err
	}
	if time.Now().After(time.Unix(expires, 0)) {
		return nil, false, nil
	}
	return bytes, true, nil
}

func (s SQLiteCache) Put(key string, expires time.Time, bytes []byte) error {
	s.writeMutex.Lock()
	defer s.writeMutex.Unlock()
	_, err := s.db.Exec("INSERT OR REPLACE INTO cache (key, expires, bytes) VALUES (?, ?, ?)", key, expires.Unix(), bytes)
	return err
}

func (s SQLiteCache) Oldest() (string, time.Time, error) {
	var key string
	var expires int64
	err := s.db.QueryRow("SELECT key, expires FROM cache WHERE expires > 0 ORDER BY expires ASC LIMIT 1").Scan(&key, &expires)
	if err != nil {
		return "", time.Time{}, err
	}
	return key, time.Unix(expires, 0), nil
}

func (s SQLiteCache) Purge(key string) {
	s.writeMutex.Lock()
	defer s.writeMutex.Unlock()
	_, err := s.db.Exec("DELETE FROM cache WHERE key = ?", key)
	if err != nil {
		panic(err)
	}
}

func (s SQLiteCache) Has(key string) bool {
	result, err := s.db.Exec("SELECT 1 FROM cache WHERE key = ?", key)
	if err != nil {
		panic(err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		panic(err)
	}
	return rows > 0
}

func (s SQLiteCache) Keys(cb func(string)) {
	rows, err := s.db.Query("SELECT key FROM cache")
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return
		}
		cb(key)
	}
}
