package main

import (
	"net/http"
	"testing"
)

func TestRuleFinder(t *testing.T) {
	makeRes := func(method, path string) *http.Response {
		req, _ := http.NewRequest(method, path, nil)
		res := http.Response{Request: req}
		return &res
	}

	rules := Rules{
		Rule{Prefix: "/wp-", Override: "no-cache"},
		Rule{Query: map[string]string{"any": ""}, Override: "any qry"},
		Rule{Query: map[string]string{"val": "value"}, Override: "val qry"},
		Rule{Override: "default"},
	}

	if rule := rules.find(makeRes("GET", "/")); rule == nil || rule.Override != "default" {
		t.Fatal("Incorrect rule")
	}
	if rule := rules.find(makeRes("GET", "/wp-admin")); rule == nil || rule.Override != "no-cache" {
		t.Fatal("Incorrect rule")
	}
	if rule := rules.find(makeRes("POST", "/wp-admin")); rule != nil {
		t.Fatal("Incorrect rule")
	}
	// test query string matching
	if rule := rules.find(makeRes("GET", "/?any")); rule == nil || rule.Override != "any qry" {
		t.Fatal("Incorrect rule")
	}
	if rule := rules.find(makeRes("GET", "/?any=something")); rule == nil || rule.Override != "any qry" {
		t.Fatal("Incorrect rule")
	}
	if rule := rules.find(makeRes("GET", "/?val=value")); rule == nil || rule.Override != "val qry" {
		t.Fatal("Incorrect rule")
	}
	if rule := rules.find(makeRes("GET", "/?val=nomatch")); rule == nil && rule.Override != "default" {
		t.Fatal("Incorrect rule")
	}
}

func TestApply(t *testing.T) {
	res := &http.Response{Header: make(http.Header)}
	ruleDefault := Rule{Default: "default"}
	ruleOverride := Rule{Override: "override"}

	// try to apply default
	applyRuleToResponse(ruleDefault, res)
	if cc := res.Header.Get("Cache-Control"); cc != "default" {
		t.Fatalf("Cache-Control header wrong, is '%s'", cc)
	}

	// change cc and check default is not set
	res.Header.Set("Cache-Control", "no-cache")
	applyRuleToResponse(ruleDefault, res)
	if cc := res.Header.Get("Cache-Control"); cc != "no-cache" {
		t.Fatalf("Cache-Control header wrong, is '%s'", cc)
	}

	// check that override works
	applyRuleToResponse(ruleOverride, res)
	if cc := res.Header.Get("Cache-Control"); cc != "override" {
		t.Fatalf("Cache-Control header wrong, is '%s'", cc)
	}
}
