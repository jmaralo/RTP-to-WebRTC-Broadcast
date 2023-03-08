package websocket_handle

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/jmaralo/rtp-to-webrtc-broadcast/signal"
)

type WebsocketHandle struct {
	upgrader     *websocket.Upgrader
	onConnection func(conn *signal.SignalHandle)
}

func NewWebsocketHandle(onConnection func(*signal.SignalHandle)) *WebsocketHandle {
	return &WebsocketHandle{
		upgrader: &websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		onConnection: onConnection,
	}
}

func (handle *WebsocketHandle) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()
	conn, err := handle.upgrader.Upgrade(writer, request, nil)
	if err != nil {
		log.Printf("WebsocketHandle: ServeHTTP: %s\n", err)
		return
	}
	handle.onConnection(signal.NewSignalHandle(conn))
}
