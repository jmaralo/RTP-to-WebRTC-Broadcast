package stream

import "github.com/pion/webrtc/v3"

type Config struct {
	Mtu      int
	Codec    webrtc.RTPCodecCapability
	Id       string
	StreamID string
}
