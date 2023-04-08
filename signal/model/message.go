package model

import (
	"encoding/json"

	"github.com/pkg/errors"
)

// Message is a message exchanged through the signaling channel
type Message struct {
	Signal  string          `json:"signal"`
	Payload json.RawMessage `json:"payload"`
}

// NewMessage creates a new message from a signal and a payload
// returns any error from marshaling the payload
func NewMessage(signal string, payload any) (Message, error) {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return Message{}, errors.Wrap(err, "NewMessage")
	}

	return Message{
		Signal:  signal,
		Payload: payloadJSON,
	}, nil
}
