package llm_test

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/llms"
)

type stubModel struct{}

func (stubModel) GenerateContent(_ context.Context, _ []llms.MessageContent, _ ...llms.CallOption) (*llms.ContentResponse, error) {
	return &llms.ContentResponse{}, nil
}

func (stubModel) Call(_ context.Context, _ string, _ ...llms.CallOption) (string, error) {
	return "", nil
}

func TestGetOrCreateModelReusesCachedInstance(t *testing.T) {
	tests := []struct {
		name      string
		modelName string
		calls     int
	}{
		{name: "single lookup", modelName: "cached-model-a", calls: 1},
		{name: "repeated lookup", modelName: "cached-model-b", calls: 3},
		{name: "another model name", modelName: "cached-model-c", calls: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			llm.ResetModelPoolForTest()
			var created int32
			llm.SetModelCreatorForTest(func(_ context.Context, modelName string) (llms.Model, string, error) {
				atomic.AddInt32(&created, 1)
				return stubModel{}, modelName, nil
			})
			t.Cleanup(llm.ResetModelPoolForTest)

			var first llms.Model
			for i := 0; i < tt.calls; i++ {
				model, name, err := llm.GetOrCreateModel(context.Background(), tt.modelName)
				require.NoError(t, err)
				assert.Equal(t, tt.modelName, name)
				if i == 0 {
					first = model
					continue
				}
				assert.Equal(t, first, model)
			}
			assert.EqualValues(t, 1, atomic.LoadInt32(&created))
		})
	}
}
