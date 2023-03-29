package peer

import (
	"encoding/json"
	"log"

	"github.com/pion/webrtc/v3"
)

func (remote *RemotePeer) onICECandidate(candidate *webrtc.ICECandidate) {
	if candidate == nil {
		log.Printf("[DEBUG] (%s) Finished fetching candidates\n", remote.id)
		return
	}

	if err := remote.channel.SendSignal("candidate", candidate.ToJSON()); err != nil {
		log.Printf("[ERROR] (%s) RemotePeer: onICECandidate: SendSignal: %s\n", remote.id, err)
		remote.Close()
		return
	}
}

func (remote *RemotePeer) onSignalCandidate(payload json.RawMessage) {
	var candidate webrtc.ICECandidateInit
	if err := json.Unmarshal(payload, &candidate); err != nil {
		log.Printf("[ERROR] (%s) RemotePeer: onSignalCandidate: json.Unmarshal: %s\n", remote.id, err)
		return
	}

	if err := remote.peer.AddICECandidate(candidate); err != nil {
		log.Printf("[ERROR] (%s) RemotePeer: onSignalCandidate: AddICECandidate: %s\n", remote.id, err)
		return
	}
}
