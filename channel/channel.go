package channel

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

type Channel struct {
	listenersMx *sync.Mutex
	listeners   map[string]func(payload json.RawMessage)
	connMx      *sync.Mutex
	conn        *websocket.Conn
}

func New(conn *websocket.Conn) *Channel {
	channel := &Channel{
		listenersMx: &sync.Mutex{},
		listeners:   make(map[string]func(payload json.RawMessage)),
		connMx:      &sync.Mutex{},
		conn:        conn,
	}

	go channel.run()

	return channel
}

func (channel *Channel) run() {
	for {
		var msg message
		if err := channel.conn.ReadJSON(&msg); err != nil {
			log.Printf("[ERROR] Channel: run: ReadJSON: %s\n", err)
			return
		}

		log.Printf("[TRACE] Read %s signal\n", msg.Signal)

		channel.listenersMx.Lock()
		channel.listeners[msg.Signal](msg.Payload)
		channel.listenersMx.Unlock()
	}
}

func (channel *Channel) AddListener(signal string, listener func(payload json.RawMessage)) {
	channel.listenersMx.Lock()
	defer channel.listenersMx.Unlock()

	log.Printf("[DEBUG] Channel add %s listener\n", signal)

	channel.listeners[signal] = listener
}

func (channel *Channel) SendSignal(signal string, payload any) error {
	log.Printf("[TRACE] Send %s signal\n", signal)

	message, err := newMessage(signal, payload)
	if err != nil {
		return err
	}

	channel.connMx.Lock()
	defer channel.connMx.Unlock()

	return channel.conn.WriteJSON(message)
}

func (channel *Channel) Close() error {
	log.Printf("[WARNING] Close Channel")

	return channel.conn.Close()
}
