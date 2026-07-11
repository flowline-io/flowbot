package llm

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
)

const (
	streamProgressInterval   = 10 * time.Second
	defaultStreamIdleTimeout = 60 * time.Second
)

// ErrStreamIdle indicates the LLM stream stopped delivering deltas for longer than the idle limit.
var ErrStreamIdle = errors.New("stream idle timeout")

// streamProgressTracker logs stream progress and cancels stalled LLM responses.
type streamProgressTracker struct {
	modelName      string
	idleTimeout    time.Duration
	cancel         context.CancelCauseFunc
	mu             sync.Mutex
	reasoningChars int
	textChars      int
	active         bool
	startedAt      time.Time
	lastDelta      time.Time
	stop           chan struct{}
	stopOnce       sync.Once
	cancelIdleOnce sync.Once
	startOnce      sync.Once
	done           sync.WaitGroup
}

func streamIdleTimeout() time.Duration {
	if d := config.App.ChatAgent.StreamIdleTimeout; d > 0 {
		return d
	}
	return defaultStreamIdleTimeout
}

func newStreamProgressTracker(modelName string, idleTimeout time.Duration, cancel context.CancelCauseFunc) *streamProgressTracker {
	return &streamProgressTracker{
		modelName:   modelName,
		idleTimeout: idleTimeout,
		cancel:      cancel,
		stop:        make(chan struct{}),
	}
}

func (t *streamProgressTracker) recordReasoning(delta string) {
	if delta == "" {
		return
	}
	t.recordDelta(len(delta), 0)
}

func (t *streamProgressTracker) recordText(delta string) {
	if delta == "" {
		return
	}
	t.recordDelta(0, len(delta))
}

func (t *streamProgressTracker) recordDelta(reasoningLen, textLen int) {
	now := time.Now()
	t.mu.Lock()
	if !t.active {
		t.active = true
		t.startedAt = now
	}
	t.reasoningChars += reasoningLen
	t.textChars += textLen
	t.lastDelta = now
	t.mu.Unlock()
}

func (t *streamProgressTracker) begin(ctx context.Context) {
	t.startOnce.Do(func() {
		t.done.Add(1)
		go t.loop(ctx)
	})
}

func (t *streamProgressTracker) loop(ctx context.Context) {
	defer t.done.Done()
	ticker := time.NewTicker(streamProgressInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.stop:
			return
		case <-ticker.C:
			t.tick(false)
		}
	}
}

func (t *streamProgressTracker) tick(final bool) {
	t.mu.Lock()
	if !t.active {
		t.mu.Unlock()
		return
	}
	elapsed := time.Since(t.startedAt).Round(time.Second)
	sinceDelta := time.Since(t.lastDelta)
	reasoningChars := t.reasoningChars
	textChars := t.textChars
	t.mu.Unlock()

	flog.Info("[agent-llm] stream progress model=%s thinking_chars=%d text_chars=%d elapsed=%s since_last_delta=%s final=%t",
		t.modelName, reasoningChars, textChars, elapsed, sinceDelta.Round(time.Second), final)

	if !final && t.idleTimeout > 0 && sinceDelta >= t.idleTimeout {
		t.cancelIdleOnce.Do(func() {
			flog.Warn("[agent-llm] stream idle timeout model=%s thinking_chars=%d text_chars=%d since_last_delta=%s limit=%s",
				t.modelName, reasoningChars, textChars, sinceDelta.Round(time.Second), t.idleTimeout)
			if t.cancel != nil {
				t.cancel(ErrStreamIdle)
			}
		})
	}
}

func (t *streamProgressTracker) end() {
	t.stopOnce.Do(func() { close(t.stop) })
	t.done.Wait()
	t.tick(true)
}

func (t *streamProgressTracker) reasoningCharsForTest() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.reasoningChars
}

func (t *streamProgressTracker) textCharsForTest() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.textChars
}
