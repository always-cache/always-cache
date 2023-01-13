package core

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func init() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
}

func startTestServer(handler *http.ServeMux, port int) (AlwaysCache, *http.Server) {
	// start server
	server := http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: handler,
	}
	go func() {
		err := server.ListenAndServe()
		if err != http.ErrServerClosed {
			panic(err)
		}
	}()
	// start set up acache
	url, _ := url.Parse(fmt.Sprintf("http://localhost:%d", port))
	acache := AlwaysCache{
		Cache:         NewMemCache(),
		OriginURL:     url,
		UpdateTimeout: time.Second / 2,
	}
	acache.Init()
	// wait a small while to ensure server is up
	time.Sleep(time.Millisecond * 200)

	return acache, &server
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
		w.Header().Add("Cache-Control", "max-age=60")
		w.Write([]byte(fmt.Sprintf("Called %d times", handleCount)))
	})
	req, _ := http.NewRequest("POST", "/update", nil)
	countReq, _ := http.NewRequest("GET", "/count", nil)

	acache, server := startTestServer(mux, 9001)
	rr := httptest.NewRecorder()

	acache.ServeHTTP(httptest.NewRecorder(), countReq)
	acache.ServeHTTP(httptest.NewRecorder(), req)
	acache.ServeHTTP(rr, countReq)

	if body, err := io.ReadAll(rr.Result().Body); err != nil || fmt.Sprintf("%s", body) != "Called 2 times" {
		t.Fatalf("Body is %s", body)
	}
	server.Shutdown(context.Background())
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
		w.Header().Add("Cache-Control", "max-age=60")
		w.Write([]byte(fmt.Sprintf("So you wanted to %s?", r.Method)))
	})
	get, _ := http.NewRequest("GET", "/", nil)
	post, _ := http.NewRequest("POST", "/", nil)
	mux := http.NewServeMux()
	mux.HandleFunc("/", handler)
	mw, server := startTestServer(mux, 9002)

	mw.ServeHTTP(httptest.NewRecorder(), get)
	assertCount(1)
	mw.ServeHTTP(httptest.NewRecorder(), get)
	assertCount(1)
	mw.ServeHTTP(httptest.NewRecorder(), post)
	// post will first do the actual post and then refresh with a get
	assertCount(3)
	mw.ServeHTTP(httptest.NewRecorder(), get)
	assertCount(3)

	server.Shutdown(context.Background())
}

func TestUpdateBeforeResponding(t *testing.T) {
	listCount := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/list", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Second)
		w.Header().Add("Cache-Control", "max-age=60")
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
	mw, server := startTestServer(mux, 9003)

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

	server.Shutdown(context.Background())
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
		w.Header().Add("cache-control", "max-age=2")
		w.Write([]byte(response))
	})
	mux := http.NewServeMux()
	mux.HandleFunc("/", handler)
	mw, server := startTestServer(mux, 9004)
	req, _ := http.NewRequest("GET", "/", nil)

	mw.ServeHTTP(httptest.NewRecorder(), req)
	response = "Hello world 2"
	time.Sleep(time.Second * 3)
	response = ""
	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, req)

	if body := rr.Body.String(); body != "Hello world 2" {
		t.Fatalf("body is %s", body)
	}

	server.Shutdown(context.Background())
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
		w.Header().Add("Cache-Control", "max-age=60")
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
	mw, server := startTestServer(mux, 9005)
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

	server.Shutdown(context.Background())
}
