package rfc9111

import (
	"net/http"
	"testing"
	"time"
)

func TestDeltaSecondsParam(t *testing.T) {
	res := &http.Response{
		Header: make(http.Header),
	}
	res.Header.Add("Age", "7200;foo=bar")
	if age, err := getAge(res); err != nil || age != time.Second*7200 {
		t.Fatalf("Age is %v", age)
	}
}
