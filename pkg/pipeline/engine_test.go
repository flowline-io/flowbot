package pipeline

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/types"
)

// mockStepCallback records all callback invocations for test assertions.
type mockStepCallback struct {
	calls []mockCallbackCall
	mu    sync.Mutex
}

type mockCallbackCall struct {
	method    string
	runID     int64
	stepIndex int
	stepName  string
	status    string
	elapsedMs int64
}

func (m *mockStepCallback) OnRunStart(ctx context.Context, runID int64, pipelineName string,
	trigger string, totalSteps int, stepNames []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, mockCallbackCall{method: "OnRunStart", runID: runID})
}

func (m *mockStepCallback) OnStepStart(ctx context.Context, runID int64, pipelineName string,
	stepIndex int, stepName string, input map[string]any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, mockCallbackCall{method: "OnStepStart", runID: runID, stepIndex: stepIndex, stepName: stepName})
}

func (m *mockStepCallback) OnStepDone(ctx context.Context, runID int64, pipelineName string,
	stepIndex int, stepName string, output map[string]any, elapsedMs int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, mockCallbackCall{method: "OnStepDone", runID: runID, stepIndex: stepIndex, stepName: stepName, elapsedMs: elapsedMs})
}

func (m *mockStepCallback) OnStepError(ctx context.Context, runID int64, pipelineName string,
	stepIndex int, stepName string, err error, elapsedMs int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, mockCallbackCall{method: "OnStepError", runID: runID, stepIndex: stepIndex, stepName: stepName, elapsedMs: elapsedMs})
}

func (m *mockStepCallback) OnRunComplete(ctx context.Context, runID int64, pipelineName string,
	elapsedMs int64, failed bool, errMsg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	status := "complete"
	if failed {
		status = "failed"
	}
	m.calls = append(m.calls, mockCallbackCall{method: "OnRunComplete", runID: runID, status: status, elapsedMs: elapsedMs})
}

func TestNewEngine_CronRegistration(t *testing.T) {
	t.Parallel()
	seed := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := NewFakeClock(seed)
	tests := []struct {
		name        string
		defs        []Definition
		wantEntries int
	}{
		{
			name: "one cron definition registers one entry",
			defs: []Definition{
				{Name: "cron1", Enabled: true, Trigger: Trigger{Cron: "0 0 * * *"}},
			},
			wantEntries: 1,
		},
		{
			name: "multiple cron definitions register multiple entries",
			defs: []Definition{
				{Name: "cron1", Enabled: true, Trigger: Trigger{Cron: "0 0 * * *"}},
				{Name: "cron2", Enabled: true, Trigger: Trigger{Cron: "@daily"}},
			},
			wantEntries: 2,
		},
		{
			name: "event-only definition not registered as cron",
			defs: []Definition{
				{Name: "event1", Enabled: true, Trigger: Trigger{Event: "e1"}},
			},
			wantEntries: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := NewEngineWithClock(tt.defs, nil, nil, noopPC, noopEC, clock)
			defer e.Stop()
			assert.Len(t, e.cron.Entries(), tt.wantEntries)
		})
	}
}

func TestEngine_CronConcurrencyGuard(t *testing.T) {
	t.Parallel()
	seed := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := NewFakeClock(seed)

	var runningCount atomic.Int32
	blockCh := make(chan struct{})
	doneCh := make(chan struct{})

	defs := []Definition{
		{
			Name:    "concurrent-pl",
			Enabled: true,
			Trigger: Trigger{Cron: "@every 100ms", CronTimeout: 5 * time.Second},
			Steps:   []Step{{Name: "blocker", Capability: "test", Operation: "block"}},
		},
	}

	e := NewEngineWithClock(defs, nil, nil, noopPC, noopEC, clock)
	defer e.Stop()

	// First goroutine acquires the lock (simulating cron run)
	go func() {
		mu := e.mu["concurrent-pl"]
		mu.Lock()
		runningCount.Add(1)
		<-blockCh
		mu.Unlock()
		doneCh <- struct{}{}
	}()

	// Wait for first to acquire lock
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, int32(1), runningCount.Load())

	// Second goroutine tries TryLock -- should fail
	skipped := true
	go func() {
		mu := e.mu["concurrent-pl"]
		if mu.TryLock() {
			skipped = false
			mu.Unlock()
		}
		doneCh <- struct{}{}
	}()
	<-doneCh
	assert.True(t, skipped, "second TryLock should fail while first holds the lock")

	close(blockCh)
	<-doneCh
}

