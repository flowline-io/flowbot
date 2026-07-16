package module

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/postgres"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/types"
)

func withSQLiteStore(t *testing.T) {
	t.Helper()
	prev := store.Database
	store.Database = postgres.NewSQLiteTestAdapter(t)
	t.Cleanup(func() { store.Database = prev })
}

func TestSettingGetAndStoreParameter(t *testing.T) {
	withSQLiteStore(t)
	runCtx := types.Context{AsUser: types.Uid("user:setting"), Topic: "default"}
	runCtx.SetContext(context.Background())

	tests := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "setting get returns stored config",
			run: func(t *testing.T) {
				require.NoError(t, store.Database.ConfigSet(runCtx.Context(), runCtx.AsUser, runCtx.Topic, "widget_theme", types.KV{"color": "blue"}))
				got, err := SettingGet(runCtx, "widget", "theme")
				require.NoError(t, err)
				assert.Equal(t, "blue", got["color"])
			},
		},
		{
			name: "store parameter persists kv",
			run: func(t *testing.T) {
				flag, err := StoreParameter(types.KV{"a": "1"}, time.Now().Add(time.Hour))
				require.NoError(t, err)
				assert.NotEmpty(t, flag)
				row, getErr := store.Database.ParameterGet(context.Background(), flag)
				require.NoError(t, getErr)
				assert.Equal(t, "1", types.KV(row.Params)["a"])
			},
		},
		{
			name: "service url includes parameter flag",
			run: func(t *testing.T) {
				prev := config.App.Flowbot.URL
				config.App.Flowbot.URL = "https://flowbot.test"
				t.Cleanup(func() { config.App.Flowbot.URL = prev })

				url := ServiceURL(runCtx, "web", "/pipelines", types.KV{"run": "1"})
				assert.Contains(t, url, "https://flowbot.test/service/web/pipelines")
				assert.Contains(t, url, "p=")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.run)
	}
}

func TestInitInvalidConfig(t *testing.T) {
	tests := []struct {
		name    string
		conf    string
		wantErr bool
	}{
		{name: "invalid json", conf: `{`, wantErr: true},
		{name: "invalid module entry", conf: `[{"name":}]`, wantErr: true},
		{name: "empty array ok", conf: `[]`, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Init(json.RawMessage(tt.conf))
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestBehaviorViaModuleHelper(t *testing.T) {
	withSQLiteStore(t)
	uid := types.Uid("user:mod-behavior")

	tests := []struct {
		name  string
		flag  string
		count int
	}{
		{name: "creates behavior", flag: MessageBotIncomingBehavior, count: 1},
		{name: "increments behavior", flag: MessageBotIncomingBehavior, count: 2},
		{name: "separate flag", flag: MessageGroupIncomingBehavior, count: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, func() { Behavior(uid, tt.flag, tt.count) })
		})
	}

	got, err := store.Database.BehaviorGet(context.Background(), uid, MessageBotIncomingBehavior)
	require.NoError(t, err)
	assert.Equal(t, int32(3), got.Count)
}
