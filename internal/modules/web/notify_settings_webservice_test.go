package web

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
	notifyrules "github.com/flowline-io/flowbot/pkg/notify/rules"
	notifytmpl "github.com/flowline-io/flowbot/pkg/notify/template"
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

			req := httptest.NewRequest(http.MethodGet, "/service/web/notifications", http.NoBody)
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

func TestNormalizeNotifySettingsTab(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		tab  string
		want string
	}{
		{name: "empty defaults to channels", tab: "", want: "channels"},
		{name: "history tab", tab: "history", want: "history"},
		{name: "playground tab", tab: "playground", want: "playground"},
		{name: "legacy notifications maps to history", tab: "notifications", want: "history"},
		{name: "unknown falls back to channels", tab: "nope", want: "channels"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := normalizeNotifySettingsTab(tt.tab); got != tt.want {
				t.Errorf("normalizeNotifySettingsTab(%q)=%q, want %q", tt.tab, got, tt.want)
			}
		})
	}
}

func TestNotificationsPageRenders(t *testing.T) {
	tests := []struct {
		name       string
		authed     bool
		query      string
		wantStatus int
		wantSub    string
	}{
		{
			name:       "authenticated renders notifications page",
			authed:     true,
			wantStatus: http.StatusOK,
			wantSub:    "Notifications",
		},
		{
			name:       "history tab selected",
			authed:     true,
			query:      "?tab=history",
			wantStatus: http.StatusOK,
			wantSub:    "tab-history",
		},
		{
			name:       "unauthenticated redirects to login",
			authed:     false,
			wantStatus: http.StatusSeeOther,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := setupTestApp()
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()

			req := httptest.NewRequest(http.MethodGet, "/service/web/notifications"+tt.query, http.NoBody)
			if tt.authed {
				req.AddCookie(&http.Cookie{Name: "accessToken", Value: "valid-token"})
			}
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("want status %d, got %d", tt.wantStatus, resp.StatusCode)
			}
			if !tt.authed {
				if !strings.Contains(resp.Header.Get("Location"), "/service/web/login") {
					t.Errorf("want login redirect, got %q", resp.Header.Get("Location"))
				}
				return
			}
			body, _ := io.ReadAll(resp.Body)
			if tt.wantSub != "" && !strings.Contains(string(body), tt.wantSub) {
				t.Errorf("want body containing %q, got %q", tt.wantSub, string(body))
			}
		})
	}
}

func TestNotifyChannelCreate(t *testing.T) {
	tests := []struct {
		name       string
		form       url.Values
		wantStatus int
		wantURI    string
		wantErrSub string
	}{
		{
			name: "creates channel with slack uri",
			form: url.Values{
				"name":     {"alerts"},
				"protocol": {"slack"},
				"uri":      {"slack://T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX"},
			},
			wantStatus: http.StatusOK,
			wantURI:    "slack://T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX",
		},
		{
			name: "rejects missing uri",
			form: url.Values{
				"name":     {"alerts"},
				"protocol": {"slack"},
				"uri":      {""},
			},
			wantStatus: http.StatusOK,
			wantErrSub: "URI is required",
		},
		{
			name: "rejects missing name",
			form: url.Values{
				"name":     {""},
				"protocol": {"slack"},
				"uri":      {"slack://T00/B00/xxx"},
			},
			wantStatus: http.StatusOK,
			wantErrSub: "Name is required",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, ts := setupTestApp()
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()

			req := httptest.NewRequest(http.MethodPost, "/service/web/notifications/channels", strings.NewReader(tt.form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "valid-token"})
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("want status %d, got %d", tt.wantStatus, resp.StatusCode)
			}
			body, _ := io.ReadAll(resp.Body)
			if tt.wantErrSub != "" {
				if !strings.Contains(string(body), tt.wantErrSub) {
					t.Errorf("want body containing %q, got %q", tt.wantErrSub, string(body))
				}
				return
			}
			if len(ts.notifyChannels) != 1 {
				t.Fatalf("want 1 channel stored, got %d", len(ts.notifyChannels))
			}
			for _, ch := range ts.notifyChannels {
				if ch.URI != tt.wantURI {
					t.Errorf("want uri %q, got %q", tt.wantURI, ch.URI)
				}
			}
		})
	}
}

