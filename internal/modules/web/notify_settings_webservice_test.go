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
