package peer

// TODO: close write channel when done

import (
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jmaralo/webrtc-broadcast/channel"
	"github.com/pion/webrtc/v3"
	"github.com/rs/zerolog/log"
)

type Remote struct {
	Errors    <-chan error
	errorChan chan<- error

	writeChan chan channel.Signal
	closeChan chan struct{}

	tracks map[uuid.UUID]*webrtc.RTPSender

	signal *channel.Channel
	peer   *webrtc.PeerConnection
	config Config
}

func New(signal *channel.Channel, onTrack func(*webrtc.TrackRemote, *webrtc.RTPReceiver), config Config) (*Remote, error) {
	peer, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		return nil, err
	}

	errorChan := make(chan error, 1)

	remote := &Remote{
		Errors:    errorChan,
		errorChan: errorChan,

		// TODO: hardcoded buffer size
		writeChan: make(chan channel.Signal, 100),
		closeChan: make(chan struct{}),

		tracks: make(map[uuid.UUID]*webrtc.RTPSender),

		signal: signal,
		peer:   peer,
		config: config,
	}

	peer.OnNegotiationNeeded(remote.onNegotiationNeeded)
	peer.OnICECandidate(remote.onCandidate)
	peer.OnTrack(onTrack)

	go remote.readSignals()
	go remote.writeSignals()
	go remote.reportReadErrors()

	return remote, nil
}

func (remote *Remote) AddTrack(id uuid.UUID, track webrtc.TrackLocal) error {
	log.Info().Str("id", id.String()).Msg("add track")
	sender, err := remote.peer.AddTrack(track)
	if err != nil {
		return err
	}

	if _, ok := remote.tracks[id]; ok {
		return errors.New("track already added")
	}

	remote.tracks[id] = sender

	go remote.consumeSender(sender, remote.config.Mtu)

	return nil
}

func (remote *Remote) consumeSender(sender *webrtc.RTPSender, mtu int) {
	readBuf := make([]byte, mtu)
	for {
		_, _, err := sender.Read(readBuf)
		if err != nil {
			return
		}
	}
}

func (remote *Remote) RemoveTrack(id uuid.UUID) {
	if sender, ok := remote.tracks[id]; ok {
		remote.peer.RemoveTrack(sender)
		delete(remote.tracks, id)
	}
}

func (remote *Remote) readSignals() {
	defer remote.Close()
	for signal := range remote.signal.Read {
		var err error
		switch signal.Name {
		case "offer":
			err = remote.onSignalOffer(signal)
		case "answer":
			err = remote.onSignalAnswer(signal)
		case "candidate":
			err = remote.onSignalCandidate(signal)
		}

		if err != nil {
			return
		}
	}
}

func (remote *Remote) reportReadErrors() {
	for err := range remote.signal.ReadErrors {
		remote.errorChan <- err
	}
}

func (remote *Remote) onNegotiationNeeded() {
	err := remote.createOffer(&remote.config.OfferOptions)
	if err != nil {
		remote.errorChan <- err
	}
}

func (remote *Remote) createOffer(options *webrtc.OfferOptions) error {
	offer, err := remote.peer.CreateOffer(options)
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

	remote.writeChan <- signal
	return nil
}

func (remote *Remote) onSignalAnswer(signal channel.Signal) error {
	var answer webrtc.SessionDescription
	err := json.Unmarshal(signal.Payload, &answer)
	if err != nil {
		return err
	}

	return remote.peer.SetRemoteDescription(answer)
}

func (remote *Remote) onSignalOffer(signal channel.Signal) error {
	var offer webrtc.SessionDescription
	err := json.Unmarshal(signal.Payload, &offer)
	if err != nil {
		return err
	}

	err = remote.peer.SetRemoteDescription(offer)
	if err != nil {
		return err
	}

	return remote.createAnswer(&remote.config.AnswerOptions)
}

func (remote *Remote) createAnswer(options *webrtc.AnswerOptions) error {
	answer, err := remote.peer.CreateAnswer(options)
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

	remote.writeChan <- signal
	return nil
}

func (remote *Remote) onSignalCandidate(signal channel.Signal) error {
	var candidate webrtc.ICECandidateInit
	err := json.Unmarshal(signal.Payload, &candidate)
	if err != nil {
		return err
	}

	return remote.peer.AddICECandidate(candidate)
}

func (remote *Remote) onCandidate(candidate *webrtc.ICECandidate) {
	if candidate == nil {
		return
	}

	signal, err := channel.NewSignal("candidate", candidate.ToJSON())
	if err != nil {
		remote.errorChan <- err
		return
	}

	remote.writeChan <- signal
}

func (remote *Remote) writeSignals() {
	defer close(remote.signal.Write)
	for {
		select {
		case <-remote.closeChan:
			return
		case signal := <-remote.writeChan:
			remote.signal.Write <- signal
			err := <-remote.signal.WriteErrors
			if err != nil {
				remote.errorChan <- err
			}
		}
	}
}

func (remote *Remote) Close() error {
	remote.closeChan <- struct{}{}
	return remote.peer.Close()
}
