package channel

import "encoding/json"

type Signal struct {
	Name    string          `json:"name"`
	Payload json.RawMessage `json:"payload"`
}

func decodeSignal(raw []byte) (Signal, error) {
	var signal Signal
	err := json.Unmarshal(raw, &signal)
	return signal, err
}

func NewSignal(name string, payload any) (Signal, error) {
	payloadBytes, err := json.Marshal(payload)
	return Signal{
		Name:    name,
		Payload: payloadBytes,
	}, err
}
