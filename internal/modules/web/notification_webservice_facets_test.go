package web

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/types/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// authOnlyNotifyAdapter mirrors BDD notifyWebAdapter: cookie auth via ParameterGet/GetDB.
// History facet loading requires ListNotifyChannels/ListNotifyRules; a nil embedded
// Adapter panics without them (CI mapped that panic to HTTP 400).
type authOnlyNotifyAdapter struct {
	store.Adapter
	client *store.Client
	uid    string
}

func (a *authOnlyNotifyAdapter) GetDB() any { return a.client }

func (a *authOnlyNotifyAdapter) ParameterGet(_ context.Context, flag string) (gen.Parameter, error) {
	return gen.Parameter{
		ID:   1,
		Flag: flag,
		Params: map[string]any{
			"uid":    a.uid,
			"topic":  "test",
			"scopes": []string{"read", "write"},
		},
		ExpiredAt: time.Now().Add(time.Hour),
	}, nil
}

func (*authOnlyNotifyAdapter) ListNotifyChannels(_ context.Context, _ store.ListNotifyChannelOptions) ([]model.NotifyChannel, error) {
	return nil, nil
}

func (*authOnlyNotifyAdapter) ListNotifyRules(_ context.Context, _ store.ListNotifyRuleOptions) ([]model.NotifyRule, error) {
	return nil, nil
}

func TestNotificationsTableWithAuthOnlyListAdapter(t *testing.T) {
	app, _, client := setupTestAppWithDB(t)
	defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()

	store.Database = &authOnlyNotifyAdapter{client: client, uid: "testuser"}

	ns := store.NewNotifyStore(client)
	_, err := ns.Record(context.Background(), "testuser", "slack", "tpl", "hello", "success", "", "", nil)
	require.NoError(t, err)

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantBody   string
	}{
		{name: "list returns 200", path: "/service/web/notifications/list", wantStatus: http.StatusOK, wantBody: "hello"},
		{name: "list empty facets still renders table", path: "/service/web/notifications/list", wantStatus: http.StatusOK, wantBody: "notifications-table"},
		{name: "list unread group", path: "/service/web/notifications/list?group=unread", wantStatus: http.StatusOK, wantBody: "notifications-table"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, http.NoBody)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "valid-token"})
			AttachCSRFForTest(req)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			assert.Contains(t, string(body), tt.wantBody)
		})
	}
}
