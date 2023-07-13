package stream

import "github.com/pion/webrtc/v3"

type Config struct {
	BufferSize int
	Codec      webrtc.RTPCodecCapability
	Id         string
	StreamID   string
	Channel    ChannelConfig
}

type ChannelConfig struct {
	Size     int
	Blocking bool
}
