package pipeline

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "valid simple name", input: "my-pipeline", wantErr: false},
		{name: "valid with underscore", input: "my_pipeline_2", wantErr: false},
		{name: "empty name rejected", input: "", wantErr: true},
		{name: "starts with hyphen rejected", input: "-bad", wantErr: true},
		{name: "contains space rejected", input: "bad name", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateName(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestStreamName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		runID int64
		want  string
	}{
		{name: "positive run id", runID: 42, want: "pipeline:run:42"},
		{name: "zero run id", runID: 0, want: "pipeline:run:0"},
		{name: "large run id", runID: 999999, want: "pipeline:run:999999"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, StreamName(tt.runID))
		})
	}
}
