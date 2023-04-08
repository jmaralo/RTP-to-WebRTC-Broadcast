package stream

import (
	"github.com/google/uuid"
	"github.com/pion/webrtc/v3"
	"github.com/pkg/errors"
)

// Track associates an uuid with a track
type Track struct {
	ID    uuid.UUID
	Track *webrtc.TrackLocalStaticRTP
}

// newTrack creates a new track from the stream
func (stream *Stream) newTrack() (Track, error) {
	track, err := webrtc.NewTrackLocalStaticRTP(stream.codec, stream.id, stream.label)
	if err != nil {
		return Track{}, errors.Wrap(err, "NewTrack")
	}

	id, observerChan := stream.addObserver()
	go stream.runTrack(track, id, observerChan)

	stream.log.Debug().Str("id", id.String()).Msg("Created new track")

	return Track{
		ID:    id,
		Track: track,
	}, nil
}

// runTrack writes to a TrackLocalStaticRTP
func (stream *Stream) runTrack(track *webrtc.TrackLocalStaticRTP, id uuid.UUID, observerChan <-chan []byte) {
	stream.log.Debug().Str("id", id.String()).Msg("Running track")
	for data := range observerChan {
		// stream.log.Trace().Str("id", id.String()).Int("size", len(data)).Msg("Writing to track")
		_, err := track.Write(data)
		if err != nil {
			// FIXME: notify the remote peer of this disconnection (not a priority right now)
			stream.log.Error().Err(err).Str("id", id.String()).Msg("Error writing to track")
			stream.closeObserver(id)
		}
	}
}
