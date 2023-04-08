package remote

import (
	"encoding/json"

	"github.com/pion/webrtc/v3"
)

// onSignalCandidate is called when the signaling channel receives a candidate
func (remote *RemotePeer) onSignalCandidate(payload json.RawMessage) {
	remote.log.Debug().Msg("received candidate")

	candidate, err := Unmarshal[webrtc.ICECandidateInit](payload)
	if err != nil {
		remote.log.Error().Err(err).Msg("unmarshal candidate signal")
		remote.Close()
		return
	}

	err = remote.peer.AddICECandidate(candidate)
	if err != nil {
		remote.log.Error().Err(err).Msg("add ICE candidate")
		remote.Close()
		return
	}
}

// onICECandidate is called when the peer connection has a new ICE candidate
func (remote *RemotePeer) onICECandidate(candidate *webrtc.ICECandidate) {
	if candidate == nil {
		remote.log.Info().Msg("ICE candidate gathering complete")
		return
	}

	remote.log.Debug().Msg("generate candidate")

	remote.signal.SendSignal("candidate", candidate.ToJSON())
}
