package remote

import (
	"encoding/json"

	"github.com/pion/webrtc/v3"
	"github.com/pkg/errors"
)

// onSignalOffer is called when the signaling channel receives an offer
func (remote *RemotePeer) onSignalOffer(payload json.RawMessage) {
	remote.log.Info().Msg("received offer")

	offer, err := Unmarshal[webrtc.SessionDescription](payload)
	if err != nil {
		remote.log.Error().Err(err).Msg("unmarshal offer signal")
		remote.Close()
		return
	}

	err = remote.handleRollback()
	if err != nil {
		remote.log.Error().Err(err).Msg("handle rollback")
		remote.Close()
		return
	}

	err = remote.handleOffer(offer)
	if err != nil {
		remote.log.Error().Err(err).Msg("handle offer")
		remote.Close()
		return
	}
}

// handleRollback is called to decide weather or not to perform a rollback
func (remote *RemotePeer) handleRollback() error {
	if remote.peer.SignalingState() == webrtc.SignalingStateStable {
		return nil
	}

	if !remote.isPolite {
		return nil
	}

	err := remote.rollbackState()
	if err != nil {
		return errors.Wrap(err, "handleRollback")
	}

	return nil
}

// rollbackState rolls back the state of the peer connection
func (remote *RemotePeer) rollbackState() error {
	remote.log.Debug().Msg("rollback state")

	err := remote.peer.SetLocalDescription(webrtc.SessionDescription{Type: webrtc.SDPTypeRollback})
	if err != nil {
		return errors.Wrap(err, "rollbackState")
	}

	return nil
}

// handleOffer handles the offer
func (remote *RemotePeer) handleOffer(offer webrtc.SessionDescription) error {
	remote.log.Debug().Msg("handle offer")

	err := remote.peer.SetRemoteDescription(offer)
	if err != nil {
		return errors.Wrap(err, "handleOfferStable")
	}

	err = remote.generateAnswer()
	if err != nil {
		return errors.Wrap(err, "handleOfferStable")
	}

	return nil
}

// generateAnswer generates the remote description for the peer connection
func (remote *RemotePeer) generateAnswer() error {
	remote.log.Debug().Msg("generate answer")

	answer, err := remote.peer.CreateAnswer(remote.answerConfiguration)
	if err != nil {
		return errors.Wrap(err, "generateRemoteDescription")
	}

	err = remote.peer.SetLocalDescription(answer)
	if err != nil {
		return errors.Wrap(err, "generateRemoteDescription")
	}

	err = remote.signal.SendSignal("answer", remote.peer.LocalDescription())
	if err != nil {
		return errors.Wrap(err, "generateRemoteDescription")
	}

	return nil
}
