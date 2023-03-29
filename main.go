package main

import (
	"flag"
	"log"
	"net/http"
	"strings"

	"github.com/jmaralo/webrtc-broadcast/connection"
	"github.com/jmaralo/webrtc-broadcast/listeners"
)

var streamURLs = flag.String("i", "192.168.0.2:6969,192.168.0.2:6970,192.168.0.2:6971", "URL for the RTP stream")
var signalURL = flag.String("o", "192.168.0.2:4040", "URL to signal the connection, the actual endpoint ws://<signal url>/signal")

func main() {
	flag.Parse()

	if err := listeners.InitListeners(strings.Split(*streamURLs, ",")); err != nil {
		log.Printf("[ERROR] InitListeners: %s\n", err)
		return
	}

	connectionHandle := connection.New()

	http.Handle("/signal", connectionHandle)
	log.Printf("[DEBUG] Listening for signals on ws://%s/signal\n", *signalURL)
	log.Printf("[ERROR] ListenAndServe: %s\n", http.ListenAndServe(*signalURL, nil))
}
