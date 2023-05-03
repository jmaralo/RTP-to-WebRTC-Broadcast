package channel

import (
	"errors"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

func (channel *Channel) poll() {
	log.Debug().Msg("start poll loop")
	defer log.Debug().Msg("stop poll loop")

	defer channel.closeGroup.Done()

	ticker := time.NewTicker(channel.config.PingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			channel.writePing()
		case <-channel.pollCloseChan:
			return
		}
	}
}

func (channel *Channel) writePing() {
	err := channel.conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(channel.config.PingInterval))
	if errors.Is(err, websocket.ErrCloseSent) {
		return
	}

	if err != nil {
		log.Error().Err(err).Msg("failed to write ping")
		channel.requestClose(websocket.ClosePolicyViolation, "ping error")
		return
	}

	select {
	case channel.pollChan <- struct{}{}:
	default:
		log.Error().Msg("too many pending pings")
		channel.requestClose(websocket.ClosePolicyViolation, "too many pending pings")
		return
	}
}

func (channel *Channel) onPong(appData string) error {
	for {
		select {
		case <-channel.pollChan:
		default:
			return nil
		}
	}
}

func (channel *Channel) requestClose(code int, reason string) {
	log.Debug().Msg("request close")
	channel.closeRoutines()
	channel.closeChan <- CloseConfig{
		Code: code,
		Text: reason,
	}
}

func (channel *Channel) closeRoutines() {
	channel.readCloseChan <- struct{}{}
	channel.writeCloseChan <- struct{}{}
	channel.pollCloseChan <- struct{}{}
}

func (channel *Channel) onClose(code int, reason string) error {
	log.Debug().Int("code", code).Str("reason", reason).Msg("remote close message")
	select {
	case <-channel.activeCloseChan:
		return channel.closeConn()
	default:
	}

	err := channel.sendClose(websocket.CloseNormalClosure, "remote closed the connection")
	channel.closeRoutines()
	if err != nil {
		log.Error().Err(err).Msg("failed to send close message")
		channel.closeConn()
		return err
	}

	log.Warn().Int("code", code).Str("reason", reason).Msg("passive close")
	return channel.closeConn()
}

func (channel *Channel) waitClose() {
	channel.closeGroup.Wait()
	reason := <-channel.closeChan
	log.Debug().Msg("closing channel")

	err := channel.sendClose(reason.Code, reason.Text)
	if err != nil {
		log.Error().Err(err).Msg("failed to send close message")
		channel.closeConn()
		return
	}
	go channel.consumeControl()

	select {
	case channel.activeCloseChan <- struct{}{}:
		log.Warn().Int("code", reason.Code).Str("reason", reason.Text).Msg("active close")
	case <-time.After(channel.config.DisconnectTimeout):
		log.Error().Msg("close timeout")
		channel.closeConn()
	}
}

func (channel *Channel) sendClose(code int, reason string) error {
	message := websocket.FormatCloseMessage(code, reason)
	err := channel.conn.WriteControl(websocket.CloseMessage, message, time.Now().Add(channel.config.DisconnectTimeout))
	if err != nil {
		channel.closeConn()
		return err
	}
	return nil
}

func (channel *Channel) consumeControl() {
	for {
		_, _, err := channel.conn.ReadMessage()
		if err != nil {
			return
		}
	}
}

func (channel *Channel) closeConn() error {
	log.Warn().Msg("channel closed")
	return channel.conn.Close()
}
