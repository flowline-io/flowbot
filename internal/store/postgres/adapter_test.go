package postgres

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store/model"
)

func TestGetPlatformChannelUsersByUserFlags_Empty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		userFlags []string
	}{
		{
			name:      "nil input returns nil slice",
			userFlags: nil,
		},
		{
			name:      "empty input returns nil slice",
			userFlags: []string{},
		},
		{
			name:      "single flag requires real database",
			userFlags: []string{"flag1"},
		},
	}

	adpt := &adapter{}
	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if len(tt.userFlags) > 0 {
				t.Skip("requires a running database and ent client")
			}
			result, err := adpt.GetPlatformChannelUsersByUserFlags(ctx, tt.userFlags)
			require.NoError(t, err)
			assert.Nil(t, result)
		})
	}
}

func TestPlatformChannelUserModelProperties(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		userFlag    string
		channelFlag string
		platformID  int64
	}{
		{
			name:        "standard platform channel user",
			userFlag:    "user:slack:U123",
			channelFlag: "ch:general",
			platformID:  1,
		},
		{
			name:        "user with empty flags",
			userFlag:    "",
			channelFlag: "",
			platformID:  0,
		},
		{
			name:        "user with special characters",
			userFlag:    "user:discord:123456789",
			channelFlag: "ch:bot-log-extra",
			platformID:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pcu := &model.PlatformChannelUser{
				UserFlag:    tt.userFlag,
				ChannelFlag: tt.channelFlag,
				PlatformID:  tt.platformID,
			}
			assert.Equal(t, tt.userFlag, pcu.UserFlag)
			assert.Equal(t, tt.channelFlag, pcu.ChannelFlag)
			assert.Equal(t, tt.platformID, pcu.PlatformID)
		})
	}
}
