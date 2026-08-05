package server

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
)

func TestResolvePlatformUserFlag(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		data protocol.MessageEventData
		want string
	}{
		{
			name: "prefers top-level user id",
			data: protocol.MessageEventData{
				Self:   protocol.Self{UserId: "self-user"},
				UserId: "event-user",
			},
			want: "event-user",
		},
		{
			name: "falls back to self user id",
			data: protocol.MessageEventData{
				Self: protocol.Self{UserId: "self-user"},
			},
			want: "self-user",
		},
		{
			name: "generates id when both user ids are missing",
			data: protocol.MessageEventData{
				Self: protocol.Self{Platform: "slack"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := resolvePlatformUserFlag(tt.data)
			if tt.want != "" {
				assert.Equal(t, tt.want, got)
				return
			}
			assert.NotEmpty(t, got)
		})
	}
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
		name         string
		seed         func(t *testing.T) protocol.MessageEventData
		wantUID      string
		wantUIDNotIn []string
		wantEmail    string
		wantErr      bool
		wantCreated  bool
		checkRepair  bool
	}{
		{
			name: "new slack user gets placeholder profile fields",
			seed: func(t *testing.T) protocol.MessageEventData {
				seedTestPlatform(t, "slack")
				return protocol.MessageEventData{
					Self:   protocol.Self{Platform: "slack"},
					UserId: "U01DMQDTV5W",
				}
			},
			wantEmail:   "U01DMQDTV5W@slack.local",
			wantCreated: true,
		},
		{
			name: "new slack user attaches to sole web account",
			seed: func(t *testing.T) protocol.MessageEventData {
				seedTestPlatform(t, "slack")
				seedSoleWebAccount(t, "admin")
				return protocol.MessageEventData{
					Self:   protocol.Self{Platform: "slack"},
					UserId: "U01DMQDTV5W",
				}
			},
			wantUID:     "user-admin",
			wantEmail:   "U01DMQDTV5W@slack.local",
			wantCreated: true,
		},
		{
			name: "orphan platform user relinks to sole web account",
			seed: func(t *testing.T) protocol.MessageEventData {
				platformID := seedTestPlatform(t, "slack")
				seedSoleWebAccount(t, "admin")
				orphan := seedTestUser(t, "orphan-slack-user")
				seedTestPlatformUser(t, platformID, orphan.ID, "U01DMQDTV5W", "", "")
				return protocol.MessageEventData{
					Self:   protocol.Self{Platform: "slack"},
					UserId: "U01DMQDTV5W",
				}
			},
			wantUID: "user-admin",
		},
		{
			name: "new slack user stays independent with multiple web accounts",
			seed: func(t *testing.T) protocol.MessageEventData {
				seedTestPlatform(t, "slack")
				seedSoleWebAccount(t, "admin")
				seedExtraWebAccount(t, "other")
				return protocol.MessageEventData{
					Self:   protocol.Self{Platform: "slack"},
					UserId: "U01DMQDTV5W",
				}
			},
			wantUIDNotIn: []string{"user-admin", "user-other"},
			wantCreated:  true,
		},
		{
			name: "orphan does not relink with multiple web accounts",
			seed: func(t *testing.T) protocol.MessageEventData {
				platformID := seedTestPlatform(t, "slack")
				seedSoleWebAccount(t, "admin")
				seedExtraWebAccount(t, "other")
				orphan := seedTestUser(t, "orphan-slack-user")
				seedTestPlatformUser(t, platformID, orphan.ID, "U01DMQDTV5W", "", "")
				return protocol.MessageEventData{
					Self:   protocol.Self{Platform: "slack"},
					UserId: "U01DMQDTV5W",
				}
			},
			wantUID: "orphan-slack-user",
		},
		{
			name: "existing platform user returns linked user flag",
			seed: func(t *testing.T) protocol.MessageEventData {
				platformID := seedTestPlatform(t, "slack")
				user := seedTestUser(t, "existing-user")
				seedTestPlatformUser(t, platformID, user.ID, "U01DMQDTV5W", "", "")
				return protocol.MessageEventData{
					Self:   protocol.Self{Platform: "slack"},
					UserId: "U01DMQDTV5W",
				}
			},
			wantUID: "existing-user",
		},
		{
			name: "missing platform returns error",
			seed: func(t *testing.T) protocol.MessageEventData {
				seedTestPlatform(t, "discord")
				return protocol.MessageEventData{
					Self:   protocol.Self{Platform: "slack"},
					UserId: "U01DMQDTV5W",
				}
			},
			wantErr: true,
		},
		{
			name: "broken platform user link is repaired with new user id",
			seed: func(t *testing.T) protocol.MessageEventData {
				platformID := seedTestPlatform(t, "slack")
				user := seedTestUser(t, "orphan-user")
				seedTestPlatformUser(t, platformID, user.ID, "U01DMQDTV5W", "", "")
				require.NoError(t, store.UserStoreFromDB().UserDelete(context.Background(), types.Uid(user.Flag), true))
				return protocol.MessageEventData{
					Self:   protocol.Self{Platform: "slack"},
					UserId: "U01DMQDTV5W",
				}
			},
			checkRepair: true,
		},
		{
			name: "broken platform user link repairs onto sole web account",
			seed: func(t *testing.T) protocol.MessageEventData {
				platformID := seedTestPlatform(t, "slack")
				seedSoleWebAccount(t, "admin")
				user := seedTestUser(t, "orphan-user")
				seedTestPlatformUser(t, platformID, user.ID, "U01DMQDTV5W", "", "")
				require.NoError(t, store.UserStoreFromDB().UserDelete(context.Background(), types.Uid(user.Flag), true))
				return protocol.MessageEventData{
					Self:   protocol.Self{Platform: "slack"},
					UserId: "U01DMQDTV5W",
				}
			},
			wantUID:     "user-admin",
			checkRepair: true,
		},
		{
			name: "missing user id gets generated platform flag",
			seed: func(t *testing.T) protocol.MessageEventData {
				seedTestPlatform(t, "slack")
				return protocol.MessageEventData{
					Self: protocol.Self{Platform: "slack"},
				}
			},
			wantCreated: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupSQLiteTestDB(t)
			data := tt.seed(t)
			platform, err := store.PlatformStoreFromDB().GetPlatformByName(context.Background(), data.Self.Platform)
			if tt.wantErr && err != nil {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			uid, err := registerPlatformUser(data, platform)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.wantUID != "" {
				assert.Equal(t, types.Uid(tt.wantUID), uid)
			} else {
				assert.NotEmpty(t, uid.String())
			}
			for _, blocked := range tt.wantUIDNotIn {
				assert.NotEqual(t, types.Uid(blocked), uid)
			}
			if tt.wantCreated {
				user, err := store.UserStoreFromDB().UserGet(context.Background(), uid)
				require.NoError(t, err)
				pus, err := store.UserStoreFromDB().GetPlatformUsersByUserId(context.Background(), user.ID)
				require.NoError(t, err)
				require.Len(t, pus, 1)
				pu := pus[0]
				assert.NotEmpty(t, pu.Flag)
				if tt.wantEmail != "" {
					assert.Equal(t, tt.wantEmail, pu.Email)
				}
				assert.Equal(t, "-", pu.AvatarURL)
				assert.NotZero(t, pu.UserID)
			}
			if tt.checkRepair {
				pu, err := store.UserStoreFromDB().GetPlatformUserByFlag(context.Background(), "U01DMQDTV5W")
				require.NoError(t, err)
				assert.NotZero(t, pu.UserID)
				user, err := store.UserStoreFromDB().GetUserById(context.Background(), pu.UserID)
				require.NoError(t, err)
				assert.NotEmpty(t, user.Flag)
			}
		})
	}
}

