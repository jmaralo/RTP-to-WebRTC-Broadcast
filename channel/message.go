package channel

import "encoding/json"

type message struct {
	Signal  string          `json:"signal"`
	Payload json.RawMessage `json:"payload"`
}

func newMessage(signal string, payload any) (message, error) {
	msgPayload, err := json.Marshal(payload)
	return message{
		Signal:  signal,
		Payload: msgPayload,
	}, err
}
