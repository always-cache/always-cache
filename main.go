package main

import (
	"flag"
	"net/url"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	configFilenameFlag      string
	portFlag                int
	originFlag              string
	providerFlag            string
	defaultCacheControlFlag string
	rewriteOriginUrlFlag    string
	legacyModeFlag          bool
	verbosityTraceFlag      bool
)

func init() {
	flag.StringVar(&configFilenameFlag, "config", "", "Path to config file")
	flag.StringVar(&originFlag, "origin", "", "Origin to proxy to (overrides config)")
	flag.IntVar(&portFlag, "port", 8080, "Port to listen on")
	flag.StringVar(&providerFlag, "provider", "sqlite", "Caching provider to use")
	flag.StringVar(&defaultCacheControlFlag, "default", "", "Default Cache-Control header (overrides config)")
	flag.StringVar(&rewriteOriginUrlFlag, "rewrite-origin-url", "", "URL to replace origin URL with for text content")
	flag.BoolVar(&legacyModeFlag, "legacy", false, "Legacy mode: do not update, only invalidate if needed")
	flag.BoolVar(&verbosityTraceFlag, "vv", false, "Verbosity: trace logging")
}

func main() {
	flag.Parse()

	logLevel := zerolog.DebugLevel
	if verbosityTraceFlag {
		logLevel = zerolog.TraceLevel
	}
	log.Logger = log.Level(logLevel).Output(zerolog.ConsoleWriter{Out: os.Stdout})

	acache := AlwaysCache{
		invalidateOnly: legacyModeFlag,
	}

	var origin string

	if configFilenameFlag != "" {
		config, err := getConfig(configFilenameFlag)
		if err != nil {
			panic(err)
		}

		if config.Port <= 0 || len(config.Origins) != 1 {
			log.Fatal().Msg("Need port and exactly one origin")
		}

		originConfig := config.Origins[0]

		if len(originConfig.Paths) > 0 {
			log.Fatal().Msg("Path-based overrides not yet supported")
		}

		// set defaults to configured origin defaults
		acache.defaults = originConfig.Defaults

		// set paths
		acache.paths = originConfig.Paths
	}

	if originFlag != "" {
		origin = originFlag
	}

	if defaultCacheControlFlag != "" {
		acache.defaults = Defaults{
			CacheControl: defaultCacheControlFlag,
			SafeMethods:  SafeMethods{},
		}
	}

	// use configured provider, panic if none specified
	switch providerFlag {
	case "sqlite":
		acache.cache = NewSQLiteCache()
	case "memory":
		acache.cache = NewMemCache()
	default:
		log.Fatal().Msgf("Unsupported cache provider: %s", providerFlag)
	}

	// if updates not disabled, update every minute
	if !legacyModeFlag {
		acache.updateTimeout = time.Second * 15
	}

	if rewriteOriginUrlFlag != "" {
		acache.replaceOriginUrl = rewriteOriginUrlFlag
	}

	if origin == "" {
		log.Fatal().Msg("Please specify origin")
	}

	// get the downstream server address
	downstreamURL, err := url.Parse(origin)
	if err != nil {
		panic(err)
	}
	acache.originURL = downstreamURL

	// set the port to listen on
	acache.port = portFlag

	// initialize
	err = acache.Run()

	if err != nil {
		panic(err)
	}
}
