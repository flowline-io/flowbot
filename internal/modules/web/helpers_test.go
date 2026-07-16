package web

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/pipeline"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
)

func TestParseTimeParam(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{name: "RFC3339", raw: "2026-01-15T10:30:00Z", wantErr: false},
		{name: "datetime-local", raw: "2026-01-15T10:30", wantErr: false},
		{name: "invalid format", raw: "not-a-time", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseTimeParam(tt.raw)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.False(t, got.IsZero())
		})
	}
}

func TestLifecycleScope(t *testing.T) {
	t.Parallel()
	tests := []struct {
		op   string
		want string
	}{
		{op: "start", want: "hub:apps:start"},
		{op: "stop", want: "hub:apps:stop"},
		{op: "restart", want: "hub:apps:restart"},
		{op: "pull", want: "hub:apps:pull"},
		{op: "update", want: "hub:apps:update"},
		{op: "unknown", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.op, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, lifecycleScope(tt.op))
		})
	}
}

func TestUniqueTypesAndProviders(t *testing.T) {
	t.Parallel()
	descriptors := []hub.Descriptor{
		{Type: "karakeep"},
		{Type: "miniflux"},
		{Type: "karakeep"},
		{Type: ""},
	}

	typesList := uniqueTypes(descriptors)
	assert.Equal(t, []string{"", "karakeep", "miniflux"}, typesList)

	providers := uniqueProviders(descriptors)
	assert.Equal(t, []string{"karakeep", "miniflux"}, providers)
}

func TestPipelineStatusLabels(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		fn   func(int) string
		in   int
		want string
	}{
		{name: "step running", fn: stepRunStatusLabel, in: 1, want: "running"},
		{name: "step done", fn: stepRunStatusLabel, in: 2, want: "done"},
		{name: "step error", fn: stepRunStatusLabel, in: 4, want: "error"},
		{name: "run failed", fn: pipelineRunStatusLabel, in: 4, want: "failed"},
		{name: "run pending default", fn: pipelineRunStatusLabel, in: 0, want: "pending"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.fn(tt.in))
		})
	}
}

func TestPipelineNameParamHandler(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		raw        string
		want       string
		wantStatus int
	}{
		{name: "plain name", raw: "demo-pipeline", want: "demo-pipeline", wantStatus: http.StatusOK},
		{name: "encoded chinese", raw: "%E6%BC%94%E7%A4%BA1", want: "演示1", wantStatus: http.StatusOK},
		{name: "underscore name", raw: "pipe_v2", want: "pipe_v2", wantStatus: http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			app := fiber.New(fiber.Config{ErrorHandler: func(c fiber.Ctx, err error) error {
				return c.Status(fiber.StatusBadRequest).SendString(err.Error())
			}})
			app.Get("/:name", func(c fiber.Ctx) error {
				got, err := pipelineNameParam(c)
				if err != nil {
					return err
				}
				return c.SendString(got)
			})
			req := httptest.NewRequest(http.MethodGet, "/"+tt.raw, http.NoBody)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			if tt.wantStatus == http.StatusOK {
				body, readErr := io.ReadAll(resp.Body)
				require.NoError(t, readErr)
				assert.Equal(t, tt.want, string(body))
			}
		})
	}
}

func TestBroadcastStreamReadArgs(t *testing.T) {
	t.Parallel()
	args := broadcastStreamReadArgs("stream:run", "123-0")
	require.NotNil(t, args)
	assert.Equal(t, []string{"stream:run", "123-0"}, args.Streams)
	assert.Equal(t, int64(10), args.Count)
	assert.Equal(t, 5*time.Second, args.Block)
}

func TestWriteHeartbeatAndSSEEvent(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	assert.False(t, writeHeartbeat(w))

	completePayload, err := sonic.MarshalString(pipeline.StepProgressEvent{StepIndex: -1, Status: "complete"})
	require.NoError(t, err)
	assert.True(t, writeSSEEvent(w, completePayload))

	runningPayload, err := sonic.MarshalString(pipeline.StepProgressEvent{StepIndex: 1, Status: "running"})
	require.NoError(t, err)
	assert.False(t, writeSSEEvent(w, runningPayload))

	assert.Contains(t, buf.String(), ": heartbeat")
	assert.Contains(t, buf.String(), "data:")
}

