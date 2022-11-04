package main

import (
	"flag"
	"io"
	"net/http"
	"os"
	"time"

	cache "github.com/ericselin/always-cache"
)

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		// this is a warkaround to remove default headers sent by an upstream proxy
		// some servers do not like the presence of these headers in the downstream request
		if k != "X-Forwarded-For" && k != "X-Forwarded-Proto" && k != "X-Forwarded-Host" {
			for _, v := range vv {
				dst.Add(k, v)
			}
		}
	}
}

var (
	host string
	port string
)

func init() {

}

func main() {
	host := flag.String("h", "", "Hostname for downstream (HTTPS) server")
	port := flag.String("p", "8080", "Local port for incoming requests")
	defaultMaxAge := flag.Duration("max-age", time.Hour, "Default max age if not set in response")
	flag.Parse()

	if *host == "" || *port == "" {
		flag.Usage()
		os.Exit(1)
	}

	acache := cache.New(cache.Config{
		Methods:       []string{"POST"},
		Cache:         cache.NewSQLiteCache(),
		DefaultMaxAge: *defaultMaxAge,
	})
	client := &http.Client{}

	downstream := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req, err := http.NewRequest(r.Method, "https://"+*host+r.URL.RequestURI(), r.Body)
		copyHeader(req.Header, r.Header)
		req.Header.Set("Host", *host)
		if err != nil {
			panic(err)
		}
		resp, err := client.Do(req)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()

		copyHeader(w.Header(), resp.Header)
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	})
	http.ListenAndServe(":"+*port, acache.Middleware(downstream))
}
