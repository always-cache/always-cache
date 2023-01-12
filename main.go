package main

import (
	"flag"
	"io"
	"net/url"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	configFilenameFlag string
	portFlag           int
	originFlag         string
	addrFlag           string
	hostFlag           string
	providerFlag       string
	legacyModeFlag     bool
	verbosityTraceFlag bool
	logFilenameFlag    string
)

func init() {
	flag.StringVar(&configFilenameFlag, "config", "", "Path to config file")
	flag.StringVar(&originFlag, "origin", "", "Origin URL to proxy to (overrides addr and host)")
	flag.StringVar(&addrFlag, "addr", "", "Origin IP address to proxy to")
	flag.StringVar(&hostFlag, "host", "", "Hostname of origin")
	flag.IntVar(&portFlag, "port", 8080, "Port to listen on")
	flag.StringVar(&providerFlag, "provider", "sqlite", "Caching provider to use")
	flag.BoolVar(&legacyModeFlag, "legacy", false, "Legacy mode: do not update, only invalidate if needed")
	flag.BoolVar(&verbosityTraceFlag, "vv", false, "Verbosity: trace logging")
	flag.StringVar(&logFilenameFlag, "log-file", "", "Log file to use (in addition to stdout)")
}

func main() {
	flag.Parse()

	logLevel := zerolog.DebugLevel
	if verbosityTraceFlag {
		logLevel = zerolog.TraceLevel
	}
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
	log.Logger = log.Level(logLevel).Output(multiWriter)

	acache := AlwaysCache{
		invalidateOnly: legacyModeFlag,
	}

	if configFilenameFlag != "" {
		config, err := getConfig(configFilenameFlag)
		if err != nil {
			log.Error().Err(err).Msg("Cannot get config")
		} else if len(config.Origins) != 1 {
			log.Error().Msg("Only exactly one origin supported")
		} else {
			acache.rules = config.Origins[0].Rules
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

	// get the downstream server address
	if originFlag != "" {
		originUrl, err := url.Parse(originFlag)
		if err != nil {
			log.Fatal().Err(err).Msg("Clould not parse url")
		}
		acache.originURL = originUrl
	} else if addrFlag != "" {
		originUrl, err := url.Parse("https://" + addrFlag)
		if err != nil {
			log.Fatal().Err(err).Msg("Clould not parse url")
		}
		acache.originURL = originUrl
		acache.originHost = hostFlag
	} else {
		log.Fatal().Msg("Please specify origin")
	}

	// set the port to listen on
	acache.port = portFlag

	// initialize
	err := acache.Run()

	if err != nil {
		panic(err)
	}
}
