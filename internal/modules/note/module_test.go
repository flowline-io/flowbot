package note

import (
	"encoding/json"
	"testing"

	"github.com/bytedance/sonic"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBotName(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{name: "should equal note", expected: "note"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, Name)
		})
	}
}

func TestInit(t *testing.T) {
	tests := []struct {
		name    string
		config  configType
		rawJSON json.RawMessage
		preInit bool
		wantErr bool
		ready   bool
	}{
		{
			name:    "enabled config",
			config:  configType{Enabled: true},
			wantErr: false,
			ready:   true,
		},
		{
			name:    "disabled config",
			config:  configType{Enabled: false},
			wantErr: false,
			ready:   false,
		},
		{
			name:    "invalid JSON",
			rawJSON: json.RawMessage(`{invalid`),
			wantErr: true,
			ready:   false,
		},
		{
			name:    "already initialized",
			rawJSON: json.RawMessage(`{"enabled":true}`),
			preInit: true,
			wantErr: true,
			ready:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.preInit {
				handler = moduleHandler{initialized: true}
			} else {
				handler = moduleHandler{}
			}

			var data json.RawMessage
			if tt.rawJSON != nil {
				data = tt.rawJSON
			} else {
				d, _ := sonic.Marshal(tt.config)
				data = d
			}

			err := handler.Init(data)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.ready, handler.IsReady())
			}
		})
	}
}

func TestRules_ReturnsAllRulesets(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "should return webservice rules"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler = moduleHandler{initialized: true}
			rules := handler.Rules()
			assert.NotEmpty(t, rules)
			assert.Len(t, rules, 1) // webserviceRules
		})
	}
}

func TestWebserviceRules_Defined(t *testing.T) {
	tests := []struct {
		name          string
		expectedPaths []string
	}{
		{
			name: "should contain CRUD endpoints",
			expectedPaths: []string{
				"/",        // list and create
				"/:id",      // get, update, delete
				"/search",
				"/health",
				"/:id/content", // get and set content
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths := make(map[string]bool)
			for _, r := range webserviceRules {
				paths[r.Path] = true
			}
			for _, expected := range tt.expectedPaths {
				assert.True(t, paths[expected], "expected path %q in webservice rules", expected)
			}
		})
	}
}

func TestWebserviceRules_NotEmpty(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "webservice rules should not be empty"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, webserviceRules)
		})
	}
}
