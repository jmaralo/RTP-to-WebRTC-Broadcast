package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/jmaralo/rtp-to-webrtc-broadcast/connection"
	"github.com/jmaralo/rtp-to-webrtc-broadcast/listener"
	websocket_handle "github.com/jmaralo/rtp-to-webrtc-broadcast/websocket"
	"github.com/pion/interceptor"
	"github.com/pion/webrtc/v3"
)

var streamURL = flag.String("i", "127.0.0.1:1234", "URL for the RTP stream")
var trackID = flag.String("tid", "rtp", "The ID that identifies the video track")
var streamID = flag.String("sid", "video", "The ID that identifies the video stream")
var mtu = flag.Int("mtu", 1500, "The MTU of the interface where the RTP stream is received")
var config = flag.String("config", "./config.json", "Configuration file for the Peers")
var signalURL = flag.String("o", "127.0.0.1:2345", "URL to signal the connection, the actual endpoint ws://<signal url>/signal")
var peerLimit = flag.Int("p", 15, "max amout of peers that can connect at once")

func main() {
	flag.Parse()

	configRaw, err := os.ReadFile(*config)
	if err != nil {
		log.Fatalln(err)
	}

	var conf = new(webrtc.Configuration)
	if err := json.Unmarshal(configRaw, conf); err != nil {
		log.Fatalln(err)
	}

	listener, err := listener.NewRTPListener(*streamURL, *mtu)
	if err != nil {
		log.Fatalf("NewRTPListener: %s\n", err)
	}

	mediaEngine := webrtc.MediaEngine{}
	if err := mediaEngine.RegisterDefaultCodecs(); err != nil {
		log.Fatalf("RegisterDefaultCodecs: %s\n", err)
	}

	interceptorRegistry := interceptor.Registry{}

	if err := webrtc.ConfigureRTCPReports(&interceptorRegistry); err != nil {
		log.Fatalf("ConfigureRTCPReports: %s\n", err)
	}

	if err := webrtc.ConfigureTWCCSender(&mediaEngine, &interceptorRegistry); err != nil {
		log.Fatalf("ConfigureTWCCSender: %s\n", err)
	}

	webrtcAPI := webrtc.NewAPI(webrtc.WithMediaEngine(&mediaEngine), webrtc.WithInterceptorRegistry(&interceptorRegistry))
	connHandle := connection.NewConnectionHandle(listener, *peerLimit, connection.NewPeerData(*trackID, *streamID, *mtu, *conf, webrtcAPI))

	wsHandle := websocket_handle.NewWebsocketHandle(connHandle.AddPeer)

	http.Handle("/signal", wsHandle)
	http.Handle("/", http.FileServer(http.Dir("./build")))
	fmt.Printf("Listening for signals on ws://%s/signal\n", *signalURL)
	log.Println(http.ListenAndServe(*signalURL, nil))
}
