package peer

import (
	"encoding/json"
	"log"
	"math/rand"
	"time"
)

func (remote *RemotePeer) onSignalKeepalive(payload json.RawMessage) {
	var seqNum uint32
	if err := json.Unmarshal(payload, &seqNum); err != nil {
		log.Printf("[ERROR] (%s) RemotePeer: onSignalKeepalive: json.Unmarshal: %s\n", remote.id, err)
		return
	}

	remote.pendingKeepalivesMx.Lock()
	defer remote.pendingKeepalivesMx.Unlock()

	obsolete := -1
	for i, seq := range remote.pendingKeepalives {
		if seq <= seqNum {
			obsolete = i
		}
	}
	log.Printf("[TRACE] (%s) Acnowledged %d keepalives with sequence number %d", remote.id, obsolete+1, seqNum)
	remote.pendingKeepalives = remote.pendingKeepalives[obsolete+1:]
}

func (remote *RemotePeer) generateKeepalives() {
	seqNum := rand.Uint32()
	// TODO: remove hardcoded interval
	ticker := time.NewTicker(KEEPALIVE_INTERVAL)
	defer ticker.Stop()
	for range ticker.C {
		remote.pendingKeepalivesMx.Lock()
		// TODO: remove hardcoded threshold
		if len(remote.pendingKeepalives) > MAX_PENDING_KEEPALIVES {
			log.Printf("[WARNING] (%s) Missed keepalive %d consecutive times", remote.id, MAX_PENDING_KEEPALIVES)
			remote.pendingKeepalivesMx.Unlock()
			remote.Close()
			return
		}

		if err := remote.channel.SendSignal("keepalive", seqNum); err != nil {
			log.Printf("[ERROR] (%s) RemotePeer: generateKeepalives: SendSignal: %s\n", remote.id, err)
			remote.pendingKeepalivesMx.Unlock()
			remote.Close()
			return
		}

		remote.pendingKeepalives = append(remote.pendingKeepalives, seqNum)
		remote.pendingKeepalivesMx.Unlock()
		log.Printf("[DEBUG] (%s) Pending keepalive with sequence number %d\n", remote.id, seqNum)

		seqNum++
	}
}
