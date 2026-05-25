package hub

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
)

func TestKanbanCommandRules_Count(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "has at least one command rule"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.NotEmpty(t, commandRules)
		})
	}
}

func TestKanbanCommandRules_Defines(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "kanban status defined with help text"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			defines := make(map[string]string)
			for _, r := range commandRules {
				defines[r.Define] = r.Help
			}
			assert.Contains(t, defines, "kanban status")
			assert.Equal(t, "Show kanban status", defines["kanban status"])
		})
	}
}

func TestKanbanCommandRules_Handlers(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "all command rules have handlers"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			for _, r := range commandRules {
				assert.NotNil(t, r.Handler, "handler for %q should not be nil", r.Define)
			}
		})
	}
}

func TestKanbanCommandRules_TokenParsing(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		define string
		input  string
		want   bool
	}{
		{
			name:   "kanban status exact match",
			define: "kanban status",
			input:  "kanban status",
			want:   true,
		},
		{
			name:   "kanban status with extra tokens",
			define: "kanban status",
			input:  "kanban status extra",
			want:   false,
		},
		{
			name:   "kanban partial match fails",
			define: "kanban status",
			input:  "kanban",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tokens, err := parser.ParseString(tt.input)
			require.NoError(t, err)

			check, err := parser.SyntaxCheck(tt.define, tokens)
			require.NoError(t, err)
			assert.Equal(t, tt.want, check)
		})
	}
}

func TestKanbanCommandRules_ProcessCommand_Unknown(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "unknown command returns nil result"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rs := command.Ruleset(commandRules)
			ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}

			result, err := rs.ProcessCommand(ctx, "unknown command xyz")
			require.NoError(t, err)
			assert.Nil(t, result)
		})
	}
}

func TestKanbanCommandRules_StatusHandler(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "status handler returns empty message type"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var statusRule *command.Rule
			for i := range commandRules {
				if commandRules[i].Define == "kanban status" {
					statusRule = &commandRules[i]
					break
				}
			}
			require.NotNil(t, statusRule)

			ctx := types.Context{Platform: "test", Topic: "test", AsUser: types.Uid("test")}
			tokens, _ := parser.ParseString("kanban status")

			payload := statusRule.Handler(ctx, tokens)
			require.NotNil(t, payload)

			msgType := types.TypeOf(payload)
			assert.Equal(t, "EmptyMsg", msgType)
		})
	}
}

func TestUnmarshal(t *testing.T) {
	t.Parallel()
	type testStruct struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	tests := []struct {
		name     string
		input    map[string]any
		expected testStruct
	}{
		{
			name: "full struct",
			input: map[string]any{
				"name": "test",
				"age":  25,
			},
			expected: testStruct{Name: "test", Age: 25},
		},
		{
			name:     "empty map",
			input:    map[string]any{},
			expected: testStruct{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var result testStruct
			err := unmarshal(tt.input, &result)
			require.NoError(t, err)
			assert.Equal(t, tt.expected.Name, result.Name)
			if tt.expected.Age != 0 {
				assert.Equal(t, tt.expected.Age, result.Age)
			}
			if tt.name == "empty map" {
				assert.Empty(t, result.Name)
			}
		})
	}
}
