package partials

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/i18n"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
)

func TestWorkflowWebPath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "simple", in: "save-url", want: "/service/web/workflows/save-url"},
		{name: "needs escape", in: "a b", want: "/service/web/workflows/a%20b"},
		{name: "unicode", in: "café", want: "/service/web/workflows/caf%C3%A9"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, WorkflowWebPath(tt.in))
		})
	}
}

func TestBuildWorkflowListEntries(t *testing.T) {
	t.Parallel()
	lastRun := time.Date(2026, 7, 21, 18, 0, 0, 0, time.UTC)
	tests := []struct {
		name         string
		defs         []model.Workflow
		triggers     []model.WorkflowTrigger
		lastRunAt    map[string]time.Time
		want         int
		wantTriggers []string
		wantLastRun  *time.Time
	}{
		{name: "empty", defs: nil, want: 0},
		{name: "single workflow", defs: []model.Workflow{{ID: 1, Name: "a", Pipeline: []string{"t1"}, Enabled: true}}, want: 1},
		{
			name: "attaches triggers by workflow id",
			defs: []model.Workflow{{ID: 7, Name: "echo", Pipeline: []string{"x"}, Enabled: true}},
			triggers: []model.WorkflowTrigger{
				{WorkflowID: 7, Type: "manual", Enabled: true},
				{WorkflowID: 7, Type: "cron", Enabled: true, Rule: map[string]any{"cron": "@hourly"}},
				{WorkflowID: 99, Type: "webhook", Enabled: true},
			},
			want:         1,
			wantTriggers: []string{"manual", "cron"},
		},
		{
			name:        "attaches last run",
			defs:        []model.Workflow{{ID: 2, Name: "echo", Pipeline: []string{"x"}}},
			lastRunAt:   map[string]time.Time{"echo": lastRun},
			want:        1,
			wantLastRun: &lastRun,
		},
		{name: "two defs", defs: []model.Workflow{{Name: "a"}, {Name: "b", Pipeline: []string{"x", "y"}}}, want: 2},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := BuildWorkflowListEntries(i18n.DefaultContext(), tt.defs, tt.triggers, tt.lastRunAt)
			require.Len(t, got, tt.want)
			if tt.want == 1 && tt.wantTriggers == nil && tt.wantLastRun == nil {
				assert.Equal(t, "a", got[0].Name)
				assert.Equal(t, 1, got[0].TaskCount)
				assert.True(t, got[0].Enabled)
				assert.Nil(t, got[0].LastRunAt)
			}
			if tt.wantTriggers != nil {
				require.Len(t, got[0].Triggers, len(tt.wantTriggers))
				for i, typ := range tt.wantTriggers {
					assert.Equal(t, typ, got[0].Triggers[i].Type)
				}
				assert.Equal(t, "M", got[0].Triggers[0].Letter)
				assert.Equal(t, "Cron: @hourly", got[0].Triggers[1].Label)
			}
			if tt.wantLastRun != nil {
				require.NotNil(t, got[0].LastRunAt)
				assert.True(t, got[0].LastRunAt.Equal(*tt.wantLastRun))
			}
			if tt.want == 2 {
				assert.Equal(t, 2, got[1].TaskCount)
			}
		})
	}
}

func TestWorkflowTriggerSummaries(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		rows       []model.WorkflowTrigger
		wantLen    int
		wantLabel  string
		wantLetter string
	}{
		{name: "nil", rows: nil, wantLen: 0},
		{name: "webhook path", rows: []model.WorkflowTrigger{{Type: "webhook", Enabled: true, Rule: map[string]any{"path": "hooks/a"}}}, wantLen: 1, wantLabel: "Webhook: hooks/a", wantLetter: "W"},
		{name: "manual", rows: []model.WorkflowTrigger{{Type: "manual", Enabled: false}}, wantLen: 1, wantLabel: "Manual", wantLetter: "M"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := WorkflowTriggerSummaries(i18n.DefaultContext(), tt.rows)
			require.Len(t, got, tt.wantLen)
			if tt.wantLen == 0 {
				return
			}
			assert.Equal(t, tt.wantLabel, got[0].Label)
			assert.Equal(t, tt.wantLetter, got[0].Letter)
		})
	}
}

