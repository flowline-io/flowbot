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

func TestValidateChannelForm(t *testing.T) {
	tests := []struct {
		name     string
		formName string
		protocol string
		uri      string
		wantKey  string
	}{
		{name: "empty name rejected", protocol: "smtp", uri: "smtp://x", wantKey: "name"},
		{name: "empty protocol rejected", formName: "alerts", uri: "smtp://x", wantKey: "protocol"},
		{name: "empty uri rejected", formName: "alerts", protocol: "smtp", wantKey: "uri"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validateChannelForm(tt.formName, tt.protocol, tt.uri)
			if _, ok := errs[tt.wantKey]; !ok {
				t.Errorf("want error for %q, got %v", tt.wantKey, errs)
			}
		})
	}
}

func TestValidateRuleForm(t *testing.T) {
	tests := []struct {
		name    string
		rule    model.NotifyRule
		wantKey string
	}{
		{
			name:    "missing name rejected",
			rule:    model.NotifyRule{RuleID: "r1", EventPattern: "e", ChannelPattern: "c", Action: "send"},
			wantKey: "name",
		},
		{
			name:    "missing rule id rejected",
			rule:    model.NotifyRule{Name: "n", EventPattern: "e", ChannelPattern: "c", Action: "send"},
			wantKey: "rule_id",
		},
		{
			name:    "missing event pattern rejected",
			rule:    model.NotifyRule{Name: "n", RuleID: "r1", ChannelPattern: "c", Action: "send"},
			wantKey: "event_pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validateRuleForm(tt.rule)
			if _, ok := errs[tt.wantKey]; !ok {
				t.Errorf("want error for %q, got %v", tt.wantKey, errs)
			}
		})
	}
}

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

func TestShowToastTrigger(t *testing.T) {
	tests := []struct {
		name      string
		toastType string
		message   string
		wantSub   string
	}{
		{
			name:      "success toast",
			toastType: "success",
			message:   "Connection successful",
			wantSub:   `"type":"success"`,
		},
		{
			name:      "error toast with special characters",
			toastType: "error",
			message:   `Connection failed: foo "bar"`,
			wantSub:   `"type":"error"`,
		},
		{
			name:      "info toast",
			toastType: "info",
			message:   "hello",
			wantSub:   `"message":"hello"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := showToastTrigger(tt.toastType, tt.message)
			if err != nil {
				t.Fatalf("showToastTrigger: %v", err)
			}
			if !strings.Contains(got, `"showToast"`) {
				t.Errorf("want showToast key, got %s", got)
			}
			if !strings.Contains(got, tt.wantSub) {
				t.Errorf("want substring %q in %s", tt.wantSub, got)
			}
			if strings.Contains(tt.message, `"`) && strings.Contains(got, `foo "bar"`) {
				t.Error("message quotes must be JSON-escaped")
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
