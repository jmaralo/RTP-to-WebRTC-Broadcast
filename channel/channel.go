package channel

import (
	"sync"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

type Channel struct {
	Read     <-chan Signal
	readChan chan<- Signal

	ReadErrors  <-chan error
	readErrChan chan<- error

	Write     chan<- Signal
	writeChan <-chan Signal

	WriteErrors  <-chan error
	writeErrChan chan<- error

	pollChan chan struct{}

	readCloseChan   chan struct{}
	writeCloseChan  chan struct{}
	pollCloseChan   chan struct{}
	closeChan       chan CloseConfig
	activeCloseChan chan struct{}
	closeGroup      *sync.WaitGroup

	conn   *websocket.Conn
	config Config
}

func New(conn *websocket.Conn, config Config) *Channel {
	log.Info().Msg("new channel")
	readChan := make(chan Signal, config.ReadBuffer)
	readErrChan := make(chan error, 1)

	writeChan := make(chan Signal, config.WriteBuffer)
	writeErrChan := make(chan error)

	channel := &Channel{
		Read:     readChan,
		readChan: readChan,

		ReadErrors:  readErrChan,
		readErrChan: readErrChan,

		Write:     writeChan,
		writeChan: writeChan,

		WriteErrors:  writeErrChan,
		writeErrChan: writeErrChan,

		pollChan: make(chan struct{}, config.MaxPendingPings),

		readCloseChan:   make(chan struct{}, 1),
		writeCloseChan:  make(chan struct{}, 1),
		pollCloseChan:   make(chan struct{}, 1),
		closeChan:       make(chan CloseConfig, 1),
		activeCloseChan: make(chan struct{}, 1),
		closeGroup:      &sync.WaitGroup{},

		conn:   conn,
		config: config,
	}

	channel.conn.SetPongHandler(channel.onPong)
	channel.conn.SetCloseHandler(channel.onClose)

	channel.closeGroup.Add(3)
	go channel.read()
	go channel.write()
	go channel.poll()
	go channel.waitClose()

	return channel
}
