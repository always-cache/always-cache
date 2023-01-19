package core

import (
	"net/http"
	"testing"
)

func TestRequestFromKey(t *testing.T) {
	keygen := CacheKeyer{"this-is-the-origin"}
	r, _ := http.NewRequest("GET", "http://dev.localhost/page", nil)
	key := keygen.GetKeyPrefix(r)
	req, _ := keygen.GetRequestFromKey(key)
	if url := req.URL.String(); url != "/page" {
		t.Fatalf("Created request url for key %s is %s", key, url)
	}
}
