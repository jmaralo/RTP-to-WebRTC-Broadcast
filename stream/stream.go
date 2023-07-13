package stream

import (
	"net"

	"github.com/google/uuid"
	"github.com/jmaralo/webrtc-broadcast/peer"
)

type Stream struct {
	channel *SPMC[[]byte]
	conn    *net.UDPConn
	config  Config
}

func New(conn *net.UDPConn, config Config) *Stream {
	stream := &Stream{
		channel: NewSPMC[[]byte](config.Channel),
		conn:    conn,
		config:  config,
	}

	go stream.run()

	return stream
}

func (stream *Stream) TrackConfig() peer.TrackConfig {
	return peer.TrackConfig{
		Codec: stream.config.Codec,
		ID:    stream.config.Id,
		Label: stream.config.StreamID,
	}
}

func (stream *Stream) run() {
	defer close(stream.channel.Input)
	for {
		readBuf := make([]byte, stream.config.BufferSize)
		n, err := stream.conn.Read(readBuf)
		if err != nil {
			return
		}

		stream.channel.Input <- readBuf[:n]
	}
}

func (stream *Stream) Subscribe(bufSize int) (uuid.UUID, <-chan []byte, error) {
	return stream.channel.AddOutput(bufSize)
}

func (stream *Stream) Unsubscribe(id uuid.UUID) {
	stream.channel.RemoveOutput(id)
}
