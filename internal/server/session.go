/******************************************************************************
 *
 *  Description :
 *
 *  Handling of user sessions/connections. One user may have multiple sesions.
 *  Each session may handle multiple topics
 *
 *****************************************************************************/

package server

import (
	"container/list"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sysatom/flowbot/internal/types"
	"github.com/sysatom/flowbot/pkg/utils"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sysatom/flowbot/pkg/logs"
)

// Maximum number of queued messages before session is considered stale and dropped.
const sendQueueLimit = 128

// Time given to a background session to terminate to avoid tiggering presence notifications.
// If session terminates (or unsubscribes from topic) in this time frame notifications are not sent at all.
const deferredNotificationsTimeout = time.Second * 5

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

// queueOut attempts to send a list of ServerComMessages to a session write loop;
// it fails if the send buffer is full.
func (s *Session) queueOutBatch(msgs []*ServerComMessage) bool {
	if s == nil {
		return true
	}
	if atomic.LoadInt32(&s.terminating) > 0 {
		return true
	}

	if s.multi != nil {
		// In case of a cluster we need to pass a copy of the actual session.
		for i := range msgs {
			msgs[i].sess = s
		}
		if s.multi.queueOutBatch(msgs) {
			return true
		}
		return false
	}

	if s.supportsMessageBatching() {
		select {
		case s.send <- msgs:
		default:
			// Never block here since it may also block the topic's run() goroutine.
			logs.Err.Println("s.queueOut: session's send queue2 full", s.sid)
			return false
		}
	} else {
		for _, msg := range msgs {
			s.queueOut(msg)
		}
	}

	return true
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

	if s.multi != nil {
		// In case of a cluster we need to pass a copy of the actual session.
		msg.sess = s
		if s.multi.queueOut(msg) {
			return true
		}
		return false
	}

	// Record latency only on {ctrl} messages and end-user sessions.
	if msg.Ctrl != nil && msg.Id != "" {
		if !msg.Ctrl.Timestamp.IsZero() {
			duration := time.Since(msg.Ctrl.Timestamp).Milliseconds()
			statsAddHistSample("RequestLatency", float64(duration))
		}
		if 200 <= msg.Ctrl.Code && msg.Ctrl.Code < 600 {
			statsInc(fmt.Sprintf("CtrlCodesTotal%dxx", msg.Ctrl.Code/100), 1)
		} else {
			logs.Warn.Println("Invalid response code: ", msg.Ctrl.Code)
		}
	}

	select {
	case s.send <- msg:
	default:
		// Never block here since it may also block the topic's run() goroutine.
		logs.Err.Println("s.queueOut: session's send queue full", s.sid)
		return false
	}
	return true
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

// Message received, convert bytes to ClientComMessage and dispatch
func (s *Session) dispatchRaw(raw []byte) {
	now := types.TimeNow()
	var msg ClientComMessage

	if atomic.LoadInt32(&s.terminating) > 0 {
		logs.Warn.Println("s.dispatch: message received on a terminating session", s.sid)
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
		logs.Warn.Println("s.dispatch", err, s.sid)
		s.queueOut(ErrMalformed("", "", now))
		return
	}

	s.dispatch(&msg)
}

func (s *Session) dispatch(msg *ClientComMessage) {
	now := types.TimeNow()
	atomic.StoreInt64(&s.lastAction, now.UnixNano())

	if msg == nil {
		// Plugin requested to silently drop the request.
		return
	}

	if msg.Extra == nil || msg.Extra.AsUser == "" {
		// Use current user's ID and auth level.
		msg.AsUser = s.uid.UserId()
	} else if fromUid := types.ParseUserId(msg.Extra.AsUser); fromUid.IsZero() {
		// Invalid msg.Extra.AsUser.
		s.queueOut(ErrMalformed("", "", now))
		logs.Warn.Println("s.dispatch: malformed msg.from: ", msg.Extra.AsUser, s.sid)
		return
	} else {
		// Use provided msg.Extra.AsUser
		msg.AsUser = msg.Extra.AsUser
	}

	msg.Timestamp = now

	var handler func(*ClientComMessage)
	var uaRefresh bool

	// Check if s.ver is defined
	checkVers := func(m *ClientComMessage, handler func(*ClientComMessage)) func(*ClientComMessage) {
		return func(m *ClientComMessage) {
			if s.ver == 0 {
				logs.Warn.Println("s.dispatch: {hi} is missing", s.sid)
				s.queueOut(ErrCommandOutOfSequence(m.Id, m.Original, msg.Timestamp))
				return
			}
			handler(m)
		}
	}

	// Check if user is logged in
	checkUser := func(m *ClientComMessage, handler func(*ClientComMessage)) func(*ClientComMessage) {
		return func(m *ClientComMessage) {
			if msg.AsUser == "" {
				logs.Warn.Println("s.dispatch: authentication required", s.sid)
				s.queueOut(ErrAuthRequiredReply(m, m.Timestamp))
				return
			}
			handler(m)
		}
	}

	switch {
	case msg.Pub != nil:
		handler = checkVers(msg, checkUser(msg, s.publish))
		msg.Id = msg.Pub.Id
		msg.Original = msg.Pub.Topic
		uaRefresh = true

	case msg.Sub != nil:
		handler = checkVers(msg, checkUser(msg, s.subscribe))
		msg.Id = msg.Sub.Id
		msg.Original = msg.Sub.Topic
		uaRefresh = true

	case msg.Leave != nil:
		handler = checkVers(msg, checkUser(msg, s.leave))
		msg.Id = msg.Leave.Id
		msg.Original = msg.Leave.Topic

	case msg.Login != nil:
		handler = checkVers(msg, s.login)
		msg.Id = msg.Login.Id

	case msg.Get != nil:
		handler = checkVers(msg, checkUser(msg, s.get))
		msg.Id = msg.Get.Id
		msg.Original = msg.Get.Topic
		uaRefresh = true

	case msg.Set != nil:
		handler = checkVers(msg, checkUser(msg, s.set))
		msg.Id = msg.Set.Id
		msg.Original = msg.Set.Topic
		uaRefresh = true

	case msg.Del != nil:
		handler = checkVers(msg, checkUser(msg, s.del))
		msg.Id = msg.Del.Id
		msg.Original = msg.Del.Topic

	case msg.Note != nil:
		// If user is not authenticated or version not set the {note} is silently ignored.
		handler = s.note
		msg.Original = msg.Note.Topic
		uaRefresh = true

	default:
		// Unknown message
		s.queueOut(ErrMalformed("", "", msg.Timestamp))
		logs.Warn.Println("s.dispatch: unknown message", s.sid)
		return
	}

	msg.sess = s
	msg.init = true
	handler(msg)

	// Notify 'me' topic that this session is currently active.
	if uaRefresh && msg.AsUser != "" && s.userAgent != "" {
		if sub := s.getSub(msg.AsUser); sub != nil {
			// The chan is buffered. If the buffer is exhaused, the session will wait for 'me' to become available
			// sub.supd <- &sessionUpdate{userAgent: s.userAgent}
		}
	}
}

// Request to subscribe to a topic.
func (s *Session) subscribe(msg *ClientComMessage) {
	if strings.HasPrefix(msg.Original, "new") || strings.HasPrefix(msg.Original, "nch") {
		// Request to create a new group/channel topic.
	} else {
		var resp *ServerComMessage
		msg.RcptTo, resp = s.expandTopicName(msg)
		if resp != nil {
			s.queueOut(resp)
			return
		}
	}

	s.inflightReqs.Add(1)
	// Session can subscribe to topic on behalf of a single user at a time.
	if sub := s.getSub(msg.RcptTo); sub != nil {
		s.queueOut(InfoAlreadySubscribed(msg.Id, msg.Original, msg.Timestamp))
		s.inflightReqs.Done()
	} else {
		select {
		default:
			// Reply with a 503 to the user.
			s.queueOut(ErrServiceUnavailableReply(msg, msg.Timestamp))
			s.inflightReqs.Done()
			logs.Err.Println("s.subscribe: hub.join queue full, topic ", msg.RcptTo, s.sid)
		}
		// Hub will send Ctrl success/failure packets back to session
	}
}

// Leave/Unsubscribe a topic
func (s *Session) leave(msg *ClientComMessage) {
	// Expand topic name
	var resp *ServerComMessage
	msg.RcptTo, resp = s.expandTopicName(msg)
	if resp != nil {
		s.queueOut(resp)
		return
	}

	s.inflightReqs.Add(1)
	if sub := s.getSub(msg.RcptTo); sub != nil {
		// Session is attached to the topic.
		if (msg.Original == "me" || msg.Original == "fnd") && msg.Leave.Unsub {
			// User should not unsubscribe from 'me' or 'find'. Just leaving is fine.
			s.queueOut(ErrPermissionDeniedReply(msg, msg.Timestamp))
			s.inflightReqs.Done()
		} else {
			// Unlink from topic, topic will send a reply.
			sub.done <- msg
		}
		return
	}
	s.inflightReqs.Done()
	if !msg.Leave.Unsub {
		// Session is not attached to the topic, wants to leave - fine, no change
		s.queueOut(InfoNotJoined(msg.Id, msg.Original, msg.Timestamp))
	} else {
		// Session wants to unsubscribe from the topic it did not join
		logs.Warn.Println("s.leave:", "must attach first", s.sid)
		s.queueOut(ErrAttachFirst(msg, msg.Timestamp))
	}
}

// Broadcast a message to all topic subscribers
func (s *Session) publish(msg *ClientComMessage) {
	var resp *ServerComMessage
	msg.RcptTo, resp = s.expandTopicName(msg)
	if resp != nil {
		s.queueOut(resp)
		return
	}

	// Add "sender" header if the message is sent on behalf of another user.
	if msg.AsUser != s.uid.UserId() {
		if msg.Pub.Head == nil {
			msg.Pub.Head = make(map[string]any)
		}
		msg.Pub.Head["sender"] = s.uid.UserId()
	} else if msg.Pub.Head != nil {
		// Clear potentially false "sender" field.
		delete(msg.Pub.Head, "sender")
		if len(msg.Pub.Head) == 0 {
			msg.Pub.Head = nil
		}
	}

	if sub := s.getSub(msg.RcptTo); sub != nil {
		// This is a post to a subscribed topic. The message is sent to the topic only
		select {
		case sub.broadcast <- msg:
		default:
			// Reply with a 503 to the user.
			s.queueOut(ErrServiceUnavailableReply(msg, msg.Timestamp))
			logs.Err.Println("s.publish: sub.broadcast channel full, topic ", msg.RcptTo, s.sid)
		}
	} else if msg.RcptTo == "sys" {
		// Publishing to "sys" topic requires no subscription.
		select {
		default:
			// Reply with a 503 to the user.
			s.queueOut(ErrServiceUnavailableReply(msg, msg.Timestamp))
			logs.Err.Println("s.publish: hub.route channel full", s.sid)
		}
	} else {
		// Publish request received without attaching to topic first.
		s.queueOut(ErrAttachFirst(msg, msg.Timestamp))
		logs.Warn.Printf("s.publish[%s]: must attach first %s", msg.RcptTo, s.sid)
	}
}

// Authenticate
func (s *Session) login(msg *ClientComMessage) {
	// msg.from is ignored here

	if msg.Login.Scheme == "reset" {
		s.queueOut(InfoAuthReset(msg.Id, msg.Timestamp))
		return
	}

	if !s.uid.IsZero() {
		// params := map[string]interface{}{"user": s.uid.UserId(), "authlvl": s.authLevel.String()}
		s.queueOut(ErrAlreadyAuthenticated(msg.Id, "", msg.Timestamp))
		return
	}
}

func (s *Session) get(msg *ClientComMessage) {
	// Expand topic name.
	var resp *ServerComMessage
	msg.RcptTo, resp = s.expandTopicName(msg)
	if resp != nil {
		s.queueOut(resp)
		return
	}

	msg.MetaWhat = parseMsgClientMeta(msg.Get.What)

	sub := s.getSub(msg.RcptTo)
	if msg.MetaWhat == 0 {
		s.queueOut(ErrMalformedReply(msg, msg.Timestamp))
		logs.Warn.Println("s.get: invalid Get message action", msg.Get.What)
	} else if sub != nil {
		select {
		case sub.meta <- msg:
		default:
			// Reply with a 503 to the user.
			s.queueOut(ErrServiceUnavailableReply(msg, msg.Timestamp))
			logs.Err.Println("s.get: sub.meta channel full, topic ", msg.RcptTo, s.sid)
		}
	} else if msg.MetaWhat&(constMsgMetaDesc|constMsgMetaSub) != 0 {
		// Request some minimal info from a topic not currently attached to.
		select {
		default:
			// Reply with a 503 to the user.
			s.queueOut(ErrServiceUnavailableReply(msg, msg.Timestamp))
			logs.Err.Println("s.get: hub.meta channel full", s.sid)
		}
	} else {
		logs.Warn.Println("s.get: subscribe first to get=", msg.Get.What)
		s.queueOut(ErrPermissionDeniedReply(msg, msg.Timestamp))
	}
}

func (s *Session) set(msg *ClientComMessage) {
	// Expand topic name.
	var resp *ServerComMessage
	msg.RcptTo, resp = s.expandTopicName(msg)
	if resp != nil {
		s.queueOut(resp)
		return
	}

	if msg.Set.Desc != nil {
		msg.MetaWhat = constMsgMetaDesc
	}
	if msg.Set.Sub != nil {
		msg.MetaWhat |= constMsgMetaSub
	}
	if msg.Set.Tags != nil {
		msg.MetaWhat |= constMsgMetaTags
	}
	if msg.Set.Cred != nil {
		msg.MetaWhat |= constMsgMetaCred
	}

	if msg.MetaWhat == 0 {
		s.queueOut(ErrMalformedReply(msg, msg.Timestamp))
		logs.Warn.Println("s.set: nil Set action")
	} else if sub := s.getSub(msg.RcptTo); sub != nil {
		select {
		case sub.meta <- msg:
		default:
			// Reply with a 503 to the user.
			s.queueOut(ErrServiceUnavailableReply(msg, msg.Timestamp))
			logs.Err.Println("s.set: sub.meta channel full, topic ", msg.RcptTo, s.sid)
		}
	} else if msg.MetaWhat&(constMsgMetaTags|constMsgMetaCred) != 0 {
		logs.Warn.Println("s.set: can Set tags/creds for subscribed topics only", msg.MetaWhat)
		s.queueOut(ErrPermissionDeniedReply(msg, msg.Timestamp))
	} else {
		// Desc.Private and Sub updates are possible without the subscription.
		select {
		default:
			// Reply with a 503 to the user.
			s.queueOut(ErrServiceUnavailableReply(msg, msg.Timestamp))
			logs.Err.Println("s.set: hub.meta channel full", s.sid)
		}
	}
}

func (s *Session) del(msg *ClientComMessage) {
	msg.MetaWhat = parseMsgClientDel(msg.Del.What)

	// Delete something other than user: topic, subscription, message(s)

	// Expand topic name and validate request.
	var resp *ServerComMessage
	msg.RcptTo, resp = s.expandTopicName(msg)
	if resp != nil {
		s.queueOut(resp)
		return
	}

	if msg.MetaWhat == 0 {
		s.queueOut(ErrMalformedReply(msg, msg.Timestamp))
		logs.Warn.Println("s.del: invalid Del action", msg.Del.What, s.sid)
		return
	}

	if sub := s.getSub(msg.RcptTo); sub != nil && msg.MetaWhat != constMsgDelTopic {
		// Session is attached, deleting subscription or messages. Send to topic.
		select {
		case sub.meta <- msg:
		default:
			// Reply with a 503 to the user.
			s.queueOut(ErrServiceUnavailableReply(msg, msg.Timestamp))
			logs.Err.Println("s.del: sub.meta channel full, topic ", msg.RcptTo, s.sid)
		}
	} else if msg.MetaWhat == constMsgDelTopic {
		// Deleting topic: for sessions attached or not attached, send request to hub first.
		// Hub will forward to topic, if appropriate.
		select {
		case globals.hub.unreg <- &topicUnreg{
			rcptTo: msg.RcptTo,
			sess:   s,
			del:    true,
		}:
		default:
			// Reply with a 503 to the user.
			s.queueOut(ErrServiceUnavailableReply(msg, msg.Timestamp))
			logs.Err.Println("s.del: hub.unreg channel full", s.sid)
		}
	} else {
		// Must join the topic to delete messages or subscriptions.
		s.queueOut(ErrAttachFirst(msg, msg.Timestamp))
		logs.Warn.Println("s.del: invalid Del action while unsubbed", msg.Del.What, s.sid)
	}
}

// Broadcast a transient message to active topic subscribers.
// Not reporting any errors.
func (s *Session) note(msg *ClientComMessage) {
	if s.ver == 0 || msg.AsUser == "" {
		// Silently ignore the message: have not received {hi} or don't know who sent the message.
		return
	}

	// Expand topic name and validate request.
	var resp *ServerComMessage
	msg.RcptTo, resp = s.expandTopicName(msg)
	if resp != nil {
		// Silently ignoring the message
		return
	}

	switch msg.Note.What {
	case "data":
		if msg.Note.Payload == nil {
			// Payload must be present in 'data' notifications.
			return
		}
	case "kp", "kpa", "kpv":
		if msg.Note.SeqId != 0 {
			return
		}
	case "read", "recv":
		if msg.Note.SeqId <= 0 {
			return
		}
	default:
		return
	}

	if sub := s.getSub(msg.RcptTo); sub != nil {
		// Pings can be sent to subscribed topics only
		select {
		case sub.broadcast <- msg:
		default:
			// Reply with a 503 to the user.
			s.queueOut(ErrServiceUnavailableReply(msg, msg.Timestamp))
			logs.Err.Println("s.note: sub.broacast channel full, topic ", msg.RcptTo, s.sid)
		}
	} else if msg.Note.What == "recv" || (msg.Note.What == "call" && (msg.Note.Event == "ringing" || msg.Note.Event == "hang-up" || msg.Note.Event == "accept")) {
		// One of the following events happened:
		// 1. Client received a pres notification about a new message, initiated a fetch
		// from the server (and detached from the topic) and acknowledges receipt.
		// 2. Client is either accepting or terminating the current video call or
		// letting the initiator of the call know that it is ringing/notifying
		// the user about the call.
		//
		// Hub will forward to topic, if appropriate.
		select {
		default:
			// Reply with a 503 to the user.
			s.queueOut(ErrServiceUnavailableReply(msg, msg.Timestamp))
			logs.Err.Println("s.note: hub.route channel full", s.sid)
		}
	} else {
		s.queueOut(ErrAttachFirst(msg, msg.Timestamp))
		logs.Warn.Println("s.note: note to invalid topic - must subscribe first", msg.Note.What, s.sid)
	}
}

// expandTopicName expands session specific topic name to global name
// Returns
//
//	topic: session-specific topic name the message recipient should see
//	routeTo: routable global topic name
//	err: *ServerComMessage with an error to return to the sender
func (s *Session) expandTopicName(msg *ClientComMessage) (string, *ServerComMessage) {
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
		statsAddHistSample("OutgoingMessageSize", float64(dataSize))
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

func (sess *Session) sendMessageLp(wrt http.ResponseWriter, msg any) bool {
	if len(sess.send) > sendQueueLimit {
		logs.Err.Println("longPoll: outbound queue limit exceeded", sess.sid)
		return false
	}

	statsInc("OutgoingMessagesLongpollTotal", 1)
	if err := lpWrite(wrt, msg); err != nil {
		logs.Err.Println("longPoll: writeOnce failed", sess.sid, err)
		return false
	}

	return true
}

func (sess *Session) writeOnce(wrt http.ResponseWriter, req *http.Request) {
	for {
		select {
		case msg, ok := <-sess.send:
			if !ok {
				return
			}
			switch v := msg.(type) {
			case *ServerComMessage: // single unserialized message
				w := sess.serializeAndUpdateStats(v)
				if !sess.sendMessageLp(wrt, w) {
					return
				}
			default: // serialized message
				if !sess.sendMessageLp(wrt, v) {
					return
				}
			}
			return

		case <-sess.bkgTimer.C:
			if sess.background {
				sess.background = false
				sess.onBackgroundTimer()
			}

		case msg := <-sess.stop:
			// Request to close the session. Make it unavailable.
			globals.sessionStore.Delete(sess)
			// Don't care if lpWrite fails.
			if msg != nil {
				lpWrite(wrt, msg)
			}
			return

		case topic := <-sess.detach:
			// Request to detach the session from a topic.
			sess.delSub(topic)
			// No 'return' statement here: continue waiting

		case <-time.After(pingPeriod):
			// just write an empty packet on timeout
			if _, err := wrt.Write([]byte{}); err != nil {
				logs.Err.Println("longPoll: writeOnce: timout", sess.sid, err)
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
	wrt.Write(msg.([]byte))
	return nil
}

func (sess *Session) readOnce(wrt http.ResponseWriter, req *http.Request) (int, error) {
	if req.ContentLength > globals.maxMessageSize {
		return http.StatusExpectationFailed, errors.New("request too large")
	}

	req.Body = http.MaxBytesReader(wrt, req.Body, globals.maxMessageSize)
	raw, err := ioutil.ReadAll(req.Body)
	if err == nil {
		// Locking-unlocking is needed because the client may issue multiple requests in parallel.
		// Should not affect performance
		sess.lock.Lock()
		statsInc("IncomingMessagesLongpollTotal", 1)
		sess.dispatchRaw(raw)
		sess.lock.Unlock()
		return 0, nil
	}

	return 0, err
}

// Obtain IP address of the client.
func getRemoteAddr(req *http.Request) string {
	var addr string
	if globals.useXForwardedFor {
		addr = req.Header.Get("X-Forwarded-For")
		if !utils.IsRoutableIP(addr) {
			addr = ""
		}
	}
	if addr != "" {
		return addr
	}
	return req.RemoteAddr
}
