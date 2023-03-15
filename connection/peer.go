package connection

import (
	"encoding/json"
	"log"

	"github.com/jmaralo/webrtc-broadcast/signal"
	"github.com/pion/webrtc/v3"
)

type RemotePeer struct {
	signal *signal.SignalHandle
	peer   *webrtc.PeerConnection
	track  *webrtc.TrackLocalStaticRTP
	stream <-chan []byte
	close  chan<- string
	data   PeerData
}

func newRemotePeer(sig *signal.SignalHandle, stream <-chan []byte, close chan<- string, data PeerData) (*RemotePeer, error) {
	log.Println("New Peer Joined:", data.id)
	peer, err := data.api.NewPeerConnection(data.config)

	if err != nil {
		sig.Close(300, err.Error())
		return nil, err
	}

	remotePeer := &RemotePeer{
		signal: sig,
		peer:   peer,
		track:  nil,
		stream: stream,
		close:  close,
		data:   data,
	}

	remotePeer.signal.AddEventListener("offer", remotePeer.onSignalOffer)
	remotePeer.signal.AddEventListener("close", remotePeer.onSignalClose)
	remotePeer.signal.AddEventListener("answer", remotePeer.onSignalAnswer)
	remotePeer.signal.AddEventListener("candidate", remotePeer.onSignalCandidate)

	remotePeer.peer.OnICECandidate(remotePeer.onICECandidate)
	remotePeer.peer.OnConnectionStateChange(remotePeer.onConnectionStateChange)

	remotePeer.addTrack()

	go remotePeer.signal.Listen()

	return remotePeer, nil
}

func (peer *RemotePeer) onSignalOffer(msg signal.Message) {
	offer := new(webrtc.SessionDescription)
	if err := json.Unmarshal(msg.Payload, offer); err != nil {
		log.Printf("RemotePeer: onOffer: Unmarshal: %s\n", err)
		peer.signal.Reject(210, msg, err.Error())
		return
	}

	err := peer.peer.SetRemoteDescription(*offer)
	if err != nil {
		log.Printf("RemotePeer: onOffer: SetRemoteDescription: %s\n", err)
		peer.signal.Reject(320, msg, err.Error())
		return
	}

	peer.sendAnswer()
}

func (peer *RemotePeer) onSignalAnswer(msg signal.Message) {
	peer.signal.Reject(110, msg, "server is not awaiting an answer")
}

func (peer *RemotePeer) onSignalCandidate(msg signal.Message) {
	candidate := new(webrtc.ICECandidateInit)
	if err := json.Unmarshal(msg.Payload, candidate); err != nil {
		log.Printf("RemotePeer: onCandidate: Unmarshal: %s\n", err)
		peer.signal.Reject(230, msg, err.Error())
		return
	}

	err := peer.peer.AddICECandidate(*candidate)
	if err != nil {
		log.Printf("RemotePeer: onCandidate: AddICECandidate: %s\n", err)
		peer.signal.Reject(330, msg, err.Error())
		return
	}
}

func (peer *RemotePeer) onSignalClose(msg signal.Message) {
	peer.Close()
}

func (peer *RemotePeer) onICECandidate(candidate *webrtc.ICECandidate) {
	if candidate == nil {
		return
	}

	if err := peer.signal.SendMessage("candidate", candidate.ToJSON()); err != nil {
		log.Printf("RemotePeer: onICECandidate: SendMessage: %s\n", err)
		return
	}
}

func (peer *RemotePeer) onConnectionStateChange(state webrtc.PeerConnectionState) {
	switch state {
	case webrtc.PeerConnectionStateConnected:
		log.Println("Peer finished handshake:", peer.data.id)
		peer.signal.FinishHandshake()
		go peer.run()
	default:
	}
}

func (peer *RemotePeer) run() {
	peer.data.running = true
	defer func() { peer.data.running = false }()

	for {
		data, ok := <-peer.stream
		if !ok {
			log.Println("RemotePeer: run: channel closed")
			peer.signal.Close(300, "no stream detected")
			return
		}

		if _, err := peer.track.Write(data); err != nil {
			log.Printf("RemotePeer: run: Write: %s\n", err)
			peer.signal.Close(300, err.Error())
			return
		}
	}
}

func (peer *RemotePeer) addTrack() {
	track, err := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264}, peer.data.trackID, peer.data.streamID)
	if err != nil {
		log.Printf("RemotePeer: addTrack: NewTrackLocalStaticRTP: %s\n", err)
		peer.signal.Close(300, err.Error())
		return
	}

	peer.track = track
	rtpSender, err := peer.peer.AddTrack(track)
	if err != nil {
		log.Printf("RemotePeer: addTrack: AddTrack: %s\n", err)
		peer.signal.Close(300, err.Error())
		return
	}

	rtpSenderBuf := make([]byte, peer.data.mtu)
	go func() {
		for {
			if _, _, err := rtpSender.Read(rtpSenderBuf); err != nil {
				log.Printf("RemotePeer: RTPSender: Read: %s\n", err)
				return
			}
		}
	}()
}

func (peer *RemotePeer) sendAnswer() {
	answer, err := peer.peer.CreateAnswer(nil)
	if err != nil {
		log.Printf("RemotePeer: sendAnswer: CreateAnswer: %s\n", err)
		peer.signal.Close(341, err.Error())
	}

	err = peer.peer.SetLocalDescription(webrtc.SessionDescription{Type: webrtc.SDPTypeAnswer, SDP: answer.SDP})
	if err != nil {
		log.Printf("RemotePeer: sendAnswer: SetLocalDescription: %s\n", err)
		peer.signal.Close(311, err.Error())
	}

	err = peer.signal.SendMessage("answer", answer)
	if err != nil {
		log.Printf("RemotePeer: sendAnswer: SendMessage: %s\n", err)
		peer.signal.Close(300, err.Error())
	}
}

func (peer *RemotePeer) Close() {
	peer.peer.Close()
	peer.close <- peer.data.id
}

func (peer *RemotePeer) Leave(reason string) error {
	return peer.signal.Close(100, reason)
}
