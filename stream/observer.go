package stream

import (
	"github.com/google/uuid"
)

// addObserver adds a new observer to the stream
// returns the observer uuid and the stream channel
func (stream *Stream) addObserver() (uuid.UUID, <-chan []byte) {
	stream.observersMx.Lock()
	defer stream.observersMx.Unlock()

	id := uuid.New()
	observerChannel := make(chan []byte)
	stream.observers[id] = observerChannel

	stream.log.Debug().Str("id", id.String()).Msg("observer added")
	return id, observerChannel
}

// closeObserver removes an observer from the stream
func (stream *Stream) closeObserver(id uuid.UUID) {
	stream.observersMx.Lock()
	defer stream.observersMx.Unlock()

	if channel, ok := stream.observers[id]; ok {
		close(channel)
	}
	delete(stream.observers, id)

	stream.log.Debug().Str("id", id.String()).Msg("observer removed")
}

// closeAllObservers closes all the observers
func (stream *Stream) closeAllObservers() {
	stream.observersMx.Lock()
	defer stream.observersMx.Unlock()

	for id, channel := range stream.observers {
		close(channel)
		delete(stream.observers, id)
		stream.log.Debug().Str("id", id.String()).Msg("observer removed")
	}

	stream.log.Warn().Msg("all observers removed")
}
