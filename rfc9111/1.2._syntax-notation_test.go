package rfc9111

import (
	"testing"
	"time"
)

func TestToDeltaSeconds(t *testing.T) {
	fiveSeconds := 5 * time.Second
	if s := toDeltaSeconds(fiveSeconds); s != "5" {
		t.Fatalf("Delta seconds is %s", s)
	}
}
