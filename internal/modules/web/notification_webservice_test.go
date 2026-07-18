package web

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/notificationrecord"
	notifypkg "github.com/flowline-io/flowbot/pkg/notify"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsConnectivityTestTemplate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		templateID string
		want       bool
	}{
		{name: "connectivity test id", templateID: "test", want: true},
		{name: "constant matches", templateID: notifypkg.ConnectivityTestTemplateID, want: true},
		{name: "real template", templateID: "bookmark.created", want: false},
		{name: "empty", templateID: "", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, notifypkg.IsConnectivityTestTemplate(tt.templateID))
		})
	}
}

func TestLookupNotifyChannelRawByName(t *testing.T) {
	tests := []struct {
		name       string
		channels   map[int64]model.NotifyChannel
		lookup     string
		wantURI    string
		wantErrSub string
	}{
		{
			name: "finds channel by name",
			channels: map[int64]model.NotifyChannel{
				1: {ID: 1, Name: "testing", Protocol: "slack", URI: "slack://T00/B00/xxx", Enabled: true},
			},
			lookup:  "testing",
			wantURI: "slack://T00/B00/xxx",
		},
		{
			name: "missing channel",
			channels: map[int64]model.NotifyChannel{
				1: {ID: 1, Name: "alerts", Protocol: "slack", URI: "slack://T00/B00/xxx", Enabled: true},
			},
			lookup:     "testing",
			wantErrSub: "not found",
		},
		{
			name:       "empty store",
			channels:   nil,
			lookup:     "testing",
			wantErrSub: "not found",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := &testStore{notifyChannels: tt.channels}
			store.Database = ts
			t.Cleanup(func() { store.Database = nil })

			got, err := lookupNotifyChannelRawByName(context.Background(), tt.lookup)
			if tt.wantErrSub != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrSub)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantURI, got.URI)
		})
	}
}

type retryMockNotifyer struct {
	protocol  string
	templates []string
	sent      int
	fail      bool
}

func (m *retryMockNotifyer) Protocol() string    { return m.protocol }
func (m *retryMockNotifyer) Templates() []string { return m.templates }
func (m *retryMockNotifyer) Send(_ types.KV, _ notifypkg.Message) error {
	m.sent++
	if m.fail {
		return assert.AnError
	}
	return nil
}

func TestRetryNotificationConnectivityTest(t *testing.T) {
	tests := []struct {
		name           string
		channel        model.NotifyChannel
		register       *retryMockNotifyer
		wantHXContains string
		wantSent       int
		wantBodyNot    string
	}{
		{
			name: "retries connectivity test via channel probe",
			channel: model.NotifyChannel{
				ID: 1, Name: "testing", Protocol: "retryslack",
				URI: "retryslack://T00/B00/xxx", Enabled: true,
			},
			register: &retryMockNotifyer{
				protocol:  "retryslack",
				templates: []string{"retryslack://{tokenA}/{tokenB}/{tokenC}"},
			},
			wantHXContains: `"type":"success"`,
			wantSent:       1,
			wantBodyNot:    "template test not found",
		},
		{
			name: "missing channel returns toast error",
			channel: model.NotifyChannel{
				ID: 1, Name: "other", Protocol: "retryslack",
				URI: "retryslack://T00/B00/xxx", Enabled: true,
			},
			register: &retryMockNotifyer{
				protocol:  "retryslack",
				templates: []string{"retryslack://{tokenA}/{tokenB}/{tokenC}"},
			},
			wantHXContains: "not found",
			wantSent:       0,
			wantBodyNot:    "template test not found",
		},
		{
			name: "send failure records error toast",
			channel: model.NotifyChannel{
				ID: 1, Name: "testing", Protocol: "retryslackfail",
				URI: "retryslackfail://T00/B00/xxx", Enabled: true,
			},
			register: &retryMockNotifyer{
				protocol:  "retryslackfail",
				templates: []string{"retryslackfail://{tokenA}/{tokenB}/{tokenC}"},
				fail:      true,
			},
			wantHXContains: `"type":"error"`,
			wantSent:       1,
			wantBodyNot:    "template test not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, ts, client := setupTestAppWithDB(t)
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()

			ts.notifyChannels = map[int64]model.NotifyChannel{tt.channel.ID: tt.channel}
			notifypkg.Register(tt.register.protocol, tt.register)
			t.Cleanup(func() { notifypkg.Unregister(tt.register.protocol) })

			ns := store.NewNotifyStore(client)
			recID, err := ns.Record(context.Background(), "testuser", "testing", notifypkg.ConnectivityTestTemplateID, "Test connectivity", string(notificationrecord.StatusFailed), "previous error", nil)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/service/web/notifications/"+strconv.FormatInt(recID, 10)+"/retry", http.NoBody)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "valid-token"})
			AttachCSRFForTest(req)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			body, _ := io.ReadAll(resp.Body)
			assert.NotContains(t, string(body), tt.wantBodyNot)
			hx := resp.Header.Get("HX-Trigger")
			assert.Contains(t, hx, tt.wantHXContains)
			assert.Equal(t, tt.wantSent, tt.register.sent)
			assert.Contains(t, string(body), "notifications-table")
		})
	}
}
