package chatagent_test

import (
	"testing"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/pkg/agent/approval"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseApprovalBody(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    approval.Mode
		wantErr bool
	}{
		{name: "auto", raw: `{"mode":"auto"}`, want: approval.ModeAuto},
		{name: "manual", raw: `{"mode":"manual"}`, want: approval.ModeManual},
		{name: "off", raw: `{"mode":"off"}`, want: approval.ModeOff},
		{name: "invalid", raw: `{"mode":"smart"}`, wantErr: true},
		{name: "bad json", raw: `{`, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := chatagent.ParseApprovalBody([]byte(tt.raw))
			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, types.ErrInvalidArgument)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseApprovalMode(t *testing.T) {
	mode, err := chatagent.ParseApprovalMode("auto")
	require.NoError(t, err)
	assert.Equal(t, approval.ModeAuto, mode)
	_, err = chatagent.ParseApprovalMode("smart")
	require.Error(t, err)
}
