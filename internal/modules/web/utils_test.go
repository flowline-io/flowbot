package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/views/partials"
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
			name: "unicode percent-encoded",
			raw:  "donn%C3%A9es1",
			want: "données1",
		},
		{
			name: "already decoded unicode",
			raw:  "données1",
			want: "données1",
		},
		{
			name:    "invalid escape sequence",
			raw:     "%ZZ",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
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

func TestParseStatsTabQuery(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		query        string
		wantDays     int
		wantGroupBy  string
		wantSinceSet bool
		wantErr      bool
	}{
		{name: "days 30", query: "days=30&groupBy=day", wantDays: 30, wantGroupBy: "day", wantSinceSet: true},
		{name: "days 0 all", query: "days=0&groupBy=week", wantDays: 0, wantGroupBy: "week", wantSinceSet: false},
		{name: "invalid days", query: "days=7&groupBy=day", wantErr: true},
		{name: "invalid groupBy", query: "days=30&groupBy=year", wantErr: true},
		{name: "legacy since ~30d", query: "since=" + time.Now().AddDate(0, 0, -30).Format("2006-01-02") + "&groupBy=day", wantDays: 30, wantGroupBy: "day", wantSinceSet: true},
		{name: "empty defaults to all", query: "", wantDays: 0, wantGroupBy: "day", wantSinceSet: false},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			app := fiber.New()
			var (
				gotSince time.Time
				gotTabs  partials.StatsTabState
				gotErr   error
			)
			app.Get("/t", func(c fiber.Ctx) error {
				gotSince, gotTabs, gotErr = parseStatsTabQuery(c)
				return nil
			})
			req := httptest.NewRequest(http.MethodGet, "/t?"+tt.query, http.NoBody)
			_, err := app.Test(req)
			require.NoError(t, err)
			if tt.wantErr {
				require.Error(t, gotErr)
				return
			}
			require.NoError(t, gotErr)
			assert.Equal(t, tt.wantDays, gotTabs.RangeDays)
			assert.Equal(t, tt.wantGroupBy, gotTabs.GroupBy)
			assert.Equal(t, tt.wantSinceSet, !gotSince.IsZero())
		})
	}
}
