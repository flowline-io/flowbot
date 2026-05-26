package server

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
)

// registerPlatformUser registers a platform user based on the provided message event data.
// It checks if the platform user already exists by its flag, and if found, retrieves the existing user flag.
// If the platform user does not exist, it creates a new user and platform user entry in the database.
// It also associates the platform user with the platform.
// Returns the user flag and an error if any operation fails.
func registerPlatformUser(data protocol.MessageEventData) (types.Uid, error) {
	platform, err := store.Database.GetPlatformByName(context.Background(), data.Self.Platform)
	if err != nil {
		return "", err
	}

	platformUser, err := store.Database.GetPlatformUserByFlag(context.Background(), data.UserId)
	if err != nil && !errors.Is(err, types.ErrNotFound) {
		return "", err
	}

	if platformUser != nil && platformUser.ID > 0 {
		user, err := store.Database.GetUserById(context.Background(), platformUser.UserID)
		if err != nil {
			return "", err
		}
		return types.Uid(user.Flag), nil
	}
	user := &gen.User{
		Flag:  types.Id(),
		Name:  "user",
		Tags:  "[]",
		State: int(schema.UserActive),
	}
	err = store.Database.UserCreate(context.Background(), user)
	if err != nil {
		return "", err
	}

	_, err = store.Database.CreatePlatformUser(context.Background(), &gen.PlatformUser{
		PlatformID: platform.ID,
		UserID:     user.ID,
		Flag:       data.UserId,
		Name:       "user",
		IsBot:      false,
	})
	if err != nil {
		return "", err
	}
	return types.Uid(user.Flag), nil
}

// registerPlatformChannel registers a platform channel based on the provided message event data.
// It checks if the platform channel already exists by its topic ID, and if found, retrieves the existing channel flag.
// If the platform channel does not exist, it creates a new channel and platform channel entry in the database.
// It also associates the platform channel with the user who triggered the event.
// Returns the channel flag and an error if any operation fails.
func registerPlatformChannel(data protocol.MessageEventData) (string, error) {
	platform, err := store.Database.GetPlatformByName(context.Background(), data.Self.Platform)
	if err != nil {
		return "", err
	}

	platformChannel, err := store.Database.GetPlatformChannelByFlag(context.Background(), data.TopicId)
	if err != nil && !errors.Is(err, types.ErrNotFound) {
		return "", err
	}

	if platformChannel != nil && platformChannel.ID > 0 {
		channel, err := store.Database.GetChannel(context.Background(), platformChannel.ChannelID)
		if err != nil {
			return "", err
		}
		return channel.Flag, nil
	}
	channel := &gen.Channel{
		Flag:  types.Id(),
		Name:  fmt.Sprintf("%s_%s", data.Self.Platform, data.TopicId),
		State: int(schema.ChannelActive),
	}
	_, err = store.Database.CreateChannel(context.Background(), channel)
	if err != nil {
		return "", err
	}

	_, err = store.Database.CreatePlatformChannel(context.Background(), &gen.PlatformChannel{
		PlatformID: platform.ID,
		ChannelID:  channel.ID,
		Flag:       data.TopicId,
	})
	if err != nil {
		return "", err
	}

	_, err = store.Database.CreatePlatformChannelUser(context.Background(), &gen.PlatformChannelUser{
		PlatformID:  platform.ID,
		ChannelFlag: data.TopicId,
		UserFlag:    data.UserId,
	})
	if err != nil {
		return "", err
	}

	return channel.Flag, nil
}

// registerAgent Register agent by uid, topic, hostid and hostname
//
// if the agent already exists, update its last online time, otherwise create a new agent
func registerAgent(uid types.Uid, topic, hostid, hostname string) error {
	if hostid == "" {
		return fmt.Errorf("hostid is empty")
	}
	agent, err := store.Database.GetAgentByHostid(context.Background(), uid, topic, hostid)
	if err != nil && !errors.Is(err, types.ErrNotFound) {
		return err
	}

	if agent != nil && agent.ID > 0 {
		err = store.Database.UpdateAgentLastOnlineAt(context.Background(), uid, topic, hostid, time.Now())
		if err != nil {
			return err
		}
	} else {
		agent = &gen.Agent{
			UID:            uid.String(),
			Topic:          topic,
			Hostid:         hostid,
			Hostname:       hostname,
			OnlineDuration: 0,
			LastOnlineAt:   time.Now(),
		}
		_, err := store.Database.CreateAgent(context.Background(), agent)
		if err != nil {
			return err
		}
	}

	return nil
}