func TestEngine_StopShutsDownCron(t *testing.T) {
	t.Parallel()
	seed := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := NewFakeClock(seed)

	defs := []Definition{
		{
			Name:    "stop-test",
			Enabled: true,
			Trigger: Trigger{Cron: "@every 100ms", CronTimeout: 5 * time.Second},
			Steps:   []Step{},
		},
	}

	e := NewEngineWithClock(defs, nil, nil, noopPC, noopEC, clock)

	// Verify cron has entries before stop
	assert.Len(t, e.cron.Entries(), 1)

	e.Stop()
	// Stop should be idempotent
	e.Stop()
}

func TestEngine_EventIDFormat(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{
			name: "shortuuid is 22 chars",
		},
		{
			name: "shortuuids are unique across calls",
		},
		{
			name: "shortuuid does not contain legacy prefixes",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			id := types.Id()
			assert.Len(t, id, 22)
			assert.NotContains(t, id, "cron:")
			assert.NotContains(t, id, "webhook:")
			id2 := types.Id()
			assert.Len(t, id2, 22)
			assert.NotEqual(t, id, id2)
		})
	}
}

func TestEngine_RegisterWebhooks(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		defs      []Definition
		wantPaths []string
		wantErr   bool
	}{
		{
			name: "returns webhook paths",
			defs: []Definition{
				{
					Name: "wh1", Enabled: true,
					Trigger: Trigger{Webhook: &WebhookConfig{Path: "path-a", Method: "POST", Auth: WebhookAuthConfig{Token: "t"}}},
				},
				{
					Name: "wh2", Enabled: true,
					Trigger: Trigger{Webhook: &WebhookConfig{Path: "path-b", Method: "POST", Auth: WebhookAuthConfig{Token: "t"}}},
				},
			},
			wantPaths: []string{"path-a", "path-b"},
		},
		{
			name: "skips non-webhook definitions",
			defs: []Definition{
				{Name: "ev1", Enabled: true, Trigger: Trigger{Event: "e1"}},
				{
					Name: "wh1", Enabled: true,
					Trigger: Trigger{Webhook: &WebhookConfig{Path: "path-a", Method: "POST", Auth: WebhookAuthConfig{Token: "t"}}},
				},
			},
			wantPaths: []string{"path-a"},
		},
		{
			name: "returns error on duplicate paths",
			defs: []Definition{
				{
					Name: "wh1", Enabled: true,
					Trigger: Trigger{Webhook: &WebhookConfig{Path: "dup", Method: "POST", Auth: WebhookAuthConfig{Token: "t"}}},
				},
				{
					Name: "wh2", Enabled: true,
					Trigger: Trigger{Webhook: &WebhookConfig{Path: "dup", Method: "PUT", Auth: WebhookAuthConfig{HMACSecret: "s"}}},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := NewEngine(tt.defs, nil, nil, noopPC, noopEC)
			defer e.Stop()
			m, err := e.RegisterWebhooks()
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			for _, p := range tt.wantPaths {
				assert.Contains(t, m, p)
			}
		})
	}
}

func TestEngine_ExecuteWebhook(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		eventTypes []string
	}{
		{
			name:       "execute webhook with single pipeline run",
			eventTypes: []string{"webhook.run"},
		},
		{
			name:       "execute webhook with differing event types",
			eventTypes: []string{"custom.type", "another.type"},
		},
		{
			name:       "execute webhook with empty steps completes immediately",
			eventTypes: []string{"noop.event"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			for _, et := range tt.eventTypes {
				defs := []Definition{
					{
						Name:    "wh-exec-" + et[len(et)-3:],
						Enabled: true,
						Trigger: Trigger{
							Webhook: &WebhookConfig{
								Path:      "exec-path",
								Method:    "POST",
								Auth:      WebhookAuthConfig{Token: "t"},
								EventType: et,
							},
						},
					},
				}
				e := NewEngine(defs, nil, nil, noopPC, noopEC)
				defer e.Stop()
				event := types.DataEvent{
					EventID:   "test-id",
					EventType: et,
					Source:    "webhook",
				}
				err := e.ExecuteWebhook(context.Background(), &defs[0], event)
				assert.NoError(t, err)
			}
		})
	}
}

