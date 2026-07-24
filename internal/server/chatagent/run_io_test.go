package chatagent

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithRunIO_RoundTrip(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		api      *APIRunOptions
		wantPub  bool
		wantGate bool
		nilAPI   bool
	}{
		{name: "nil api leaves context empty", nilAPI: true},
		{name: "empty api leaves context empty", api: &APIRunOptions{}},
		{name: "publisher only", api: &APIRunOptions{Publisher: NewChannelPublisher(1)}, wantPub: true},
		{name: "confirm only", api: &APIRunOptions{Confirm: NewConfirmGate("s", nil, nil)}, wantGate: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var api *APIRunOptions
			if !tt.nilAPI {
				api = tt.api
			}
			ctx := withRunIO(context.Background(), api)
			io := runIOFromContext(ctx)
			if !tt.wantPub && !tt.wantGate {
				assert.Nil(t, io)
				return
			}
			require.NotNil(t, io)
			if tt.wantPub {
				assert.NotNil(t, io.Publisher)
			} else {
				assert.Nil(t, io.Publisher)
			}
			if tt.wantGate {
				assert.NotNil(t, io.Confirm)
			} else {
				assert.Nil(t, io.Confirm)
			}
		})
	}
}
