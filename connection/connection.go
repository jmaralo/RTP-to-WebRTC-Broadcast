package connection

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/jmaralo/webrtc-broadcast/peer"
)

type WebSocketHandle struct {
	upgrader *websocket.Upgrader
}

func New() *WebSocketHandle {
	return &WebSocketHandle{
		upgrader: &websocket.Upgrader{
			CheckOrigin: func(*http.Request) bool { return true },
		},
	}
}

func (handle *WebSocketHandle) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()
	log.Println("[DEBUG] New WebSocket connection")

	conn, err := handle.upgrader.Upgrade(writer, request, nil)
	if err != nil {
		log.Printf("[ERROR] WebSocketHandle: ServeHTTP: Upgrade: %s\n", err)
		return
	}

	remote, err := peer.New(conn)
	if err != nil {
		log.Printf("[ERROR] WebSocketHandle: ServeHTTP: peer.New: %s\n", err)
	}

	fmt.Println(remote)
}
