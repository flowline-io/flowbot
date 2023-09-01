package server

import (
	"fmt"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/stats"
	"github.com/gorilla/websocket"
	json "github.com/json-iterator/go"
	"net/http"
	"sync/atomic"
	"time"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = idleSessionTimeout

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10
)

func (s *Session) closeWS() {
	if s.proto == WEBSOCK {
		_ = s.ws.Close()
	}
}

func (s *Session) sendMessage(msg any) bool {
	if len(s.send) > sendQueueLimit {
		flog.Error(fmt.Errorf("ws: outbound queue limit exceeded %v", s.sid))
		return false
	}

	stats.Inc("OutgoingMessagesWebsockTotal", 1)
	if err := wsWrite(s.ws, websocket.TextMessage, msg); err != nil {
		if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure,
			websocket.CloseNormalClosure) {
			flog.Error(err)
		}
		return false
	}
	return true
}

func (s *Session) writeLoop() {
	ticker := time.NewTicker(pingPeriod)

	defer func() {
		ticker.Stop()
		// Break readLoop.
		s.closeWS()
	}()

	for {
		select {
		case msg, ok := <-s.send:
			if !ok {
				// Channel closed.
				return
			}
			switch v := msg.(type) {
			case []*ServerComMessage: // batch of unserialized messages
				for _, msg := range v {
					w := s.serializeAndUpdateStats(msg)
					if !s.sendMessage(w) {
						return
					}
				}
			case *ServerComMessage: // single unserialized message
				w := s.serializeAndUpdateStats(v)
				if !s.sendMessage(w) {
					return
				}
			default: // serialized message
				if !s.sendMessage(v) {
					return
				}
			}

		case <-s.bkgTimer.C:
			if s.background {
				s.background = false
				s.onBackgroundTimer()
			}

		case msg := <-s.stop:
			// Shutdown requested, don't care if the message is delivered
			if msg != nil {
				_ = wsWrite(s.ws, websocket.TextMessage, msg)
			}
			return

		case topic := <-s.detach:
			s.delSub(topic)

		case <-ticker.C:
			if err := wsWrite(s.ws, websocket.PingMessage, nil); err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure,
					websocket.CloseNormalClosure) {
					flog.Error(err)
				}
				return
			}
		}
	}
}

// queueOut attempts to send a ServerComMessage to a session write loop;
// it fails, if the send buffer is full.
func (s *Session) queueOut(msg *ServerComMessage) bool {
	if s == nil {
		return true
	}
	if atomic.LoadInt32(&s.terminating) > 0 {
		return true
	}

	flog.Info("s.queueOutExtra: msg send %v %v", s.sid, s.uid)

	data, err := json.Marshal(msg)
	if err != nil {
		flog.Error(err)
		return false
	}

	select {
	case s.send <- data:
	default:
		// Never block here since it may also block the topic's run() goroutine.
		flog.Error(err)
		return false
	}
	return true
}

// read loop
func (s *Session) readLoop() {
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
				flog.Error(err)
			}
			return
		}
		stats.Inc("IncomingMessagesWebsockTotal", 1)
		s.dispatchRaw(raw)
	}
}

// Message received, convert bytes to ClientComMessage and dispatch
func (s *Session) dispatchRaw(raw []byte) {
	now := types.TimeNow()
	var msg ClientComMessage

	if atomic.LoadInt32(&s.terminating) > 0 {
		flog.Warn("s.dispatchExtra: message received on a terminating session %v", s.sid)
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
	flog.Info("in: '%s%s' sid='%s' uid='%s'", toLog, truncated, s.sid, s.uid)

	if err := json.Unmarshal(raw, &msg); err != nil {
		// Malformed message
		flog.Warn("s.dispatchExtra %v %v", err, s.sid)
		s.queueOut(ErrMalformed("", "", now))
		return
	}

	s.dispatch(&msg)
}

func (s *Session) dispatch(msg *ClientComMessage) {
	//result, err := linkitAction(s.uid, msg.Data)
	//if err != nil {
	//	flog.Error(err)
	//	return
	//}
	//if result != nil {
	//	s.queueOut(result)
	//}
}

// Writes a message with the given message type (mt) and payload.
func wsWrite(ws *websocket.Conn, mt int, msg any) error {
	var bits []byte
	if msg != nil {
		bits = msg.([]byte)
	} else {
		bits = []byte{}
	}
	_ = ws.SetWriteDeadline(time.Now().Add(writeWait))
	return ws.WriteMessage(mt, bits)
}

// Handles websocket requests from peers.
var upgrader = websocket.Upgrader{
	ReadBufferSize:    1024,
	WriteBufferSize:   1024,
	EnableCompression: globals.wsCompression,
	// Allow connections from any Origin
	CheckOrigin: func(r *http.Request) bool { return true },
}
