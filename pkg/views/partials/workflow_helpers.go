package partials

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
)

// WorkflowWebPath returns the encoded web UI path for a workflow name.
func WorkflowWebPath(name string) string {
	return "/service/web/workflows/" + url.PathEscape(name)
}

// WorkflowListEntry is one row in the workflows list table.
type WorkflowListEntry struct {
	// Name is the workflow identifier.
	Name string
	// Describe is the optional human description.
	Describe string
	// Enabled reports whether cron/webhook triggers are active.
	Enabled bool
	// Triggers lists configured triggers for badge display.
	Triggers []PipelineTriggerSummary
	// TaskCount is the number of pipeline steps.
	TaskCount int
	// LastRunAt is the latest run started_at when known.
	LastRunAt *time.Time
	// Stats holds recent completed-run latency aggregates when available.
	Stats *types.RunLatencyStats
}

// BuildWorkflowListEntries maps store rows to list entries.
// triggers may be nil; when provided they are grouped by workflow_id.
// lastRunAt maps workflow name to the latest run started_at; missing keys mean never run.
func BuildWorkflowListEntries(defs []model.Workflow, triggers []model.WorkflowTrigger, lastRunAt map[string]time.Time) []WorkflowListEntry {
	byWF := make(map[int64][]model.WorkflowTrigger)
	for _, tr := range triggers {
		byWF[tr.WorkflowID] = append(byWF[tr.WorkflowID], tr)
	}
	entries := make([]WorkflowListEntry, 0, len(defs))
	for _, def := range defs {
		var last *time.Time
		if t, ok := lastRunAt[def.Name]; ok {
			lastCopy := t
			last = &lastCopy
		}
		entries = append(entries, WorkflowListEntry{
			Name:      def.Name,
			Describe:  def.Describe,
			Enabled:   def.Enabled,
			Triggers:  WorkflowTriggerSummaries(byWF[def.ID]),
			TaskCount: len(def.Pipeline),
			LastRunAt: last,
		})
	}
	return entries
}

// WorkflowTriggerSummaries converts stored workflow triggers into list badge summaries.
func WorkflowTriggerSummaries(rows []model.WorkflowTrigger) []PipelineTriggerSummary {
	if len(rows) == 0 {
		return nil
	}
	out := make([]PipelineTriggerSummary, 0, len(rows))
	for _, tr := range rows {
		out = append(out, PipelineTriggerSummary{
			Type:    tr.Type,
			Label:   workflowTriggerLabel(tr),
			Enabled: tr.Enabled,
			Letter:  PipelineTriggerLetter(tr.Type),
		})
	}
	return out
}

func workflowTriggerLabel(tr model.WorkflowTrigger) string {
	switch tr.Type {
	case "manual":
		return "Manual"
	case "cron":
		spec := workflowRuleString(tr.Rule, "cron")
		if spec == "" {
			spec = workflowRuleString(tr.Rule, "expression")
		}
		if spec == "" {
			return "Cron"
		}
		return "Cron: " + spec
	case "webhook":
		path := workflowRuleString(tr.Rule, "path")
		if path == "" {
			return "Webhook"
		}
		return "Webhook: " + path
	default:
		if tr.Type == "" {
			return "Trigger"
		}
		return tr.Type
	}
}

