package stream

import (
	"net"

	"github.com/pion/webrtc/v3"
	"github.com/rs/zerolog/log"
)

type Stream struct {
	Errors    <-chan error
	errorChan chan<- error

	stopChan chan struct{}

	track  *webrtc.TrackLocalStaticRTP
	conn   *net.UDPConn
	config Config
}

func New(conn *net.UDPConn, config Config) (*Stream, error) {
	log.Info().Str("raddr", conn.LocalAddr().String()).Msg("new stream")
	track, err := webrtc.NewTrackLocalStaticRTP(config.Codec, config.Id, config.StreamID)
	if err != nil {
		return nil, err
	}

	errorChan := make(chan error, 1)

	stream := &Stream{
		Errors:    errorChan,
		errorChan: errorChan,

		stopChan: make(chan struct{}),

		track:  track,
		conn:   conn,
		config: config,
	}

	go stream.listen(config.Mtu)

	return stream, nil
}

func (stream *Stream) Track() *webrtc.TrackLocalStaticRTP {
	return stream.track
}

func (stream *Stream) listen(mtu int) {
	log.Debug().Msg("stream listening")
	defer log.Debug().Msg("stream stopped listening")
	defer stream.conn.Close()

	for {
		buffer := make([]byte, mtu)
		n, err := stream.conn.Read(buffer)
		if err != nil {
			log.Error().Err(err).Msg("failed to read from conn")
			stream.errorChan <- err
			return
		}

		log.Trace().Msg("writing to track")
		if _, err := stream.track.Write(buffer[:n]); err != nil {
			log.Error().Err(err).Msg("failed to write to track")
			stream.errorChan <- err
			return
		}

		select {
		case <-stream.stopChan:
			log.Trace().Msg("cancel listen received")
			return
		default:
		}
	}
}

func (stream *Stream) Stop() {
	stream.stopChan <- struct{}{}
}
