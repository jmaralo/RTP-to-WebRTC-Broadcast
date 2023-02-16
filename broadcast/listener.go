package broadcast

import (
	"io"
	"log"
	"net"

	"github.com/google/uuid"
	"github.com/jmaralo/rtp-to-webrtc-broadcast/common"
)

type RTPHandle struct {
	listener *net.UDPConn
	peers    *common.SyncMap[string, io.WriteCloser]
	mtu      uint
}

func NewRTPHandle(addr *net.UDPAddr, mtu uint) (*RTPHandle, error) {
	listener, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}

	if mtu < 0 {
		mtu = DEFAULT_MTU
	}

	handle := &RTPHandle{
		listener: listener,
		peers:    common.NewSyncMap[string, io.WriteCloser](),
		mtu:      mtu,
	}

	go handle.broadcast()

	return handle, nil
}

func (rtph *RTPHandle) AddPeer(peer io.WriteCloser) string {
	id := uuid.New().String()
	rtph.peers.Set(id, peer)
	return id
}

func (rtph *RTPHandle) broadcast() {
	var buf = make([]byte, rtph.mtu)
	for {
		n, err := rtph.listener.Read(buf)
		if err != nil {
			log.Printf("RTPHandle broadcast listen: %s\n", err)
			return
		}

		del := make([]string, 0, rtph.peers.Len())
		rtph.peers.ForEach(func(conn string, writer io.WriteCloser) {
			if _, err := writer.Write(buf[:n]); err != nil {
				log.Printf("RTPHandle broadcast peer %s: %s\n", conn, err)
				del = append(del, conn)
				writer.Close()
			}
		})

		for _, conn := range del {
			rtph.RemovePeer(conn)
		}
	}
}

func (rtph *RTPHandle) RemovePeer(id string) {
	rtph.peers.Del(id)
}
