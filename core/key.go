package core

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/always-cache/always-cache/rfc9111"
)

var errorMethodNotSupported = fmt.Errorf("Method not supported")

type CacheKeyer struct {
	// Unique identifier for the origin.
	// Usually this should be the origin - well - origin.
	OriginId string
}

// getKeyPrefix returns the cache key for a request without the vary headers (i.e. a key prefix).
// The returned key is suitable for finding all stored response variants for a porticular request.
// If it is a GET request, the key depends only on the URL.
// If it is a POST request, it will also depend on the request body.
func (c CacheKeyer) GetKeyPrefix(r *http.Request) string {
	key := r.Method + ":" + c.OriginId + r.URL.RequestURI() + "\t"
	if r.Method == "POST" {
		if multipartHash := multipartHash(r); multipartHash != "" {
			return key + multipartHash
		} else if bodyHash := bodyHash(r); bodyHash != "" {
			return key + bodyHash
		}
	}
	return key
}

// addVaryKeys returns the full cache key (including vary headers) based on a previously generated
// cache key prefix and the request and response involved.
func (c CacheKeyer) AddVaryKeys(prefix string, req *http.Request, res *http.Response) string {
	key := prefix
	for _, name := range rfc9111.GetListHeader(res.Header, "Vary") {
		if !rfc9111.FieldAbsent(req.Header, name) {
			key = key + "\n" + strings.ToLower(name) + ": " + req.Header.Get(name)
		}
	}
	return key
}

// getRequestFromKey generates a caching-wise equal request than the request that resulted in the
// provided key. This means it takes vary headers into account.
// It returns an error if the request cannot for some reason be deducted.
func (c CacheKeyer) GetRequestFromKey(key string) (*http.Request, error) {
	if !strings.HasPrefix(key, "GET:") {
		return nil, errorMethodNotSupported
	}
	keyNoOriginId := strings.TrimPrefix(key, c.OriginId)
	keyNoVary := strings.Split(keyNoOriginId, "\t")[0]
	uri := strings.TrimSpace(strings.TrimLeft(keyNoVary, "GET:"))
	return http.NewRequest("GET", uri, nil)
}

// getVaryHeaders creates a http.Header instance containing all the vary keys included in a key.
func (c CacheKeyer) GetVaryHeaders(key string) http.Header {
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
