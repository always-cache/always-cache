package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
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
	configFile := flag.String("config", "config.yml", "Path to config file")
	flag.Parse()

	config, err := getConfig(*configFile)
	if err != nil {
		panic(err)
	}

	if config.Port <= 0 || len(config.Origins) != 1 {
		fmt.Println("Need port and exactly one origin")
		os.Exit(1)
	}

	origin := config.Origins[0]

	if len(origin.Paths) > 0 {
		log.Fatal().Msg("Path-based overrides not yet supported")
	}

	acache := AlwaysCache{}

	// if updates not disabled, update every minute
	if !origin.DisableUpdate {
		acache.updateTimeout = time.Minute
	}

	// set defaults to configured origin defaults
	acache.defaults = origin.Defaults

	// set paths
	acache.paths = origin.Paths

	// use configured provider, panic if none specified
	switch config.Provider {
	case "sqlite":
		acache.cache = NewSQLiteCache()
	case "memory":
		acache.cache = NewMemCache()
	default:
		panic(fmt.Sprintf("Unsupported cache provider: %s", config.Provider))
	}

	// get the downstream server address
	downstreamURL, err := url.Parse(origin.Origin)
	if err != nil {
		panic(err)
	}
	acache.originURL = downstreamURL

	// initialize
	acache.Init()

	log.Info().Msgf("Proxying port %v to %s", config.Port, downstreamURL)
	err = http.ListenAndServe(fmt.Sprintf(":%d", config.Port), &acache)
	if err != nil {
		panic(err)
	}
}
