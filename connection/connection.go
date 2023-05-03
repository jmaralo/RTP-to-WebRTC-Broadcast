package connection

// TODO: limit max connections

import (
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/jmaralo/webrtc-broadcast/channel"
	"github.com/jmaralo/webrtc-broadcast/peer"
	"github.com/jmaralo/webrtc-broadcast/stream"
	"github.com/pion/webrtc/v3"
	"github.com/rs/zerolog/log"
)

type Manager struct {
	streams      map[uuid.UUID]*stream.Stream
	upgrader     *websocket.Upgrader
	remotesMx    *sync.Mutex
	remotes      map[uuid.UUID]*peer.Remote
	signalConfig channel.Config
	peerConfig   peer.Config
	onTrack      func(*webrtc.TrackRemote, *webrtc.RTPReceiver)
	maxConns     int
}

func NewManager(peerConfig peer.Config, signalConfig channel.Config, onTrack func(*webrtc.TrackRemote, *webrtc.RTPReceiver), maxConns int) *Manager {
	return &Manager{
		streams: make(map[uuid.UUID]*stream.Stream),
		upgrader: &websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},
		remotesMx:    &sync.Mutex{},
		remotes:      make(map[uuid.UUID]*peer.Remote),
		signalConfig: signalConfig,
		peerConfig:   peerConfig,
		onTrack:      onTrack,
		maxConns:     maxConns,
	}
}

func (manager *Manager) ServeHTTP(writter http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()

	if manager.remotesLen() >= manager.maxConns {
		http.Error(writter, "max connections reached", http.StatusServiceUnavailable)
		return
	}

	conn, err := manager.upgrader.Upgrade(writter, request, nil)
	if err != nil {
		return
	}

	signal := channel.New(conn, manager.signalConfig)

	remote, err := peer.New(signal, manager.onTrack, manager.peerConfig)
	if err != nil {
		return
	}

	for id, stream := range manager.streams {
		err := remote.AddTrack(id, stream.Track())
		if err != nil {
			remote.Close()
			return
		}
	}

	id, err := uuid.NewRandom()
	if err != nil {
		return
	}

	manager.addRemote(id, remote)
}

func (manager *Manager) remotesLen() int {
	manager.remotesMx.Lock()
	defer manager.remotesMx.Unlock()

	return len(manager.remotes)
}

func (manager *Manager) addRemote(id uuid.UUID, remote *peer.Remote) {
	manager.remotesMx.Lock()
	defer manager.remotesMx.Unlock()

	manager.remotes[id] = remote
	go manager.consumeRemoteErr(id, remote)
	log.Info().Int("connections", len(manager.remotes)).Msg("new connection")
}

func (manager *Manager) consumeRemoteErr(id uuid.UUID, remote *peer.Remote) {
	for range remote.Errors {
		manager.removeRemote(id)
	}
}

func (manager *Manager) removeRemote(id uuid.UUID) {
	manager.remotesMx.Lock()
	defer manager.remotesMx.Unlock()

	delete(manager.remotes, id)
	log.Info().Int("connections", len(manager.remotes)).Msg("connection closed")
}

func (manager *Manager) AddStream(stream *stream.Stream) error {
	id, err := uuid.NewRandom()
	if err != nil {
		return err
	}

	manager.streams[id] = stream
	go manager.consumeStreamErr(id, stream)

	for _, remote := range manager.remotes {
		remoteErr := remote.AddTrack(id, stream.Track())
		if remoteErr != nil {
			err = remoteErr
		}
	}

	return err
}

func (manager *Manager) consumeStreamErr(id uuid.UUID, stream *stream.Stream) {
	for range stream.Errors {
		manager.removeStream(id)
	}
}

func (manager *Manager) removeStream(id uuid.UUID) {
	for _, remote := range manager.remotes {
		remote.RemoveTrack(id)
	}
}
