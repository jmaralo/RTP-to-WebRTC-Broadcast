package signal

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jmaralo/webrtc-broadcast/common"
)

var recoverable []int = []int{201, 230, 330}

type eventHandle = func(Message)
type handleMap = common.SyncMap[string, eventHandle]

type HandleConfiguration struct {
	HandshakeTimeout     time.Duration
	KeepaliveTimeout     time.Duration
	MaxPendingKeepalives int
}

type SignalHandle struct {
	socket                 *websocket.Conn
	events                 *handleMap
	defaultEvents          *handleMap
	pendingKeepalives      *common.SyncSlice[uint32]
	config                 HandleConfiguration
	lastHandshakeTimestamp time.Time
	notifyHandshake        chan struct{}
	notifyProgress         chan struct{}
	isClosed               bool
	writeMutex             *sync.Mutex
}

func NewSignalHandle(socket *websocket.Conn, config HandleConfiguration) *SignalHandle {
	handle := &SignalHandle{
		socket:                 socket,
		events:                 common.NewSyncMap[string, eventHandle](),
		defaultEvents:          common.NewSyncMap[string, eventHandle](),
		pendingKeepalives:      common.NewSyncSlice[uint32](),
		config:                 config,
		lastHandshakeTimestamp: time.Now(),
		notifyHandshake:        make(chan struct{}),
		notifyProgress:         make(chan struct{}),
		isClosed:               false,
		writeMutex:             &sync.Mutex{},
	}

	handle.registerDefaultEvents()
	go handle.checkHandshake()

	return handle
}

func (handle *SignalHandle) AddEventListener(event string, action eventHandle) {
	handle.events.Set(event, action)
}

func (handle *SignalHandle) RemoveEventListener(event string) {
	handle.events.Del(event)
}

func (handle *SignalHandle) Send(msg *Message) error {
	handle.writeMutex.Lock()
	defer handle.writeMutex.Unlock()
	return handle.socket.WriteJSON(msg)
}

func (handle *SignalHandle) SendMessage(signal string, payload any) error {
	msg, err := NewMessage(signal, payload)
	if err != nil {
		return err
	}

	return handle.Send(msg)
}

func (handle *SignalHandle) Listen() {
	message := new(Message)
	for {
		if err := handle.socket.ReadJSON(message); err != nil {
			log.Printf("SignalHandle; Listen: ReadJSON: %s\n", err)
			handle.Close(300, err.Error())
			return
		}

		handle.onReceive(message)
	}
}

func (handle *SignalHandle) onReceive(message *Message) {
	if action, ok := handle.defaultEvents.Get(message.Signal); ok {
		action(*message)
	}

	if action, ok := handle.events.Get(message.Signal); ok {
		action(*message)
	}
}

func (handle *SignalHandle) FinishHandshake() {
	handle.notifyHandshake <- struct{}{}
}

func (handle *SignalHandle) generateKeepalives() {
	nextSeq := rand.Uint32()
	for {
		if handle.pendingKeepalives.Len() >= handle.config.MaxPendingKeepalives {
			break
		}

		<-time.After(handle.config.KeepaliveTimeout)

		if handle.isClosed {
			return
		}

		if err := handle.SendMessage("keepalive", nextSeq); err != nil {
			log.Printf("SignalHandle: generateKeepalives: SendMessage: %s\n", err)
			break
		}
		handle.pendingKeepalives.Append(nextSeq)
		nextSeq++

	}
	handle.Close(101, fmt.Sprintf("failed to answer keepalive %d consecutive times", handle.config.MaxPendingKeepalives))
}

func (handle *SignalHandle) checkHandshake() {
	for {
		select {
		case <-time.After(handle.config.HandshakeTimeout):
			handle.Close(101, "connection handshake timeout")
			return
		case <-handle.notifyHandshake:
			handle.generateKeepalives()
			return
		case <-handle.notifyProgress:
		}
	}
}

func (handle *SignalHandle) Close(code int, reason string) error {
	message, err := NewClose(code, reason)
	if err != nil {
		log.Printf("SignalHandle: Close: NewClose: %s\n", err)
	}

	if handle.isClosed {
		return err
	}

	err = handle.Send(message)
	if err != nil {
		log.Printf("SignalHandle: Close: Send: %s\n", err)
	}

	handle.onReceive(message)

	return err
}

func (handle *SignalHandle) Reject(code int, origin Message, reason string) error {
	message, err := NewReject(code, origin, reason)
	if err != nil {
		log.Printf("SignalHandle: Reject: NewReject: %s\n", err)
	}
	err = handle.Send(message)
	if err != nil {
		log.Printf("SignalHandle: Reject: Send: %s\n", err)
	}

	handle.onReceive(message)

	return err
}

func (handle *SignalHandle) registerDefaultEvents() {
	handle.defaultEvents.Set("offer", handle.defaultNotifyProgress)
	handle.defaultEvents.Set("answer", handle.defaultNotifyProgress)
	handle.defaultEvents.Set("candidate", handle.defaultNotifyProgress)
	handle.defaultEvents.Set("close", handle.defaultCloseHandle)
	handle.defaultEvents.Set("reject", handle.defaultRejectHandle)
	handle.defaultEvents.Set("keepalive", handle.defaultKeepaliveHandle)
}

func (handle *SignalHandle) defaultNotifyProgress(msg Message) {
	handle.notifyProgress <- struct{}{}
}

func (handle *SignalHandle) defaultCloseHandle(msg Message) {
	closePayload := new(ClosePayload)
	if err := json.Unmarshal(msg.Payload, closePayload); err != nil {
		log.Printf("SignalHandle: defaultCloseHandle: Unmarshal: %s\n", err)
	} else {
		log.Printf("Closing signal channel, code: %d, reason: \"%s\"\n", closePayload.Code, closePayload.Reason)
	}

	handle.isClosed = true
	handle.socket.WriteControl(websocket.CloseMessage, []byte{}, time.Now().Add(time.Second*5))
	handle.socket.Close()
}

func (handle *SignalHandle) defaultRejectHandle(msg Message) {
	rejectPayload := new(RejectPayload)
	if err := json.Unmarshal(msg.Payload, rejectPayload); err != nil {
		log.Printf("SignalHandle: defaultRejectHandle: Unmarshal: %s\n", err)
	} else {
		log.Printf("Reject signal, code: %d, reason: \"%s\", origin: %v", rejectPayload.Code, rejectPayload.Reason, rejectPayload.Origin)
	}

	for _, code := range recoverable {
		if code == rejectPayload.Code {
			return
		}
	}

	handle.Close(rejectPayload.Code, rejectPayload.Reason)
}

func (handle *SignalHandle) defaultKeepaliveHandle(msg Message) {
	seqNum := new(uint32)
	if err := json.Unmarshal(msg.Payload, seqNum); err != nil {
		log.Printf("SignalHandle: defaultKeepaliveHandle: Unmarshal: %s\n", err)
	}

	threshold := 0
	handle.pendingKeepalives.ForEach(func(_ int, seq uint32) {
		if seq <= *seqNum {
			threshold++
		}
	})

	handle.pendingKeepalives.SliceUpper(int(threshold))
}
