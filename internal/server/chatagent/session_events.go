package chatagent

import "sync"

// SessionEventHub fans out stream events to multiple SSE subscribers for one session.
type SessionEventHub struct {
	mu   sync.RWMutex
	subs map[string]*ChannelPublisher
}

var sessionEventHubs sync.Map

// GetSessionEventHub returns the event hub for one session.
func GetSessionEventHub(sessionID string) *SessionEventHub {
	if sessionID == "" {
		return &SessionEventHub{subs: make(map[string]*ChannelPublisher)}
	}
	if raw, ok := sessionEventHubs.Load(sessionID); ok {
		if hub, ok := raw.(*SessionEventHub); ok {
			return hub
		}
	}
	hub := &SessionEventHub{subs: make(map[string]*ChannelPublisher)}
	actual, _ := sessionEventHubs.LoadOrStore(sessionID, hub)
	if existing, ok := actual.(*SessionEventHub); ok {
		return existing
	}
	return hub
}

func clearSessionEventHub(sessionID string) {
	if sessionID == "" {
		return
	}
	if raw, ok := sessionEventHubs.LoadAndDelete(sessionID); ok {
		if hub, ok := raw.(*SessionEventHub); ok {
			hub.closeAll()
		}
	}
}

// Subscribe registers one SSE consumer on the session hub.
func (h *SessionEventHub) Subscribe(id string, buffer int) *ChannelPublisher {
	pub := NewChannelPublisher(buffer)
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.subs == nil {
		h.subs = make(map[string]*ChannelPublisher)
	}
	h.subs[id] = pub
	return pub
}

// Detach removes one SSE consumer from hub fan-out without closing its publisher.
// Use this when the publisher is still owned by an in-flight run (primary messages SSE).
func (h *SessionEventHub) Detach(id string) {
	h.mu.Lock()
	delete(h.subs, id)
	h.mu.Unlock()
}

// Unsubscribe removes one SSE consumer from the session hub and closes its publisher.
func (h *SessionEventHub) Unsubscribe(id string) {
	h.mu.Lock()
	pub, ok := h.subs[id]
	delete(h.subs, id)
	h.mu.Unlock()
	if ok && pub != nil {
		pub.Close()
	}
}

func (h *SessionEventHub) publish(event StreamEvent) {
	h.mu.RLock()
	pubs := make([]*ChannelPublisher, 0, len(h.subs))
	for _, pub := range h.subs {
		pubs = append(pubs, pub)
	}
	h.mu.RUnlock()
	for _, pub := range pubs {
		_ = pub.Publish(event)
	}
}

func (h *SessionEventHub) closeAll() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for id, pub := range h.subs {
		if pub != nil {
			pub.Close()
		}
		delete(h.subs, id)
	}
}

// WritePendingConfirmIfAny writes a waiting confirm event for late /events subscribers.
// Returns true when the writer failed and the stream should stop.
func WritePendingConfirmIfAny(sessionID string, write func(StreamEvent) bool) bool {
	ev, ok := LookupPendingConfirm(sessionID)
	if !ok {
		return false
	}
	return write(ev)
}

// hubPublisher publishes events to every subscriber on a session hub.
type hubPublisher struct {
	sessionID string
}

func (p hubPublisher) Publish(event StreamEvent) error {
	GetSessionEventHub(p.sessionID).publish(event)
	return nil
}

// PublishSessionEvent delivers one event to all SSE subscribers for a session.
func PublishSessionEvent(sessionID string, event StreamEvent) {
	GetSessionEventHub(sessionID).publish(event)
}

// ResetSessionEventHubsForTest clears all session event hubs.
func ResetSessionEventHubsForTest() {
	sessionEventHubs = sync.Map{}
}
