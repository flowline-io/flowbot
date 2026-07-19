package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalize_Postgres(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		pg         PostgresConfig
		wantDSN    string
		wantMax    int
		wantOpen   int
		wantSQLTO  bool
		wantSQLVal int
	}{
		{
			name:    "dsn only",
			pg:      PostgresConfig{DSN: "postgres://u:p@localhost/db?sslmode=disable"},
			wantDSN: "postgres://u:p@localhost/db?sslmode=disable",
			wantMax: 0,
		},
		{
			name: "with pool overrides",
			pg: PostgresConfig{
				DSN:          "postgres://u:p@localhost/db",
				MaxResults:   512,
				MaxOpenConns: 40,
				SQLTimeout:   15,
			},
			wantDSN:    "postgres://u:p@localhost/db",
			wantMax:    512,
			wantOpen:   40,
			wantSQLTO:  true,
			wantSQLVal: 15,
		},
		{
			name:    "empty dsn still builds adapter map",
			pg:      PostgresConfig{},
			wantDSN: "",
			wantMax: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := Type{Postgres: tt.pg}
			cfg.Normalize()

			assert.Equal(t, "postgres", cfg.Store.UseAdapter)
			assert.Equal(t, tt.wantMax, cfg.Store.MaxResults)
			require.NotNil(t, cfg.Store.Adapters)
			adapter, ok := cfg.Store.Adapters["postgres"].(map[string]any)
			require.True(t, ok)
			assert.Equal(t, tt.wantDSN, adapter["dsn"])
			if tt.wantOpen != 0 {
				assert.Equal(t, tt.wantOpen, adapter["max_open_conns"])
			} else {
				_, has := adapter["max_open_conns"]
				assert.False(t, has)
			}
			if tt.wantSQLTO {
				assert.Equal(t, tt.wantSQLVal, adapter["sql_timeout"])
			}
		})
	}
}

func TestNormalize_Media(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		media      *mediaConfig
		wantNil    bool
		wantMax    int64
		wantPeriod int
		wantBlock  int
	}{
		{
			name:    "nil media unchanged",
			media:   nil,
			wantNil: true,
		},
		{
			name:       "zero values get defaults",
			media:      &mediaConfig{UseHandler: "fs"},
			wantMax:    defaultMediaMaxSize,
			wantPeriod: defaultMediaGcPeriod,
			wantBlock:  defaultMediaGcBlockSize,
		},
		{
			name: "explicit values preserved",
			media: &mediaConfig{
				UseHandler:        "fs",
				MaxFileUploadSize: 2048,
				GcPeriod:          10,
				GcBlockSize:       5,
			},
			wantMax:    2048,
			wantPeriod: 10,
			wantBlock:  5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := Type{Media: tt.media}
			cfg.Normalize()
			if tt.wantNil {
				assert.Nil(t, cfg.Media)
				return
			}
			require.NotNil(t, cfg.Media)
			assert.Equal(t, tt.wantMax, cfg.Media.MaxFileUploadSize)
			assert.Equal(t, tt.wantPeriod, cfg.Media.GcPeriod)
			assert.Equal(t, tt.wantBlock, cfg.Media.GcBlockSize)
		})
	}
}
