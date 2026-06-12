package server

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/stretchr/testify/assert"
)

func TestPlatformMessageID(t *testing.T) {
	tests := []struct {
		name string
		resp protocol.Response
		want string
	}{
		{
			name: "string map",
			resp: protocol.NewSuccessResponse(map[string]string{"message_id": "1700000000.0001"}),
			want: "1700000000.0001",
		},
		{
			name: "any map",
			resp: protocol.NewSuccessResponse(map[string]any{"message_id": "ts-2"}),
			want: "ts-2",
		},
		{
			name: "failed response",
			resp: protocol.NewFailedResponse(protocol.ErrInternalHandler.New("fail")),
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, platformMessageID(tt.resp))
		})
	}
}
