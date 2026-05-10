package skills

import (
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
)

func TestExtractArgsFromUse(t *testing.T) {
	tests := []struct {
		name string
		use  string
		want string
	}{
		{name: "no args", use: "list", want: ""},
		{name: "one arg", use: "get <id>", want: "<id>"},
		{name: "two args", use: "update <task_id> <subtask_id>", want: "<task_id> <subtask_id>"},
		{name: "flags in use", use: "list [flags]", want: ""},
		{name: "arg not bracketed", use: "create something", want: ""},
		{name: "nested brackets match", use: "run <<file>>", want: "<<file>>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractArgsFromUse(tt.use)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []string
		wantNil bool
	}{
		{name: "empty", input: "", wantNil: true},
		{name: "one arg", input: "<id>", want: []string{"id"}},
		{name: "two args", input: "<task_id> <subtask_id>", want: []string{"task_id", "subtask_id"}},
		{name: "no brackets splits words anyway", input: "hello world", want: []string{"hello", "world"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseArgs(tt.input)
			if tt.wantNil {
				require.Nil(t, got)
			} else {
				require.Equal(t, tt.want, got)
			}
		})
	}
}

func TestBuildCLIString(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		argsUsage string
		flags     []flagSpec
		want      string
	}{
		{
			name:      "no args no flags",
			path:      "bookmark list",
			argsUsage: "",
			flags:     nil,
			want:      "flowbot bookmark list",
		},
		{
			name:      "with args",
			path:      "bookmark get",
			argsUsage: "<id>",
			flags:     nil,
			want:      "flowbot bookmark get <id>",
		},
		{
			name:      "mixed required and optional",
			path:      "bookmark create",
			argsUsage: "",
			flags: []flagSpec{
				{Name: "url", Shorthand: "u", Type: "string", Required: true, Description: "Bookmark URL"},
				{Name: "output", Shorthand: "o", Type: "string", Required: false, Description: "Format"},
			},
			want: "flowbot bookmark create --url <url> [flags]",
		},
		{
			name:      "with required bool flag and optional",
			path:      "archive",
			argsUsage: "<id>",
			flags: []flagSpec{
				{Name: "force", Shorthand: "f", Type: "bool", Required: true, Description: "Force"},
				{Name: "output", Shorthand: "o", Type: "string", Required: false, Description: "Format"},
			},
			want: "flowbot archive <id> --force [flags]",
		},
		{
			name: "all required no optional",
			path: "kanban set",
			flags: []flagSpec{
				{Name: "name", Type: "string", Required: true},
			},
			want: "flowbot kanban set --name <name>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildCLIString(tt.path, tt.argsUsage, tt.flags)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestZeroDefault(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "zeroDefault returns value or fallback"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, "hello", zeroDefault("hello", "fallback"))
			require.Equal(t, "fallback", zeroDefault("", "fallback"))
		})
	}
}

func TestExtractFlags(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "extractFlags returns correct flag specs"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
			fs.StringP("output", "o", "table", "Output format")
			fs.StringP("url", "u", "", "Bookmark URL")
			fs.String("name", "", "Tag name")
			fs.BoolP("force", "f", false, "Force")

			result := extractFlags(fs)

			// "output" should be skipped
			for _, f := range result {
				require.NotEqual(t, "output", f.Name, "output flag should be skipped")
			}

			// Check "url" is present
			var urlSpec *flagSpec
			for i := range result {
				if result[i].Name == "url" {
					urlSpec = &result[i]
					break
				}
			}
			require.NotNil(t, urlSpec)
			require.Equal(t, "u", urlSpec.Shorthand)
			require.Equal(t, "string", urlSpec.Type)

			// Check "name" is present
			var nameSpec *flagSpec
			for i := range result {
				if result[i].Name == "name" {
					nameSpec = &result[i]
					break
				}
			}
			require.NotNil(t, nameSpec)

			// Check "force" is present
			var forceSpec *flagSpec
			for i := range result {
				if result[i].Name == "force" {
					forceSpec = &result[i]
					break
				}
			}
			require.NotNil(t, forceSpec)
			require.Equal(t, "f", forceSpec.Shorthand)
			require.Equal(t, "bool", forceSpec.Type)
		})
	}
}

func TestExtractFlagsNoRequiredAnnotation(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "extractFlags does not set required annotation by default"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
			fs.StringP("title", "t", "", "Title")
			fs.IntP("limit", "n", 20, "Limit")

			result := extractFlags(fs)
			require.Len(t, result, 2)

			for _, f := range result {
				require.False(t, f.Required, "flag %s should not be required", f.Name)
			}
		})
	}
}
