package ability

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStringParam(t *testing.T) {
	tests := []struct {
		name   string
		params map[string]any
		key    string
		want   string
		wantOk bool
	}{
		{"found string", map[string]any{"key": "val"}, "key", "val", true},
		{"not found", map[string]any{}, "key", "", false},
		{"nil value", map[string]any{"key": nil}, "key", "", false},
		{"int value fmt", map[string]any{"key": 123}, "key", "123", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := StringParam(tt.params, tt.key)
			assert.Equal(t, tt.wantOk, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRequiredString(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]any
		key     string
		wantErr bool
		want    string
	}{
		{"valid string returns value", map[string]any{"key": "val"}, "key", false, "val"},
		{"missing key returns error", map[string]any{}, "key", true, ""},
		{"empty string returns error", map[string]any{"key": ""}, "key", true, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RequiredString(tt.params, tt.key)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestIntParam(t *testing.T) {
	tests := []struct {
		name   string
		params map[string]any
		key    string
		want   int
		wantOk bool
	}{
		{"int value", map[string]any{"key": 42}, "key", 42, true},
		{"float value", map[string]any{"key": float64(42)}, "key", 42, true},
		{"string value", map[string]any{"key": "42"}, "key", 42, true},
		{"invalid string", map[string]any{"key": "abc"}, "key", 0, false},
		{"not found", map[string]any{}, "key", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := IntParam(tt.params, tt.key)
			assert.Equal(t, tt.wantOk, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBoolParam(t *testing.T) {
	tests := []struct {
		name   string
		params map[string]any
		key    string
		want   bool
		wantOk bool
	}{
		{"true", map[string]any{"key": true}, "key", true, true},
		{"false", map[string]any{"key": false}, "key", false, true},
		{"string true", map[string]any{"key": "true"}, "key", true, true},
		{"string false", map[string]any{"key": "false"}, "key", false, true},
		{"invalid", map[string]any{"key": "invalid"}, "key", false, false},
		{"not found", map[string]any{}, "key", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := BoolParam(tt.params, tt.key)
			assert.Equal(t, tt.wantOk, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPageRequestFromParams(t *testing.T) {
	tests := []struct {
		name   string
		params map[string]any
		want   PageRequest
	}{
		{
			"populated params",
			map[string]any{
				"limit":      20,
				"cursor":     "next-page",
				"sort_by":    "created_at",
				"sort_order": "desc",
			},
			PageRequest{Limit: 20, Cursor: "next-page", SortBy: "created_at", SortOrder: "desc"},
		},
		{
			"empty params",
			map[string]any{},
			PageRequest{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr := PageRequestFromParams(tt.params)
			assert.Equal(t, tt.want.Limit, pr.Limit)
			assert.Equal(t, tt.want.Cursor, pr.Cursor)
			assert.Equal(t, tt.want.SortBy, pr.SortBy)
			assert.Equal(t, tt.want.SortOrder, pr.SortOrder)
		})
	}
}

func TestRequiredInt(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]any
		key     string
		wantErr bool
		want    int
	}{
		{"valid int returns value", map[string]any{"key": 42}, "key", false, 42},
		{"missing key returns error", map[string]any{}, "key", true, 0},
		{"invalid type returns error", map[string]any{"key": "abc"}, "key", true, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := RequiredInt(tt.params, tt.key)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, v)
			}
		})
	}
}

func TestInt64Param(t *testing.T) {
	tests := []struct {
		name   string
		params map[string]any
		key    string
		want   int64
		wantOk bool
	}{
		{"int64 value", map[string]any{"key": int64(42)}, "key", 42, true},
		{"int value", map[string]any{"key": 42}, "key", 42, true},
		{"float64 value", map[string]any{"key": float64(99.0)}, "key", 99, true},
		{"not found", map[string]any{}, "key", 0, false},
		{"nil value", map[string]any{"key": nil}, "key", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := Int64Param(tt.params, tt.key)
			assert.Equal(t, tt.wantOk, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRequiredInt64(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]any
		key     string
		wantErr bool
		want    int64
	}{
		{"valid int64 returns value", map[string]any{"key": int64(100)}, "key", false, 100},
		{"missing key returns error", map[string]any{}, "key", true, 0},
		{"nil value returns error", map[string]any{"key": nil}, "key", true, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := RequiredInt64(tt.params, tt.key)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, v)
			}
		})
	}
}
