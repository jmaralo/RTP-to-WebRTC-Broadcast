package listeners

import (
	"log"
	"net"
	"sync"

	"github.com/google/uuid"
)

const (
	OBSERVER_CHAN_BUF = 100
	MTU               = 1500
)

var (
	listeners []*Listener
)

func Get() []*Listener {
	return listeners
}

func InitListeners(addrs []string) error {
	listeners = make([]*Listener, len(addrs))
	for i, addr := range addrs {
		if listener, err := newListener(addr); err != nil {
			return err
		} else {
			listeners[i] = listener
		}
	}

	return nil
}

type Listener struct {
	conn        *net.UDPConn
	observersMx *sync.Mutex
	observers   map[uuid.UUID]chan<- []byte
}

func newListener(addr string) (*Listener, error) {
	sourceAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp", sourceAddr)
	if err != nil {
		return nil, err
	}

	listener := &Listener{
		conn:        conn,
		observersMx: &sync.Mutex{},
		observers:   make(map[uuid.UUID]chan<- []byte),
	}

	go listener.run()

	return listener, nil
}

func (listener *Listener) NewObserver() (uuid.UUID, <-chan []byte) {
	// TODO: remove hardcoded size
	observer := make(chan []byte, OBSERVER_CHAN_BUF)

	id, err := uuid.NewRandom()
	if err != nil {
		log.Panicf("[ERROR] Listener: NewObserver: uuid.NewRandom: %s\n", err)
		close(observer)
	}

	listener.observersMx.Lock()
	defer listener.observersMx.Unlock()

	listener.observers[id] = observer

	return id, observer
}

func (listener *Listener) run() {
	// TODO: remove hardcoded size
	readBuf := make([]byte, MTU)
	for {
		n, err := listener.conn.Read(readBuf)
		if err != nil {
			log.Printf("[ERROR] Listener: run: conn.Read: %s\n", err)
			listener.Close()
			return
		}

		listener.observersMx.Lock()
		for _, observer := range listener.observers {
			select {
			case observer <- append([]byte{}, readBuf[:n]...):
			default:
				// log.Printf("[DEBUG] Observer blocked on write\n")
			}
		}
		listener.observersMx.Unlock()
	}
}

func (listener *Listener) Close() error {
	log.Printf("[WARNING] Close Listener")
	var err error

	if connErr := listener.conn.Close(); err != nil {
		log.Printf("[ERROR] Listener: Close: conn.Close: %s\n", connErr)
		err = connErr
	}

	listener.observersMx.Lock()
	defer listener.observersMx.Unlock()

	for _, observer := range listener.observers {
		close(observer)
	}

	return err
}
