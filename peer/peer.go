package peer

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/jmaralo/webrtc-broadcast/channel"
	"github.com/jmaralo/webrtc-broadcast/listeners"
	"github.com/pion/webrtc/v3"
)

const (
	MAX_PENDING_KEEPALIVES = 3
	KEEPALIVE_INTERVAL     = time.Second * 10
	MTU                    = 1500
)

var (
	CAPABILITIES = webrtc.RTPCodecCapability{
		MimeType: webrtc.MimeTypeH264,
	}
)

type RemotePeer struct {
	channel *channel.Channel
	peer    *webrtc.PeerConnection
	id      uuid.UUID

	pendingKeepalivesMx *sync.Mutex
	pendingKeepalives   []uint32

	negotiationChannel chan struct{}
}

func New(conn *websocket.Conn) (*RemotePeer, error) {
	// TODO: remove hardcoded configuration
	peer, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		conn.Close()
		return nil, err
	}

	remoteID, err := uuid.NewRandom()
	if err != nil {
		peer.Close()
		conn.Close()
		return nil, err
	}

	remote := &RemotePeer{
		channel: channel.New(conn),
		peer:    peer,
		id:      remoteID,

		pendingKeepalivesMx: &sync.Mutex{},
		pendingKeepalives:   make([]uint32, 0, MAX_PENDING_KEEPALIVES),

		negotiationChannel: make(chan struct{}),
	}

	remote.channel.AddListener("answer", remote.onSignalAnswer)
	remote.channel.AddListener("candidate", remote.onSignalCandidate)
	remote.channel.AddListener("keepalive", remote.onSignalKeepalive)
	remote.channel.AddListener("close", remote.onSignalClose)

	remote.peer.OnICECandidate(remote.onICECandidate)

	go remote.runNegotiations()
	remote.peer.OnNegotiationNeeded(remote.onNegotiationNeeded)

	remote.peer.OnConnectionStateChange(remote.onConnectionStateChange)

	go remote.generateKeepalives()

	for i, listener := range listeners.Get() {
		if err := remote.addTrack(listener, fmt.Sprintf("listener_%d", i)); err != nil {
			remote.Close()
			return nil, err
		}
	}

	return remote, nil
}

func (remote *RemotePeer) onConnectionStateChange(state webrtc.PeerConnectionState) {
	log.Printf("[DEBUG] (%s) Connection state changed to %s\n", remote.id, state.String())
}

func (remote *RemotePeer) onSignalClose(payload json.RawMessage) {
	remote.Close()
}

func (remote *RemotePeer) Close() error {
	var err error
	log.Printf("[WARNING] (%s) Close RemotePeer\n", remote.id)

	if signalErr := remote.channel.SendSignal("close", nil); err != nil {
		log.Printf("[ERROR] (%s) RemotePeer: Close: SendSignal: %s\n", remote.id, err)
		err = signalErr
	}

	if peerErr := remote.peer.Close(); err != nil {
		log.Printf("[ERROR] (%s) RemotePeer: Close: peer.Close: %s\n", remote.id, err)
		err = peerErr
	}

	if channelErr := remote.channel.Close(); err != nil {
		log.Printf("[ERROR] (%s) RemotePeer: Close: channel.Close: %s\n", remote.id, err)
		err = channelErr
	}

	return err
}