func TestHandleStreamRead(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		result []redis.XStream
		err    error
		done   bool
	}{
		{name: "canceled context ends stream", err: context.Canceled, done: true},
		{name: "redis nil sends heartbeat", err: redis.Nil, done: false},
		{name: "generic error retries", err: errors.New("boom"), done: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			w := bufio.NewWriter(&buf)
			lastID := "0-0"
			done := handleStreamRead(w, tt.result, tt.err, &lastID)
			assert.Equal(t, tt.done, done)
		})
	}
}

func TestRenderError(t *testing.T) {
	t.Parallel()
	app := fiber.New()
	app.Get("/", func(c fiber.Ctx) error {
		return renderError(c, "bad input")
	})
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	resp, err := app.Test(req)
	require.NoError(t, err)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "bad input")
	assert.Contains(t, resp.Header.Get("Content-Type"), "text/html")
}

func TestValidateThrottleAndAggregateParams(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		params     map[string]any
		wantSubstr string
		agg        bool
	}{
		{name: "throttle missing window", params: map[string]any{"limit": float64(1)}, wantSubstr: "Window is required"},
		{name: "throttle invalid limit", params: map[string]any{"window": "1m", "limit": float64(0)}, wantSubstr: "Limit must be > 0"},
		{name: "aggregate missing window", params: map[string]any{"digest_tpl_id": "digest"}, wantSubstr: "Window is required", agg: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			errs := map[string]string{}
			if tt.agg {
				validateAggregateParams(tt.params, &errs)
			} else {
				validateThrottleParams(tt.params, &errs)
			}
			assert.Contains(t, errs["params_json"], tt.wantSubstr)
		})
	}
}

func TestParseIDHandler(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		param      string
		wantStatus int
	}{
		{name: "valid id", param: "42", wantStatus: http.StatusOK},
		{name: "invalid id", param: "abc", wantStatus: fiber.StatusBadRequest},
		{name: "zero id allowed", param: "0", wantStatus: http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			app := fiber.New(fiber.Config{ErrorHandler: func(c fiber.Ctx, err error) error {
				return c.Status(fiber.StatusBadRequest).SendString(err.Error())
			}})
			app.Get("/:id", func(c fiber.Ctx) error {
				_, err := parseID(c)
				if err != nil {
					return err
				}
				return c.SendStatus(http.StatusOK)
			})
			req := httptest.NewRequest(http.MethodGet, "/"+tt.param, http.NoBody)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestParseTimeRangeInvalidOrder(t *testing.T) {
	t.Parallel()
	app := fiber.New()
	var opts store.ListDataEventsOptions
	app.Get("/", func(c fiber.Ctx) error {
		opts = store.ListDataEventsOptions{}
		parseTimeRange(c, &opts)
		return nil
	})
	req := httptest.NewRequest(http.MethodGet, "/?time_start=2026-01-20T10:00&time_end=2026-01-10T10:00", http.NoBody)
	_, err := app.Test(req)
	require.NoError(t, err)
	assert.Nil(t, opts.TimeStart)
	assert.Nil(t, opts.TimeEnd)
}

func TestGetProtocolsAndTemplateIDs(t *testing.T) {
	t.Parallel()
	protocols := getProtocolsList()
	assert.NotNil(t, protocols)

	templates := getTemplateIDs()
	assert.NotNil(t, templates)
}

func TestHtmlEscape(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in   string
		want string
	}{
		{in: "plain", want: "plain"},
		{in: "<script>", want: "&lt;script&gt;"},
		{in: "a & b", want: "a &amp; b"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, htmlEscape(tt.in))
		})
	}
}

