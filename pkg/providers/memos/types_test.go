package memos

import (
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoUnmarshalJSON_Enums(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		raw            string
		wantVisibility string
		wantState      string
		wantErr        bool
	}{
		{
			name:           "string enums from REST protojson",
			raw:            `{"name":"memos/1","state":"NORMAL","visibility":"PRIVATE"}`,
			wantVisibility: "PRIVATE",
			wantState:      "NORMAL",
		},
		{
			name:           "numeric enums from webhook encoding/json",
			raw:            `{"name":"memos/1","state":1,"visibility":1,"property":{}}`,
			wantVisibility: "PRIVATE",
			wantState:      "NORMAL",
		},
		{
			name:           "numeric public and archived",
			raw:            `{"name":"memos/2","state":2,"visibility":3}`,
			wantVisibility: "PUBLIC",
			wantState:      "ARCHIVED",
		},
		{
			name:           "protected visibility number",
			raw:            `{"name":"memos/3","visibility":2}`,
			wantVisibility: "PROTECTED",
		},
		{
			name:    "invalid enum type object",
			raw:     `{"name":"memos/4","visibility":{"bad":true}}`,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var m Memo
			err := sonic.Unmarshal([]byte(tt.raw), &m)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantVisibility, m.Visibility)
			assert.Equal(t, tt.wantState, m.State)
		})
	}
}
