package peer

import (
	"encoding/json"
	"log"

	"github.com/pion/webrtc/v3"
)

func (remote *RemotePeer) onNegotiationNeeded() {
	log.Printf("[DEBUG] (%s) Negotiation Needed", remote.id)

	select {
	case remote.negotiationChannel <- struct{}{}:
		log.Printf("[DEBUG] (%s) Sent negotiation signal", remote.id)
	default:
	}
}

func (remote *RemotePeer) runNegotiations() {
	for range remote.negotiationChannel {
		if err := remote.handleNegotiation(); err != nil {
			log.Printf("[ERROR] (%s) RemotePeer: runNegotiations: handleNegotiation: %s\n", remote.id, err)
		}
	}
	log.Printf("[DEBUG] (%s) negotiation channel closed\n", remote.id)
}

func (remote *RemotePeer) handleNegotiation() error {
	// TODO: remove hardcoded configuration
	offer, err := remote.peer.CreateOffer(nil)
	if err != nil {
		return err
	}

	if err := remote.peer.SetLocalDescription(offer); err != nil {
		return err
	}

	if err := remote.channel.SendSignal("offer", offer); err != nil {
		return err
	}

	return nil
}

func (remote *RemotePeer) onSignalAnswer(payload json.RawMessage) {
	var answer webrtc.SessionDescription
	if err := json.Unmarshal(payload, &answer); err != nil {
		log.Printf("[ERROR] (%s) RemotePeer: onSignalAnswer: json.Unmarshal: %s\n", remote.id, err)
		remote.Close()
		return
	}

	if err := remote.peer.SetRemoteDescription(answer); err != nil {
		log.Printf("[ERROR] (%s) RemotePeer: onSignalAnswer: peer.SetRemoteDescription: %s\n", remote.id, err)
		remote.Close()
		return
	}
}
