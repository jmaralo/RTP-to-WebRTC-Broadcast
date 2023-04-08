package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/jmaralo/webrtc-broadcast/connection"
	"github.com/jmaralo/webrtc-broadcast/stream"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
)

var streamURLs = flag.String("i", "192.168.0.2:6969,192.168.0.2:6970,192.168.0.2:6971", "URL for the RTP stream")
var signalURL = flag.String("o", "192.168.0.2:4040", "URL to signal the connection, the actual endpoint ws://<signal url>/signal")
var logLevel = flag.String("logLevel", "info", "Log level to use")
var maxPeers = flag.Int("maxPeers", 10, "Maximum number of peers to allow")

func main() {
	flag.Parse()
	initLogger()

	err := stream.Init(strings.Split(*streamURLs, ","))
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize streams")
		return
	}
	defer stream.Get().Close()
	log.Info().Str("addrs", *streamURLs).Msg("Streams initialized")

	connectionHandle := connection.New(*maxPeers)
	log.Info().Msg("Connection handle initialized")

	http.Handle("/", connectionHandle)
	log.Info().Str("addr", fmt.Sprintf("ws://%s", *signalURL)).Msg("Listening for connections")
	err = http.ListenAndServe(*signalURL, nil)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to start server")
		return
	}
}

// initLogger initializes the global logger with good defaults
func initLogger() {
	zerolog.CallerMarshalFunc = func(pc uintptr, file string, line int) string {
		short := file
		for i := len(file) - 1; i > 0; i-- {
			if file[i] == '/' {
				short = file[i+1:]
				break
			}
		}
		file = short
		return file + ":" + strconv.Itoa(line)
	}
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixNano
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack

	consoleWriter := zerolog.ConsoleWriter{Out: os.Stdout}

	globalLogger := zerolog.New(consoleWriter).With().Timestamp().Caller().Logger()
	log.Logger = globalLogger

	if level, ok := logLevelMap[*logLevel]; ok {
		zerolog.SetGlobalLevel(level)
	} else {
		log.Fatal().Msg("invalid log level selected")
	}
}

var logLevelMap = map[string]zerolog.Level{
	"fatal":   zerolog.FatalLevel,
	"error":   zerolog.ErrorLevel,
	"warn":    zerolog.WarnLevel,
	"info":    zerolog.InfoLevel,
	"debug":   zerolog.DebugLevel,
	"trace":   zerolog.TraceLevel,
	"disable": zerolog.Disabled,
}
