package stream

import (
	"net"
	"sync"

	"github.com/google/uuid"
	"github.com/pion/webrtc/v3"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// TODO: remove hardcoded value
const (
	// READ_BUFFER_SIZE is the size of the read buffer
	READ_BUFFER_SIZE = 1500
)

// Stream reads and broadcast an UDP stream
type Stream struct {
	conn *net.UDPConn
	// observerMx is used to protect the observers map from concurrent access
	observersMx *sync.Mutex
	observers   map[uuid.UUID]chan<- []byte

	codec webrtc.RTPCodecCapability
	id    string
	label string

	log zerolog.Logger
}

// new creates a new stream
func new(conn *net.UDPConn, codec webrtc.RTPCodecCapability, id string, label string) *Stream {
	stream := &Stream{
		conn:        conn,
		observersMx: &sync.Mutex{},
		observers:   make(map[uuid.UUID]chan<- []byte),
		codec:       codec,
		id:          id,
		label:       label,
		log:         log.With().Str("streamID", id).Str("streamLabel", label).Logger(),
	}

	go stream.listen()

	return stream
}

// listen listens for incoming data
// the data readed from the udp connection is copied and broadcasted to all the observers
func (stream *Stream) listen() {
	stream.log.Debug().Msg("stream listening")
	readBuffer := make([]byte, READ_BUFFER_SIZE)
	for {
		n, err := stream.conn.Read(readBuffer)
		if err != nil {
			stream.close()
			stream.log.Error().Err(err).Msg("stream closed")
			return
		}

		stream.broadcast(readBuffer[:n])
	}
}

// broadcast broadcasts the data to all the observers
// it first copies the data to avoid passing the same slice to all the observers by reference
// this method should not block on read when broadcasting to the observers
func (stream *Stream) broadcast(data []byte) {
	stream.observersMx.Lock()
	defer stream.observersMx.Unlock()

	for id, observer := range stream.observers {
		dataCopy := make([]byte, len(data))
		copy(dataCopy, data)

		if !sendBlocking(dataCopy, observer) {
			stream.log.Trace().Str("id", id.String()).Msg("observer channel full")
		}
	}
}

// sendNonBlocking sends data through a channel without blocking
// it returns true if the data was sent and false if the channel is full
func sendNonBlocking(data []byte, channel chan<- []byte) bool {
	select {
	case channel <- data:
		return true
	default:
		return false
	}
}

// sendBlocking sends data through a channel blocking until the data is sent
func sendBlocking(data []byte, channel chan<- []byte) bool {
	channel <- data
	return true
}

// close closes the stream
func (stream *Stream) close() error {
	stream.log.Warn().Msg("closing stream")
	stream.closeAllObservers()
	return stream.conn.Close()
}
