package ability

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
	_, err := RequiredString(map[string]any{"key": "val"}, "key")
	assert.NoError(t, err)

	_, err = RequiredString(map[string]any{}, "key")
	assert.Error(t, err)

	_, err = RequiredString(map[string]any{"key": ""}, "key")
	assert.Error(t, err)
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
	params := map[string]any{
		"limit":      20,
		"cursor":     "next-page",
		"sort_by":    "created_at",
		"sort_order": "desc",
	}
	pr := PageRequestFromParams(params)
	assert.Equal(t, 20, pr.Limit)
	assert.Equal(t, "next-page", pr.Cursor)
	assert.Equal(t, "created_at", pr.SortBy)
	assert.Equal(t, "desc", pr.SortOrder)
}

func TestPageRequestFromParamsEmpty(t *testing.T) {
	pr := PageRequestFromParams(map[string]any{})
	assert.Equal(t, 0, pr.Limit)
	assert.Equal(t, "", pr.Cursor)
	assert.Equal(t, "", pr.SortBy)
	assert.Equal(t, "", pr.SortOrder)
}
