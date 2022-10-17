package cache

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

func TestMiddlewareReturnsResponse(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello world"))
	})
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()

	Middleware(handler).ServeHTTP(rr, req)

	if body, err := io.ReadAll(rr.Result().Body); err != nil || fmt.Sprintf("%s", body) != "Hello world" {
		t.Fatalf("Body is %s", body)
	}
}

func TestMiddlewareReturnsSecondRequestFromCache(t *testing.T) {
	var handleCount int
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleCount++
		w.Write([]byte("Hello world"))
	})
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	mw := Middleware(handler)

	mw.ServeHTTP(httptest.NewRecorder(), req)
	mw.ServeHTTP(rr, req)

	if handleCount != 1 {
		t.Fatalf("Next handler called %d times", handleCount)
	}
	if body, err := io.ReadAll(rr.Result().Body); err != nil || fmt.Sprintf("%s", body) != "Hello world" {
		t.Fatalf("Body is %s", body)
	}
}

func TestCacheHeaders(t *testing.T) {
	var handleCount int
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleCount++
		w.Header().Add("content-type", "text/test")
		w.Write([]byte("Hello world"))
	})
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	mw := Middleware(handler)

	mw.ServeHTTP(httptest.NewRecorder(), req)
	mw.ServeHTTP(rr, req)

	if ct := rr.Result().Header.Get("content-type"); ct != "text/test" {
		body, _ := io.ReadAll(rr.Result().Body)
		t.Fatalf("Content-Type header is %s with body %s", ct, body)
	}
}

func TestCacheUpdate(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/update", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("cache-update", "/count")
		w.Write([]byte("Hello world"))
	})
	var handleCount int
	mux.HandleFunc("/count", func(w http.ResponseWriter, r *http.Request) {
		handleCount++
		w.Write([]byte(fmt.Sprintf("Called %d times", handleCount)))
	})
	mw := Middleware(mux)
	req, _ := http.NewRequest("POST", "/update", nil)
	countReq, _ := http.NewRequest("GET", "/count", nil)

	rr := httptest.NewRecorder()

	mw.ServeHTTP(httptest.NewRecorder(), countReq)
	mw.ServeHTTP(httptest.NewRecorder(), req)
	mw.ServeHTTP(rr, countReq)

	if body, err := io.ReadAll(rr.Result().Body); err != nil || fmt.Sprintf("%s", body) != "Called 2 times" {
		t.Fatalf("Body is %s", body)
	}
}

func TestUpdateOnPost(t *testing.T) {
	handleCount := 0
	assertCount := func(count int) {
		if count != handleCount {
			t.Fatalf("Handler called %d times, expected %d", handleCount, count)
		}
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleCount++
		w.Write([]byte(fmt.Sprintf("So you wanted to %s?", r.Method)))
	})
	get, _ := http.NewRequest("GET", "/", nil)
	post, _ := http.NewRequest("POST", "/", nil)
	mw := Middleware(handler)

	mw.ServeHTTP(httptest.NewRecorder(), get)
	assertCount(1)
	mw.ServeHTTP(httptest.NewRecorder(), get)
	assertCount(1)
	mw.ServeHTTP(httptest.NewRecorder(), post)
	// post will first do the actual post and then refresh with a get
	assertCount(3)
	mw.ServeHTTP(httptest.NewRecorder(), get)
	assertCount(3)
}

func TestUpdateBeforeResponding(t *testing.T) {
	listCount := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/list", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Second)
		w.Write([]byte(fmt.Sprintf("%d elements", listCount)))
	})
	mux.HandleFunc("/add", func(w http.ResponseWriter, r *http.Request) {
		// only add if post request
		if r.Method == "POST" {
			listCount++
			w.Header().Add("cache-update", "/list")
			w.Write([]byte("done"))
		} else {
			w.Write([]byte("nothing to do on get"))
		}
	})
	mw := Middleware(mux)

	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, httptest.NewRequest("GET", "/list", nil))
	if body := rr.Body.String(); body != "0 elements" {
		t.Fatalf("body is %s", body)
	}
	// create post, which will update the cache, and return when the response is done
	mw.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/add", nil))

	rr = httptest.NewRecorder()
	mw.ServeHTTP(rr, httptest.NewRequest("GET", "/list", nil))
	if body := rr.Body.String(); body != "1 elements" {
		t.Fatalf("body is %s", body)
	}
}

func TestCacheOnlySuccess(t *testing.T) {
	handleCount := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleCount++
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte("Hello world"))
	})
	req, _ := http.NewRequest("GET", "/", nil)
	mw := Middleware(handler)

	mw.ServeHTTP(httptest.NewRecorder(), req)
	mw.ServeHTTP(httptest.NewRecorder(), req)

	if handleCount != 2 {
		t.Fatalf("Handler called %d times", handleCount)
	}
}

func TestMaxAge(t *testing.T) {
	handleCount := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleCount++
		w.Header().Add("cache-control", "max-age=1")
		w.Write([]byte("Hello world"))
	})
	req, _ := http.NewRequest("GET", "/", nil)
	mw := Middleware(handler)

	mw.ServeHTTP(httptest.NewRecorder(), req)
	time.Sleep(time.Second)
	mw.ServeHTTP(httptest.NewRecorder(), req)

	if handleCount != 2 {
		t.Fatalf("Handler called %d times", handleCount)
	}
}

func TestChiMiddleware(t *testing.T) {
	listLength := 0
	r := chi.NewRouter()
	r.Get("/chi", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(fmt.Sprintf("List %d items", listLength)))
	})
	r.Get("/chi-list", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(fmt.Sprintf("List %d items", listLength)))
	})
	r.Post("/chi", func(w http.ResponseWriter, r *http.Request) {
		listLength++
		w.Header().Add("cache-update", "/chi-list")
		w.Write([]byte("post"))
	})
	handler := Middleware(r)

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/chi", nil))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/chi", nil))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/chi", nil))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/chi", nil))

	if rec.Result().StatusCode != http.StatusOK {
		t.Fatalf("Status code is %d", rec.Result().StatusCode)
	}
	if rec.Body.String() != "List 1 items" {
		t.Fatalf("body is %s", rec.Body.String())
	}
}
