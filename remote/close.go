package remote

import "encoding/json"

// onSignalClose is called when the signaling channel receives a close
func (remote *RemotePeer) onSignalClose(payload json.RawMessage) {
	remote.log.Trace().Msg("received close")

	err := remote.Close()
	if err != nil {
		remote.log.Error().Err(err).Msg("close")
		return
	}
}

// sendClose sends a close signal to the signaling channel
func (remote *RemotePeer) sendClose() {
	remote.log.Trace().Msg("send close")

	err := remote.signal.SendSignal("close", nil)
	if err != nil {
		remote.log.Error().Err(err).Msg("send close")
		return
	}
}
