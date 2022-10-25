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

	New(Config{}).Middleware(handler).ServeHTTP(rr, req)

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
	mw := New(Config{}).Middleware(handler)

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
	mw := New(Config{}).Middleware(handler)

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
	mw := New(Config{}).Middleware(mux)
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
	mw := New(Config{}).Middleware(handler)

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
	mw := New(Config{}).Middleware(mux)

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
	mw := New(Config{}).Middleware(handler)

	mw.ServeHTTP(httptest.NewRecorder(), req)
	mw.ServeHTTP(httptest.NewRecorder(), req)

	if handleCount != 2 {
		t.Fatalf("Handler called %d times", handleCount)
	}
}

// TestMaxAgeUpdate tests that the cache is updated when the max-age is reached.
// This is done by setting the max-age to 1 second and the update timeout to 0.5 seconds.
//
// This is what we will do and what we expect to happen:
// 1. Request the resource, which will be cached for 1 second.
// 2. Change the response to something new.
// 2. Sleep for 1 second, and in the meantime the cached resource should be updated.
// 3. Turn off the handler, so we know the next request will be served from the cache.
// 4. Request the resource again, which should be the updated resource served from the cache.
func TestMaxAgeUpdate(t *testing.T) {
	response := "Hello world"
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if response == "" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Add("cache-control", "max-age=1")
		w.Write([]byte(response))
	})
	mw := New(Config{UpdateTimeout: time.Second / 2}).Middleware(handler)
	req, _ := http.NewRequest("GET", "/", nil)

	mw.ServeHTTP(httptest.NewRecorder(), req)
	response = "Hello world 2"
	time.Sleep(time.Second)
	response = ""
	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, req)

	if body := rr.Body.String(); body != "Hello world 2" {
		t.Fatalf("body is %s", body)
	}
}

// TestUpdateDelay tests that the `delay` directive works as expected.
// It should delay the update of the cache by the specified amount of time.
//
// This is what we will do and what we expect to happen:
// 1. Request the resource, with a default max-age of 60 seconds.
// 2. POST an update to the resource, with a delay of 1 second.
// 3. Sleep for 100 ms just in case.
// 4. Update the response to something new.
// 5. Sleep for 1 second, and in the meantime the cached resource should be updated.
// 6. Turn off the handler, so we know the next request will be served from the cache.
// 7. Request the resource again, which should be the updated resource served from the cache.
func TestUpdateDelay(t *testing.T) {
	response := "Hello world"
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if response == "" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Write([]byte(response))
	})
	mux.HandleFunc("/update", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			w.Header().Add("cache-update", "/; delay=1")
			w.Write([]byte("done"))
		} else {
			http.Error(w, "nothing to do on get", http.StatusMethodNotAllowed)
		}
	})
	mw := New(Config{}).Middleware(mux)
	get, _ := http.NewRequest("GET", "/", nil)
	post, _ := http.NewRequest("POST", "/update", nil)
	rr := httptest.NewRecorder()
	time.Sleep(time.Millisecond * 100)

	mw.ServeHTTP(httptest.NewRecorder(), get)  // 1.
	mw.ServeHTTP(httptest.NewRecorder(), post) // 2.
	time.Sleep(time.Millisecond * 100)         // 3.
	response = "Hello world 2"                 // 4.
	time.Sleep(time.Second)                    // 5.
	response = ""                              // 6.
	mw.ServeHTTP(rr, get)                      // 7.

	if body := rr.Body.String(); body != "Hello world 2" {
		t.Fatalf("body is %s", body)
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
	handler := New(Config{}).Middleware(r)

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
