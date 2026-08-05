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

// resolvePlatformUserFlag returns the stable platform-scoped user identifier from inbound event data.
// Some adapters omit user_id on edge-case events; fall back to self.user_id or a generated id so
// required PlatformUser.flag validation does not fail on empty strings.
func resolvePlatformUserFlag(data protocol.MessageEventData) string {
	if data.UserId != "" {
		return data.UserId
	}
	if data.Self.UserId != "" {
		return data.Self.UserId
	}
	return types.Id()
}

// registerPlatformUser registers a platform user based on the provided message event data.
// platform must already be resolved by the caller (one lookup per inbound message).
// It checks if the platform user already exists by its flag, and if found, retrieves the existing user flag.
// If the platform user does not exist, it creates a new user and platform user entry in the database.
// It also associates the platform user with the platform.
//
// Single-web-account (homelab) installs attach platform identities to that account's user so
// Chat Agent Permissions / approval mode configured in the Web UI apply to Slack/Discord chat.
// Relink updates platform_users only; chat sessions created under an orphan uid keep that uid
// until the user runs platform "end" then "chat" (or otherwise starts a new session).
// Returns the user flag and an error if any operation fails.
func registerPlatformUser(data protocol.MessageEventData, platform *gen.Platform) (types.Uid, error) {
	if platform == nil {
		return "", fmt.Errorf("register platform user: platform is nil")
	}
	ctx := context.Background()

	platformUserFlag := resolvePlatformUserFlag(data)

	platformUser, err := store.UserStoreFromDB().GetPlatformUserByFlag(ctx, platformUserFlag)
	if err != nil && !errors.Is(err, types.ErrNotFound) {
		return "", err
	}

	if platformUser != nil && platformUser.ID > 0 {
		user, err := store.UserStoreFromDB().GetUserById(ctx, platformUser.UserID)
		if err == nil {
			user, err = maybeRelinkPlatformUserToSoleWebAccount(ctx, platformUser, user)
			if err != nil {
				return "", err
			}
			return types.Uid(user.Flag), nil
		}
		if !errors.Is(err, types.ErrNotFound) {
			return "", err
		}
		user, err = soleWebAccountOrNewUser(ctx)
		if err != nil {
			return "", err
		}
		platformUser.UserID = user.ID
		if err = store.UserStoreFromDB().UpdatePlatformUser(ctx, platformUser); err != nil {
			return "", err
		}
		return types.Uid(user.Flag), nil
	}
	user, err := soleWebAccountOrNewUser(ctx)
	if err != nil {
		return "", err
	}

	email, avatarURL := platformUserProfileDefaults(data.Self.Platform, platformUserFlag)
	_, err = store.UserStoreFromDB().CreatePlatformUser(ctx, &gen.PlatformUser{
		PlatformID: platform.ID,
		UserID:     user.ID,
		Flag:       platformUserFlag,
		Name:       "user",
		Email:      email,
		AvatarURL:  avatarURL,
		IsBot:      false,
	})
	if err != nil {
		return "", err
	}
	return types.Uid(user.Flag), nil
}

func soleWebAccountOrNewUser(ctx context.Context) (*gen.User, error) {
	if accountUser, ok, err := soleWebAccountUser(ctx); err != nil {
		return nil, err
	} else if ok {
		return accountUser, nil
	}
	return newUserRecord()
}

func soleWebAccountUser(ctx context.Context) (*gen.User, bool, error) {
	account, ok, err := store.WebAccountStoreFromDB().SoleAccount(ctx)
	if err != nil || !ok {
		return nil, ok, err
	}
	user, err := store.UserStoreFromDB().UserGet(ctx, types.Uid(account.UID))
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return user, true, nil
}

// maybeRelinkPlatformUserToSoleWebAccount rewrites orphan platform identities onto the
// sole web account so Web UI permission/approval settings apply to existing Slack users.
// Platform users already linked to a web account are left unchanged (multi-user safe).
func maybeRelinkPlatformUserToSoleWebAccount(ctx context.Context, platformUser *gen.PlatformUser, current *gen.User) (*gen.User, error) {
	if platformUser == nil || current == nil {
		return current, nil
	}
	_, err := store.WebAccountStoreFromDB().GetByUID(ctx, current.Flag)
	if err == nil {
		return current, nil
	}
	if !errors.Is(err, types.ErrNotFound) {
		return nil, err
	}
	accountUser, ok, err := soleWebAccountUser(ctx)
	if err != nil || !ok {
		return current, err
	}
	if accountUser.ID == current.ID {
		return current, nil
	}
	platformUser.UserID = accountUser.ID
	if err := store.UserStoreFromDB().UpdatePlatformUser(ctx, platformUser); err != nil {
		return nil, err
	}
	return accountUser, nil
}

