package loopdetect_test

import (
	"fmt"
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/loopdetect"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func textResult(text string, isErr bool) msg.ToolResultMessage {
	return msg.ToolResultMessage{
		Parts:   []msg.ContentPart{msg.TextPart{Text: text}},
		IsError: isErr,
	}
}

func observeN(d *loopdetect.Detector, tool string, args map[string]any, result msg.ToolResultMessage, n int) {
	for range n {
		d.ObserveResult(tool, args, result)
	}
}

func TestDetectorGenericRepeat(t *testing.T) {
	cfg := loopdetect.DefaultConfig()
	cfg.GenericWarn = 3
	cfg.GenericCritical = 5
	d := loopdetect.NewDetector(cfg)
	args := map[string]any{"path": "a.go"}

	observeN(d, "read_file", args, textResult("ok", false), 2)
	dec := d.Check("read_file", args)
	assert.Equal(t, loopdetect.LevelWarn, dec.Level)
	assert.Equal(t, loopdetect.DetectorGenericRepeat, dec.Detector)

	observeN(d, "read_file", args, textResult("ok", false), 2)
	dec = d.Check("read_file", args)
	assert.Equal(t, loopdetect.LevelCritical, dec.Level)
	assert.False(t, dec.Terminate)
}

func TestDetectorNoProgress(t *testing.T) {
	cfg := loopdetect.DefaultConfig()
	cfg.NoProgressCritical = 3
	cfg.GenericCritical = 100
	cfg.GenericWarn = 100
	d := loopdetect.NewDetector(cfg)
	args := map[string]any{"command": "ls"}
	same := textResult("exit=0 same", false)

	observeN(d, "run_terminal", args, same, 2) // 2 completed + pending = 3
	dec := d.Check("run_terminal", args)
	assert.Equal(t, loopdetect.LevelCritical, dec.Level)
	assert.Equal(t, loopdetect.DetectorNoProgress, dec.Detector)
	assert.Equal(t, 3, dec.Count)

	// Changing result breaks the streak.
	d.ObserveResult("run_terminal", args, textResult("exit=0 different", false))
	dec = d.Check("run_terminal", args)
	assert.Equal(t, loopdetect.LevelNone, dec.Level)
}

func TestDetectorGlobalCircuitBreaker(t *testing.T) {
	cfg := loopdetect.DefaultConfig()
	cfg.GlobalCircuitBreaker = 4
	cfg.NoProgressCritical = 100
	cfg.GenericCritical = 100
	cfg.GenericWarn = 100
	d := loopdetect.NewDetector(cfg)
	args := map[string]any{"command": "true"}
	observeN(d, "run_terminal", args, textResult("ok", false), 3) // 3 completed + pending = 4
	dec := d.Check("run_terminal", args)
	assert.Equal(t, loopdetect.LevelCritical, dec.Level)
	assert.Equal(t, loopdetect.DetectorGlobalCircuitBreaker, dec.Detector)
	assert.True(t, dec.Terminate)
	assert.Equal(t, 4, dec.Count)
}

func TestDetectorPingPong(t *testing.T) {
	cfg := loopdetect.DefaultConfig()
	cfg.PingPongWarn = 4
	cfg.PingPongCritical = 6
	cfg.GenericWarn = 100
	cfg.GenericCritical = 100
	cfg.NoProgressCritical = 100
	d := loopdetect.NewDetector(cfg)
	a := map[string]any{"path": "a"}
	b := map[string]any{"path": "b"}
	ra := textResult("A", false)
	rb := textResult("B", false)

	// History: A B A B A  — pending B continues alternation (count 5+1=6)
	seq := []struct {
		args map[string]any
		res  msg.ToolResultMessage
	}{
		{a, ra}, {b, rb}, {a, ra}, {b, rb}, {a, ra},
	}
	for _, step := range seq {
		d.ObserveResult("read_file", step.args, step.res)
	}
	dec := d.Check("read_file", b)
	assert.Equal(t, loopdetect.LevelCritical, dec.Level)
	assert.Equal(t, loopdetect.DetectorPingPong, dec.Detector)
}

