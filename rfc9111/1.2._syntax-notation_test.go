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

func TestHttpDateRFC850(t *testing.T) {
	_, err := HttpDate("Thursday, 18-Aug-50 02:01:18 GMT")
	if err != nil {
		t.Fatalf("Error parsing date %+v", err)
	}
}

func TestHttpDateTZCase(t *testing.T) {
	_, err := HttpDate("Thu, 18 Aug 2050 02:01:18 gMT")
	if err != nil {
		t.Fatalf("Error parsing date %+v", err)
	}
}
