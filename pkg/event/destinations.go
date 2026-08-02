package event

import (
	"context"
	"fmt"
	"sync"
)

// DestinationUser is the user identity subset needed to resolve message destinations.
type DestinationUser struct {
	ID int64
}

// DestinationPlatformUser links a user to a platform for delivery.
type DestinationPlatformUser struct {
	PlatformID int64
	Flag       string
}

// DestinationPlatformChannel identifies a platform channel by routing metadata.
type DestinationPlatformChannel struct {
	PlatformID int64
}

// DestinationPlatform is a messaging platform used for delivery.
type DestinationPlatform struct {
	ID   int64
	Name string
}

// DestinationChannelUser maps a platform user flag to a channel flag.
type DestinationChannelUser struct {
	UserFlag    string
	ChannelFlag string
}

// MessageDestinations resolves users, platforms, and channels for message delivery.
type MessageDestinations interface {
	GetUserByFlag(ctx context.Context, flag string) (*DestinationUser, error)
	GetPlatformUsersByUserId(ctx context.Context, userID int64) ([]*DestinationPlatformUser, error)
	GetPlatformChannelByFlag(ctx context.Context, flag string) (*DestinationPlatformChannel, error)
	GetPlatform(ctx context.Context, id int64) (*DestinationPlatform, error)
	GetPlatforms(ctx context.Context) ([]*DestinationPlatform, error)
	GetPlatformChannelUsersByUserFlags(ctx context.Context, userFlags []string) ([]*DestinationChannelUser, error)
}

var (
	messageDestinationsMu sync.RWMutex
	messageDestinations   MessageDestinations
)

// SetMessageDestinations wires the persistence backend used by SendMessage.
func SetMessageDestinations(s MessageDestinations) {
	messageDestinationsMu.Lock()
	defer messageDestinationsMu.Unlock()
	messageDestinations = s
}

// GetMessageDestinations returns the injected message destinations store.
func GetMessageDestinations() MessageDestinations {
	messageDestinationsMu.RLock()
	defer messageDestinationsMu.RUnlock()
	return messageDestinations
}

func requireMessageDestinations() (MessageDestinations, error) {
	s := GetMessageDestinations()
	if s == nil {
		return nil, fmt.Errorf("event: message destinations store is not configured")
	}
	return s, nil
}
