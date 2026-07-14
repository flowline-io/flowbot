package web

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodePathParam(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		want    string
		wantErr bool
	}{
		{
			name: "empty string",
			raw:  "",
			want: "",
		},
		{
			name: "ascii unchanged",
			raw:  "my-pipeline",
			want: "my-pipeline",
		},
		{
			name: "chinese percent-encoded",
			raw:  "%E6%BC%94%E7%A4%BA1",
			want: "演示1",
		},
		{
			name: "already decoded chinese",
			raw:  "演示1",
			want: "演示1",
		},
		{
			name:    "invalid escape sequence",
			raw:     "%ZZ",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := decodePathParam(tt.raw)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
