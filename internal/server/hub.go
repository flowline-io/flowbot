package server

import (
	"github.com/flowline-io/flowbot/internal/types"
	"sync"
)

// Request to hub to remove the topic
type topicUnreg struct {
	// Session making the request, could be nil.
	sess *Session
	// Routable name of the topic to drop. Duplicated here because pkt could be nil.
	rcptTo string
	// UID of the user being deleted. Duplicated here because pkt could be nil.
	forUser types.Uid
	// Unregister then delete the topic.
	del bool
	// Channel for reporting operation completion when deleting topics for a user.
	done chan<- bool
}

type userStatusReq struct {
	// UID of the user being affected.
	forUser types.Uid
	// New topic state value. Only types.StateSuspended is supported at this time.
	//state types.ObjState
}

// Hub is the core structure which holds topics.
type Hub struct {

	// Topics must be indexed by name
	topics *sync.Map

	// Current number of loaded topics
	numTopics int

	// Remove topic from hub, possibly deleting it afterwards, buffered at 32
	unreg chan *topicUnreg

	// Channel for suspending/resuming users, buffered 128.
	userStatus chan *userStatusReq

	// Cluster request to rehash topics, unbuffered
	rehash chan bool

	// Request to shutdown, unbuffered
	shutdown chan chan<- bool
}