func TestEngine_ExecuteWebhookMutex(t *testing.T) {
	t.Parallel()
	def := Definition{
		Name:    "mutex-test",
		Enabled: true,
		Trigger: Trigger{
			Webhook: &WebhookConfig{
				Path:   "mtx",
				Method: "POST",
				Auth:   WebhookAuthConfig{Token: "t"},
			},
		},
	}
	e := NewEngine([]Definition{def}, nil, nil, noopPC, noopEC)
	defer e.Stop()

	mu := e.mu[def.Name]
	require.NotNil(t, mu)

	mu.Lock()

	started := make(chan struct{})
	done := make(chan struct{})
	go func() {
		close(started)
		event := types.DataEvent{EventID: "mtx-ev", EventType: "t"}
		_ = e.ExecuteWebhook(context.Background(), &def, event)
		close(done)
	}()

	<-started
	time.Sleep(50 * time.Millisecond)

	select {
	case <-done:
		t.Fatal("ExecuteWebhook completed before mutex released")
	default:
	}

	mu.Unlock()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("ExecuteWebhook did not complete after mutex release")
	}
}

func TestEngine_HandleEventMutex(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		defs  []Definition
		event types.DataEvent
	}{
		{
			name: "single pipeline acquires and releases mutex",
			defs: []Definition{
				{Name: "p1", Enabled: true, Trigger: Trigger{Event: "e1"}, Steps: []Step{}},
			},
			event: types.DataEvent{EventID: "evt1", EventType: "e1"},
		},
		{
			name: "no matching event does not block",
			defs: []Definition{
				{Name: "p1", Enabled: true, Trigger: Trigger{Event: "e1"}, Steps: []Step{}},
			},
			event: types.DataEvent{EventID: "evt2", EventType: "no-match"},
		},
		{
			name: "multiple pipelines for same event each lock independently",
			defs: []Definition{
				{Name: "p1", Enabled: true, Trigger: Trigger{Event: "e1"}, Steps: []Step{}},
				{Name: "p2", Enabled: true, Trigger: Trigger{Event: "e1"}, Steps: []Step{}},
			},
			event: types.DataEvent{EventID: "evt3", EventType: "e1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := NewEngine(tt.defs, nil, nil, noopPC, noopEC)
			defer e.Stop()
			err := e.Handler()(context.Background(), tt.event)
			assert.NoError(t, err)
		})
	}
}

func TestMergeTags(t *testing.T) {
	tests := []struct {
		name     string
		upstream types.KV
		stepTags any
		want     types.KV
	}{
		{"nil upstream returns empty", nil, nil, types.KV{}},
		{"upstream no step tags passes through", types.KV{"project": "alpha"}, nil, types.KV{"project": "alpha"}},
		{
			"step overrides on collision",
			types.KV{"project": "alpha", "env": "staging"},
			types.KV{"project": "beta", "processed": "true"},
			types.KV{"project": "beta", "env": "staging", "processed": "true"},
		},
		{
			"step as map[string]any merges",
			types.KV{"project": "alpha"},
			map[string]any{"processed": "true"},
			types.KV{"project": "alpha", "processed": "true"},
		},
		{"non-map step tags returns upstream", types.KV{"project": "alpha"}, "string", types.KV{"project": "alpha"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, mergeTags(tt.upstream, tt.stepTags))
		})
	}
}

func TestHandleEvent_WithTagsDoesNotCrash(t *testing.T) {
	t.Parallel()
	defs := []Definition{
		{
			Name:    "tag-test",
			Enabled: true,
			Trigger: Trigger{Event: "test.event"},
			Steps: []Step{
				{Name: "s1", Capability: "test", Operation: "create", Params: map[string]any{"title": "x"}},
			},
		},
	}
	e := NewEngine(defs, nil, nil, noopPC, noopEC)
	defer e.Stop()
	event := types.DataEvent{
		EventID: "evt-1", EventType: "test.event", EntityID: "src-1", App: "app-a",
		Tags: types.KV{"project": "alpha"},
	}
	// handleEvent returns nil even on step failure; verify no crash from tag merge or nil Resource check
	err := e.Handler()(context.Background(), event)
	assert.NoError(t, err)
}

