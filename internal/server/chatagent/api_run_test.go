package chatagent

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTrySetAPIRunState(t *testing.T) {
	tests := []struct {
		name      string
		setup     func() string
		wantErr   bool
		wantClear bool
	}{
		{
			name: "registers first run",
			setup: func() string {
				return "sess-a"
			},
		},
		{
			name: "rejects concurrent run",
			setup: func() string {
				sessionID := "sess-b"
				pub := NewChannelPublisher(4)
				gate := NewConfirmGate(sessionID, pub)
				require.NoError(t, TrySetAPIRunState(sessionID, NewAPIRunState(pub, gate)))
				return sessionID
			},
			wantErr: true,
		},
		{
			name: "clear only matching state",
			setup: func() string {
				sessionID := "sess-c"
				pub := NewChannelPublisher(4)
				gate := NewConfirmGate(sessionID, pub)
				state := NewAPIRunState(pub, gate)
				require.NoError(t, TrySetAPIRunState(sessionID, state))
				ClearAPIRunState(sessionID, state)
				return sessionID
			},
			wantClear: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessionID := tt.setup()
			t.Cleanup(func() { ClearAPIRunState(sessionID, nil) })

			if tt.wantClear {
				_, ok := GetAPIRunState(sessionID)
				assert.False(t, ok)
				return
			}

			pub := NewChannelPublisher(4)
			gate := NewConfirmGate(sessionID, pub)
			state := NewAPIRunState(pub, gate)
			err := TrySetAPIRunState(sessionID, state)
			if tt.wantErr {
				require.ErrorIs(t, err, ErrRunInFlight)
				return
			}
			require.NoError(t, err)
			got, ok := GetAPIRunState(sessionID)
			require.True(t, ok)
			assert.Equal(t, state, got)
		})
	}
}