func TestRegisterPlatformChannel(t *testing.T) {
	tests := []struct {
		name              string
		seed              func(t *testing.T) protocol.MessageEventData
		wantErr           bool
		wantCreateChannel bool
		wantExistingFlag  string
		wantRepair        bool
	}{
		{
			name: "new topic creates platform channel with persisted channel id",
			seed: func(t *testing.T) protocol.MessageEventData {
				seedTestPlatform(t, "slack")
				return protocol.MessageEventData{
					Self:    protocol.Self{Platform: "slack"},
					TopicId: "D01DMRLE0HW",
					UserId:  "U01DMQDTV5W",
				}
			},
			wantCreateChannel: true,
		},
		{
			name: "existing topic returns linked channel flag",
			seed: func(t *testing.T) protocol.MessageEventData {
				platformID := seedTestPlatform(t, "slack")
				channelID := seedTestChannel(t, "existing-channel")
				seedTestPlatformChannel(t, platformID, channelID, "D01DMRLE0HW")
				return protocol.MessageEventData{
					Self:    protocol.Self{Platform: "slack"},
					TopicId: "D01DMRLE0HW",
					UserId:  "U01DMQDTV5W",
				}
			},
			wantExistingFlag: "existing-channel",
		},
		{
			name: "broken platform channel link is repaired with new channel id",
			seed: func(t *testing.T) protocol.MessageEventData {
				platformID := seedTestPlatform(t, "slack")
				now := time.Now()
				_, err := store.PlatformStoreFromDB().CreatePlatformChannel(context.Background(), &gen.PlatformChannel{
					PlatformID: platformID,
					ChannelID:  0,
					Flag:       "D01DMRLE0HW",
					CreatedAt:  now,
					UpdatedAt:  now,
				})
				require.NoError(t, err)
				return protocol.MessageEventData{
					Self:    protocol.Self{Platform: "slack"},
					TopicId: "D01DMRLE0HW",
					UserId:  "U01DMQDTV5W",
				}
			},
			wantCreateChannel: true,
			wantRepair:        true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupSQLiteTestDB(t)
			data := tt.seed(t)
			platform, err := store.PlatformStoreFromDB().GetPlatformByName(context.Background(), data.Self.Platform)
			require.NoError(t, err)

			flag, err := registerPlatformChannel(data, platform)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotEmpty(t, flag)
			if tt.wantExistingFlag != "" {
				assert.Equal(t, tt.wantExistingFlag, flag)
				return
			}
			pc, err := store.PlatformStoreFromDB().GetPlatformChannelByFlag(context.Background(), data.TopicId)
			require.NoError(t, err)
			if tt.wantCreateChannel {
				assert.NotZero(t, pc.ChannelID)
				ch, err := store.PlatformStoreFromDB().GetChannel(context.Background(), pc.ChannelID)
				require.NoError(t, err)
				assert.Equal(t, flag, ch.Flag)
			}
			if tt.wantRepair {
				assert.NotZero(t, pc.ChannelID)
			}
		})
	}
}
