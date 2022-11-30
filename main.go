package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	configFilenameFlag string
	legacyModeFlag     bool
	verbosityTraceFlag bool
)

func init() {
	flag.StringVar(&configFilenameFlag, "config", "config.yml", "Path to config file")
	flag.BoolVar(&legacyModeFlag, "legacy", false, "Legacy mode: do not update, only invalidate if needed")
	flag.BoolVar(&verbosityTraceFlag, "vv", false, "Verbosity: trace logging")
	flag.Parse()
}

func main() {
	logLevel := zerolog.DebugLevel
	if verbosityTraceFlag {
		logLevel = zerolog.TraceLevel
	}
	log.Logger = log.Level(logLevel).Output(zerolog.ConsoleWriter{Out: os.Stdout})

	config, err := getConfig(configFilenameFlag)
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

	acache := AlwaysCache{
		invalidateOnly: legacyModeFlag,
	}

	// if updates not disabled, update every minute
	if !legacyModeFlag && !origin.DisableUpdate {
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

	// set the port to listen on
	acache.port = config.Port

	// initialize
	err = acache.Run()

	if err != nil {
		panic(err)
	}
}
