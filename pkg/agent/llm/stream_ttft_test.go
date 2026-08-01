package llm

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStreamStartTrackerMark(t *testing.T) {
	tests := []struct {
		name        string
		marks       int
		wantStarted bool
		wantRecords int
	}{
		{name: "no mark leaves unstarted", marks: 0, wantStarted: false, wantRecords: 0},
		{name: "first mark records ttft", marks: 1, wantStarted: true, wantRecords: 1},
		{name: "second mark is idempotent", marks: 2, wantStarted: true, wantRecords: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			type observation struct {
				model   string
				seconds float64
			}
			var got []observation
			prev := recordLLMTTFT
			t.Cleanup(func() { recordLLMTTFT = prev })
			recordLLMTTFT = func(model string, seconds float64) {
				got = append(got, observation{model: model, seconds: seconds})
			}

			tracker := &streamStartTracker{
				start:     time.Now().Add(-20 * time.Millisecond),
				modelName: "probe-model",
			}
			for range tt.marks {
				tracker.mark()
			}

			assert.Equal(t, tt.wantStarted, tracker.hasStarted())
			require.Len(t, got, tt.wantRecords)
			if tt.wantRecords > 0 {
				assert.Equal(t, "probe-model", got[0].model)
				assert.GreaterOrEqual(t, got[0].seconds, 0.015)
			}
		})
	}
}
