package rules

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateCondition(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		condition string
		wantErr   bool
		errSub    string
	}{
		{
			name:      "empty condition is valid",
			condition: "",
			wantErr:   false,
		},
		{
			name:      "simple comparison is valid",
			condition: "time.hour >= 23",
			wantErr:   false,
		},
		{
			name:      "and expression is valid",
			condition: "time.hour >= 22 && time.hour < 6",
			wantErr:   false,
		},
		{
			name:      "or expression is valid",
			condition: "time.hour >= 23 || time.hour < 6",
			wantErr:   false,
		},
		{
			name:      "empty part after or is invalid",
			condition: "time.hour >= 23 ||",
			wantErr:   true,
			errSub:    "empty expression after ||",
		},
		{
			name:      "empty part after and is invalid",
			condition: "time.hour >= 23 &&",
			wantErr:   true,
			errSub:    "empty expression after &&",
		},
		{
			name:      "unknown operator is invalid",
			condition: "time.hour != 10",
			wantErr:   true,
			errSub:    "unknown operator",
		},
		{
			name:      "non time.hour expression is invalid",
			condition: "day.hour >= 10",
			wantErr:   true,
			errSub:    "expected 'time.hour",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateCondition(tt.condition)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errSub)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestValidateTimeExpression(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		expr    string
		wantErr bool
	}{
		{name: "greater than or equal", expr: "time.hour >= 10", wantErr: false},
		{name: "less than", expr: "time.hour < 6", wantErr: false},
		{name: "equals", expr: "time.hour == 14", wantErr: false},
		{name: "greater than", expr: "time.hour > 9", wantErr: false},
		{name: "missing operator", expr: "time.hour 10", wantErr: true},
		{name: "wrong prefix", expr: "hour >= 10", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateTimeExpression(tt.expr)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
