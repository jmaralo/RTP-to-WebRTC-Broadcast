package peer

import "github.com/pion/webrtc/v3"

type Config struct {
	OfferOptions  webrtc.OfferOptions
	AnswerOptions webrtc.AnswerOptions
	Mtu           int
}
