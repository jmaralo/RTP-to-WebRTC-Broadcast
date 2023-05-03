package channel

import "time"

type Config struct {
	ReadBuffer        int
	WriteBuffer       int
	DataType          int
	PingInterval      time.Duration
	MaxPendingPings   int
	DisconnectTimeout time.Duration
}

type CloseConfig struct {
	Code int
	Text string
}
