package server

import (
	"github.com/flowline-io/flowbot/pkg/logs"
	"github.com/gorilla/websocket"
	"net/http"
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

func (sess *Session) closeWS() {
	if sess.proto == WEBSOCK {
		_ = sess.ws.Close()
	}
}

func (sess *Session) sendMessage(msg any) bool {
	if len(sess.send) > sendQueueLimit {
		logs.Err.Println("ws: outbound queue limit exceeded", sess.sid)
		return false
	}

	statsInc("OutgoingMessagesWebsockTotal", 1)
	if err := wsWrite(sess.ws, websocket.TextMessage, msg); err != nil {
		if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure,
			websocket.CloseNormalClosure) {
			logs.Err.Println("ws: writeLoop", sess.sid, err)
		}
		return false
	}
	return true
}

func (sess *Session) writeLoop() {
	ticker := time.NewTicker(pingPeriod)

	defer func() {
		ticker.Stop()
		// Break readLoop.
		sess.closeWS()
	}()

	for {
		select {
		case msg, ok := <-sess.send:
			if !ok {
				// Channel closed.
				return
			}
			switch v := msg.(type) {
			case []*ServerComMessage: // batch of unserialized messages
				for _, msg := range v {
					w := sess.serializeAndUpdateStats(msg)
					if !sess.sendMessage(w) {
						return
					}
				}
			case *ServerComMessage: // single unserialized message
				w := sess.serializeAndUpdateStats(v)
				if !sess.sendMessage(w) {
					return
				}
			default: // serialized message
				if !sess.sendMessage(v) {
					return
				}
			}

		case <-sess.bkgTimer.C:
			if sess.background {
				sess.background = false
				sess.onBackgroundTimer()
			}

		case msg := <-sess.stop:
			// Shutdown requested, don't care if the message is delivered
			if msg != nil {
				_ = wsWrite(sess.ws, websocket.TextMessage, msg)
			}
			return

		case topic := <-sess.detach:
			sess.delSub(topic)

		case <-ticker.C:
			if err := wsWrite(sess.ws, websocket.PingMessage, nil); err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure,
					websocket.CloseNormalClosure) {
					logs.Err.Println("ws: writeLoop ping", sess.sid, err)
				}
				return
			}
		}
	}
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
