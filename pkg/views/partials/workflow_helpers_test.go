package partials

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/types"
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
		{name: "unicode", in: "工作流", want: "/service/web/workflows/%E5%B7%A5%E4%BD%9C%E6%B5%81"},
	}
	for _, tt := range tests {
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
		defs         []*gen.Workflow
		triggers     []*gen.WorkflowTrigger
		lastRunAt    map[string]time.Time
		want         int
		wantTriggers []string
		wantLastRun  *time.Time
	}{
		{name: "empty", defs: nil, want: 0},
		{name: "skips nil", defs: []*gen.Workflow{nil, {ID: 1, Name: "a", Pipeline: []string{"t1"}, Enabled: true}}, want: 1},
		{
			name: "attaches triggers by workflow id",
			defs: []*gen.Workflow{{ID: 7, Name: "echo", Pipeline: []string{"x"}, Enabled: true}},
			triggers: []*gen.WorkflowTrigger{
				{WorkflowID: 7, Type: "manual", Enabled: true},
				{WorkflowID: 7, Type: "cron", Enabled: true, Rule: map[string]any{"cron": "@hourly"}},
				{WorkflowID: 99, Type: "webhook", Enabled: true},
			},
			want:         1,
			wantTriggers: []string{"manual", "cron"},
		},
		{
			name:        "attaches last run",
			defs:        []*gen.Workflow{{ID: 2, Name: "echo", Pipeline: []string{"x"}}},
			lastRunAt:   map[string]time.Time{"echo": lastRun},
			want:        1,
			wantLastRun: &lastRun,
		},
		{name: "two defs", defs: []*gen.Workflow{{Name: "a"}, {Name: "b", Pipeline: []string{"x", "y"}}}, want: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := BuildWorkflowListEntries(tt.defs, tt.triggers, tt.lastRunAt)
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
		rows       []*gen.WorkflowTrigger
		wantLen    int
		wantLabel  string
		wantLetter string
	}{
		{name: "nil", rows: nil, wantLen: 0},
		{name: "webhook path", rows: []*gen.WorkflowTrigger{{Type: "webhook", Enabled: true, Rule: map[string]any{"path": "hooks/a"}}}, wantLen: 1, wantLabel: "Webhook: hooks/a", wantLetter: "W"},
		{name: "manual", rows: []*gen.WorkflowTrigger{{Type: "manual", Enabled: false}}, wantLen: 1, wantLabel: "Manual", wantLetter: "M"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := WorkflowTriggerSummaries(tt.rows)
			require.Len(t, got, tt.wantLen)
			if tt.wantLen == 0 {
				return
			}
			assert.Equal(t, tt.wantLabel, got[0].Label)
			assert.Equal(t, tt.wantLetter, got[0].Letter)
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
		{name: "done", status: int(schema.WorkflowRunDone), wantClass: "flowbot-chip flowbot-chip-success", wantText: "Done"},
		{name: "failed", status: int(schema.WorkflowRunFailed), wantClass: "flowbot-chip flowbot-chip-error", wantText: "Failed"},
		{name: "running", status: int(schema.WorkflowRunRunning), wantClass: "flowbot-chip flowbot-chip-warning", wantText: "Running"},
		{name: "unknown", status: int(schema.WorkflowRunStateUnknown), wantClass: "flowbot-chip flowbot-chip-muted", wantText: "Unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.wantClass, WorkflowRunStatusClass(tt.status))
			assert.Equal(t, tt.wantText, WorkflowRunStatusText(tt.status))
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
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, WorkflowDAGNodeIndex(view, tt.li, tt.ni))
		})
	}
}