func TestWorkflowWebhookURLPath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		tr   model.WorkflowTrigger
		want string
	}{
		{name: "zero", tr: model.WorkflowTrigger{}, want: ""},
		{name: "manual", tr: model.WorkflowTrigger{Type: "manual"}, want: ""},
		{name: "webhook missing path", tr: model.WorkflowTrigger{Type: "webhook", Rule: map[string]any{"payload": "raw"}}, want: ""},
		{
			name: "path without leading slash",
			tr:   model.WorkflowTrigger{Type: "webhook", Rule: map[string]any{"path": "hooks/a"}},
			want: "/webhook/workflow/hooks/a",
		},
		{
			name: "path with leading slash and token",
			tr: model.WorkflowTrigger{Type: "webhook", Rule: map[string]any{
				"path": "/hooks/my-workflow",
				"auth": map[string]any{"token": "secret+value"},
			}},
			want: "/webhook/workflow/hooks/my-workflow?token=secret%2Bvalue",
		},
		{
			name: "hmac only skips token query",
			tr: model.WorkflowTrigger{Type: "webhook", Rule: map[string]any{
				"path": "hooks/b",
				"auth": map[string]any{"hmac_secret": "hmac"},
			}},
			want: "/webhook/workflow/hooks/b",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, WorkflowWebhookURLPath(tt.tr))
		})
	}
}

func TestWorkflowWebhookURL(t *testing.T) {
	t.Parallel()
	tr := model.WorkflowTrigger{Type: "webhook", Rule: map[string]any{"path": "hooks/a", "auth": map[string]any{"token": "t"}}}
	tests := []struct {
		name   string
		tr     model.WorkflowTrigger
		origin string
		want   string
	}{
		{name: "zero", tr: model.WorkflowTrigger{}, origin: "https://bot.example", want: ""},
		{name: "relative when origin empty", tr: tr, origin: "", want: "/webhook/workflow/hooks/a?token=t"},
		{name: "absolute", tr: tr, origin: "https://bot.example/", want: "https://bot.example/webhook/workflow/hooks/a?token=t"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, WorkflowWebhookURL(tt.tr, tt.origin))
		})
	}
}

func TestWorkflowTriggersTable_webhookURLAndCopy(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		tr     model.WorkflowTrigger
		origin string
		want   []string
		absent []string
	}{
		{
			name: "webhook shows absolute url and copy",
			tr: model.WorkflowTrigger{
				ID:   7,
				Type: "webhook",
				Rule: map[string]any{
					"path": "/hooks/bookmark",
					"auth": map[string]any{"token": "tok"},
				},
			},
			origin: "https://bot.example",
			want: []string{
				`data-testid="workflow-webhook-url-7"`,
				`https://bot.example/webhook/workflow/hooks/bookmark?token=tok`,
				`data-testid="btn-copy-workflow-webhook-url-7"`,
				`data-clip-copy`,
				`data-clip-markdown="https://bot.example/webhook/workflow/hooks/bookmark?token=tok"`,
			},
			absent: []string{`"payload"`, `data-absolute-url-path`, `POST`},
		},
		{
			name: "manual keeps rule preview",
			tr: model.WorkflowTrigger{
				ID:   1,
				Type: "manual",
				Rule: nil,
			},
			origin: "https://bot.example",
			want:   []string{`data-testid="workflow-trigger-1"`, `manual`},
			absent: []string{`data-clip-markdown`, `workflow-webhook-url`},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			if err := WorkflowTriggersTable(context.Background(), "demo", []model.WorkflowTrigger{tt.tr}, tt.origin).Render(context.Background(), &buf); err != nil {
				t.Fatalf("render: %v", err)
			}
			html := buf.String()
			for _, w := range tt.want {
				if !strings.Contains(html, w) {
					t.Fatalf("want %q in html\nhtml=%s", w, html)
				}
			}
			for _, a := range tt.absent {
				if strings.Contains(html, a) {
					t.Fatalf("did not want %q in html\nhtml=%s", a, html)
				}
			}
		})
	}
}

