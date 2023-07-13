package peer

// TODO: close write channel when done

import (
	"encoding/json"
	"errors"
	"sync"

	"github.com/google/uuid"
	"github.com/jmaralo/webrtc-broadcast/channel"
	"github.com/pion/webrtc/v3"
)

type Remote struct {
	stopChan  chan struct{}
	closeChan chan struct{}

	writeMx *sync.Mutex

	signal *channel.Channel
	peer   *webrtc.PeerConnection
	config Config
	id     uuid.UUID
}

func New(id uuid.UUID, signal *channel.Channel, config Config, api *webrtc.API) (*Remote, error) {
	peer, err := getPeer(api, config.PeerConfig)
	if err != nil {
		return nil, err
	}

	remote := &Remote{
		stopChan:  make(chan struct{}),
		closeChan: make(chan struct{}),

		writeMx: &sync.Mutex{},

		signal: signal,
		peer:   peer,
		config: config,
		id:     id,
	}

	remote.peer.OnTrack(remote.config.OnTrack)
	remote.peer.OnICECandidate(remote.onCandidate)
	remote.peer.OnNegotiationNeeded(remote.onNegotiationNeeded)

	go remote.read()
	go remote.close()

	return remote, nil
}

func (remote *Remote) AddTrack(id uuid.UUID, data <-chan []byte, config TrackConfig, cleanup func(uuid.UUID)) error {
	track, err := webrtc.NewTrackLocalStaticRTP(config.Codec, config.ID, config.Label)
	if err != nil {
		cleanup(id)
		return err
	}

	sender, err := remote.peer.AddTrack(track)
	if err != nil {
		cleanup(id)
		return err
	}

	go remote.runSender(id, sender, cleanup)
	go remote.runTrack(id, data, track, cleanup)
	return nil
}

func (remote *Remote) runSender(id uuid.UUID, sender *webrtc.RTPSender, cleanup func(uuid.UUID)) {
	defer cleanup(id)
	readBuf := make([]byte, remote.config.Mtu)
	for {
		_, _, err := sender.Read(readBuf)
		if err != nil {
			return
		}
	}
}

func (remote *Remote) runTrack(id uuid.UUID, data <-chan []byte, track *webrtc.TrackLocalStaticRTP, cleanup func(uuid.UUID)) {
	defer cleanup(id)
	for payload := range data {
		payloadCopy := make([]byte, len(payload))
		copy(payloadCopy, payload)
		_, err := track.Write(payloadCopy)
		if err != nil {
			return
		}
	}
}

func (remote *Remote) Close() {
	remote.tryClose()
}

func getPeer(api *webrtc.API, config webrtc.Configuration) (*webrtc.PeerConnection, error) {
	if api != nil {
		return api.NewPeerConnection(config)
	} else {
		return webrtc.NewPeerConnection(config)
	}
}

func (remote *Remote) read() {
	defer remote.tryClose()
	for {
		select {
		case signal, ok := <-remote.signal.Read:
			if !ok {
				return
			}

			err := remote.handleSignal(signal)
			if err != nil {
				// TODO
				return
			}
		case <-remote.stopChan:
			return
		}
	}
}

func (remote *Remote) handleSignal(signal channel.Signal) error {
	switch signal.Name {
	case "offer":
		return remote.handleSignalOffer(signal.Payload)
	case "answer":
		return remote.onSignalAnswer(signal.Payload)
	case "candidate":
		return remote.onSignalCandidate(signal.Payload)
	}

	return errors.New("unknown signal")
}

func (remote *Remote) handleSignalOffer(payload json.RawMessage) error {
	var offer webrtc.SessionDescription
	err := json.Unmarshal(payload, &offer)
	if err != nil {
		return err
	}

	err = remote.peer.SetRemoteDescription(offer)
	if err != nil {
		return err
	}

	return remote.createAnswer(remote.config.AnswerOptions)
}

func (remote *Remote) createAnswer(options webrtc.AnswerOptions) error {
	remote.writeMx.Lock()
	defer remote.writeMx.Unlock()
	answer, err := remote.peer.CreateAnswer(&options)
	if err != nil {
		return err
	}

	err = remote.peer.SetLocalDescription(answer)
	if err != nil {
		return err
	}

	signal, err := channel.NewSignal("answer", answer)
	if err != nil {
		return err
	}

	remote.signal.Write <- signal
	return nil
}

func (remote *Remote) createOffer(options webrtc.OfferOptions) error {
	remote.writeMx.Lock()
	defer remote.writeMx.Unlock()
	offer, err := remote.peer.CreateOffer(&options)
	if err != nil {
		return err
	}

	err = remote.peer.SetLocalDescription(offer)
	if err != nil {
		return err
	}

	signal, err := channel.NewSignal("offer", offer)
	if err != nil {
		return err
	}

	remote.signal.Write <- signal
	return nil
}

func (remote *Remote) onSignalAnswer(payload json.RawMessage) error {
	var answer webrtc.SessionDescription
	err := json.Unmarshal(payload, &answer)
	if err != nil {
		return err
	}

	err = remote.peer.SetRemoteDescription(answer)
	if err != nil {
		return err
	}

	return nil
}

func (remote *Remote) onCandidate(candidate *webrtc.ICECandidate) {
	remote.writeMx.Lock()
	defer remote.writeMx.Unlock()
	if candidate == nil {
		return
	}

	signal, err := channel.NewSignal("candidate", candidate.ToJSON())
	if err != nil {
		return
	}

	remote.signal.Write <- signal
}

func (remote *Remote) onSignalCandidate(payload json.RawMessage) error {
	var candidate webrtc.ICECandidateInit
	err := json.Unmarshal(payload, &candidate)
	if err != nil {
		return err
	}

	err = remote.peer.AddICECandidate(candidate)
	if err != nil {
		return err
	}

	return nil
}

func (remote *Remote) onNegotiationNeeded() {
	err := remote.createOffer(remote.config.OfferOptions)
	if err != nil {
		// TODO
		remote.tryClose()
	}
}

func (remote *Remote) tryClose() bool {
	select {
	case remote.closeChan <- struct{}{}:
		return true
	default:
		return false
	}
}

func (remote *Remote) close() {
	<-remote.closeChan
	remote.peer.OnNegotiationNeeded(func() {})                          // Prevent new offers from being created
	remote.peer.OnICECandidate(func(candidate *webrtc.ICECandidate) {}) // Prevent new ice candidates from being created
	remote.writeMx.Lock()
	defer remote.writeMx.Unlock()
	close(remote.signal.Write)
	remote.peer.Close()
	remote.config.OnClose(remote.id)
}
