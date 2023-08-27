package server

import (
	"container/list"
	"encoding/json"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/logs"
	"github.com/gorilla/websocket"
	"sync/atomic"
	"time"
)

// NewExtraSessionStore initializes an extra session store.
func NewExtraSessionStore(lifetime time.Duration) *SessionStore {
	ss := &SessionStore{
		lru:      list.New(),
		lifeTime: lifetime,

		sessCache: make(map[string]*Session),
	}
	return ss
}

// queueOut attempts to send a ServerComMessage to a session write loop;
// it fails, if the send buffer is full.
func (s *Session) queueOutExtra(msg *types.ServerComMessage) bool {
	if s == nil {
		return true
	}
	if atomic.LoadInt32(&s.terminating) > 0 {
		return true
	}

	logs.Info.Println("s.queueOutExtra: msg send", s.sid, s.uid)

	data, err := json.Marshal(msg)
	if err != nil {
		logs.Err.Println("s.queueOutExtra: msg marshal failed", s.sid)
		return false
	}

	select {
	case s.send <- data:
	default:
		// Never block here since it may also block the topic's run() goroutine.
		logs.Err.Println("s.queueOutExtra: session's send queue full", s.sid)
		return false
	}
	return true
}

// read loop
func (s *Session) readLoopExtra() {
	defer func() {
		s.closeWS()
		s.cleanUp(false)
	}()

	s.ws.SetReadLimit(globals.maxMessageSize)
	_ = s.ws.SetReadDeadline(time.Now().Add(pongWait))
	s.ws.SetPongHandler(func(string) error {
		_ = s.ws.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		// Read a ClientComMessage
		_, raw, err := s.ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure,
				websocket.CloseNormalClosure) {
				logs.Err.Println("ws: readLoopExtra", s.sid, err)
			}
			return
		}
		statsInc("IncomingMessagesWebsockTotal", 1)
		s.dispatchRawExtra(raw)
	}
}

// Message received, convert bytes to ClientComMessage and dispatch
func (s *Session) dispatchRawExtra(raw []byte) {
	now := types.TimeNow()
	var msg types.ClientComMessage

	if atomic.LoadInt32(&s.terminating) > 0 {
		logs.Warn.Println("s.dispatchExtra: message received on a terminating session", s.sid)
		s.queueOut(ErrLocked("", "", now))
		return
	}

	if len(raw) == 1 && raw[0] == 0x31 {
		// 0x31 == '1'. This is a network probe message. Respond with a '0':
		s.queueOutBytes([]byte{0x30})
		return
	}

	toLog := raw
	truncated := ""
	if len(raw) > 512 {
		toLog = raw[:512]
		truncated = "<...>"
	}
	logs.Info.Printf("in: '%s%s' sid='%s' uid='%s'", toLog, truncated, s.sid, s.uid)

	if err := json.Unmarshal(raw, &msg); err != nil {
		// Malformed message
		logs.Warn.Println("s.dispatchExtra", err, s.sid)
		s.queueOut(ErrMalformed("", "", now))
		return
	}

	s.dispatchExtra(&msg)
}

func (s *Session) dispatchExtra(msg *types.ClientComMessage) {
	result, err := linkitAction(s.uid, msg.Data)
	if err != nil {
		logs.Err.Println(err)
		return
	}
	if result != nil {
		s.queueOutExtra(types.OkMessage(result))
	}
}
