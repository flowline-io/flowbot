package server

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
)

type registrationStore struct {
	testStoreAdapter
	platform          *gen.Platform
	platformUser      *gen.PlatformUser
	user              *gen.User
	createPlatform    *gen.PlatformUser
	updatedPlatform   *gen.PlatformUser
	platformUserErr   error
	userCreateErr     error
	createPlatformErr error
	updatePlatformErr error
}

func (s *registrationStore) GetPlatformByName(_ context.Context, name string) (*gen.Platform, error) {
	if s.platform == nil {
		return nil, types.ErrNotFound
	}
	if s.platform.Name != name {
		return nil, types.ErrNotFound
	}
	return s.platform, nil
}

func (s *registrationStore) GetPlatformUserByFlag(_ context.Context, flag string) (*gen.PlatformUser, error) {
	if s.platformUserErr != nil {
		return nil, s.platformUserErr
	}
	if s.platformUser == nil || s.platformUser.Flag != flag {
		return nil, types.ErrNotFound
	}
	return s.platformUser, nil
}

func (s *registrationStore) UserCreate(_ context.Context, user *gen.User) error {
	if s.userCreateErr != nil {
		return s.userCreateErr
	}
	user.ID = 42
	s.user = user
	return nil
}

func (s *registrationStore) GetUserById(_ context.Context, id int64) (*gen.User, error) {
	if s.user == nil || s.user.ID != id {
		return nil, types.ErrNotFound
	}
	return s.user, nil
}

func (s *registrationStore) CreatePlatformUser(_ context.Context, item *gen.PlatformUser) (int64, error) {
	if s.createPlatformErr != nil {
		return 0, s.createPlatformErr
	}
	s.createPlatform = item
	return 99, nil
}

func (s *registrationStore) UpdatePlatformUser(_ context.Context, item *gen.PlatformUser) error {
	if s.updatePlatformErr != nil {
		return s.updatePlatformErr
	}
	s.updatedPlatform = item
	s.platformUser = item
	return nil
}

func TestPlatformUserProfileDefaults(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		platformName string
		flag         string
		wantEmail    string
		wantAvatar   string
	}{
		{
			name:         "slack user gets platform-scoped placeholder email",
			platformName: "slack",
			flag:         "U01DMQDTV5W",
			wantEmail:    "U01DMQDTV5W@slack.local",
			wantAvatar:   "-",
		},
		{
			name:         "missing platform falls back to unknown domain",
			platformName: "",
			flag:         "user-1",
			wantEmail:    "user-1@unknown.local",
			wantAvatar:   "-",
		},
		{
			name:         "missing flag falls back to generic user id",
			platformName: "discord",
			flag:         "",
			wantEmail:    "user@discord.local",
			wantAvatar:   "-",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			email, avatarURL := platformUserProfileDefaults(tt.platformName, tt.flag)
			assert.Equal(t, tt.wantEmail, email)
			assert.Equal(t, tt.wantAvatar, avatarURL)
		})
	}
}

func TestRegisterPlatformUser(t *testing.T) {
	tests := []struct {
		name        string
		store       *registrationStore
		data        protocol.MessageEventData
		wantUID     string
		wantEmail   string
		wantErr     bool
		wantErrText string
		wantCreated bool
	}{
		{
			name: "new slack user gets placeholder profile fields",
			store: &registrationStore{
				platform: &gen.Platform{ID: 7, Name: "slack"},
			},
			data: protocol.MessageEventData{
				Self:   protocol.Self{Platform: "slack"},
				UserId: "U01DMQDTV5W",
			},
			wantEmail:   "U01DMQDTV5W@slack.local",
			wantCreated: true,
		},
		{
			name: "existing platform user returns linked user flag",
			store: &registrationStore{
				platform: &gen.Platform{ID: 7, Name: "slack"},
				platformUser: &gen.PlatformUser{
					ID:     11,
					UserID: 42,
					Flag:   "U01DMQDTV5W",
				},
				user: &gen.User{ID: 42, Flag: "existing-user"},
			},
			data: protocol.MessageEventData{
				Self:   protocol.Self{Platform: "slack"},
				UserId: "U01DMQDTV5W",
			},
			wantUID: "existing-user",
		},
		{
			name: "missing platform returns error",
			store: &registrationStore{
				platform: &gen.Platform{ID: 7, Name: "discord"},
			},
			data: protocol.MessageEventData{
				Self:   protocol.Self{Platform: "slack"},
				UserId: "U01DMQDTV5W",
			},
			wantErr: true,
		},
		{
			name: "platform user lookup error is propagated",
			store: &registrationStore{
				platform:        &gen.Platform{ID: 7, Name: "slack"},
				platformUserErr: errors.New("lookup failed"),
			},
			data: protocol.MessageEventData{
				Self:   protocol.Self{Platform: "slack"},
				UserId: "U01DMQDTV5W",
			},
			wantErr:     true,
			wantErrText: "lookup failed",
		},
		{
			name: "broken platform user link is repaired with new user id",
			store: &registrationStore{
				platform: &gen.Platform{ID: 7, Name: "slack"},
				platformUser: &gen.PlatformUser{
					ID:     11,
					UserID: 0,
					Flag:   "U01DMQDTV5W",
				},
			},
			data: protocol.MessageEventData{
				Self:   protocol.Self{Platform: "slack"},
				UserId: "U01DMQDTV5W",
			},
			wantCreated: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orig := store.Database
			store.Database = tt.store
			t.Cleanup(func() { store.Database = orig })

			uid, err := registerPlatformUser(tt.data)
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrText != "" {
					require.ErrorContains(t, err, tt.wantErrText)
				}
				return
			}
			require.NoError(t, err)
			if tt.wantUID != "" {
				assert.Equal(t, types.Uid(tt.wantUID), uid)
			} else {
				assert.NotEmpty(t, uid.String())
			}
			if tt.wantCreated {
				require.NotNil(t, tt.store.createPlatform)
				assert.Equal(t, tt.wantEmail, tt.store.createPlatform.Email)
				assert.Equal(t, "-", tt.store.createPlatform.AvatarURL)
				assert.Equal(t, int64(42), tt.store.createPlatform.UserID)
			} else {
				assert.Nil(t, tt.store.createPlatform)
			}
			if tt.name == "broken platform user link is repaired with new user id" {
				require.NotNil(t, tt.store.updatedPlatform)
				assert.Equal(t, int64(42), tt.store.updatedPlatform.UserID)
			}
		})
	}
}

