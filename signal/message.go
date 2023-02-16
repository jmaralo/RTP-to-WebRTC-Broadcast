package signal

import "encoding/json"

type Message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

func NewMessage(kind string, payload any) (*Message, error) {
	payloadRaw, err := json.Marshal(payload)
	return &Message{
		Type:    kind,
		Payload: payloadRaw,
	}, err
}
