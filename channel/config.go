package channel

import "time"

type Config struct {
	ReadBuffer        int
	WriteBuffer       int
	PingInterval      time.Duration
	MaxPendingPings   int
	DisconnectTimeout time.Duration
}

type closeConfig struct {
	Code int
	Text string
}
