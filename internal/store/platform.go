package store

import (
	"context"
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/bot"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/channel"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/platform"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/platformchannel"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/platformchanneluser"
	"github.com/flowline-io/flowbot/pkg/types"
)

// PlatformStore persists platforms, channels, bots, and platform channel links.
type PlatformStore struct {
	client *gen.Client
}

// NewPlatformStore creates a PlatformStore with the given ent client.
func NewPlatformStore(client *gen.Client) *PlatformStore {
	return &PlatformStore{client: client}
}

// PlatformStoreFromDB returns a PlatformStore using the global database client.
func PlatformStoreFromDB() *PlatformStore {
	return NewPlatformStore(ClientFromDB())
}

// Client returns the underlying ent client.
func (s *PlatformStore) Client() *gen.Client {
	if s == nil {
		return nil
	}
	return s.client
}

// GetPlatformChannelByFlag returns the platform channel by flag.
func (s *PlatformStore) GetPlatformChannelByFlag(ctx context.Context, flag string) (*gen.PlatformChannel, error) {
	u, err := s.client.PlatformChannel.Query().Where(platformchannel.FlagEQ(flag)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: get platform channel by flag: %w", err)
	}
	return u, nil
}

// GetPlatformChannelsByPlatformIds returns the platform channels by platform ids.
func (s *PlatformStore) GetPlatformChannelsByPlatformIds(ctx context.Context, platformIds []int64) ([]*gen.PlatformChannel, error) {
	channels, err := s.client.PlatformChannel.Query().Where(platformchannel.PlatformIDIn(platformIds...)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: get platform channels by platform ids: %w", err)
	}
	result := make([]*gen.PlatformChannel, len(channels))
	copy(result, channels)
	return result, nil
}

// GetPlatformChannelsByChannelId returns the platform channels by channel id.
func (s *PlatformStore) GetPlatformChannelsByChannelId(ctx context.Context, channelId int64) (*gen.PlatformChannel, error) {
	u, err := s.client.PlatformChannel.Query().Where(platformchannel.ChannelIDEQ(channelId)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: get platform channel by channel id: %w", err)
	}
	return u, nil
}

// CreatePlatformChannel persists a new platform channel.
func (s *PlatformStore) CreatePlatformChannel(ctx context.Context, item *gen.PlatformChannel) (int64, error) {
	u, err := s.client.PlatformChannel.Create().
		SetPlatformID(item.PlatformID).
		SetChannelID(item.ChannelID).
		SetFlag(item.Flag).
		SetCreatedAt(item.CreatedAt).
		SetUpdatedAt(item.UpdatedAt).
		Save(ctx)
	if err != nil {
		return 0, fmt.Errorf("postgres: create platform channel: %w", err)
	}
	return u.ID, nil
}

// UpdatePlatformChannelChannelID updates the platform channel channel id.
func (s *PlatformStore) UpdatePlatformChannelChannelID(ctx context.Context, platformChannelID, channelID int64) error {
	n, err := s.client.PlatformChannel.Update().
		Where(platformchannel.IDEQ(platformChannelID)).
		SetChannelID(channelID).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: update platform channel channel id: %w", err)
	}
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// CreatePlatformChannelUser persists a new platform channel user.
func (s *PlatformStore) CreatePlatformChannelUser(ctx context.Context, item *gen.PlatformChannelUser) (int64, error) {
	u, err := s.client.PlatformChannelUser.Create().
		SetPlatformID(item.PlatformID).
		SetChannelFlag(item.ChannelFlag).
		SetUserFlag(item.UserFlag).
		SetCreatedAt(item.CreatedAt).
		SetUpdatedAt(item.UpdatedAt).
		Save(ctx)
	if err != nil {
		return 0, fmt.Errorf("postgres: create platform channel user: %w", err)
	}
	return u.ID, nil
}

// GetPlatformChannelUsersByUserFlag returns the platform channel users by user flag.
func (s *PlatformStore) GetPlatformChannelUsersByUserFlag(ctx context.Context, userFlag string) ([]*gen.PlatformChannelUser, error) {
	users, err := s.client.PlatformChannelUser.Query().
		Where(platformchanneluser.UserFlagEQ(userFlag)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: get platform channel users by user flag: %w", err)
	}
	result := make([]*gen.PlatformChannelUser, len(users))
	copy(result, users)
	return result, nil
}

// GetPlatformChannelUsersByUserFlags returns platform channel user records for a batch of user flags.
func (s *PlatformStore) GetPlatformChannelUsersByUserFlags(ctx context.Context, userFlags []string) ([]*gen.PlatformChannelUser, error) {
	if len(userFlags) == 0 {
		return nil, nil
	}
	users, err := s.client.PlatformChannelUser.Query().
		Where(platformchanneluser.UserFlagIn(userFlags...)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: get platform channel users by user flags: %w", err)
	}
	result := make([]*gen.PlatformChannelUser, len(users))
	copy(result, users)
	return result, nil
}

// GetBot returns the bot.
func (s *PlatformStore) GetBot(ctx context.Context, id int64) (*gen.Bot, error) {
	b, err := s.client.Bot.Query().Where(bot.IDEQ(id)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: getbot: %w", err)
	}
	return b, nil
}

// GetBotByName returns the bot by name.
func (s *PlatformStore) GetBotByName(ctx context.Context, name string) (*gen.Bot, error) {
	b, err := s.client.Bot.Query().Where(bot.NameEQ(name)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: getbotbyname: %w", err)
	}
	return b, nil
}

