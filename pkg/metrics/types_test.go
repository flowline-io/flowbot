package metrics

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestSanitizeLabel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "no change", input: "archive-items", want: "archive-items"},
		{name: "spaces to underscore", input: "my pipeline", want: "my_pipeline"},
		{name: "special chars", input: "a/b?c:d#e", want: "a_b_c_d_e"},
		{name: "truncate long values", input: strings.Repeat("x", 200), want: strings.Repeat("x", 128)},
		{name: "empty string", input: "", want: ""},
		{name: "chinese characters", input: "管道", want: "__"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := sanitizeLabel(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRecoverLog(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		call func()
	}{
		{
			name: "recovers from panic and logs",
			call: func() {
				defer recoverLog("test_metric_panic")
				panic("label mismatch simulation")
			},
		},
		{
			name: "noop when no panic occurs",
			call: func() {
				defer recoverLog("test_metric_ok")
			},
		},
		{
			name: "collector method with valid labels does not panic",
			call: func() {
				runTotal := prometheus.NewCounterVec(
					prometheus.CounterOpts{Name: "pipeline_run_total_recover_ok_test"},
					[]string{"pipeline", "status"},
				)
				c := &PipelineCollector{runTotal: runTotal}
				c.IncRunTotal("p", "done")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.NotPanics(t, tt.call)
		})
	}
}
