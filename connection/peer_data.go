package connection

import "github.com/pion/webrtc/v3"

type PeerData struct {
	id       string
	running  bool
	mtu      int
	streamID string
	trackID  string
	config   webrtc.Configuration
	api      *webrtc.API
	closed   bool
}

func NewPeerData(trackID, streamID string, mtu int, config webrtc.Configuration, api *webrtc.API) *PeerData {
	return &PeerData{
		id:       "",
		running:  false,
		mtu:      mtu,
		streamID: streamID,
		trackID:  trackID,
		config:   config,
		api:      api,
		closed:   false,
	}
}

func (data *PeerData) SetID(id string) {
	data.id = id
}
