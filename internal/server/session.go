package server

import (
	"container/list"
	"errors"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/logs"
	"github.com/flowline-io/flowbot/pkg/stats"
	"github.com/gorilla/websocket"
	json "github.com/json-iterator/go"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// Maximum number of queued messages before session is considered stale and dropped.
const sendQueueLimit = 128

// SessionProto is the type of the wire transport.
type SessionProto int

// Constants defining individual types of wire transports.
const (
	// NONE is undefined/not set.
	NONE SessionProto = iota
	// WEBSOCK represents websocket connection.
	WEBSOCK
	// LPOLL represents a long polling connection.
	LPOLL
)

// Session represents a single WS connection or a long polling session. A user may have multiple
// sessions.
type Session struct {
	// protocol - NONE (unset), WEBSOCK, LPOLL, GRPC, PROXY, MULTIPLEX
	proto SessionProto

	// Session ID
	sid string

	// Websocket. Set only for websocket sessions.
	ws *websocket.Conn

	// Pointer to session's record in sessionStore. Set only for Long Poll sessions.
	lpTracker *list.Element

	// Reference to multiplexing session. Set only for proxy sessions.
	multi        *Session
	proxiedTopic string

	// IP address of the client. For long polling this is the IP of the last poll.
	remoteAddr string

	// User agent, a string provived by an authenticated client in {login} packet.
	userAgent string

	// Protocol version of the client: ((major & 0xff) << 8) | (minor & 0xff).
	ver int

	// Device ID of the client
	deviceID string
	// Platform: web, ios, android
	platf string
	// Human language of the client
	lang string
	// Country code of the client
	countryCode string

	// ID of the current user. Could be zero if session is not authenticated
	// or for multiplexing sessions.
	uid types.Uid

	// Time when the long polling session was last refreshed
	lastTouched time.Time

	// Time when the session received any packer from client
	lastAction int64

	// Timer which triggers after some seconds to mark background session as foreground.
	bkgTimer *time.Timer

	// Number of subscribe/unsubscribe requests in flight.
	inflightReqs *boundedWaitGroup
	// Synchronizes access to session store in cluster mode:
	// subscribe/unsubscribe replies are asynchronous.
	sessionStoreLock sync.Mutex
	// Indicates that the session is terminating.
	// After this flag's been flipped to true, there must not be any more writes
	// into the session's send channel.
	// Read/written atomically.
	// 0 = false
	// 1 = true
	terminating int32

	// Background session: subscription presence notifications and online status are delayed.
	background bool

	// Outbound mesages, buffered.
	// The content must be serialized in format suitable for the session.
	send chan any

	// Channel for shutting down the session, buffer 1.
	// Content in the same format as for 'send'
	stop chan any

	// detach - channel for detaching session from topic, buffered.
	// Content is topic name to detach from.
	detach chan string

	// Map of topic subscriptions, indexed by topic name.
	// Don't access directly. Use getters/setters.
	subs map[string]*Subscription
	// Mutex for subs access: both topic go routines and network go routines access
	// subs concurrently.
	subsLock sync.RWMutex

	// Needed for long polling and grpc.
	lock sync.Mutex

	// Field used only in cluster mode by topic master node.
}

// Subscription is a mapper of sessions to topics.
type Subscription struct {
	// Channel to communicate with the topic, copy of Topic.clientMsg
	broadcast chan<- *ClientComMessage

	// Session sends a signal to Topic when this session is unsubscribed
	// This is a copy of Topic.unreg
	done chan<- *ClientComMessage

	// Channel to send {meta} requests, copy of Topic.meta
	meta chan<- *ClientComMessage
}

func (s *Session) addSub(topic string, sub *Subscription) {
	if s.multi != nil {
		s.multi.addSub(topic, sub)
		return
	}
	s.subsLock.Lock()

	// Sessions that serve as an interface between proxy topics and their masters (proxy sessions)
	// may have only one subscription, that is, to its master topic.
	// Normal sessions may be subscribed to multiple topics.

	s.subsLock.Unlock()
}

func (s *Session) getSub(topic string) *Subscription {
	// Don't check s.multi here. Let it panic if called for proxy session.

	s.subsLock.RLock()
	defer s.subsLock.RUnlock()

	return s.subs[topic]
}

func (s *Session) delSub(topic string) {
	if s.multi != nil {
		s.multi.delSub(topic)
		return
	}
	s.subsLock.Lock()
	delete(s.subs, topic)
	s.subsLock.Unlock()
}

func (s *Session) countSub() int {
	if s.multi != nil {
		return s.multi.countSub()
	}
	return len(s.subs)
}

// Inform topics that the session is being terminated.
// No need to check for s.multi because it's not called for PROXY sessions.
func (s *Session) unsubAll() {
	s.subsLock.RLock()
	defer s.subsLock.RUnlock()

	for _, sub := range s.subs {
		// sub.done is the same as topic.unreg
		// The whole session is being dropped; ClientComMessage is a wrapper for session, ClientComMessage.init is false.
		// keep redundant init: false so it can be searched for.
		sub.done <- &ClientComMessage{sess: s, init: false}
	}
}

func (s *Session) supportsMessageBatching() bool {
	switch s.proto {
	case WEBSOCK:
		return true
	default:
		return false
	}
}

// queueOutBytes attempts to send a ServerComMessage already serialized to []byte.
// If the send buffer is full, it fails.
func (s *Session) queueOutBytes(data []byte) bool {
	if s == nil || atomic.LoadInt32(&s.terminating) > 0 {
		return true
	}

	select {
	case s.send <- data:
	default:
		logs.Err.Println("s.queueOutBytes: session's send queue full", s.sid)
		return false
	}
	return true
}

func (s *Session) maybeScheduleClusterWriteLoop() {
	if s.multi != nil {
		return
	}
}

func (s *Session) detachSession(fromTopic string) {
	if atomic.LoadInt32(&s.terminating) == 0 {
		s.detach <- fromTopic
		s.maybeScheduleClusterWriteLoop()
	}
}

func (s *Session) stopSession(data any) {
	s.stop <- data
	s.maybeScheduleClusterWriteLoop()
}

func (s *Session) purgeChannels() {
	for len(s.send) > 0 {
		<-s.send
	}
	for len(s.stop) > 0 {
		<-s.stop
	}
	for len(s.detach) > 0 {
		<-s.detach
	}
}

// cleanUp is called when the session is terminated to perform resource cleanup.
func (s *Session) cleanUp(expired bool) {
	atomic.StoreInt32(&s.terminating, 1)
	s.purgeChannels()
	s.inflightReqs.Wait()
	s.inflightReqs = nil
	if !expired {
		s.sessionStoreLock.Lock()
		globals.sessionStore.Delete(s)
		s.sessionStoreLock.Unlock()
	}

	s.background = false
	s.bkgTimer.Stop()
	s.unsubAll()
	// Stop the write loop.
	s.stopSession(nil)
}

// expandTopicName expands session specific topic name to global name
// Returns
//
//	topic: session-specific topic name the message recipient should see
//	routeTo: routable global topic name
//	err: *ServerComMessage with an error to return to the sender
func (s *Session) expandTopicName(msg *ClientComMessage) (string, *types.ServerComMessage) {
	if msg.Original == "" {
		logs.Warn.Println("s.etn: empty topic name", s.sid)
		return "", ErrMalformed(msg.Id, "", msg.Timestamp)
	}

	routeTo := msg.Original

	return routeTo, nil
}

func (s *Session) serializeAndUpdateStats(msg *ServerComMessage) any {
	dataSize, data := s.serialize(msg)
	if dataSize >= 0 {
		stats.AddHistSample("OutgoingMessageSize", float64(dataSize))
	}
	return data
}

func (s *Session) serialize(msg *ServerComMessage) (int, any) {

	out, _ := json.Marshal(msg)
	return len(out), out
}

// onBackgroundTimer marks background session as foreground and informs topics it's subscribed to.
func (s *Session) onBackgroundTimer() {
	s.subsLock.RLock()
	defer s.subsLock.RUnlock()
}

func (s *Session) sendMessageLp(wrt http.ResponseWriter, msg any) bool {
	if len(s.send) > sendQueueLimit {
		logs.Err.Println("longPoll: outbound queue limit exceeded", s.sid)
		return false
	}

	stats.Inc("OutgoingMessagesLongpollTotal", 1)
	if err := lpWrite(wrt, msg); err != nil {
		logs.Err.Println("longPoll: writeOnce failed", s.sid, err)
		return false
	}

	return true
}

func (s *Session) writeOnce(wrt http.ResponseWriter, req *http.Request) {
	for {
		select {
		case msg, ok := <-s.send:
			if !ok {
				return
			}
			switch v := msg.(type) {
			case *ServerComMessage: // single unserialized message
				w := s.serializeAndUpdateStats(v)
				if !s.sendMessageLp(wrt, w) {
					return
				}
			default: // serialized message
				if !s.sendMessageLp(wrt, v) {
					return
				}
			}
			return

		case <-s.bkgTimer.C:
			if s.background {
				s.background = false
				s.onBackgroundTimer()
			}

		case msg := <-s.stop:
			// Request to close the session. Make it unavailable.
			globals.sessionStore.Delete(s)
			// Don't care if lpWrite fails.
			if msg != nil {
				_ = lpWrite(wrt, msg)
			}
			return

		case topic := <-s.detach:
			// Request to detach the session from a topic.
			s.delSub(topic)
			// No 'return' statement here: continue waiting

		case <-time.After(pingPeriod):
			// just write an empty packet on timeout
			if _, err := wrt.Write([]byte{}); err != nil {
				logs.Err.Println("longPoll: writeOnce: timout", s.sid, err)
			}
			return

		case <-req.Context().Done():
			// HTTP request canceled or connection lost.
			return
		}
	}
}

func lpWrite(wrt http.ResponseWriter, msg any) error {
	// This will panic if msg is not []byte. This is intentional.
	_, _ = wrt.Write(msg.([]byte))
	return nil
}

func (s *Session) readOnce(wrt http.ResponseWriter, req *http.Request) (int, error) {
	if req.ContentLength > globals.maxMessageSize {
		return http.StatusExpectationFailed, errors.New("request too large")
	}

	req.Body = http.MaxBytesReader(wrt, req.Body, globals.maxMessageSize)
	raw, err := io.ReadAll(req.Body)
	if err == nil {
		// Locking-unlocking is needed because the client may issue multiple requests in parallel.
		// Should not affect performance
		s.lock.Lock()
		stats.Inc("IncomingMessagesLongpollTotal", 1)
		s.dispatchRaw(raw)
		s.lock.Unlock()
		return 0, nil
	}

	return 0, err
}