func TestCollectFormArgs(t *testing.T) {
	t.Parallel()
	app := fiber.New()
	app.Post("/", func(c fiber.Ctx) error {
		args := collectFormArgs(c)
		return c.JSON(args)
	})
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("rules={}&mode=ask"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := app.Test(req)
	require.NoError(t, err)
	var got map[string]string
	require.NoError(t, sonic.ConfigDefault.NewDecoder(resp.Body).Decode(&got))
	assert.Equal(t, "{}", got["rules"])
	assert.Equal(t, "ask", got["mode"])
}

func TestInvalidTokenUsageRequest(t *testing.T) {
	t.Parallel()
	app := fiber.New()
	app.Get("/", func(c fiber.Ctx) error {
		return invalidTokenUsageRequest(c, types.Errorf(types.ErrInvalidArgument, "bad range"))
	})
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestMapAgentSession(t *testing.T) {
	t.Parallel()
	now := time.Now()
	row := &gen.ChatSession{
		Flag: "sess-1", UID: "user-1", Title: "Demo", State: int(schema.ChatSessionActive),
		CreatedAt: now, UpdatedAt: now,
	}
	got := mapAgentSession(row)
	assert.Equal(t, "sess-1", got.Flag)
	assert.Equal(t, "user-1", got.UID)
	assert.Equal(t, "Active", got.State)
}

func TestValidateChannelForm(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		channel   string
		protocol  string
		uri       string
		wantField string
		wantSub   string
	}{
		{name: "valid channel", channel: "alerts", protocol: "webhook", uri: "https://example.com/hook"},
		{name: "missing name", channel: "", protocol: "webhook", uri: "https://example.com", wantField: "name", wantSub: "Name is required"},
		{name: "missing protocol", channel: "alerts", protocol: "", uri: "https://example.com", wantField: "protocol", wantSub: "Protocol is required"},
		{name: "missing uri", channel: "alerts", protocol: "webhook", uri: "", wantField: "uri", wantSub: "URI is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			errs := validateChannelForm(tt.channel, tt.protocol, tt.uri)
			if tt.wantField == "" {
				assert.Empty(t, errs)
				return
			}
			assert.Contains(t, errs[tt.wantField], tt.wantSub)
		})
	}
}

func TestValidateRuleForm(t *testing.T) {
	t.Parallel()
	validRule := model.NotifyRule{
		Name: "demo", RuleID: "rule-1", EventPattern: "pipeline.failed",
		ChannelPattern: "webhook:*", Action: "notify",
	}
	tests := []struct {
		name      string
		rule      model.NotifyRule
		wantField string
		wantSub   string
	}{
		{name: "valid notify rule", rule: validRule},
		{name: "missing name", rule: model.NotifyRule{RuleID: "r1", EventPattern: "x", ChannelPattern: "y", Action: "notify"}, wantField: "name", wantSub: "Name is required"},
		{name: "missing rule id", rule: model.NotifyRule{Name: "demo", EventPattern: "x", ChannelPattern: "y", Action: "notify"}, wantField: "rule_id", wantSub: "Rule ID is required"},
		{name: "missing event pattern", rule: model.NotifyRule{Name: "demo", RuleID: "r1", ChannelPattern: "y", Action: "notify"}, wantField: "event_pattern", wantSub: "Event pattern is required"},
		{name: "missing action", rule: model.NotifyRule{Name: "demo", RuleID: "r1", EventPattern: "x", ChannelPattern: "y"}, wantField: "action", wantSub: "Action is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			errs := validateRuleForm(tt.rule)
			if tt.wantField == "" {
				assert.Empty(t, errs)
				return
			}
			assert.Contains(t, errs[tt.wantField], tt.wantSub)
		})
	}
}

func TestParseConfigValue(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		raw   string
		check func(t *testing.T, got types.KV)
	}{
		{
			name: "json object",
			raw:  `{"k":"v"}`,
			check: func(t *testing.T, got types.KV) {
				assert.Equal(t, "v", got["k"])
			},
		},
		{
			name: "empty string",
			raw:  "",
			check: func(t *testing.T, got types.KV) {
				assert.Empty(t, got)
			},
		},
		{
			name: "json scalar wrapped",
			raw:  `"hello"`,
			check: func(t *testing.T, got types.KV) {
				assert.Equal(t, "hello", got["value"])
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.check(t, parseConfigValue(tt.raw))
		})
	}
}

func TestShowToastTrigger(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		typ     string
		message string
		wantSub string
	}{
		{name: "success toast", typ: "success", message: "saved", wantSub: "saved"},
		{name: "error toast with quotes escaped", typ: "error", message: `failed: foo "bar"`, wantSub: `"type":"error"`},
		{name: "info toast", typ: "info", message: "hello", wantSub: "showToast"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := showToastTrigger(tt.typ, tt.message)
			require.NoError(t, err)
			assert.Contains(t, got, tt.wantSub)
			if strings.Contains(tt.message, `"`) {
				assert.NotContains(t, got, `foo "bar"`)
			}
		})
	}
}

func TestChatSessionStateLabelHelper(t *testing.T) {
	t.Parallel()
	tests := []struct {
		state int
		want  string
	}{
		{state: int(schema.ChatSessionActive), want: "Active"},
		{state: int(schema.ChatSessionClosed), want: "Closed"},
		{state: 999, want: "Unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, chatSessionStateLabel(tt.state))
		})
	}
}
