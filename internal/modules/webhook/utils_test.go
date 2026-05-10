package webhook

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/internal/store/model"
)

func TestStateStr(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		state model.WebhookState
		want  string
	}{
		{
			name:  "active state returns active",
			state: model.WebhookActive,
			want:  "active",
		},
		{
			name:  "inactive state returns inactive",
			state: model.WebhookInactive,
			want:  "inactive",
		},
		{
			name:  "unknown state returns unknown",
			state: model.WebhookState(99),
			want:  "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, stateStr(tt.state))
		})
	}
}
