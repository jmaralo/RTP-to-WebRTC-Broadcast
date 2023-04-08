package stream

import (
	"fmt"
	"net"

	"github.com/pion/webrtc/v3"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// handle stores the global StreamHandle
var handle *StreamHandle

// InitHandle initializes the global StreamHandle
func Init(addrs []string) error {
	var err error
	handle, err = newHandle(addrs)
	if err != nil {
		return errors.Wrap(err, "InitHandle")
	}

	return nil
}

// GetHandle returns the global StreamHandle
func Get() *StreamHandle {
	return handle
}

// StreamHandle will manage opening and closing new observers on the streams
type StreamHandle struct {
	streams map[string]*Stream
}

// newHandle creates a new StreamHandle from some addrs
func newHandle(addrs []string) (*StreamHandle, error) {
	streams, err := createStreams(addrs)
	if err != nil {
		return nil, errors.Wrap(err, "NewHandle")
	}

	return &StreamHandle{
		streams: streams,
	}, nil
}

// GetTracks returns a map of tracks created from the streams
func (handle *StreamHandle) AddTracks() (map[string]Track, error) {
	log.Info().Msg("Adding tracks")
	tracks := make(map[string]Track)
	for addr, stream := range handle.streams {
		track, err := stream.newTrack()
		if err != nil {
			return nil, errors.Wrap(err, "GetTracks")
		}

		tracks[addr] = track
		log.Debug().Str("addr", addr).Msg("Added track")
	}

	return tracks, nil
}

// RemoveTracks closes the observers for the given track map
func (handle *StreamHandle) RemoveTracks(tracks map[string]Track) {
	log.Warn().Msg("Removing tracks")
	for _, track := range tracks {
		for _, stream := range handle.streams {
			stream.closeObserver(track.ID)
			log.Debug().Msg("Removed track")
		}
	}
}

// Close closes the stream handle
func (handle *StreamHandle) Close() error {
	log.Warn().Msg("Closing stream handle")
	var err error
	for _, stream := range handle.streams {
		closeErr := stream.close()
		if closeErr != nil {
			err = errors.Wrap(closeErr, "Close")
		}
	}
	return err
}

// createStreams uses createConn to create a map of streams
func createStreams(addrs []string) (map[string]*Stream, error) {
	streams := make(map[string]*Stream)
	for _, addr := range addrs {
		conn, err := createConn(addr)
		if err != nil {
			return nil, errors.Wrap(err, "createStreams")
		}

		streams[addr] = new(conn, webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264}, "video", fmt.Sprintf("stream_%s", addr))
		log.Debug().Str("addr", addr).Msg("Created stream")
	}

	return streams, nil
}

// createConn resolves laddr and creates and udp listener
func createConn(laddr string) (*net.UDPConn, error) {
	addr, err := net.ResolveUDPAddr("udp", laddr)
	if err != nil {
		return nil, errors.Wrap(err, "createConn")
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, errors.Wrap(err, "createConn")
	}

	return conn, nil
}