func TestDetectorPostCompaction(t *testing.T) {
	cfg := loopdetect.DefaultConfig()
	cfg.PostCompactionIdentical = 2
	cfg.GenericCritical = 100
	cfg.GenericWarn = 100
	cfg.NoProgressCritical = 100
	cfg.GlobalCircuitBreaker = 100
	d := loopdetect.NewDetector(cfg)
	args := map[string]any{"path": "x"}
	res := textResult("same", false)

	d.ArmPostCompaction()
	observeN(d, "read_file", args, res, 2)
	dec := d.Check("read_file", args)
	assert.Equal(t, loopdetect.LevelCritical, dec.Level)
	assert.Equal(t, loopdetect.DetectorPostCompaction, dec.Detector)
	assert.True(t, dec.Terminate)

	d2 := loopdetect.NewDetector(cfg)
	observeN(d2, "read_file", args, res, 2)
	dec = d2.Check("read_file", args)
	assert.Equal(t, loopdetect.LevelNone, dec.Level, "unarmed must not fire post-compaction")

	// Pre-arm identical history must not count after arm.
	d3 := loopdetect.NewDetector(cfg)
	observeN(d3, "read_file", args, res, 2)
	d3.ArmPostCompaction()
	dec = d3.Check("read_file", args)
	assert.Equal(t, loopdetect.LevelNone, dec.Level, "pre-arm history must not trigger post-compaction")
	observeN(d3, "read_file", args, res, 2)
	dec = d3.Check("read_file", args)
	assert.Equal(t, loopdetect.LevelCritical, dec.Level)
	assert.Equal(t, loopdetect.DetectorPostCompaction, dec.Detector)
}

func TestDetectorResetFingerprint(t *testing.T) {
	cfg := loopdetect.DefaultConfig()
	cfg.GenericCritical = 3
	cfg.GenericWarn = 2
	d := loopdetect.NewDetector(cfg)
	args := map[string]any{"path": "x"}
	observeN(d, "read_file", args, textResult("ok", false), 2)
	dec := d.Check("read_file", args)
	require.Equal(t, loopdetect.LevelCritical, dec.Level)

	d.ResetFingerprint(dec.ArgsHash)
	dec = d.Check("read_file", args)
	assert.Equal(t, loopdetect.LevelNone, dec.Level)
}

func TestDetectorDisabled(t *testing.T) {
	cfg := loopdetect.DefaultConfig()
	cfg.Enabled = false
	cfg.GenericCritical = 1
	d := loopdetect.NewDetector(cfg)
	args := map[string]any{"path": "x"}
	observeN(d, "read_file", args, textResult("ok", false), 5)
	dec := d.Check("read_file", args)
	assert.Equal(t, loopdetect.LevelNone, dec.Level)
}

func TestArgsHashStable(t *testing.T) {
	a := loopdetect.ArgsHash("t", map[string]any{"b": 1, "a": 2})
	b := loopdetect.ArgsHash("t", map[string]any{"a": 2, "b": 1})
	assert.Equal(t, a, b)
	assert.NotEqual(t, a, loopdetect.ArgsHash("t", map[string]any{"a": 3, "b": 1}))
}

func TestResultHashStripsVolatileShellMeta(t *testing.T) {
	r1 := textResult("exit_code=0 pid=123 duration=45ms hello", false)
	r2 := textResult("exit_code=0 pid=999 duration=1ms hello", false)
	h1 := loopdetect.ResultHash("run_terminal", r1)
	h2 := loopdetect.ResultHash("run_terminal", r2)
	assert.Equal(t, h1, h2)

	r3 := textResult("exit_code=1 pid=123 duration=45ms hello", false)
	assert.NotEqual(t, h1, loopdetect.ResultHash("run_terminal", r3))
}

func TestDetectorWindowTrim(t *testing.T) {
	cfg := loopdetect.DefaultConfig()
	cfg.Window = 3
	cfg.GenericWarn = 100
	cfg.GenericCritical = 100
	d := loopdetect.NewDetector(cfg)
	for i := range 5 {
		d.ObserveResult("read_file", map[string]any{"path": fmt.Sprintf("%d", i)}, textResult("x", false))
	}
	// Only last 3 remain; first path should not contribute to generic count.
	dec := d.Check("read_file", map[string]any{"path": "0"})
	assert.Equal(t, loopdetect.LevelNone, dec.Level)
}