func workflowRuleString(rule map[string]any, key string) string {
	if rule == nil {
		return ""
	}
	v, ok := rule[key]
	if !ok || v == nil {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

// WorkflowWebhookURLPath returns the relative workflow webhook URL path
// (/webhook/workflow/{path}), appending ?token= when auth.token is set.
// Returns empty when the trigger is not a webhook or path is missing.
func WorkflowWebhookURLPath(tr model.WorkflowTrigger) string {
	if !strings.EqualFold(tr.Type, "webhook") {
		return ""
	}
	path := strings.TrimPrefix(workflowRuleString(tr.Rule, "path"), "/")
	if path == "" {
		return ""
	}
	out := "/webhook/workflow/" + path
	if token := workflowRuleAuthString(tr.Rule, "token"); token != "" {
		out += "?token=" + url.QueryEscape(token)
	}
	return out
}

// WorkflowWebhookURL returns the absolute webhook URL when publicOrigin is set,
// otherwise the relative path from WorkflowWebhookURLPath.
func WorkflowWebhookURL(tr model.WorkflowTrigger, publicOrigin string) string {
	path := WorkflowWebhookURLPath(tr)
	if path == "" {
		return ""
	}
	base := strings.TrimRight(strings.TrimSpace(publicOrigin), "/")
	if base == "" {
		return path
	}
	return base + path
}

func workflowRuleAuthString(rule map[string]any, key string) string {
	if rule == nil {
		return ""
	}
	auth, ok := rule["auth"]
	if !ok || auth == nil {
		return ""
	}
	switch m := auth.(type) {
	case map[string]any:
		return workflowRuleString(m, key)
	case types.KV:
		return workflowRuleString(map[string]any(m), key)
	default:
		return ""
	}
}

// WorkflowRunStatusClass returns the flowbot-chip CSS class for a workflow run status.
func WorkflowRunStatusClass(status int) string {
	if c, ok := workflowRunStatusMeta[types.WorkflowRunState(status)]; ok {
		return c.class
	}
	return "flowbot-chip flowbot-chip-muted"
}

// WorkflowRunStatusText returns a short label for a workflow run status.
func WorkflowRunStatusText(status int) string {
	if c, ok := workflowRunStatusMeta[types.WorkflowRunState(status)]; ok {
		return c.text
	}
	return "Unknown"
}

type workflowRunStatusInfo struct {
	class string
	text  string
}

var workflowRunStatusMeta = map[types.WorkflowRunState]workflowRunStatusInfo{
	types.WorkflowRunDone:    {class: "flowbot-chip flowbot-chip-success", text: "Done"},
	types.WorkflowRunFailed:  {class: "flowbot-chip flowbot-chip-error", text: "Failed"},
	types.WorkflowRunRunning: {class: "flowbot-chip flowbot-chip-warning", text: "Running"},
}

// WorkflowRunDuration formats the elapsed time for a workflow run.
func WorkflowRunDuration(r model.WorkflowRun) string {
	end := time.Now()
	if r.CompletedAt != nil {
		end = *r.CompletedAt
	}
	start := r.StartedAt
	if start.IsZero() {
		start = r.CreatedAt
	}
	d := end.Sub(start)
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return d.Round(time.Second).String()
}

// WorkflowStepRunDuration formats the elapsed time for a workflow step run.
func WorkflowStepRunDuration(sr model.WorkflowStepRun) string {
	if sr.CompletedAt == nil {
		return "-"
	}
	d := sr.CompletedAt.Sub(sr.StartedAt)
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return d.Round(time.Second).String()
}

// stepDisplayName prefers StepName, falling back to StepID.
func stepDisplayName(s model.WorkflowStepRun) string {
	if s.StepName != "" {
		return s.StepName
	}
	return s.StepID
}

// WorkflowStepKey returns a stable identifier for step-run test ids and UI keys.
func WorkflowStepKey(s model.WorkflowStepRun) string {
	if s.StepID != "" {
		return s.StepID
	}
	return s.StepName
}

// workflowStepHasDetail reports whether a step run should render an expandable detail row.
func workflowStepHasDetail(s model.WorkflowStepRun) bool {
	return len(s.Params) > 0 || len(s.Result) > 0 || s.Error != "" || types.WorkflowRunState(s.Status) == types.WorkflowRunFailed
}

// workflowStepDetailOpen reports whether the detail row should start expanded (failed steps).
func workflowStepDetailOpen(s model.WorkflowStepRun) bool {
	return s.Error != "" || types.WorkflowRunState(s.Status) == types.WorkflowRunFailed
}

// WorkflowDAGNode is one task box in the workflow DAG view.
type WorkflowDAGNode struct {
	// ID is the task identifier.
	ID string
	// Describe is the optional human description.
	Describe string
	// Action is the task action string.
	Action string
	// Deps lists upstream task IDs (Conn).
	Deps []string
}

// WorkflowDAGView is a layered layout of workflow tasks for read-only display.
type WorkflowDAGView struct {
	// Layers groups tasks by topological depth (roots first).
	Layers [][]WorkflowDAGNode
	// Sequential reports whether the graph is a linear pipeline (no branching Conn).
	Sequential bool
	// ParallelRuntime reports whether max_concurrency enables parallel DAG execution.
	ParallelRuntime bool
}

// BuildWorkflowDAGView builds a layered DAG view from tasks and pipeline order.
// When no Conn edges exist, pipeline order (falling back to task order) is shown as a chain.
// ParallelRuntime is true only when maxConcurrency > 1 (matching the runner).
func BuildWorkflowDAGView(tasks []types.WorkflowTask, pipeline []string, maxConcurrency int) WorkflowDAGView {
	if len(tasks) == 0 {
		return WorkflowDAGView{}
	}
	byID := make(map[string]types.WorkflowTask, len(tasks))
	hasConn := false
	for _, t := range tasks {
		byID[t.ID] = t
		if len(t.Conn) > 0 {
			hasConn = true
		}
	}
	var view WorkflowDAGView
	if !hasConn {
		view = buildSequentialDAGView(tasks, pipeline, byID)
	} else {
		view = buildConnDAGView(tasks, byID)
	}
	view.ParallelRuntime = maxConcurrency > 1
	return view
}

func buildSequentialDAGView(tasks []types.WorkflowTask, pipeline []string, byID map[string]types.WorkflowTask) WorkflowDAGView {
	order := make([]string, 0, len(tasks))
	seen := make(map[string]bool, len(tasks))
	for _, id := range pipeline {
		if _, ok := byID[id]; !ok || seen[id] {
			continue
		}
		order = append(order, id)
		seen[id] = true
	}
	for _, t := range tasks {
		if seen[t.ID] {
			continue
		}
		order = append(order, t.ID)
		seen[t.ID] = true
	}
	layers := make([][]WorkflowDAGNode, 0, len(order))
	for i, id := range order {
		t := byID[id]
		deps := t.Conn
		if len(deps) == 0 && i > 0 {
			deps = []string{order[i-1]}
		}
		layers = append(layers, []WorkflowDAGNode{dagNodeFromTask(t, deps)})
	}
	return WorkflowDAGView{Layers: layers, Sequential: true}
}

func buildConnDAGView(tasks []types.WorkflowTask, byID map[string]types.WorkflowTask) WorkflowDAGView {
	level := workflowDAGLevels(tasks, byID)
	maxLevel := 0
	for _, t := range tasks {
		if level[t.ID] > maxLevel {
			maxLevel = level[t.ID]
		}
	}
	layers := make([][]WorkflowDAGNode, maxLevel+1)
	for _, t := range tasks {
		lv := level[t.ID]
		layers[lv] = append(layers[lv], dagNodeFromTask(t, t.Conn))
	}
	return WorkflowDAGView{Layers: layers, Sequential: false}
}

// workflowDAGLevels assigns topological depth to each task using Conn edges.
func workflowDAGLevels(tasks []types.WorkflowTask, byID map[string]types.WorkflowTask) map[string]int {
	children := make(map[string][]string, len(tasks))
	inDegree := make(map[string]int, len(tasks))
	for _, t := range tasks {
		inDegree[t.ID] = 0
	}
	for _, t := range tasks {
		for _, dep := range t.Conn {
			if _, ok := byID[dep]; !ok {
				continue
			}
			children[dep] = append(children[dep], t.ID)
			inDegree[t.ID]++
		}
	}

	level := make(map[string]int, len(tasks))
	queue := make([]string, 0, len(tasks))
	for _, t := range tasks {
		if inDegree[t.ID] == 0 {
			queue = append(queue, t.ID)
			level[t.ID] = 0
		}
	}
	for i := 0; i < len(queue); i++ {
		u := queue[i]
		for _, v := range children[u] {
			next := level[u] + 1
			if cur, ok := level[v]; !ok || next > cur {
				level[v] = next
			}
			inDegree[v]--
			if inDegree[v] == 0 {
				queue = append(queue, v)
			}
		}
	}
	for _, t := range tasks {
		if _, ok := level[t.ID]; !ok {
			level[t.ID] = 0
		}
	}
	return level
}

func dagNodeFromTask(t types.WorkflowTask, deps []string) WorkflowDAGNode {
	outDeps := append([]string(nil), deps...)
	return WorkflowDAGNode{
		ID:       t.ID,
		Describe: t.Describe,
		Action:   t.Action,
		Deps:     outDeps,
	}
}

// WorkflowDAGNodeIndex returns a 1-based display index for a node in the layered view.
func WorkflowDAGNodeIndex(view WorkflowDAGView, layerIdx, nodeIdx int) int {
	if layerIdx < 0 || nodeIdx < 0 || layerIdx >= len(view.Layers) {
		return 0
	}
	n := 1
	for i := range layerIdx {
		n += len(view.Layers[i])
	}
	return n + nodeIdx
}

// workflowDAGRepeat returns a slice used to render n connector stems in templ.
func workflowDAGRepeat(n int) []struct{} {
	if n < 1 {
		n = 1
	}
	return make([]struct{}, n)
}

// workflowDAGConnectorStyle returns CSS variables shared by layer and connector grids.
func workflowDAGConnectorStyle(fromCount, toCount int) string {
	fromCount, toCount, rail := workflowDAGNormalizeCounts(fromCount, toCount)
	return fmt.Sprintf("--from-cols: %d; --to-cols: %d; --rail-cols: %d", fromCount, toCount, rail)
}

// workflowDAGConnectorClass returns connector classes, including single-rail mode.
func workflowDAGConnectorClass(fromCount, toCount int) string {
	_, _, rail := workflowDAGNormalizeCounts(fromCount, toCount)
	if rail == 1 {
		return "workflow-dag-connector workflow-dag-connector-single"
	}
	return "workflow-dag-connector"
}

// workflowDAGNormalizeCounts clamps layer sizes and returns the shared rail column count.
func workflowDAGNormalizeCounts(fromCount, toCount int) (int, int, int) {
	if fromCount < 1 {
		fromCount = 1
	}
	if toCount < 1 {
		toCount = 1
	}
	rail := max(toCount, fromCount)
	return fromCount, toCount, rail
}

func workflowDAGNodeTitle(node WorkflowDAGNode) string {
	if len(node.Deps) == 0 {
		return node.ID
	}
	return node.ID + " ← " + strings.Join(node.Deps, ", ")
}

// WorkflowInputDefaultString formats an input default for form placeholders.
func WorkflowInputDefaultString(def types.WorkflowInputDef) string {
	if def.Default == nil {
		return ""
	}
	switch v := def.Default.(type) {
	case string:
		return v
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		b, err := sonic.Marshal(def.Default)
		if err != nil {
			return fmt.Sprint(def.Default)
		}
		return string(b)
	}
}

// WorkflowInputIsBoolean reports whether the input type is boolean.
func WorkflowInputIsBoolean(typ string) bool {
	return typ == types.WorkflowInputTypeBoolean
}

// WorkflowInputIsJSON reports whether the input type is json.
func WorkflowInputIsJSON(typ string) bool {
	return typ == types.WorkflowInputTypeJSON
}

// WorkflowInputIsNumber reports whether the input type is number.
func WorkflowInputIsNumber(typ string) bool {
	return typ == types.WorkflowInputTypeNumber
}
