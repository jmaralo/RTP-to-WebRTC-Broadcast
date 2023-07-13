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
	"github.com/pion/interceptor"
	"github.com/pion/webrtc/v3"
	"github.com/rs/zerolog/log"
)

type Manager struct {
	streams      []*stream.Stream
	upgrader     *websocket.Upgrader
	signalConfig channel.Config
	peerConfig   peer.Config
	config       Config
	remotesMx    *sync.Mutex
	remotes      map[uuid.UUID]*peer.Remote
	api          *webrtc.API
}

func NewManager(streams []*stream.Stream, peerConfig peer.Config, signalConfig channel.Config, config Config) (*Manager, error) {
	media := &webrtc.MediaEngine{}
	if err := media.RegisterDefaultCodecs(); err != nil {
		return nil, err
	}

	interceptors := &interceptor.Registry{}
	if err := webrtc.ConfigureRTCPReports(interceptors); err != nil {
		return nil, err
	}

	if err := webrtc.ConfigureTWCCSender(media, interceptors); err != nil {
		return nil, err
	}

	api := webrtc.NewAPI(webrtc.WithMediaEngine(media), webrtc.WithInterceptorRegistry(interceptors))

	manager := &Manager{
		streams: streams,
		upgrader: &websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},
		signalConfig: signalConfig,
		peerConfig:   peerConfig,
		config:       config,
		remotesMx:    &sync.Mutex{},
		remotes:      make(map[uuid.UUID]*peer.Remote),
		api:          api,
	}

	manager.peerConfig.OnClose = manager.removeRemote

	return manager, nil
}

func (manager *Manager) ServeHTTP(writter http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()

	if manager.remotesLen() >= manager.config.MaxPeers {
		http.Error(writter, "max connections reached", http.StatusServiceUnavailable)
		return
	}

	conn, err := manager.upgrader.Upgrade(writter, request, nil)
	if err != nil {
		return
	}

	id, err := uuid.NewRandom()
	if err != nil {
		return
	}

	signal := channel.New(conn, manager.signalConfig)

	remote, err := peer.New(id, signal, manager.peerConfig, manager.api)
	if err != nil {
		return
	}

	for _, stream := range manager.streams {
		id, data, err := stream.Subscribe(100)
		if err != nil {
			remote.Close()
			return
		}

		err = remote.AddTrack(id, data, stream.TrackConfig(), stream.Unsubscribe)
		if err != nil {
			remote.Close()
			return
		}
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
	log.Info().Int("peers", len(manager.remotes)).Msg("new peer")
}

func (manager *Manager) removeRemote(id uuid.UUID) {
	manager.remotesMx.Lock()
	defer manager.remotesMx.Unlock()
	delete(manager.remotes, id)
	log.Info().Int("peers", len(manager.remotes)).Msg("remove peer")
}
