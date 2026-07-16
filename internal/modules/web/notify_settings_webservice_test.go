package web

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/types/model"
)

func TestNotifySettingsPageUnauthenticated(t *testing.T) {
	tests := []struct {
		name       string
		wantStatus int
	}{
		{name: "redirects to login without cookie", wantStatus: http.StatusSeeOther},
		{name: "requires authentication", wantStatus: http.StatusSeeOther},
		{name: "blocks unauthenticated access", wantStatus: http.StatusSeeOther},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := setupTestApp()
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()

			req := httptest.NewRequest(http.MethodGet, "/service/web/notify-settings", http.NoBody)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test: %v", err)
			}
			defer resp.Body.Close()

			if tt.wantStatus != resp.StatusCode {
				t.Errorf("want status %d, got %d", tt.wantStatus, resp.StatusCode)
			}

			body, _ := io.ReadAll(resp.Body)
			if len(body) > 0 && strings.Contains(string(body), "Notify Settings") {
				t.Error("unauthenticated request should not render notify settings page")
			}
		})
	}
}

func TestNotifyChannelTest(t *testing.T) {
	tests := []struct {
		name           string
		channelID      string
		channel        *model.NotifyChannel
		wantStatus     int
		wantHXContains string
	}{
		{
			name:       "missing channel returns not found",
			channelID:  "999",
			wantStatus: http.StatusOK,
		},
		{
			name:      "unknown protocol returns error toast",
			channelID: "1",
			channel: &model.NotifyChannel{
				ID:       1,
				Name:     "alerts",
				Protocol: "nosuchproto",
				URI:      "nosuchproto://token",
				Enabled:  true,
			},
			wantStatus:     http.StatusOK,
			wantHXContains: `"type":"error"`,
		},
		{
			name:      "relative URI builds scheme from protocol",
			channelID: "2",
			channel: &model.NotifyChannel{
				ID:       2,
				Name:     "relative",
				Protocol: "nosuchproto",
				URI:      "token-only",
				Enabled:  true,
			},
			wantStatus:     http.StatusOK,
			wantHXContains: `"type":"error"`,
		},
		{
			name:      "http URI with unknown protocol still errors on protocol not scheme",
			channelID: "3",
			channel: &model.NotifyChannel{
				ID:       3,
				Name:     "http-uri",
				Protocol: "nosuchproto",
				URI:      "http://ntfy.example.com/mytopic",
				Enabled:  true,
			},
			wantStatus:     http.StatusOK,
			wantHXContains: `unknown protocol`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, ts := setupTestApp()
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()
			if tt.channel != nil {
				ts.notifyChannels = map[int64]model.NotifyChannel{tt.channel.ID: *tt.channel}
			}

			req := httptest.NewRequest(http.MethodPost, "/service/web/notify-settings/channels/"+tt.channelID+"/test", http.NoBody)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "valid-token"})
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("want status %d, got %d", tt.wantStatus, resp.StatusCode)
			}
			if tt.wantHXContains != "" {
				hx := resp.Header.Get("HX-Trigger")
				if !strings.Contains(hx, tt.wantHXContains) {
					t.Errorf("want HX-Trigger containing %q, got %q", tt.wantHXContains, hx)
				}
			}
		})
	}
}
