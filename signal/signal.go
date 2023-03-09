package signal

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/jmaralo/webrtc-broadcast/common"
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

func (handle *SignalHandle) Send(message Message) error {
	handle.peerMx.Lock()
	defer handle.peerMx.Unlock()
	return handle.peer.WriteJSON(message)
}

func (handle *SignalHandle) SendMessage(event string, payload any) error {
	message, err := NewMessage(event, payload)
	if err != nil {
		return err
	}

	return handle.Send(*message)
}

func (handle *SignalHandle) Listen() {
	var msg = new(Message)
	for {
		if err := handle.peer.ReadJSON(msg); err != nil {
			log.Println(err)
			return
		}

		if action, ok := handle.events.Get(msg.Event); ok {
			action(msg.Payload)
		}
	}
}

func (handle *SignalHandle) Close() error {
	return handle.peer.Close()
}

func (handle *SignalHandle) SetEvent(key string, value func(json.RawMessage)) {
	handle.events.Set(key, value)
}

func (handle *SignalHandle) UnsetEvent(key string) {
	handle.events.Del(key)
}
