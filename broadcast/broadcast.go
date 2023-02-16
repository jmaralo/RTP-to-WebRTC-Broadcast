package broadcast

import (
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/jmaralo/rtp-to-webrtc-broadcast/signal"
	"github.com/pion/webrtc/v3"
)

const DEFAULT_MTU = 65536

type BroadcastHandle struct {
	listener *RTPHandle
	upgrader *websocket.Upgrader
	config   webrtc.Configuration
	id       string
	streamID string
	mtu      uint
}

func NewBroadcastHandle(addr, id, streamID string, mtu uint) (*BroadcastHandle, error) {
	if mtu == 0 {
		mtu = DEFAULT_MTU
	}

	ipAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}

	rtpHandle, err := NewRTPHandle(ipAddr, mtu)
	if err != nil {
		return nil, err
	}

	return &BroadcastHandle{
		listener: rtpHandle,
		upgrader: &websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				defer r.Body.Close()
				return true
			},
		},
		config:   webrtc.Configuration{},
		id:       id,
		streamID: streamID,
		mtu:      mtu,
	}, nil
}

func (bh *BroadcastHandle) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	conn, err := bh.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Broadcast ServeHTTP: %s\n", err)
	}

	go bh.handleConn(conn)
}

func (bh *BroadcastHandle) handleConn(conn *websocket.Conn) {
	log.Println("new peer")
	peer, err := webrtc.NewPeerConnection(bh.config)
	if err != nil {
		log.Printf("Broadcast: handleConn: newPeer: %s\n", err)
		return
	}

	track, err := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264}, bh.id, bh.streamID)
	if err != nil {
		log.Printf("Broadcast: handleConn: newPeer: %s\n", err)
		return
	}

	rtpSender, err := peer.AddTrack(track)
	if err != nil {
		log.Printf("Broadcast: handleConn: newPeer: %s\n", err)
		return
	}

	go func(sender *webrtc.RTPSender) {
		buf := make([]byte, bh.mtu)
		for {
			if _, _, err := rtpSender.Read(buf); err != nil {
				log.Printf("Broadcast: readRTPSender: %s\n", err)
				return
			}
		}
	}(rtpSender)

	sig := signal.NewSignalHandle(conn)
	sig.SetEvent("offer", func(message json.RawMessage) {
		offer := new(string)
		if err := json.Unmarshal(message, offer); err != nil {
			log.Printf("Broadcast: handleOffer: %s\n", err)
			return
		}
		if err := peer.SetRemoteDescription(webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: *offer}); err != nil {
			log.Printf("Broadcast: handleOffer: %s\n", err)
			return
		}

		answer, err := peer.CreateAnswer(nil)
		if err != nil {
			log.Printf("Broadcast: handleOffer: %s\n", err)
			return
		}

		if err := peer.SetLocalDescription(answer); err != nil {
			log.Printf("Broadcast: handleOffer: %s\n", err)
			return
		}

		reply, err := signal.NewMessage("answer", answer.SDP)
		if err != nil {
			log.Printf("Broadcast: handleOffer: %s\n", err)
			return
		}

		if err := sig.Send(*reply); err != nil {
			log.Printf("Broadcast: handleOffer: %s\n", err)
			return
		}
	})

	sig.SetEvent("candidate", func(message json.RawMessage) {
		candidate := new(webrtc.ICECandidateInit)
		if err := json.Unmarshal(message, candidate); err != nil {
			log.Printf("Broadcast: handleCandidate: %s\n", err)
			return
		}

		if err := peer.AddICECandidate(*candidate); err != nil {
			log.Printf("Broadcast: handleCandidate: %s\n", err)
			return
		}
	})

	go sig.Listen()

	peer.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}

		message, err := signal.NewMessage("candidate", candidate.ToJSON())
		if err != nil {
			log.Printf("Broadcast: onICECandidate: %s\n", err)
			return
		}

		if err := sig.Send(*message); err != nil {
			log.Printf("Broadcast: onICECandidate: %s\n", err)
			return
		}
	})

	connected := make(chan struct{})
	peer.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		if state == webrtc.PeerConnectionStateConnected {
			connected <- struct{}{}
		}
	})

	<-connected

	data, pipe := io.Pipe()
	id := bh.listener.AddPeer(pipe)
	defer bh.listener.RemovePeer(id)

	log.Println("streaming...")

	buf := make([]byte, bh.mtu)
	for {
		n, err := data.Read(buf)
		if err != nil {
			log.Printf("Broadcast: readData: %s\n", err)
			return
		}

		if _, err := track.Write(buf[:n]); err != nil {
			log.Printf("Broadcast: Write: %s\n", err)
			return
		}
	}

}
