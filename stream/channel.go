package stream

import (
	"sync"

	"github.com/google/uuid"
)

type SPMC[T any] struct {
	Input      chan<- T
	inputChan  <-chan T
	outputMx   *sync.Mutex
	outputChan map[uuid.UUID]chan<- T
	config     ChannelConfig
}

func NewSPMC[T any](config ChannelConfig) *SPMC[T] {
	inputChan := make(chan T, config.Size)

	channel := &SPMC[T]{
		Input:      inputChan,
		inputChan:  inputChan,
		outputMx:   &sync.Mutex{},
		outputChan: make(map[uuid.UUID]chan<- T),
		config:     config,
	}

	go channel.run()

	return channel
}

func (channel *SPMC[T]) AddOutput(bufSize int) (uuid.UUID, <-chan T, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return id, nil, err
	}

	outputChan := make(chan T, bufSize)

	channel.outputMx.Lock()
	defer channel.outputMx.Unlock()
	channel.outputChan[id] = outputChan
	return id, outputChan, nil
}

func (channel *SPMC[T]) RemoveOutput(id uuid.UUID) {
	channel.outputMx.Lock()
	defer channel.outputMx.Unlock()
	if output, ok := channel.outputChan[id]; ok {
		close(output)
		delete(channel.outputChan, id)
	}
}

func (channel *SPMC[T]) run() {
	defer channel.close()
	for input := range channel.inputChan {
		channel.broadcast(input)
	}
}

func (channel *SPMC[T]) close() {
	channel.outputMx.Lock()
	defer channel.outputMx.Unlock()
	for id, output := range channel.outputChan {
		close(output)
		delete(channel.outputChan, id)
	}
}

func (channel *SPMC[T]) broadcast(data T) {
	channel.outputMx.Lock()
	defer channel.outputMx.Unlock()
	for _, output := range channel.outputChan {
		select {
		case output <- data:
		default:
		}
	}
}
