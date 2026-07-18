package web

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
	notifypkg "github.com/flowline-io/flowbot/pkg/notify"
	notifytmpl "github.com/flowline-io/flowbot/pkg/notify/template"
	"github.com/flowline-io/flowbot/pkg/types/model"
)

func TestValidatePlaygroundRequest(t *testing.T) {
	// Subtests mutate the global template engine; do not run in parallel.
	tests := []struct {
		name      string
		req       playgroundRequest
		templates []notifypkg.Template
		wantField string
		wantSub   string
	}{
		{
			name:      "missing channel",
			req:       playgroundRequest{Mode: "template", TemplateID: "a.created", PayloadJSON: "{}"},
			templates: []notifypkg.Template{{ID: "a.created", DefaultTemplate: "x"}},
			wantField: "channel_id",
			wantSub:   "Channel is required",
		},
		{
			name:      "template mode requires template id",
			req:       playgroundRequest{Mode: "template", ChannelID: 1, PayloadJSON: "{}"},
			wantField: "template_id",
			wantSub:   "Template is required",
		},
		{
			name:      "custom mode requires body",
			req:       playgroundRequest{Mode: "custom", ChannelID: 1, PayloadJSON: "{}"},
			wantField: "custom_template",
			wantSub:   "Custom template is required",
		},
		{
			name:      "invalid payload json",
			req:       playgroundRequest{Mode: "custom", ChannelID: 1, CustomTemplate: "hi", PayloadJSON: "{"},
			wantField: "payload_json",
			wantSub:   "Invalid JSON",
		},
		{
			name:      "unknown template",
			req:       playgroundRequest{Mode: "template", ChannelID: 1, TemplateID: "missing", PayloadJSON: "{}"},
			templates: []notifypkg.Template{{ID: "a.created", DefaultTemplate: "x"}},
			wantField: "template_id",
			wantSub:   "Unknown template",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notifytmpl.ResetForTest()
			if tt.templates != nil {
				require.NoError(t, notifytmpl.Init(tt.templates))
			}
			t.Cleanup(notifytmpl.ResetForTest)
			errs := validatePlaygroundRequest(tt.req)
			assert.Contains(t, errs[tt.wantField], tt.wantSub)
		})
	}
}

func TestParsePlaygroundPriority(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		raw  string
		want notifypkg.Priority
	}{
		{name: "low", raw: "low", want: notifypkg.Low},
		{name: "emergency", raw: "emergency", want: notifypkg.Emergency},
		{name: "default normal", raw: "", want: notifypkg.Normal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, parsePlaygroundPriority(tt.raw))
		})
	}
}

func TestPlaygroundHistoryTemplateID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		req  playgroundRequest
		want string
	}{
		{name: "custom uses playground id", req: playgroundRequest{Mode: "custom"}, want: notifypkg.PlaygroundTemplateID},
		{name: "template uses selected id", req: playgroundRequest{Mode: "template", TemplateID: "agent.status"}, want: "agent.status"},
		{name: "empty mode treated as template", req: playgroundRequest{TemplateID: "x"}, want: "x"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, playgroundHistoryTemplateID(tt.req))
		})
	}
}

func TestNotifyPlaygroundFormUnauthenticated(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		method     string
		wantStatus int
	}{
		{name: "form requires auth", path: "/service/web/notifications/playground", method: http.MethodGet, wantStatus: http.StatusSeeOther},
		{name: "preview requires auth", path: "/service/web/notifications/playground/preview", method: http.MethodPost, wantStatus: http.StatusSeeOther},
		{name: "send requires auth", path: "/service/web/notifications/playground/send", method: http.MethodPost, wantStatus: http.StatusSeeOther},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := setupTestApp()
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()

			req := httptest.NewRequest(tt.method, tt.path, http.NoBody)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestNotifyPlaygroundPreviewValidation(t *testing.T) {
	tests := []struct {
		name    string
		form    url.Values
		wantSub string
	}{
		{
			name: "rejects missing channel",
			form: url.Values{
				"mode":            {"custom"},
				"custom_template": {"**Hi**"},
				"payload_json":    {"{}"},
			},
			wantSub: "Channel is required",
		},
		{
			name: "rejects missing custom template",
			form: url.Values{
				"mode":         {"custom"},
				"channel_id":   {"1"},
				"payload_json": {"{}"},
			},
			wantSub: "Custom template is required",
		},
		{
			name: "previews custom template",
			form: url.Values{
				"mode":            {"custom"},
				"channel_id":      {"1"},
				"custom_template": {"**Hello {{ .name }}**\nLine 2"},
				"format":          {"markdown"},
				"payload_json":    {"{\"name\":\"world\"}"},
				"priority":        {"normal"},
			},
			wantSub: "Hello world",
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
			ts.notifyChannels = map[int64]model.NotifyChannel{
				1: {ID: 1, Name: "alerts", Protocol: "slack", URI: "slack://T00/B00/xxx", Enabled: true},
			}

			req := httptest.NewRequest(http.MethodPost, "/service/web/notifications/playground/preview", strings.NewReader(tt.form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "valid-token"})
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			body, _ := io.ReadAll(resp.Body)
			assert.Contains(t, string(body), tt.wantSub)
		})
	}
}

func TestNotifyPlaygroundSamplePayload(t *testing.T) {
	tests := []struct {
		name       string
		templateID string
		templates  []notifypkg.Template
		wantSub    string
	}{
		{
			name:       "empty template id returns default payload",
			templateID: "",
			wantSub:    "playground",
		},
		{
			name:       "known template returns extracted fields",
			templateID: "bookmark.created",
			templates: []notifypkg.Template{
				{ID: "bookmark.created", DefaultTemplate: `{{ .url }} {{ .title }}`},
			},
			wantSub: "https://example.com",
		},
		{
			name:       "unknown template falls back to default",
			templateID: "missing.id",
			templates:  []notifypkg.Template{{ID: "other", DefaultTemplate: `{{ .x }}`}},
			wantSub:    "playground",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := setupTestApp()
			defer func() {
				store.Database = nil
				handler = moduleHandler{}
				config = configType{}
				notifytmpl.ResetForTest()
			}()
			require.NoError(t, notifytmpl.Init(tt.templates))

			path := "/service/web/notifications/playground/sample-payload"
			if tt.templateID != "" {
				path += "?template_id=" + url.QueryEscape(tt.templateID)
			}
			req := httptest.NewRequest(http.MethodGet, path, http.NoBody)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "valid-token"})
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			body, _ := io.ReadAll(resp.Body)
			assert.Contains(t, string(body), "playground-payload")
			assert.Contains(t, string(body), tt.wantSub)
		})
	}
}

func TestNormalizeNotifySettingsTabIncludesPlayground(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "playground", normalizeNotifySettingsTab("playground"))
}