func TestWorkflowRunStatusHelpers(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		status    int
		wantClass string
		wantText  string
	}{
		{name: "done", status: int(types.WorkflowRunDone), wantClass: "flowbot-chip flowbot-chip-success", wantText: "Done"},
		{name: "failed", status: int(types.WorkflowRunFailed), wantClass: "flowbot-chip flowbot-chip-error", wantText: "Failed"},
		{name: "running", status: int(types.WorkflowRunRunning), wantClass: "flowbot-chip flowbot-chip-warning", wantText: "Running"},
		{name: "unknown", status: int(types.WorkflowRunStateUnknown), wantClass: "flowbot-chip flowbot-chip-muted", wantText: "Unknown"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.wantClass, WorkflowRunStatusClass(tt.status))
			assert.Equal(t, tt.wantText, WorkflowRunStatusText(i18n.DefaultContext(), tt.status))
		})
	}
}

func TestWorkflowInputDefaultString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		def  types.WorkflowInputDef
		want string
	}{
		{name: "nil default", def: types.WorkflowInputDef{Name: "a"}, want: ""},
		{name: "string default", def: types.WorkflowInputDef{Default: "hello"}, want: "hello"},
		{name: "bool true", def: types.WorkflowInputDef{Default: true}, want: "true"},
		{name: "number", def: types.WorkflowInputDef{Default: 3}, want: "3"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, WorkflowInputDefaultString(tt.def))
		})
	}
}

func TestWorkflowInputTypeHelpers(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		typ        string
		wantBool   bool
		wantJSON   bool
		wantNumber bool
	}{
		{name: "boolean", typ: types.WorkflowInputTypeBoolean, wantBool: true},
		{name: "json", typ: types.WorkflowInputTypeJSON, wantJSON: true},
		{name: "number", typ: types.WorkflowInputTypeNumber, wantNumber: true},
		{name: "string", typ: types.WorkflowInputTypeString},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.wantBool, WorkflowInputIsBoolean(tt.typ))
			assert.Equal(t, tt.wantJSON, WorkflowInputIsJSON(tt.typ))
			assert.Equal(t, tt.wantNumber, WorkflowInputIsNumber(tt.typ))
		})
	}
}

