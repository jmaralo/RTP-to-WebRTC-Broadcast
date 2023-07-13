package peer

import (
	"github.com/google/uuid"
	"github.com/pion/webrtc/v3"
)

type Config struct {
	OfferOptions  webrtc.OfferOptions
	AnswerOptions webrtc.AnswerOptions
	Mtu           int
	PeerConfig    webrtc.Configuration
	OnTrack       func(*webrtc.TrackRemote, *webrtc.RTPReceiver)
	OnClose       func(uuid.UUID)
}

type TrackConfig struct {
	Codec webrtc.RTPCodecCapability
	ID    string
	Label string
}
