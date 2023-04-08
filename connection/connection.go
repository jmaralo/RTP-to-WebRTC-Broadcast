package connection

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/jmaralo/webrtc-broadcast/remote"
	"github.com/pion/webrtc/v3"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// ConnectionHandle manages the remote peer connections
type ConnectionHandle struct {
	maxPeers int
	upgrader *websocket.Upgrader
	log      zerolog.Logger
}

// New creates a new connection handle
func New(maxPeers int) *ConnectionHandle {
	return &ConnectionHandle{
		maxPeers: maxPeers,
		upgrader: &websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		log: log.With().Logger(),
	}
}

// ServeHTTP handles the HTTP requests and upgrades them to websockets
func (handle *ConnectionHandle) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()
	handle.log.Debug().Msg("New connection")

	if remote.ConnectedPeers > handle.maxPeers {
		http.Error(writer, "Too many peers", http.StatusForbidden)
		return
	}

	// set CORS headers
	writer.Header().Set("Access-Control-Allow-Origin", "*")

	conn, err := handle.upgrader.Upgrade(writer, request, nil)
	if err != nil {
		handle.log.Error().Err(err).Msg("Error upgrading connection")
		return
	}

	err = handle.addRemote(conn)
	if err != nil {
		conn.Close()
		handle.log.Error().Err(err).Msg("Error adding remote")
		return
	}

	handle.log.Warn().Int("current", remote.ConnectedPeers).Int("max", handle.maxPeers).Msg("Connected peers")
}

// addRemote creates a new remote peer and adds it to the list of remotes
func (handle *ConnectionHandle) addRemote(conn *websocket.Conn) error {
	// TODO: remove hardcoded configuration
	_, err := remote.New(conn, false, webrtc.Configuration{})
	if err != nil {
		return errors.Wrap(err, "addRemote")
	}

	handle.log.Info().Msg("New remote added")
	return nil
}
