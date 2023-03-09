package connection

import (
	"log"

	"github.com/jmaralo/webrtc-broadcast/common"
	"github.com/jmaralo/webrtc-broadcast/listener"
	"github.com/jmaralo/webrtc-broadcast/signal"
)

type ConnectionHandle struct {
	peers     *common.SyncMap[string, *RemotePeer]
	listener  *listener.RTPListener
	peerData  *PeerData
	maxPeers  int
	closeChan chan string
}

func NewConnectionHandle(listener *listener.RTPListener, maxPeers int, data *PeerData) *ConnectionHandle {
	handle := &ConnectionHandle{
		peers:     common.NewSyncMap[string, *RemotePeer](),
		listener:  listener,
		peerData:  data,
		maxPeers:  maxPeers,
		closeChan: make(chan string),
	}

	go handle.listenClose()

	return handle
}

func (handle *ConnectionHandle) AddPeer(conn *signal.SignalHandle) {
	log.Println("clients:", handle.peers.Len()+1)
	if handle.peers.Len() >= handle.maxPeers {
		conn.SendMessage("close", "max peers exceeded")
		return
	}

	id, stream := handle.listener.NewClient()

	handle.peerData.SetID(id)
	peer, err := newRemotePeer(conn, stream, handle.closeChan, *handle.peerData)
	if err != nil {
		log.Printf("ConnectionHandle: AddPeer: %s\n", err)
		return
	}

	handle.peers.Set(id, peer)
}

func (handle *ConnectionHandle) RemovePeer(id string) {
	handle.listener.RemoveClient(id)
	handle.peers.Del(id)
}

func (handle *ConnectionHandle) listenClose() {
	for {
		remove := <-handle.closeChan
		log.Println("Remove Peer:", remove)
		handle.RemovePeer(remove)
	}
}
