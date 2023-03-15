package signal

import "encoding/json"

// Possible Evnets: "offer", "answer", "candidate", "close", "reject"
type Message struct {
	Signal  string          `json:"signal"`
	Payload json.RawMessage `json:"payload"`
}

func NewMessage(kind string, payload any) (*Message, error) {
	payloadRaw, err := json.Marshal(payload)
	return &Message{
		Signal:  kind,
		Payload: payloadRaw,
	}, err
}

type ClosePayload struct {
	Code   int    `json:"code"`
	Reason string `json:"reason,omitempty"`
}

func NewClose(code int, reason string) (*Message, error) {
	return NewMessage("close", RejectPayload{
		Code:   code,
		Reason: reason,
	})
}

type RejectPayload struct {
	Code   int     `json:"code"`
	Reason string  `json:"reason,omitempty"`
	Origin Message `json:"origin,omitempty"`
}

func NewReject(code int, origin Message, reason string) (*Message, error) {
	return NewMessage("reject", RejectPayload{
		Code:   code,
		Reason: reason,
		Origin: origin,
	})
}