// CreateBot persists a new bot.
func (s *PlatformStore) CreateBot(ctx context.Context, botModel *gen.Bot) (int64, error) {
	b, err := s.client.Bot.Create().
		SetName(botModel.Name).
		SetState(int(botModel.State)).
		SetCreatedAt(botModel.CreatedAt).
		SetUpdatedAt(botModel.UpdatedAt).
		Save(ctx)
	if err != nil {
		return 0, fmt.Errorf("postgres: createbot: %w", err)
	}
	return b.ID, nil
}

// UpdateBot updates the bot.
func (s *PlatformStore) UpdateBot(ctx context.Context, botModel *gen.Bot) error {
	_, err := s.client.Bot.Update().Where(bot.IDEQ(botModel.ID)).
		SetName(botModel.Name).
		SetState(int(botModel.State)).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return types.ErrNotFound
		}
		return fmt.Errorf("postgres: updatebot: %w", err)
	}
	return nil
}

// DeleteBot deletes the bot.
func (s *PlatformStore) DeleteBot(ctx context.Context, name string) error {
	_, err := s.client.Bot.Delete().Where(bot.NameEQ(name)).Exec(ctx)
	if err != nil {
		return fmt.Errorf("postgres: deletebot: %w", err)
	}
	return nil
}

// GetBots returns the bots.
func (s *PlatformStore) GetBots(ctx context.Context) ([]*gen.Bot, error) {
	bots, err := s.client.Bot.Query().Order(gen.Asc(bot.FieldCreatedAt)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: getbots: %w", err)
	}
	result := make([]*gen.Bot, len(bots))
	copy(result, bots)
	return result, nil
}

// GetPlatform returns the platform.
func (s *PlatformStore) GetPlatform(ctx context.Context, id int64) (*gen.Platform, error) {
	p, err := s.client.Platform.Query().Where(platform.IDEQ(id)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: getplatform: %w", err)
	}
	return p, nil
}

// GetPlatformByName returns the platform by name.
func (s *PlatformStore) GetPlatformByName(ctx context.Context, name string) (*gen.Platform, error) {
	p, err := s.client.Platform.Query().Where(platform.NameEQ(name)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: getplatformbyname: %w", err)
	}
	return p, nil
}

// GetPlatforms returns the platforms.
func (s *PlatformStore) GetPlatforms(ctx context.Context) ([]*gen.Platform, error) {
	platforms, err := s.client.Platform.Query().Order(gen.Asc(platform.FieldCreatedAt)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: getplatforms: %w", err)
	}
	result := make([]*gen.Platform, len(platforms))
	copy(result, platforms)
	return result, nil
}

// CreatePlatform persists a new platform.
func (s *PlatformStore) CreatePlatform(ctx context.Context, platformModel *gen.Platform) (int64, error) {
	p, err := s.client.Platform.Create().
		SetName(platformModel.Name).
		SetCreatedAt(platformModel.CreatedAt).
		SetUpdatedAt(platformModel.UpdatedAt).
		Save(ctx)
	if err != nil {
		return 0, fmt.Errorf("postgres: createplatform: %w", err)
	}
	return p.ID, nil
}

// GetChannel returns the channel.
func (s *PlatformStore) GetChannel(ctx context.Context, id int64) (*gen.Channel, error) {
	c, err := s.client.Channel.Query().Where(channel.IDEQ(id)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: getchannel: %w", err)
	}
	return c, nil
}

// GetChannelByName returns the channel by name.
func (s *PlatformStore) GetChannelByName(ctx context.Context, name string) (*gen.Channel, error) {
	c, err := s.client.Channel.Query().Where(channel.NameEQ(name)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: getchannelbyname: %w", err)
	}
	return c, nil
}

// CreateChannel persists a new channel.
func (s *PlatformStore) CreateChannel(ctx context.Context, channelModel *gen.Channel) (int64, error) {
	c, err := s.client.Channel.Create().
		SetName(channelModel.Name).
		SetFlag(channelModel.Flag).
		SetState(int(channelModel.State)).
		SetCreatedAt(channelModel.CreatedAt).
		SetUpdatedAt(channelModel.UpdatedAt).
		Save(ctx)
	if err != nil {
		return 0, fmt.Errorf("postgres: createchannel: %w", err)
	}
	channelModel.ID = c.ID
	return c.ID, nil
}

// UpdateChannel updates the channel.
func (s *PlatformStore) UpdateChannel(ctx context.Context, channelModel *gen.Channel) error {
	_, err := s.client.Channel.Update().Where(channel.IDEQ(channelModel.ID)).
		SetName(channelModel.Name).
		SetFlag(channelModel.Flag).
		SetState(int(channelModel.State)).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return types.ErrNotFound
		}
		return fmt.Errorf("postgres: updatechannel: %w", err)
	}
	return nil
}

// DeleteChannel deletes the channel.
func (s *PlatformStore) DeleteChannel(ctx context.Context, name string) error {
	_, err := s.client.Channel.Delete().Where(channel.NameEQ(name)).Exec(ctx)
	if err != nil {
		return fmt.Errorf("postgres: deletechannel: %w", err)
	}
	return nil
}

// GetChannels returns the channels.
func (s *PlatformStore) GetChannels(ctx context.Context) ([]*gen.Channel, error) {
	channels, err := s.client.Channel.Query().Order(gen.Asc(channel.FieldCreatedAt)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: getchannels: %w", err)
	}
	result := make([]*gen.Channel, len(channels))
	copy(result, channels)
	return result, nil
}

