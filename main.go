package cache

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
)

func bytesToResponse(b []byte) (*http.Response, error) {
	return http.ReadResponse(bufio.NewReader(bytes.NewReader(b)), nil)
}

func copyHeadersTo(dst, src http.Header) {
	for name, values := range src {
		for _, value := range values {
			dst.Set(name, value)
		}
	}
}

type Cache interface {
	Get(key string) ([]byte, bool)
	Put(key string, bytes []byte) error
	Purge(key string)
}

type MemCache struct {
	db map[string][]byte
}

func (m MemCache) Get(key string) ([]byte, bool) {
	bytes, ok := m.db[key]
	return bytes, ok
}
func (m MemCache) Put(key string, bytes []byte) error {
	m.db[key] = bytes
	return nil
}
func (m MemCache) Purge(key string) {
	delete(m.db, key)
}

type ResponseWriterTee struct {
	rw           http.ResponseWriter
	b            *bytes.Buffer
	status       int
	wroteHeaders bool
}

func (t *ResponseWriterTee) Header() http.Header {
	return t.rw.Header()
}

func (t *ResponseWriterTee) WriteHeader(statusCode int) {
	t.wroteHeaders = true
	t.status = statusCode
	t.b.WriteString(fmt.Sprintf("HTTP/1.1 %d %s\n", statusCode, http.StatusText(statusCode)))
	t.rw.Header().Write(t.b)
	t.b.WriteString("\n")
	t.rw.WriteHeader(statusCode)
}

func (t *ResponseWriterTee) Write(b []byte) (int, error) {
	if !t.wroteHeaders {
		t.WriteHeader(http.StatusOK)
	}
	t.b.Write(b)
	return t.rw.Write(b)
}

func (t *ResponseWriterTee) Response() []byte {
	return t.b.Bytes()
}

func (t *ResponseWriterTee) Updates() []string {
	return t.rw.Header().Values("cache-update")
}

func NewResponseWriter(w http.ResponseWriter) *ResponseWriterTee {
	buf := new(bytes.Buffer)
	return &ResponseWriterTee{rw: w, b: buf}
}

type DummyResponseWriter struct {
	http.ResponseWriter
	header http.Header
}

func (d DummyResponseWriter) Write(b []byte) (int, error) {
	return 0, nil
}
func (d DummyResponseWriter) WriteHeader(statusCode int) {
}
func (d DummyResponseWriter) Header() http.Header {
	if d.header == nil {
		d.header = make(http.Header)
	}
	return d.header
}

type Cacher struct {
	Cache    Cache
	expiries map[string]time.Time
}

func (c *Cacher) Get(key string) ([]byte, bool) {
	if expiry, ok := c.expiries[key]; ok {
		if time.Now().After(expiry) {
			c.Cache.Purge(key)
			delete(c.expiries, key)
			return nil, false
		}
	}
	return c.Cache.Get(key)
}

func (c *Cacher) Put(key string, rw *ResponseWriterTee) error {
	cacheControl := rw.Header().Get("Cache-Control")
	maxAge := time.Hour
	if matches := regexp.MustCompile(`(?i)\bmax-age=(\d)+`).FindStringSubmatch(cacheControl); matches != nil {
		if duration, err := time.ParseDuration(matches[1] + "s"); err == nil {
			maxAge = duration
		}
	}
	c.expiries[key] = time.Now().Add(maxAge)
	return c.Cache.Put(key, rw.Response())
}

func Middleware(next http.Handler) http.Handler {
	queue := make(map[string](chan bool))
	cache := MemCache{make(map[string][]byte)}
	cacher := Cacher{cache, make(map[string]time.Time)}
	fn := func(w http.ResponseWriter, r *http.Request) {
		logger := hlog.FromRequest(r)
		if logger.GetLevel() == zerolog.Disabled {
			logger = &log.Logger
		}
		key := r.URL.RequestURI()

		if isCacheable(r) {
			// if there is a cache update going, wait for that
			if c, ok := queue[key]; ok {
				logger.Trace().Str("key", key).Msg("Waiting for cache update")
				go func() {
					time.Sleep(time.Second * 5)
					logger.Warn().Str("key", key).Msg("Cache update timed out")
					c <- false
				}()
				<-c
			}
			// check if we have a cached version
			if cachedResponse, ok := cacher.Get(key); ok {
				logger.Trace().Str("key", key).Msg("Cache hit and serving")
				resp, err := bytesToResponse(cachedResponse)
				if err != nil {
					http.Error(w, "Cannot read response", http.StatusInternalServerError)
					return
				}
				copyHeadersTo(w.Header(), resp.Header)
				io.Copy(w, resp.Body)
			} else {
				rw := NewResponseWriter(w)
				next.ServeHTTP(rw, r)
				if shouldCache(rw) {
					logger.Trace().Str("key", key).Msg("Cache miss and write")
					cacher.Put(key, rw)
				} else {
					logger.Trace().Str("key", key).Msg("Cache miss and not write")
				}
			}
		} else {
			rw := NewResponseWriter(w)
			next.ServeHTTP(rw, r)
			// if we have a cached get request, update
			if _, found := cache.Get(key); found {
				queue[key] = make(chan bool, 1)
				rw := NewResponseWriter(DummyResponseWriter{})
				req, _ := http.NewRequest("GET", r.URL.RequestURI(), nil)
				req.Header.Add("Authorization", r.Header.Get("Authorization"))
				next.ServeHTTP(rw, req)
				if shouldCache(rw) {
					logger.Trace().Str("key", key).Msg("Cache update based on path")
					cacher.Put(key, rw)
					queue[key] <- true
				} else {
					logger.Trace().Str("key", key).Msg("Could not cache update")
					queue[key] <- false
				}
				delete(queue, key)
			}
			// update cache based on cache-update header
			for _, update := range rw.Updates() {
				queue[update] = make(chan bool, 1)
				rw := NewResponseWriter(DummyResponseWriter{})
				req, _ := http.NewRequest("GET", update, nil)
				req.Header.Add("Authorization", r.Header.Get("Authorization"))
				next.ServeHTTP(rw, req)
				if shouldCache(rw) {
					logger.Trace().Str("key", key).Str("update", update).Msg("Cache update based on headers")
					cacher.Put(update, rw)
					queue[update] <- true
				} else {
					logger.Trace().Str("key", key).Str("update", update).Msg("Could not cache update")
					queue[update] <- false
				}
				delete(queue, update)
			}
		}
	}
	return http.HandlerFunc(fn)
}

func isCacheable(r *http.Request) bool {
	return r.Method == "GET"
}

func shouldCache(rw *ResponseWriterTee) bool {
	return rw.status == http.StatusOK
}
