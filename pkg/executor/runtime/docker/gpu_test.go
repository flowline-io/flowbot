package docker

import (
	"testing"

	"github.com/moby/moby/api/types/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_parseGPUDeviceRequests(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   string
		want    []container.DeviceRequest
		wantErr string
	}{
		{
			name:  "count all",
			value: "all",
			want: []container.DeviceRequest{{
				Count:        -1,
				Capabilities: [][]string{{"gpu"}},
				Options:      map[string]string{},
			}},
		},
		{
			name:  "driver and count",
			value: "driver=nvidia,count=2",
			want: []container.DeviceRequest{{
				Driver:       "nvidia",
				Count:        2,
				Capabilities: [][]string{{"gpu"}},
				Options:      map[string]string{},
			}},
		},
		{
			name:    "duplicate key rejected",
			value:   "count=1,count=2",
			wantErr: "can be specified only once",
		},
		{
			name:    "unexpected key rejected",
			value:   "foo=bar",
			wantErr: "unexpected key",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseGPUDeviceRequests(tt.value)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_parseGPUCount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   string
		want    int
		wantErr bool
	}{
		{name: "all", value: "all", want: -1},
		{name: "integer", value: "3", want: 3},
		{name: "invalid", value: "x", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseGPUCount(tt.value)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
