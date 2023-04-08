package remote

import (
	"encoding/json"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jmaralo/webrtc-broadcast/signal"
	"github.com/jmaralo/webrtc-broadcast/stream"
	"github.com/pion/webrtc/v3"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var ConnectedPeers = 0

// RemotePeer abstracts the peer connection and signaling channel providing automatic negotiation
// it follows this article to negotiate during the connection: https://blog.mozilla.org/webrtc/perfect-negotiation-in-webrtc/
type RemotePeer struct {
	peer   *webrtc.PeerConnection
	signal *signal.Channel
	tracks map[string]stream.Track

	// a polite peer will perform rollback if needed
	isPolite        bool
	pollingInterval time.Duration
	stopPoll        chan struct{}

	// configuration for generating offers and answers
	// TODO: pass values to these options
	offerConfiguration  *webrtc.OfferOptions
	answerConfiguration *webrtc.AnswerOptions

	log zerolog.Logger
}

// New creates a new remote peer
func New(conn *websocket.Conn, isPolite bool, configuration webrtc.Configuration) (*RemotePeer, error) {
	log.Info().Msg("New remote peer")
	peer, err := webrtc.NewPeerConnection(configuration)
	if err != nil {
		conn.Close()
		return nil, errors.Wrap(err, "NewPeerConnection")
	}

	remote := &RemotePeer{
		peer:     peer,
		signal:   signal.New(conn),
		isPolite: isPolite,
		log:      log.With().Bool("isPolite", isPolite).Str("addr", conn.RemoteAddr().String()).Logger(),
		// TODO: make these configurable
		pollingInterval: time.Second * 5,
		stopPoll:        make(chan struct{}),
	}

	remote.registerCallbacks()
	err = remote.addTracks()
	if err != nil {
		remote.Close()
		return nil, errors.Wrap(err, "AddTracks")
	}

	go remote.runPoll()

	ConnectedPeers++
	return remote, nil
}

// registerCallbacks registers the callbacks for the peer connection
func (remote *RemotePeer) registerCallbacks() {
	remote.peer.OnICECandidate(remote.onICECandidate)
	remote.peer.OnNegotiationNeeded(remote.onNegotiationNeeded)
	remote.peer.OnConnectionStateChange(remote.onConnectionStateChange)

	remote.signal.AddListener("answer", remote.onSignalAnswer)
	remote.signal.AddListener("offer", remote.onSignalOffer)
	remote.signal.AddListener("candidate", remote.onSignalCandidate)
	remote.signal.AddListener("close", remote.onSignalClose)
}

// onConnectionStateChange handles the connection state change
func (remote *RemotePeer) onConnectionStateChange(state webrtc.PeerConnectionState) {
	remote.log.Info().Str("state", state.String()).Msg("Connection state changed")
}

// Unmarshal is a helper function to unmarshal a json payload into a struct
func Unmarshal[T any](payload []byte) (T, error) {
	var data T
	err := json.Unmarshal(payload, &data)
	return data, errors.Wrap(err, "Unmarshal")
}

// Close closes the peer connection and signaling channel
func (remote *RemotePeer) Close() error {
	remote.log.Warn().Msg("Close remote peer")
	var err error

	remote.sendClose()

	remote.removeTracks()

	peerErr := remote.peer.Close()
	if peerErr != nil {
		err = errors.Wrap(peerErr, "Close")
	}

	signalErr := remote.signal.Close()
	if signalErr != nil {
		err = errors.Wrap(signalErr, "Close")
	}

	remote.stopPolling()

	ConnectedPeers--
	remote.log.Warn().Int("connected", ConnectedPeers).Msg("Closed remote peer")
	return err
}