func TestNotifyChannelUpdate(t *testing.T) {
	tests := []struct {
		name       string
		form       url.Values
		wantStatus int
		wantURI    string
		wantErrSub string
	}{
		{
			name: "accepts filled slack uri on update",
			form: url.Values{
				"name":     {"alerts"},
				"protocol": {"slack"},
				"uri":      {"slack://T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX"},
				"enabled":  {"on"},
			},
			wantStatus: http.StatusOK,
			wantURI:    "slack://T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX",
		},
		{
			name: "empty uri keeps existing secret",
			form: url.Values{
				"name":     {"alerts"},
				"protocol": {"slack"},
				"uri":      {""},
				"enabled":  {"on"},
			},
			wantStatus: http.StatusOK,
			wantURI:    "slack://KEEP/EXISTING/SECRET",
		},
		{
			name: "rejects missing name",
			form: url.Values{
				"name":     {""},
				"protocol": {"slack"},
				"uri":      {"slack://T00/B00/xxx"},
			},
			wantStatus: http.StatusOK,
			wantErrSub: "Name is required",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, ts := setupTestApp()
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()
			ts.notifyChannels = map[int64]model.NotifyChannel{
				1: {
					ID:       1,
					Name:     "alerts",
					Protocol: "slack",
					URI:      "slack://KEEP/EXISTING/SECRET",
					Enabled:  true,
				},
			}

			req := httptest.NewRequest(http.MethodPut, "/service/web/notifications/channels/1", strings.NewReader(tt.form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "valid-token"})
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("want status %d, got %d", tt.wantStatus, resp.StatusCode)
			}
			body, _ := io.ReadAll(resp.Body)
			if tt.wantErrSub != "" {
				if !strings.Contains(string(body), tt.wantErrSub) {
					t.Errorf("want body containing %q, got %q", tt.wantErrSub, string(body))
				}
				return
			}
			if strings.Contains(string(body), "URI is required") {
				t.Fatalf("update unexpectedly rejected URI: %s", string(body))
			}
			ch := ts.notifyChannels[1]
			if ch.URI != tt.wantURI {
				t.Errorf("want uri %q, got %q", tt.wantURI, ch.URI)
			}
		})
	}
}

func TestNotifyTemplatesTable(t *testing.T) {
	tests := []struct {
		name       string
		templates  map[int64]model.NotifyTemplate
		wantStatus int
		wantSub    string
		wantAbsent string
	}{
		{
			name: "renders loaded templates with edit controls",
			templates: map[int64]model.NotifyTemplate{
				1: {ID: 1, TemplateID: "bookmark.created", Name: "New bookmark", Description: "on create", DefaultFormat: "markdown", DefaultTemplate: "**hi**", OverridesJSON: "[]"},
			},
			wantStatus: http.StatusOK,
			wantSub:    "data-testid=\"template-edit\"",
		},
		{
			name:       "empty templates shows placeholder",
			templates:  map[int64]model.NotifyTemplate{},
			wantStatus: http.StatusOK,
			wantSub:    "No notification templates",
		},
		{
			name: "exposes delete controls",
			templates: map[int64]model.NotifyTemplate{
				1: {ID: 1, TemplateID: "agent.status", Name: "Agent", DefaultFormat: "markdown", DefaultTemplate: "{{ .status }}", OverridesJSON: "[]"},
			},
			wantStatus: http.StatusOK,
			wantSub:    "data-testid=\"template-delete\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, ts := setupTestApp()
			defer func() {
				store.Database = nil
				handler = moduleHandler{}
				config = configType{}
			}()
			ts.notifyTemplates = tt.templates

			req := httptest.NewRequest(http.MethodGet, "/service/web/notifications/templates/list", http.NoBody)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "valid-token"})
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("want status %d, got %d", tt.wantStatus, resp.StatusCode)
			}
			body, _ := io.ReadAll(resp.Body)
			if !strings.Contains(string(body), tt.wantSub) {
				t.Errorf("want body containing %q, got %q", tt.wantSub, string(body))
			}
			if tt.wantAbsent != "" && strings.Contains(string(body), tt.wantAbsent) {
				t.Errorf("body should not contain %q", tt.wantAbsent)
			}
		})
	}
}

func TestNotifyRulesTable(t *testing.T) {
	tests := []struct {
		name       string
		rules      map[int64]model.NotifyRule
		wantStatus int
		wantSub    string
		wantAbsent string
	}{
		{
			name: "renders rule fields with edit controls",
			rules: map[int64]model.NotifyRule{
				1: {ID: 1, RuleID: "night_mute", Name: "Night mute", Action: "mute", EventPattern: "*", ChannelPattern: "*", Priority: 100, Enabled: true},
			},
			wantStatus: http.StatusOK,
			wantSub:    "data-testid=\"rule-edit\"",
		},
		{
			name:       "empty rules shows placeholder",
			rules:      map[int64]model.NotifyRule{},
			wantStatus: http.StatusOK,
			wantSub:    "No notification rules",
		},
		{
			name: "exposes delete controls",
			rules: map[int64]model.NotifyRule{
				1: {ID: 1, RuleID: "drop_test", Name: "Drop", Action: "drop", EventPattern: "test.*", ChannelPattern: "*", Priority: 1, Enabled: false},
			},
			wantStatus: http.StatusOK,
			wantSub:    "data-testid=\"rule-delete\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, ts := setupTestApp()
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()
			ts.notifyRules = tt.rules

			req := httptest.NewRequest(http.MethodGet, "/service/web/notifications/rules/list", http.NoBody)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "valid-token"})
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("want status %d, got %d", tt.wantStatus, resp.StatusCode)
			}
			body, _ := io.ReadAll(resp.Body)
			if !strings.Contains(string(body), tt.wantSub) {
				t.Errorf("want body containing %q, got %q", tt.wantSub, string(body))
			}
			if tt.wantAbsent != "" && strings.Contains(string(body), tt.wantAbsent) {
				t.Errorf("body should not contain %q", tt.wantAbsent)
			}
		})
	}
}

