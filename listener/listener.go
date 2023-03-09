package listener

import (
	"log"
	"net"

	"github.com/google/uuid"
	"github.com/jmaralo/webrtc-broadcast/common"
)

// Small buffer for data channel, this prevents duplicate and unordered payloads
const CHANNEL_BUFFER = 100

type RTPListener struct {
	stream  *net.UDPConn
	clients *common.SyncMap[string, chan<- []byte]
}

func NewRTPListener(addr string, mtu int) (*RTPListener, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		log.Printf("NewRTPListener: ResolveUDPAddr: %s\n", err)
		return nil, err
	}

	udpListener, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Printf("NewRTPListener: ListenUDP: %s\n", err)
		return nil, err
	}

	listener := &RTPListener{
		stream:  udpListener,
		clients: common.NewSyncMap[string, chan<- []byte](),
	}

	go listener.broadcast(mtu)

	return listener, nil
}

func (listener *RTPListener) broadcast(mtu int) {
	buf := make([]byte, mtu)
	for {
		n, err := listener.stream.Read(buf)
		if err != nil {
			log.Printf("RTPListener: broadcast: %s\n", err)
			continue
		}

		listener.clients.ForEach(func(id string, stream chan<- []byte) {
			data := make([]byte, n)
			copy(data, buf)
			select {
			case stream <- data:
			default:
			}
		})
	}
}

func (listener *RTPListener) NewClient() (id string, stream <-chan []byte) {
	data := make(chan []byte, CHANNEL_BUFFER)

	defer func() {
		if err := recover(); err != nil {
			log.Printf("RTPListener: NewClient: NewUUID: %s\n", err)
		}
	}()
	id = uuid.NewString()

	listener.clients.Set(id, data)
	return id, data
}

func (listener *RTPListener) RemoveClient(id string) {
	if channel, ok := listener.clients.Get(id); ok {
		close(channel)
	}
	listener.clients.Del(id)
}
