package remote

import (
	"encoding/json"

	"github.com/pion/webrtc/v3"
	"github.com/pkg/errors"
)

// onNegotiationNeeded is called when the peer connection needs to negotiate
func (remote *RemotePeer) onNegotiationNeeded() {
	remote.log.Info().Msg("negotiation needed")

	err := remote.generateOffer()
	if err != nil {
		remote.log.Error().Err(err).Msg("generate offer")
		remote.Close()
		return
	}
}

// generateOffer generates the local description for the peer connection
func (remote *RemotePeer) generateOffer() error {
	remote.log.Debug().Msg("generate offer")

	offer, err := remote.peer.CreateOffer(remote.offerConfiguration)
	if err != nil {
		return errors.Wrap(err, "generateLocalDescription")
	}

	err = remote.peer.SetLocalDescription(offer)
	if err != nil {
		return errors.Wrap(err, "generateLocalDescription")
	}

	err = remote.signal.SendSignal("offer", remote.peer.LocalDescription())
	if err != nil {
		return errors.Wrap(err, "generateLocalDescription")
	}

	return nil
}

// onSignalAnswer is called when the signaling channel receives an answer
func (remote *RemotePeer) onSignalAnswer(payload json.RawMessage) {
	remote.log.Info().Msg("received answer")

	answer, err := Unmarshal[webrtc.SessionDescription](payload)
	if err != nil {
		remote.log.Error().Err(err).Msg("unmarshal answer signal")
		remote.Close()
		return
	}

	err = remote.peer.SetRemoteDescription(answer)
	if err != nil {
		remote.log.Error().Err(err).Msg("set remote description")
		remote.Close()
		return
	}
}
