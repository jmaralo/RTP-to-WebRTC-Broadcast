package channel

import (
	"errors"
	"time"

	"github.com/gorilla/websocket"
)

type Channel struct {
	Read     <-chan Signal
	readChan chan<- Signal

	Write     chan<- Signal
	writeChan <-chan Signal

	Errors     <-chan error
	errorsChan chan<- error

	pollChan chan struct{}

	closeChan chan closeConfig

	conn   *websocket.Conn
	config Config
}

func New(conn *websocket.Conn, config Config) *Channel {
	readChan := make(chan Signal, config.ReadBuffer)
	writeChan := make(chan Signal, config.WriteBuffer)
	errorsChan := make(chan error, 5)

	channel := &Channel{
		Read:     readChan,
		readChan: readChan,

		Write:     writeChan,
		writeChan: writeChan,

		Errors:     errorsChan,
		errorsChan: errorsChan,

		pollChan: make(chan struct{}, config.MaxPendingPings),

		closeChan: make(chan closeConfig),

		conn:   conn,
		config: config,
	}

	channel.conn.SetPongHandler(channel.onPong)
	channel.conn.SetCloseHandler(channel.onClose)

	go channel.read()
	go channel.write()
	go channel.ping()
	go channel.close()

	return channel
}

func (channel *Channel) read() {
	defer close(channel.readChan)
	for {
		var signal Signal
		err := channel.conn.ReadJSON(&signal)
		if err != nil {
			channel.tryClose(websocket.CloseInternalServerErr, err.Error())
			channel.errorsChan <- err
			return
		}

		channel.readChan <- signal
	}
}

func (channel *Channel) write() {
	defer channel.tryClose(websocket.CloseNormalClosure, "no more data to send")
	for signal := range channel.writeChan {
		err := channel.conn.WriteJSON(signal)
		if err != nil {
			channel.tryClose(websocket.CloseInternalServerErr, err.Error())
			channel.errorsChan <- err
			return
		}
	}
}

func (channel *Channel) ping() {
	ticker := time.NewTicker(channel.config.PingInterval)
	for range ticker.C {
		err := channel.conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(channel.config.PingInterval))
		if err != nil {
			channel.tryClose(websocket.CloseInternalServerErr, err.Error())
			channel.errorsChan <- err
			return
		}

		select {
		case channel.pollChan <- struct{}{}:
		default:
			channel.tryClose(websocket.ClosePolicyViolation, "too many pending pings")
			channel.errorsChan <- errors.New("too many pending pings")
			return
		}
	}
}

func (channel *Channel) onPong(appData string) error {
	for {
		select {
		case <-channel.pollChan:
		default:
			return nil
		}
	}
}

func (channel *Channel) onClose(code int, text string) error {
	channel.tryClose(websocket.CloseNormalClosure, "ok")
	return nil
}

func (channel *Channel) tryClose(code int, reason string) bool {
	select {
	case channel.closeChan <- closeConfig{Code: code, Text: reason}:
		return true
	default:
		return false
	}
}

func (channel *Channel) close() {
	defer channel.conn.Close()

	options := <-channel.closeChan

	deadline := time.Now().Add(channel.config.DisconnectTimeout)
	message := websocket.FormatCloseMessage(options.Code, options.Text)
	err := channel.conn.WriteControl(websocket.CloseMessage, message, deadline)
	if err != nil {
		channel.errorsChan <- err
	}
}