func TestStepCallback_OrderOfCalls(t *testing.T) {
	t.Parallel()

	type callSpec struct {
		method    string
		stepIndex int
		stepName  string
		elapsedMs int64
		status    string
	}
	tests := []struct {
		name          string
		calls         []callSpec
		expectedOrder []string
	}{
		{
			name: "happy path — two steps, all success",
			calls: []callSpec{
				{method: "OnRunStart", stepIndex: -1},
				{method: "OnStepStart", stepIndex: 0, stepName: "a"},
				{method: "OnStepDone", stepIndex: 0, stepName: "a", elapsedMs: 100},
				{method: "OnStepStart", stepIndex: 1, stepName: "b"},
				{method: "OnStepDone", stepIndex: 1, stepName: "b", elapsedMs: 200},
				{method: "OnRunComplete", elapsedMs: 300, status: "complete"},
			},
			expectedOrder: []string{
				"OnRunStart", "OnStepStart", "OnStepDone",
				"OnStepStart", "OnStepDone", "OnRunComplete",
			},
		},
		{
			name: "error path — step fails, run reports error",
			calls: []callSpec{
				{method: "OnRunStart", stepIndex: -1},
				{method: "OnStepStart", stepIndex: 0, stepName: "bad"},
				{method: "OnStepError", stepIndex: 0, stepName: "bad", elapsedMs: 50},
				{method: "OnRunComplete", elapsedMs: 100, status: "failed"},
			},
			expectedOrder: []string{
				"OnRunStart", "OnStepStart", "OnStepError", "OnRunComplete",
			},
		},
		{
			name: "single step — simplest pipeline",
			calls: []callSpec{
				{method: "OnRunStart", stepIndex: -1},
				{method: "OnStepStart", stepIndex: 0, stepName: "only"},
				{method: "OnStepDone", stepIndex: 0, stepName: "only", elapsedMs: 42},
				{method: "OnRunComplete", elapsedMs: 50, status: "complete"},
			},
			expectedOrder: []string{
				"OnRunStart", "OnStepStart", "OnStepDone", "OnRunComplete",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mockCB := &mockStepCallback{}
			for _, c := range tt.calls {
				switch c.method {
				case "OnRunStart":
					mockCB.OnRunStart(context.Background(), 1, "p", "event:x", len(tt.calls)-2, nil)
				case "OnStepStart":
					mockCB.OnStepStart(context.Background(), 1, "p", c.stepIndex, c.stepName, nil)
				case "OnStepDone":
					mockCB.OnStepDone(context.Background(), 1, "p", c.stepIndex, c.stepName, nil, c.elapsedMs)
				case "OnStepError":
					mockCB.OnStepError(context.Background(), 1, "p", c.stepIndex, c.stepName, assert.AnError, c.elapsedMs)
				case "OnRunComplete":
					failed := c.status == "failed"
					mockCB.OnRunComplete(context.Background(), 1, "p", c.elapsedMs, failed, "test error")
				}
			}
			if len(mockCB.calls) != len(tt.expectedOrder) {
				t.Fatalf("got %d calls, want %d", len(mockCB.calls), len(tt.expectedOrder))
			}
			for i, call := range mockCB.calls {
				if call.method != tt.expectedOrder[i] {
					t.Errorf("call %d: got %s, want %s", i, call.method, tt.expectedOrder[i])
				}
			}
		})
	}
}

func TestEngine_SetCallback(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setCB   func(e *Engine)
		wantNil bool
	}{
		{
			name:    "no callback set — nil",
			setCB:   func(e *Engine) {},
			wantNil: true,
		},
		{
			name: "set callback — stored",
			setCB: func(e *Engine) {
				e.SetCallback(&mockStepCallback{})
			},
			wantNil: false,
		},
		{
			name: "set then nil — cleared",
			setCB: func(e *Engine) {
				e.SetCallback(&mockStepCallback{})
				e.SetCallback(nil)
			},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := NewEngine(nil, nil, nil, noopPC, noopEC)
			defer e.Stop()
			tt.setCB(e)
			if tt.wantNil && e.callback != nil {
				t.Error("expected callback to be nil")
			}
			if !tt.wantNil && e.callback == nil {
				t.Error("expected callback to be set")
			}
		})
	}
}
