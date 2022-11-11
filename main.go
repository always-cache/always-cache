package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		// this is a warkaround to remove default headers sent by an upstream proxy
		// some servers do not like the presence of these headers in the downstream request
		// also remove conditional request headers, since they are not supported
		if k != "X-Forwarded-For" && k != "X-Forwarded-Proto" && k != "X-Forwarded-Host" &&
			k != "If-None-Match" && k != "If-Modified-Since" {
			for _, v := range vv {
				dst.Add(k, v)
			}
		}
	}
}

func main() {
	downstream := flag.String("downstream", "", "URL for downstream server")
	port := flag.String("port", "8080", "Local port for incoming requests")
	defaultMaxAge := flag.Duration("max-age", time.Hour, "Default max age if not set in response")
	methods := flag.String("methods", "", "Additional methods to cache, comma-separated")
	provider := flag.String("provider", "sqlite", "Cache provider to use")
	noUpdate := flag.Bool("no-update", false, "Disable automatic updates of stale content")
	flag.Parse()

	// print set flags for debugging
	flag.VisitAll(func(flag *flag.Flag) {
		log.Debug().Msgf("Flag %s: %s", flag.Name, flag.Value)
	})

	if *downstream == "" || *port == "" {
		flag.Usage()
		os.Exit(1)
	}

	conf := Config{
		DefaultMaxAge:  *defaultMaxAge,
		DisableUpdates: *noUpdate,
	}
	switch *provider {
	case "sqlite":
		conf.Cache = NewSQLiteCache()
	case "memory":
		conf.Cache = NewMemCache()
	default:
		panic(fmt.Sprintf("Unsupported cache provider: %s", *provider))
	}
	if *methods != "" {
		conf.Methods = strings.Split(*methods, ",")
	}
	acache := New(conf)

	downstreamURL, err := url.Parse(*downstream)
	if err != nil {
		panic(err)
	}

	client := &http.Client{
		// do not follow redirects
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req, err := http.NewRequest(r.Method, downstreamURL.String()+r.URL.RequestURI(), r.Body)
		copyHeader(req.Header, r.Header)
		req.Header.Set("Host", downstreamURL.Host)
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
	log.Info().Msgf("Proxying port %s to %s", *port, downstreamURL)
	err = http.ListenAndServe(":"+*port, acache.Middleware(handler))
	if err != nil {
		panic(err)
	}
}
