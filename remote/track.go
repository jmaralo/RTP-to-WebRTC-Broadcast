package remote

import (
	"github.com/jmaralo/webrtc-broadcast/stream"
	"github.com/pion/webrtc/v3"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// addTracks adds tracks to a peer connection from the global handle
func (remote *RemotePeer) addTracks() error {
	remote.log.Info().Msg("Adding tracks")

	tracks, err := stream.Get().AddTracks()
	if err != nil {
		return errors.Wrap(err, "AddTracks")
	}

	for _, track := range tracks {
		err = remote.addTrack(track)
		if err != nil {
			stream.Get().RemoveTracks(tracks)
			return errors.Wrap(err, "AddTracks")
		}
	}

	remote.tracks = tracks

	return nil
}

// addTrack adds and runs a single track to the remote peer
func (remote *RemotePeer) addTrack(track stream.Track) error {
	remote.log.Debug().Str("trackID", track.ID.String()).Msg("Adding track")
	rtpSender, err := remote.peer.AddTrack(track.Track)
	if err != nil {
		return errors.Wrap(err, "AddTrack")
	}

	go readRTCPFromRTPSender(rtpSender)

	return nil
}

// removeTracks removes the peer connecion tracks from the global handler
func (remote *RemotePeer) removeTracks() {
	remote.log.Warn().Msg("Removing tracks")
	stream.Get().RemoveTracks(remote.tracks)
}

// readRTCPFromRTPSender reads from the rtp sender to propagate rtcp packets
func readRTCPFromRTPSender(rtpSender *webrtc.RTPSender) {
	log.Trace().Msg("Reading RTCP from RTPSender")
	for {
		_, _, err := rtpSender.ReadRTCP()
		if err != nil {
			log.Error().Err(err).Msg("Error reading RTCP from RTPSender")
			return
		}
	}
}
