package signal

import "encoding/json"

// Possible Evnets: "offer", "answer", "candidate", "close", "reject"
// Close msg: "max peers exceeded", "error creating peer connection"
type Message struct {
	Event   string          `json:"event"`
	Payload json.RawMessage `json:"payload"`
}

func NewMessage(kind string, payload any) (*Message, error) {
	payloadRaw, err := json.Marshal(payload)
	return &Message{
		Event:   kind,
		Payload: payloadRaw,
	}, err
}
