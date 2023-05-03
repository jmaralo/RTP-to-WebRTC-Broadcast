package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jmaralo/webrtc-broadcast/channel"
	"github.com/jmaralo/webrtc-broadcast/connection"
	"github.com/jmaralo/webrtc-broadcast/peer"
	"github.com/jmaralo/webrtc-broadcast/stream"
	"github.com/pion/webrtc/v3"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
)

var streamsAddr = flag.String("i", "192.168.0.9:9090,192.168.0.9:9091,192.168.0.9:9092", "comma separated list of RTP streams")
var localAddr = flag.String("o", "192.168.0.9:4040", "address to listen on")
var maxPeers = flag.Uint("p", 300, "maximum number of peers")
var logLevel = flag.String("l", "info", "logging level")
var pingInterval = flag.Duration("ping", time.Second*5, "ping interval")
var disconnectTimeout = flag.Duration("disconnect", time.Second*10, "disconnect timeout")
var mtu = flag.Int("mtu", 1500, "MTU")

func main() {
	flag.Parse()
	initLogger()

	manager := connection.NewManager(peer.Config{
		OfferOptions:  webrtc.OfferOptions{},
		AnswerOptions: webrtc.AnswerOptions{},
		Mtu:           *mtu,
	}, channel.Config{
		ReadBuffer:        100,
		WriteBuffer:       100,
		DataType:          websocket.TextMessage,
		PingInterval:      *pingInterval,
		MaxPendingPings:   3,
		DisconnectTimeout: *disconnectTimeout,
	}, consumeTrack, int(*maxPeers))

	addrs := strings.Split(*streamsAddr, ",")
	conns := make([]*net.UDPConn, len(addrs))
	for i, addr := range addrs {
		raddr, err := net.ResolveUDPAddr("udp", addr)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to resolve UDP address")
		}
		conn, err := net.ListenUDP("udp", raddr)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to listen on UDP address")
		}
		conns[i] = conn
	}

	for i, conn := range conns {
		stream, err := stream.New(conn, stream.Config{
			Codec: webrtc.RTPCodecCapability{
				MimeType:  webrtc.MimeTypeH264,
				ClockRate: 90000,
			},
			Id:       fmt.Sprint(i),
			StreamID: fmt.Sprint(i),
			Mtu:      *mtu,
		})
		if err != nil {
			log.Fatal().Err(err).Msg("failed to create stream")
		}
		manager.AddStream(stream)
	}

	http.Handle("/", manager)
	log.Info().Str("addr", *localAddr).Msg("listening")
	http.ListenAndServe(*localAddr, nil)
}

func consumeTrack(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
	go consumeReceiver(receiver)
	go consumeTrackRemote(track)
}

func consumeReceiver(receiver *webrtc.RTPReceiver) {
	for {
		_, _, err := receiver.Read(make([]byte, 1500))
		if err != nil {
			log.Error().Err(err).Msg("failed to read from receiver")
			return
		}
	}
}

func consumeTrackRemote(track *webrtc.TrackRemote) {
	for {
		_, _, err := track.Read(make([]byte, 1500))
		if err != nil {
			log.Error().Err(err).Msg("failed to read from track")
			return
		}
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
