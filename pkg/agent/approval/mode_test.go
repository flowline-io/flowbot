package approval_test

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/approval"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseMode(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    approval.Mode
		wantErr bool
	}{
		{name: "manual", raw: "manual", want: approval.ModeManual},
		{name: "auto", raw: "auto", want: approval.ModeAuto},
		{name: "off", raw: "off", want: approval.ModeOff},
		{name: "empty defaults manual", raw: "", want: approval.ModeManual},
		{name: "invalid", raw: "smart", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := approval.ParseMode(tt.raw)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