// newUserRecord creates a flowbot user row for first-time platform registration.
func newUserRecord() (*gen.User, error) {
	user := &gen.User{
		Flag:  types.Id(),
		Name:  "user",
		Tags:  "[]",
		State: int(schema.UserActive),
	}
	if err := store.UserStoreFromDB().UserCreate(context.Background(), user); err != nil {
		return nil, err
	}
	return user, nil
}

// newChannelForTopic creates a channel row for an inbound platform topic.
func newChannelForTopic(data protocol.MessageEventData) (*gen.Channel, error) {
	channel := &gen.Channel{
		Flag:  types.Id(),
		Name:  fmt.Sprintf("%s_%s", data.Self.Platform, data.TopicId),
		State: int(schema.ChannelActive),
	}
	channelID, err := store.PlatformStoreFromDB().CreateChannel(context.Background(), channel)
	if err != nil {
		return nil, err
	}
	channel.ID = channelID
	return channel, nil
}

// platformUserProfileDefaults returns placeholder profile fields for platform users created
// from inbound chat events that do not include email or avatar metadata.
func platformUserProfileDefaults(platformName, flag string) (email, avatarURL string) {
	if platformName == "" {
		platformName = "unknown"
	}
	if flag == "" {
		flag = "user"
	}
	return fmt.Sprintf("%s@%s.local", flag, platformName), "-"
}

// registerPlatformChannel registers a platform channel based on the provided message event data.
// platform must already be resolved by the caller (one lookup per inbound message).
// It checks if the platform channel already exists by its topic ID, and if found, retrieves the existing channel flag.
// If the platform channel does not exist, it creates a new channel and platform channel entry in the database.
// It also associates the platform channel with the user who triggered the event.
// Returns the channel flag and an error if any operation fails.
func registerPlatformChannel(data protocol.MessageEventData, platform *gen.Platform) (string, error) {
	if platform == nil {
		return "", fmt.Errorf("register platform channel: platform is nil")
	}

	platformChannel, err := store.PlatformStoreFromDB().GetPlatformChannelByFlag(context.Background(), data.TopicId)
	if err != nil && !errors.Is(err, types.ErrNotFound) {
		return "", err
	}

	if platformChannel != nil && platformChannel.ID > 0 {
		channel, err := store.PlatformStoreFromDB().GetChannel(context.Background(), platformChannel.ChannelID)
		if err == nil {
			return channel.Flag, nil
		}
		if !errors.Is(err, types.ErrNotFound) {
			return "", err
		}
		channel, err = newChannelForTopic(data)
		if err != nil {
			return "", err
		}
		if err = store.PlatformStoreFromDB().UpdatePlatformChannelChannelID(context.Background(), platformChannel.ID, channel.ID); err != nil {
			return "", err
		}
		return channel.Flag, nil
	}
	channel, err := newChannelForTopic(data)
	if err != nil {
		return "", err
	}

	_, err = store.PlatformStoreFromDB().CreatePlatformChannel(context.Background(), &gen.PlatformChannel{
		PlatformID: platform.ID,
		ChannelID:  channel.ID,
		Flag:       data.TopicId,
	})
	if err != nil {
		return "", err
	}

	_, err = store.PlatformStoreFromDB().CreatePlatformChannelUser(context.Background(), &gen.PlatformChannelUser{
		PlatformID:  platform.ID,
		ChannelFlag: data.TopicId,
		UserFlag:    resolvePlatformUserFlag(data),
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
	agent, err := store.RuntimeAgentStoreFromDB().GetAgentByHostid(context.Background(), uid, topic, hostid)
	if err != nil && !errors.Is(err, types.ErrNotFound) {
		return err
	}

	if agent != nil && agent.ID > 0 {
		err = store.RuntimeAgentStoreFromDB().UpdateAgentLastOnlineAt(context.Background(), uid, topic, hostid, time.Now())
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
		_, err := store.RuntimeAgentStoreFromDB().CreateAgent(context.Background(), agent)
		if err != nil {
			return err
		}
	}

	return nil
}