type channelRegistrationStore struct {
	testStoreAdapter
	platform         *gen.Platform
	platformChannel  *gen.PlatformChannel
	channel          *gen.Channel
	createdChannel   *gen.Channel
	createdPlatform  *gen.PlatformChannel
	updatedChannelID int64
}

func (s *channelRegistrationStore) GetPlatformByName(_ context.Context, name string) (*gen.Platform, error) {
	if s.platform == nil || s.platform.Name != name {
		return nil, types.ErrNotFound
	}
	return s.platform, nil
}

func (s *channelRegistrationStore) GetPlatformChannelByFlag(_ context.Context, flag string) (*gen.PlatformChannel, error) {
	if s.platformChannel == nil || s.platformChannel.Flag != flag {
		return nil, types.ErrNotFound
	}
	return s.platformChannel, nil
}

func (s *channelRegistrationStore) GetChannel(_ context.Context, id int64) (*gen.Channel, error) {
	if s.channel == nil || s.channel.ID != id {
		return nil, types.ErrNotFound
	}
	return s.channel, nil
}

func (s *channelRegistrationStore) CreateChannel(_ context.Context, channel *gen.Channel) (int64, error) {
	s.createdChannel = channel
	channel.ID = 501
	return channel.ID, nil
}

func (s *channelRegistrationStore) CreatePlatformChannel(_ context.Context, item *gen.PlatformChannel) (int64, error) {
	s.createdPlatform = item
	return 77, nil
}

func (s *channelRegistrationStore) UpdatePlatformChannelChannelID(_ context.Context, platformChannelID, channelID int64) error {
	if s.platformChannel == nil || s.platformChannel.ID != platformChannelID {
		return types.ErrNotFound
	}
	s.updatedChannelID = channelID
	s.platformChannel.ChannelID = channelID
	return nil
}

func (*channelRegistrationStore) CreatePlatformChannelUser(context.Context, *gen.PlatformChannelUser) (int64, error) {
	return 1, nil
}

func TestRegisterPlatformChannel(t *testing.T) {
	tests := []struct {
		name              string
		store             *channelRegistrationStore
		data              protocol.MessageEventData
		wantErr           bool
		wantCreateChannel bool
		wantLinkChannelID int64
		wantRepair        bool
	}{
		{
			name: "new topic creates platform channel with persisted channel id",
			store: &channelRegistrationStore{
				platform: &gen.Platform{ID: 7, Name: "slack"},
			},
			data: protocol.MessageEventData{
				Self:    protocol.Self{Platform: "slack"},
				TopicId: "D01DMRLE0HW",
				UserId:  "U01DMQDTV5W",
			},
			wantCreateChannel: true,
			wantLinkChannelID: 501,
		},
		{
			name: "existing topic returns linked channel flag",
			store: &channelRegistrationStore{
				platform: &gen.Platform{ID: 7, Name: "slack"},
				platformChannel: &gen.PlatformChannel{
					ID:        12,
					ChannelID: 88,
					Flag:      "D01DMRLE0HW",
				},
				channel: &gen.Channel{ID: 88, Flag: "existing-channel"},
			},
			data: protocol.MessageEventData{
				Self:    protocol.Self{Platform: "slack"},
				TopicId: "D01DMRLE0HW",
				UserId:  "U01DMQDTV5W",
			},
			wantCreateChannel: false,
		},
		{
			name: "broken platform channel link is repaired with new channel id",
			store: &channelRegistrationStore{
				platform: &gen.Platform{ID: 7, Name: "slack"},
				platformChannel: &gen.PlatformChannel{
					ID:        12,
					ChannelID: 0,
					Flag:      "D01DMRLE0HW",
				},
			},
			data: protocol.MessageEventData{
				Self:    protocol.Self{Platform: "slack"},
				TopicId: "D01DMRLE0HW",
				UserId:  "U01DMQDTV5W",
			},
			wantCreateChannel: true,
			wantLinkChannelID: 501,
			wantRepair:        true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orig := store.Database
			store.Database = tt.store
			t.Cleanup(func() { store.Database = orig })

			flag, err := registerPlatformChannel(tt.data)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotEmpty(t, flag)
			if tt.wantCreateChannel {
				require.NotNil(t, tt.store.createdChannel)
				assert.Equal(t, int64(501), tt.store.createdChannel.ID)
			} else {
				assert.Nil(t, tt.store.createdChannel)
				assert.Equal(t, "existing-channel", flag)
			}
			if tt.wantRepair {
				assert.Equal(t, tt.wantLinkChannelID, tt.store.updatedChannelID)
			} else if tt.wantLinkChannelID > 0 {
				require.NotNil(t, tt.store.createdPlatform)
				assert.Equal(t, tt.wantLinkChannelID, tt.store.createdPlatform.ChannelID)
			}
		})
	}
}