func TestNotifyTemplateCreate(t *testing.T) {
	tests := []struct {
		name       string
		form       url.Values
		wantStatus int
		wantSub    string
		wantOOB    bool
	}{
		{
			name: "creates template and clears empty state",
			form: url.Values{
				"template_id":      {"bookmark.created"},
				"name":             {"Bookmark"},
				"default_format":   {"markdown"},
				"default_template": {"**hi** {{ .url }}"},
				"overrides_json":   {"[]"},
			},
			wantStatus: http.StatusOK,
			wantSub:    "bookmark.created",
			wantOOB:    true,
		},
		{
			name: "rejects missing template id",
			form: url.Values{
				"name":             {"Bookmark"},
				"default_format":   {"markdown"},
				"default_template": {"hi"},
				"overrides_json":   {"[]"},
			},
			wantStatus: http.StatusOK,
			wantSub:    "Template ID is required",
		},
		{
			name: "rejects invalid overrides json",
			form: url.Values{
				"template_id":      {"x"},
				"name":             {"X"},
				"default_format":   {"markdown"},
				"default_template": {"hi"},
				"overrides_json":   {"{bad"},
			},
			wantStatus: http.StatusOK,
			wantSub:    "Invalid JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, ts := setupTestApp()
			defer func() {
				store.Database = nil
				handler = moduleHandler{}
				config = configType{}
				notifytmpl.ResetForTest()
			}()

			req := httptest.NewRequest(http.MethodPost, "/service/web/notifications/templates", strings.NewReader(tt.form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "valid-token"})
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("want status %d, got %d", tt.wantStatus, resp.StatusCode)
			}
			body, _ := io.ReadAll(resp.Body)
			bodyStr := string(body)
			if !strings.Contains(bodyStr, tt.wantSub) {
				t.Errorf("want body containing %q, got %q", tt.wantSub, bodyStr)
			}
			if tt.wantOOB {
				if !strings.Contains(bodyStr, `hx-swap-oob="delete"`) {
					t.Errorf("want empty-state oob delete, got %q", bodyStr)
				}
				if len(ts.notifyTemplates) != 1 {
					t.Errorf("want 1 template stored, got %d", len(ts.notifyTemplates))
				}
			}
		})
	}
}

func TestNotifyRuleCreate(t *testing.T) {
	tests := []struct {
		name       string
		form       url.Values
		wantStatus int
		wantSub    string
		wantStored bool
	}{
		{
			name: "creates enabled rule",
			form: url.Values{
				"rule_id":         {"night_mute"},
				"name":            {"Night mute"},
				"action":          {"drop"},
				"event_pattern":   {"*"},
				"channel_pattern": {"*"},
				"priority":        {"10"},
				"enabled":         {"on"},
			},
			wantStatus: http.StatusOK,
			wantSub:    "night_mute",
			wantStored: true,
		},
		{
			name: "rejects missing rule id",
			form: url.Values{
				"name":            {"Night mute"},
				"action":          {"drop"},
				"event_pattern":   {"*"},
				"channel_pattern": {"*"},
			},
			wantStatus: http.StatusOK,
			wantSub:    "Rule ID is required",
		},
		{
			name: "creates disabled rule without loading into engine list filter",
			form: url.Values{
				"rule_id":         {"disabled_drop"},
				"name":            {"Disabled"},
				"action":          {"drop"},
				"event_pattern":   {"*"},
				"channel_pattern": {"*"},
				"priority":        {"1"},
			},
			wantStatus: http.StatusOK,
			wantSub:    "disabled_drop",
			wantStored: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, ts := setupTestApp()
			defer func() {
				store.Database = nil
				handler = moduleHandler{}
				config = configType{}
			}()
			require.NoError(t, notifyrules.Init(nil, nil))

			req := httptest.NewRequest(http.MethodPost, "/service/web/notifications/rules", strings.NewReader(tt.form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "valid-token"})
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("want status %d, got %d", tt.wantStatus, resp.StatusCode)
			}
			body, _ := io.ReadAll(resp.Body)
			if !strings.Contains(string(body), tt.wantSub) {
				t.Errorf("want body containing %q, got %q", tt.wantSub, string(body))
			}
			if tt.wantStored && len(ts.notifyRules) != 1 {
				t.Errorf("want 1 rule stored, got %d", len(ts.notifyRules))
			}
			if !tt.wantStored && len(ts.notifyRules) != 0 {
				t.Errorf("want no rules stored, got %d", len(ts.notifyRules))
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

			req := httptest.NewRequest(http.MethodPost, "/service/web/notifications/channels/"+tt.channelID+"/test", http.NoBody)
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
