package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	alwayscache "github.com/always-cache/always-cache"
	"github.com/always-cache/always-cache/cache"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	// CLI flags
	portFlag           int
	originFlag         string
	addrFlag           string
	hostFlag           string
	dbFilenameFlag     string
	legacyModeFlag     bool
	verbosityTraceFlag bool
	logFilenameFlag    string

	// this is set by goreleaser
	version string
)

func init() {
	flag.StringVar(&originFlag, "origin", "", "Origin URL to proxy to (overrides addr and host)")
	flag.StringVar(&addrFlag, "addr", "", "Origin IP address to proxy to")
	flag.StringVar(&hostFlag, "host", "", "Hostname of origin")
	flag.IntVar(&portFlag, "port", 8080, "Port to listen on")
	flag.StringVar(&dbFilenameFlag, "db", "cache.db", "Cache DB file name (use 'memory' for in-memory db)")
	flag.BoolVar(&legacyModeFlag, "legacy", false, "Legacy mode: do not update, only invalidate if needed")
	flag.BoolVar(&verbosityTraceFlag, "vv", false, "Verbosity: trace logging")
	flag.StringVar(&logFilenameFlag, "log-file", "", "Log file to use (in addition to stdout)")

	if version == "" {
		version = "DEV"
	}
}

func main() {
	flag.Parse()

	// set log level
	logLevel := zerolog.DebugLevel
	if verbosityTraceFlag {
		logLevel = zerolog.TraceLevel
	}

	// set up log output to stdout
	// also output to logfile if specified
	logOutputs := make([]io.Writer, 0)
	logOutputs = append(logOutputs, zerolog.ConsoleWriter{Out: os.Stdout})
	if logFilenameFlag != "" {
		if logFileOutput, err := os.OpenFile(logFilenameFlag, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644); err != nil {
			log.Fatal().Err(err).Msg("Cannot open log file")
		} else {
			logOutputs = append(logOutputs, logFileOutput)
		}
	}
	multiWriter := zerolog.MultiLevelWriter(logOutputs...)
	log.Logger = log.Level(logLevel).Output(multiWriter).
		With().Str("version", version).Logger()

	// set up sqlite memory provider
	dbFilename := dbFilenameFlag
	if dbFilename == "memory" {
		dbFilename = "file::memory:?cache=shared"
	}

	// always-cache origin instance
	cacheConfig := alwayscache.Config{
		Cache:          cache.NewSQLiteCache(dbFilename),
		DisableUpdates: legacyModeFlag,
	}

	// get the downstream server address
	if originFlag != "" {
		originUrl, err := url.Parse(originFlag)
		if err != nil {
			log.Fatal().Err(err).Msg("Clould not parse url")
		}
		cacheConfig.OriginURL = *originUrl
	} else if addrFlag != "" {
		originUrl, err := url.Parse("https://" + addrFlag)
		if err != nil {
			log.Fatal().Err(err).Msg("Clould not parse url")
		}
		cacheConfig.OriginURL = *originUrl
		cacheConfig.OriginHost = hostFlag
	} else {
		log.Fatal().Msg("Please specify origin")
	}

	acache := alwayscache.CreateCache(cacheConfig)
	log.Info().Msgf("Proxying port %v to %s (with hostname '%s')", portFlag, cacheConfig.OriginURL.String(), cacheConfig.OriginHost)
	err := http.ListenAndServe(fmt.Sprintf(":%d", portFlag), acache)

	if err != nil {
		panic(err)
	}
}
