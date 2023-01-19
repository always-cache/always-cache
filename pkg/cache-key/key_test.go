package cachekey

import (
	"net/http"
	"strings"
	"testing"
)

func TestRequestFromKey(t *testing.T) {
	keygen := NewCacheKeyer("this-is-the-origin")
	r, _ := http.NewRequest("GET", "http://dev.localhost/page", nil)
	key := keygen.GetKeyPrefix(r)
	req, err := keygen.GetRequestFromKey(key)
	if err != nil {
		t.Fatalf("%s: %s", key, err)
	}
	if url := req.URL.String(); url != "/page" {
		t.Fatalf("Created request url for key %s is %s", key, url)
	}
}

func TestOriginPrefixIncludesOrigin(t *testing.T) {
	origin := "this-is-the-origin"
	keygen := NewCacheKeyer(origin)
	if !strings.Contains(keygen.OriginPrefix, origin) {
		t.Fatalf("OriginPrefix is %s", keygen.OriginPrefix)
	}
}
