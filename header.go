package main

import "strings"

type CacheControl struct {
	m map[string]string
}

func (c CacheControl) Get(directive string) (string, bool) {
	val, ok := c.m[directive]
	return val, ok
}

func ParseCacheControl(header string) CacheControl {
	m := make(map[string]string)
	for _, directive := range strings.Split(header, ", ") {
		parts := strings.SplitN(directive, "=", 2)
		var val string
		if len(parts) > 1 {
			val = parts[1]
		}
		m[parts[0]] = val
	}
	return CacheControl{m}
}
