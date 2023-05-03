package channel

import (
	"encoding/json"
	"io"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

func (channel *Channel) read() {
	log.Debug().Msg("start reading loop")
	defer log.Debug().Msg("stop reading loop")

	defer channel.closeGroup.Done()
	for {
		_, reader, err := channel.conn.NextReader()
		if err != nil {
			log.Error().Err(err).Msg("failed to get next message")
			channel.readErrChan <- err
			return
		}

		signal, err := channel.readNext(reader)
		if err != nil {
			log.Error().Err(err).Msg("failed to read signal")
			channel.readErrChan <- err
			return
		}

		channel.readChan <- signal
		log.Trace().Str("name", signal.Name).Msg("read signal")

		select {
		case <-channel.readCloseChan:
			close(channel.readChan)
			close(channel.readErrChan)
			return
		default:
		}
	}
}

func (channel *Channel) readNext(reader io.Reader) (Signal, error) {
	raw, err := io.ReadAll(reader)
	if err != nil {
		return Signal{}, err
	}

	return decodeSignal(raw)
}

func (channel *Channel) write() {
	log.Debug().Msg("start write loop")
	defer log.Debug().Msg("stop write loop")

	defer channel.closeGroup.Done()
	for {
		select {
		case signal, ok := <-channel.writeChan:
			if !ok {
				log.Debug().Msg("write channel closed")
				channel.requestClose(websocket.CloseNormalClosure, "no more data to write")
			} else {
				err := channel.writeSignal(signal)
				if err != nil {
					log.Error().Err(err).Msg("failed to write signal")
				}
				channel.writeErrChan <- err
			}
		default:
		}

		select {
		case <-channel.writeCloseChan:
			close(channel.writeErrChan)
			return
		default:
		}
	}
}

func (channel *Channel) writeSignal(signal Signal) error {
	writer, err := channel.conn.NextWriter(channel.config.DataType)
	if err != nil {
		return err
	}

	return channel.writeNext(writer, signal)
}

func (channel *Channel) writeNext(writer io.WriteCloser, signal Signal) error {
	defer writer.Close()

	raw, err := json.Marshal(signal)
	if err != nil {
		return err
	}

	n, err := writer.Write(raw)
	if err != nil {
		return err
	}

	if n < len(raw) {
		return io.ErrShortWrite
	}

	log.Trace().Str("name", signal.Name).Msg("wrote signal")
	return nil
}