func TestBuildWorkflowDAGView(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		tasks           []types.WorkflowTask
		pipeline        []string
		maxConcurrency  int
		wantLayers      [][]string
		wantSequential  bool
		wantParallelRun bool
	}{
		{
			name:       "empty",
			tasks:      nil,
			wantLayers: nil,
		},
		{
			name: "sequential from pipeline without conn",
			tasks: []types.WorkflowTask{
				{ID: "a", Action: "mapper:"},
				{ID: "b", Action: "mapper:", Describe: "second"},
				{ID: "c", Action: "shell:echo"},
			},
			pipeline:        []string{"a", "b", "c"},
			maxConcurrency:  3,
			wantLayers:      [][]string{{"a"}, {"b"}, {"c"}},
			wantSequential:  true,
			wantParallelRun: true,
		},
		{
			name: "diamond parallel dag",
			tasks: []types.WorkflowTask{
				{ID: "fetch_data", Action: "capability:x.list", Describe: "Fetch"},
				{ID: "archive_url", Action: "capability:a.create", Conn: []string{"fetch_data"}},
				{ID: "create_task", Action: "capability:k.create", Conn: []string{"fetch_data"}},
				{ID: "notify", Action: "capability:n.send", Conn: []string{"archive_url", "create_task"}},
			},
			pipeline:        []string{"fetch_data", "archive_url", "create_task", "notify"},
			maxConcurrency:  3,
			wantLayers:      [][]string{{"fetch_data"}, {"archive_url", "create_task"}, {"notify"}},
			wantSequential:  false,
			wantParallelRun: true,
		},
		{
			name: "diamond conn but serial runtime",
			tasks: []types.WorkflowTask{
				{ID: "fetch_data", Action: "capability:x.list"},
				{ID: "archive_url", Action: "capability:a.create", Conn: []string{"fetch_data"}},
				{ID: "create_task", Action: "capability:k.create", Conn: []string{"fetch_data"}},
			},
			pipeline:        []string{"fetch_data", "archive_url", "create_task"},
			maxConcurrency:  1,
			wantLayers:      [][]string{{"fetch_data"}, {"archive_url", "create_task"}},
			wantSequential:  false,
			wantParallelRun: false,
		},
		{
			name: "single root task",
			tasks: []types.WorkflowTask{
				{ID: "only", Action: "mapper:", Describe: "solo"},
			},
			pipeline:        []string{"only"},
			maxConcurrency:  0,
			wantLayers:      [][]string{{"only"}},
			wantSequential:  true,
			wantParallelRun: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := BuildWorkflowDAGView(tt.tasks, tt.pipeline, tt.maxConcurrency)
			assert.Equal(t, tt.wantSequential, got.Sequential)
			assert.Equal(t, tt.wantParallelRun, got.ParallelRuntime)
			require.Len(t, got.Layers, len(tt.wantLayers))
			for i, wantIDs := range tt.wantLayers {
				gotIDs := make([]string, 0, len(got.Layers[i]))
				for _, n := range got.Layers[i] {
					gotIDs = append(gotIDs, n.ID)
				}
				assert.Equal(t, wantIDs, gotIDs)
			}
			if tt.name == "diamond parallel dag" {
				require.Len(t, got.Layers[1], 2)
				assert.Equal(t, []string{"fetch_data"}, got.Layers[1][0].Deps)
				assert.Equal(t, []string{"archive_url", "create_task"}, got.Layers[2][0].Deps)
			}
			if tt.name == "sequential from pipeline without conn" {
				assert.Equal(t, []string{"a"}, got.Layers[1][0].Deps)
			}
		})
	}
}

func TestWorkflowDAGConnectorStyle(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		from int
		to   int
		want string
		cls  string
	}{
		{
			name: "one to one",
			from: 1,
			to:   1,
			want: "--from-cols: 1; --to-cols: 1; --rail-cols: 1",
			cls:  "workflow-dag-connector workflow-dag-connector-single",
		},
		{
			name: "one to two fork",
			from: 1,
			to:   2,
			want: "--from-cols: 1; --to-cols: 2; --rail-cols: 2",
			cls:  "workflow-dag-connector",
		},
		{
			name: "two to one join",
			from: 2,
			to:   1,
			want: "--from-cols: 2; --to-cols: 1; --rail-cols: 2",
			cls:  "workflow-dag-connector",
		},
		{
			name: "clamps zero",
			from: 0,
			to:   0,
			want: "--from-cols: 1; --to-cols: 1; --rail-cols: 1",
			cls:  "workflow-dag-connector workflow-dag-connector-single",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, workflowDAGConnectorStyle(tt.from, tt.to))
			assert.Equal(t, tt.cls, workflowDAGConnectorClass(tt.from, tt.to))
		})
	}
}

func TestWorkflowDAGNodeIndex(t *testing.T) {
	t.Parallel()
	view := WorkflowDAGView{
		Layers: [][]WorkflowDAGNode{
			{{ID: "a"}},
			{{ID: "b"}, {ID: "c"}},
			{{ID: "d"}},
		},
	}
	tests := []struct {
		name string
		li   int
		ni   int
		want int
	}{
		{name: "first", li: 0, ni: 0, want: 1},
		{name: "parallel second", li: 1, ni: 1, want: 3},
		{name: "last", li: 2, ni: 0, want: 4},
		{name: "out of range", li: 9, ni: 0, want: 0},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, WorkflowDAGNodeIndex(view, tt.li, tt.ni))
		})
	}
}
