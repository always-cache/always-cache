package cache

import (
	"database/sql"
	"sync"
	"time"

	_ "github.com/glebarez/go-sqlite"
)

// CacheProvider is an interface for a cache provider.
// It stores and retrieves []byte values, which represent HTTP responses.
// It also keeps track of expiration times of cache entries.
// Operating on specific keys or origin-specific prefixes is very important
// in order for many origins to be able to be stored in the same cache.
//
// Implementations must be thread-safe!
type CacheProvider interface {
	// Keys calls the given callback for each key with the given prefix.
	// It calls the callback in order to enable very large lists of keys to be
	// processable (provider implementation might use paging, for instance).
	AllKeys(prefix string, cb func(string))
	// All returns all cache entries that have the specific key prefix
	All(prefix string) ([]CacheEntry, error)
	// Get returns the cached response for the given key, if it exists.
	// It also returns a boolean indicating whether retrieval was successful.
	// If the cache entry has expired, the boolean should be false.
	// (In this case, the cache provider should also purge the entry.)
	Get(key string) ([]byte, bool, error)
	// Put stores the given response in the cache under the given key.
	// It also sets an expiration time for the entry.
	Put(key string, expires time.Time, bytes []byte) error
	PutCE(CacheEntry) error
	// Oldest returns the key and expiration time of the oldest entry in the cache.
	// The oldest entry is the one with the earliest expiration time.
	// It should not return items where the expiry is zero
	Oldest(prefix string) (string, time.Time, error)
	// Purge removes the cache entry for the given key.
	// It is a utility method that is not used by the cache middleware.
	Purge(key string)
	// Has checks if the specified key exists in the cache.
	Has(key string) bool
}

type CacheEntry struct {
	Key         string
	Expires     time.Time
	RequestedAt time.Time
	ReceivedAt  time.Time
	Bytes       []byte
}

type SQLiteCache struct {
	db         *sql.DB
	writeMutex *sync.Mutex
}

// NewSQLiteCache creates a new cache with the given filename as the db.
// If file name is empty, a new in-memory db is opened.
func NewSQLiteCache(filename string) SQLiteCache {
	if filename == "" {
		filename = "file::memory:?cache=shared"
	}
	db, err := sql.Open("sqlite", filename)
	if err != nil {
		panic(err)
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS cache (
		key TEXT PRIMARY KEY,
		expires INTEGER,
		requested_at INTEGER,
		received_at INTEGER,
		bytes BLOB
	)`)
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
	rows, err := s.db.Query(`SELECT 
		key, expires, requested_at, received_at, bytes
		FROM cache WHERE key LIKE ?`, prefix+"%")
	if err != nil {
		return entries, err
	}
	for rows.Next() {
		var entry CacheEntry
		var exp, req, rec int64
		if err := rows.Scan(&entry.Key, &exp, &req, &rec, &entry.Bytes); err != nil {
			return entries, err
		}
		entry.Expires = time.Unix(exp, 0)
		entry.RequestedAt = time.Unix(req, 0)
		entry.ReceivedAt = time.Unix(rec, 0)
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

func (s SQLiteCache) PutCE(ce CacheEntry) error {
	s.writeMutex.Lock()
	defer s.writeMutex.Unlock()
	_, err := s.db.Exec(`INSERT OR REPLACE INTO cache 
		(key, expires, requested_at, received_at, bytes) VALUES (?, ?, ?, ?, ?)`,
		ce.Key, ce.Expires.Unix(), ce.RequestedAt.Unix(), ce.ReceivedAt.Unix(), ce.Bytes)
	return err
}

func (s SQLiteCache) Oldest(prefix string) (string, time.Time, error) {
	var key string
	var expires int64
	err := s.db.QueryRow(
		"SELECT key, expires FROM cache WHERE key LIKE ? AND expires > 0 ORDER BY expires ASC LIMIT 1",
		prefix+"%",
	).Scan(&key, &expires)
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

func (s SQLiteCache) AllKeys(prefix string, cb func(string)) {
	rows, err := s.db.Query("SELECT key FROM cache WHERE key LIKE ?", prefix+"%")
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
