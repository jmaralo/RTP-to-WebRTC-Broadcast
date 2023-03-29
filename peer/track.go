package peer

import (
	"log"

	"github.com/google/uuid"
	"github.com/jmaralo/webrtc-broadcast/listeners"
	"github.com/pion/webrtc/v3"
)

func (remote *RemotePeer) addTrack(listener *listeners.Listener, streamID string) error {
	// TODO: remove hardcoded capabilities
	trackLocal, err := webrtc.NewTrackLocalStaticRTP(CAPABILITIES, "video", streamID)
	if err != nil {
		return err
	}

	control, err := remote.peer.AddTrack(trackLocal)
	if err != nil {
		return err
	}

	remote.handleTrack(trackLocal, control, listener)

	return nil
}

func (remote *RemotePeer) handleTrack(trackLocal *webrtc.TrackLocalStaticRTP, control *webrtc.RTPSender, listener *listeners.Listener) {
	observerID, stream := listener.NewObserver()
	go func(control *webrtc.RTPSender, observerID uuid.UUID) {
		// TODO: remove hardcoded size
		readBuf := make([]byte, MTU)
		for {
			if _, _, err := control.Read(readBuf); err != nil {
				log.Printf("[ERROR] (%s) (%s) RemotePeer: handleTrack: control.Read: %s\n", remote.id, observerID, err)
				return
			}
		}
	}(control, observerID)

	go func(trackLocal *webrtc.TrackLocalStaticRTP, stream <-chan []byte, observerID uuid.UUID) {
		for payload := range stream {
			_, err := trackLocal.Write(payload)
			if err != nil {
				log.Printf("[ERROR] (%s) (%s) RemotePeer: handleTrack: trackLocal.Write: %s\n", remote.id, observerID, err)
				return
			}
		}
	}(trackLocal, stream, observerID)
}
