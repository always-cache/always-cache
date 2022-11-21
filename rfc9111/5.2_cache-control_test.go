package rfc9111

import "testing"

func TestMaxAge(t *testing.T) {
	cc := ParseCacheControl([]string{"max-age=60"})
	val, ok := cc.Get("max-age")
	if !ok {
		t.Fatal("Could not get directive")
	}
	if val != "60" {
		t.Fatalf("Value is %s", val)
	}
}

func TestReal(t *testing.T) {
	cc := ParseCacheControl([]string{"public, max-age=0, s-maxage=600"})
	if val, ok := cc.Get("public"); !ok || val != "" {
		t.Fatalf("val: '%s', ok: %v", val, ok)
	}
	if val, ok := cc.Get("max-age"); !ok || val != "0" {
		t.Fatalf("val: '%s', ok: %v", val, ok)
	}
	if val, ok := cc.Get("s-maxage"); !ok || val != "600" {
		t.Fatalf("val: '%s', ok: %v", val, ok)
	}
}
