package notify

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/postgres"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
)

func seedNotifyTestChannel(t *testing.T, name, protocol, uri string, enabled, isDefault bool) int64 {
	t.Helper()
	ctx := context.Background()
	id, err := store.NotifyConfigStoreFromDB().CreateNotifyChannel(ctx, name, protocol, uri)
	require.NoError(t, err)
	if !enabled {
		require.NoError(t, store.NotifyConfigStoreFromDB().UpdateNotifyChannel(ctx, id, name, protocol, uri, false))
	}
	if isDefault && enabled {
		require.NoError(t, store.NotifyConfigStoreFromDB().SetDefaultNotifyChannel(ctx, id))
	}
	return id
}

func seedNotifyTestTemplate(t *testing.T, tmpl model.NotifyTemplate, isDefault bool) int64 {
	t.Helper()
	ctx := context.Background()
	if tmpl.Name == "" {
		tmpl.Name = tmpl.TemplateID
	}
	if tmpl.Name == "" {
		tmpl.Name = "test-template"
	}
	if tmpl.DefaultFormat == "" {
		tmpl.DefaultFormat = "markdown"
	}
	if tmpl.DefaultTemplate == "" {
		tmpl.DefaultTemplate = "{{ .message }}"
	}
	if tmpl.OverridesJSON == "" {
		tmpl.OverridesJSON = "[]"
	}
	id, err := store.NotifyConfigStoreFromDB().CreateNotifyTemplate(ctx, tmpl)
	require.NoError(t, err)
	if isDefault {
		require.NoError(t, store.NotifyConfigStoreFromDB().SetDefaultNotifyTemplate(ctx, id))
	}
	return id
}

func seedUserNotifyConfig(t *testing.T, uid types.Uid, channel, uri string) {
	t.Helper()
	require.NoError(t, store.ModuleDataStoreFromDB().ConfigSet(
		context.Background(),
		uid,
		"",
		notifyConfigKeyPrefix+channel,
		types.KV{"value": uri},
	))
}

func setupNotifySQLiteDB(t *testing.T) {
	t.Helper()
	replaceDatabaseForTest(t, postgres.NewSQLiteTestAdapter(t))
}
