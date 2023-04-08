package signal

import (
	"encoding/json"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/jmaralo/webrtc-broadcast/signal/model"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Channel abstracts the signaling channel
// underneath it uses a websocket connection
type Channel struct {
	// connMx is used to protect the conn
	connMx *sync.Mutex
	conn   *websocket.Conn
	// listenersMx is used to protect the listeners map
	listenersMx *sync.Mutex
	listeners   map[string]func(json.RawMessage)

	log zerolog.Logger
}

// New creates a new signaling channel
func New(conn *websocket.Conn) *Channel {
	channel := &Channel{
		conn:        conn,
		connMx:      &sync.Mutex{},
		listenersMx: &sync.Mutex{},
		listeners:   make(map[string]func(json.RawMessage)),
		log:         log.With().Str("addr", conn.RemoteAddr().String()).Logger(),
	}

	go channel.listen()

	return channel
}

// listen listens for incoming messages and calls the corresponding listener
func (channel *Channel) listen() {
	channel.log.Info().Msg("Listening for signals")
	defer channel.Close()

	for {
		signal, err := channel.readSignal()
		if err != nil {
			channel.log.Error().Err(err).Msg("Error reading signal")
			channel.Close()
			return
		}

		channel.callListener(signal)
	}
}

// readSignal return the next signal received from the signaling channel
func (channel *Channel) readSignal() (model.Message, error) {
	var message model.Message
	err := channel.conn.ReadJSON(&message)
	return message, errors.Wrap(err, "readSignal")
}

// callListener calls the listener for the given signal
func (channel *Channel) callListener(signal model.Message) {
	channel.log.Debug().Str("signal", signal.Signal).Msg("Received signal")
	channel.listenersMx.Lock()
	defer channel.listenersMx.Unlock()

	if listener, ok := channel.listeners[signal.Signal]; ok {
		listener(signal.Payload)
	} else {
		channel.log.Warn().Str("signal", signal.Signal).Msg("Listener not found")
	}
}

// SendSignal sends a signal through the signaling channel
func (channel *Channel) SendSignal(signal string, payload any) error {
	channel.log.Debug().Str("signal", signal).Msg("Sending signal")

	message, err := model.NewMessage(signal, payload)
	if err != nil {
		return errors.Wrap(err, "SendSignal")
	}

	err = channel.writeSignal(message)
	if err != nil {
		return errors.Wrap(err, "SendSignal")
	}

	return nil
}

// writeSignal writes a signal through the signaling channel
func (channel *Channel) writeSignal(signal model.Message) error {
	channel.connMx.Lock()
	defer channel.connMx.Unlock()

	err := channel.conn.WriteJSON(signal)
	if err != nil {
		return errors.Wrap(err, "writeSignal")
	}

	return nil
}

// AddListener adds a listener for the given signal
func (channel *Channel) AddListener(signal string, listener func(json.RawMessage)) {
	channel.listenersMx.Lock()
	defer channel.listenersMx.Unlock()

	channel.log.Info().Str("signal", signal).Msg("Adding listener")
	channel.listeners[signal] = listener
}

// RemoveListener removes the listener for the given signal
func (channel *Channel) RemoveListener(signal string) {
	channel.listenersMx.Lock()
	defer channel.listenersMx.Unlock()

	channel.log.Warn().Str("signal", signal).Msg("Removing listener")
	delete(channel.listeners, signal)
}

// Close closes the signaling channel
func (channel *Channel) Close() error {
	channel.log.Warn().Msg("Closing signaling channel")
	return channel.conn.Close()
}
