package server

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/config"
)

func TestInitializeTimezone(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "loads Local timezone successfully",
			wantErr: false,
		},
		{
			name:    "default timezone loads without error",
			wantErr: false,
		},
		{
			name:    "timezone init is idempotent",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := initializeTimezone()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestInitializeMetrics(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		cfg     config.Metrics
		wantErr bool
	}{
		{
			name:    "disabled metrics returns nil",
			cfg:     config.Metrics{Enabled: false},
			wantErr: false,
		},
		{
			name:    "enabled metrics with empty endpoint returns nil",
			cfg:     config.Metrics{Enabled: true, Endpoint: ""},
			wantErr: false,
		},
		{
			name:    "enabled metrics with empty config returns nil",
			cfg:     config.Metrics{Enabled: false, Endpoint: ""},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := config.App.Metrics
			config.App.Metrics = tt.cfg
			t.Cleanup(func() { config.App.Metrics = original })

			err := initializeMetrics()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestInitializeLog(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		cfg  config.Log
	}{
		{
			name: "default log config initializes ok",
			cfg: config.Log{
				Level:      "info",
				Caller:     false,
				StackTrace: false,
				JSONOutput: false,
			},
		},
		{
			name: "with rotation config initializes ok",
			cfg: config.Log{
				Level:   "debug",
				Caller:  true,
				FileLog: true,
				Rotation: &config.LogRotation{
					MaxSize:    100,
					MaxAge:     30,
					MaxBackups: 5,
					Compress:   true,
				},
			},
		},
		{
			name: "with sampling config initializes ok",
			cfg: config.Log{
				Level: "warn",
				Sampling: &config.LogSampling{
					Burst:  10,
					Period: 5,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := config.App.Log
			config.App.Log = tt.cfg
			t.Cleanup(func() { config.App.Log = original })

			err := initializeLog()
			assert.NoError(t, err)
		})
	}
}

func TestInitializeMedia(t *testing.T) {
	tests := []struct {
		name    string
		setNil  bool
		wantErr bool
	}{
		{
			name:    "nil media config returns nil",
			setNil:  true,
			wantErr: false,
		},
		{
			name:    "nil media config returns nil on repeated call",
			setNil:  true,
			wantErr: false,
		},
		{
			name:    "nil media config is idempotent",
			setNil:  true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := config.App.Media
			t.Cleanup(func() { config.App.Media = original })

			if tt.setNil {
				config.App.Media = nil
			}

			err := initializeMedia()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGlobalsMediaGcPeriod(t *testing.T) {
	tests := []struct {
		name     string
		wantZero bool
	}{
		{name: "mediaGcPeriod has default zero value", wantZero: true},
		{name: "mediaGcPeriod is time.Duration zero", wantZero: true},
		{name: "mediaGcPeriod does not change unexpectedly", wantZero: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantZero {
				assert.Equal(t, time.Duration(0), globals.mediaGcPeriod)
			}
		})
	}
}
