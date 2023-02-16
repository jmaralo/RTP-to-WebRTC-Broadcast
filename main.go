package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/jmaralo/rtp-to-webrtc-broadcast/broadcast"
	"github.com/pion/webrtc/v3"
)

var streamURL = flag.String("i", "127.0.0.1:1234", "URL for the RTP stream")
var trackID = flag.String("tid", "rtp", "The ID that identifies the video track")
var streamID = flag.String("sid", "video", "The ID that identifies the video stream")
var mtu = flag.Uint("mtu", 1500, "The MTU of the interface where the RTP stream is received")
var config = flag.String("config", "./config.json", "Configuration file for the Peers")
var signalURL = flag.String("o", "127.0.0.1:2345", "URL to signal the connection, the actual endpoint ws://<signal url>/signal")

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

	broadcastHandle, err := broadcast.NewBroadcastHandle(*streamURL, *streamID, *trackID, *mtu, webrtc.Configuration{})
	if err != nil {
		log.Println(err)
	}

	http.Handle("/signal", broadcastHandle)
	fmt.Printf("Listening for signals on ws://%s/signal\n", *signalURL)
	log.Println(http.ListenAndServe(*signalURL, nil))
}
