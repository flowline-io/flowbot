package server

import (
	"context"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/event"
)

// messageDestinations adapts UserStore/PlatformStore to event.MessageDestinations.
type messageDestinations struct{}

func (messageDestinations) GetUserByFlag(ctx context.Context, flag string) (*event.DestinationUser, error) {
	user, err := store.UserStoreFromDB().GetUserByFlag(ctx, flag)
	if err != nil {
		return nil, err
	}
	return &event.DestinationUser{ID: user.ID}, nil
}

func (messageDestinations) GetPlatformUsersByUserId(ctx context.Context, userID int64) ([]*event.DestinationPlatformUser, error) {
	rows, err := store.UserStoreFromDB().GetPlatformUsersByUserId(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]*event.DestinationPlatformUser, 0, len(rows))
	for _, row := range rows {
		out = append(out, &event.DestinationPlatformUser{
			PlatformID: row.PlatformID,
			Flag:       row.Flag,
		})
	}
	return out, nil
}

func (messageDestinations) GetPlatformChannelByFlag(ctx context.Context, flag string) (*event.DestinationPlatformChannel, error) {
	ch, err := store.PlatformStoreFromDB().GetPlatformChannelByFlag(ctx, flag)
	if err != nil {
		return nil, err
	}
	return &event.DestinationPlatformChannel{PlatformID: ch.PlatformID}, nil
}

func (messageDestinations) GetPlatform(ctx context.Context, id int64) (*event.DestinationPlatform, error) {
	p, err := store.PlatformStoreFromDB().GetPlatform(ctx, id)
	if err != nil {
		return nil, err
	}
	return &event.DestinationPlatform{ID: p.ID, Name: p.Name}, nil
}

func (messageDestinations) GetPlatforms(ctx context.Context) ([]*event.DestinationPlatform, error) {
	rows, err := store.PlatformStoreFromDB().GetPlatforms(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*event.DestinationPlatform, 0, len(rows))
	for _, row := range rows {
		out = append(out, &event.DestinationPlatform{ID: row.ID, Name: row.Name})
	}
	return out, nil
}

func (messageDestinations) GetPlatformChannelUsersByUserFlags(ctx context.Context, userFlags []string) ([]*event.DestinationChannelUser, error) {
	rows, err := store.PlatformStoreFromDB().GetPlatformChannelUsersByUserFlags(ctx, userFlags)
	if err != nil {
		return nil, err
	}
	out := make([]*event.DestinationChannelUser, 0, len(rows))
	for _, row := range rows {
		out = append(out, &event.DestinationChannelUser{
			UserFlag:    row.UserFlag,
			ChannelFlag: row.ChannelFlag,
		})
	}
	return out, nil
}

// WireMessageDestinations injects the store-backed message destinations adapter into event.
func WireMessageDestinations() {
	event.SetMessageDestinations(messageDestinations{})
}
