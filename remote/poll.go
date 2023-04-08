package remote

import "time"

// runPolling starts the polling loop to check the connection is still alive
func (remote *RemotePeer) runPoll() {
	remote.log.Debug().Msg("start polling")

	ticker := time.NewTicker(remote.pollingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			err := remote.sendPoll()
			if err != nil {
				remote.log.Error().Err(err).Msg("poll")
				remote.Close()
				return
			}
		case <-remote.stopPoll:
			remote.log.Debug().Msg("stop polling")
			return
		}
	}
}

// stopPolling stops the polling loop
func (remote *RemotePeer) stopPolling() {
	remote.log.Debug().Msg("stop polling")

	select {
	case remote.stopPoll <- struct{}{}:
	default:
	}
	close(remote.stopPoll)
}

// sendPoll sends a poll signal to the signaling channel
func (remote *RemotePeer) sendPoll() error {
	remote.log.Trace().Msg("send poll")

	err := remote.signal.SendSignal("poll", nil)
	if err != nil {
		return err
	}

	return nil
}
