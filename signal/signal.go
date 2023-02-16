package signal

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/jmaralo/rtp-to-webrtc-broadcast/common"
)

type SignalHandle struct {
	peer   *websocket.Conn
	peerMx *sync.Mutex
	events *common.SyncMap[string, func(json.RawMessage)]
}

func NewSignalHandle(conn *websocket.Conn) *SignalHandle {
	return &SignalHandle{
		peer:   conn,
		peerMx: &sync.Mutex{},
		events: common.NewSyncMap[string, func(json.RawMessage)](),
	}
}

func (sh *SignalHandle) Send(message Message) error {
	sh.peerMx.Lock()
	defer sh.peerMx.Unlock()
	return sh.peer.WriteJSON(message)
}

func (sh *SignalHandle) Listen() {
	var msg = new(Message)
	for {
		if err := sh.peer.ReadJSON(msg); err != nil {
			log.Println(err)
			return
		}

		if action, exists := sh.events.Get(msg.Type); exists {
			action(msg.Payload)
		}
	}
}

func (sh *SignalHandle) Close() error {
	return sh.peer.Close()
}

func (sh *SignalHandle) SetEvent(key string, value func(json.RawMessage)) {
	sh.events.Set(key, value)
}

func (sh *SignalHandle) UnsetEvent(key string) {
	sh.events.Del(key)
}
