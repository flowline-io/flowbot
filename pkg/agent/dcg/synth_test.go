package dcg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSynthCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		language string
		code     string
		want     string
		wantErr  bool
	}{
		{
			name:     "python alias py",
			language: "py",
			code:     "print(1)",
			want:     `python -c "print(1)"`,
		},
		{
			name:     "shell alias bash",
			language: "bash",
			code:     "echo hi",
			want:     `sh -c "echo hi"`,
		},
		{
			name:     "empty language",
			language: "  ",
			code:     "print(1)",
			wantErr:  true,
		},
		{
			name:     "empty code",
			language: "python",
			code:     "  ",
			wantErr:  true,
		},
		{
			name:     "unknown language",
			language: "rust",
			code:     "fn main(){}",
			wantErr:  true,
		},
		{
			name:     "python3 rejected like run_code",
			language: "python3",
			code:     "print(1)",
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := SynthCommand(tt.language, tt.code)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCommandForTool(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		tool    string
		args    map[string]any
		want    string
		wantOK  bool
		wantErr bool
	}{
		{
			name:   "run_terminal command",
			tool:   "run_terminal",
			args:   map[string]any{"command": " git status "},
			want:   "git status",
			wantOK: true,
		},
		{
			name:    "run_terminal empty",
			tool:    "run_terminal",
			args:    map[string]any{"command": "  "},
			wantErr: true,
		},
		{
			name:    "run_terminal nil command",
			tool:    "run_terminal",
			args:    map[string]any{"command": nil},
			wantErr: true,
		},
		{
			name:   "run_code python",
			tool:   "run_code",
			args:   map[string]any{"language": "python", "code": "print(1)"},
			want:   `python -c "print(1)"`,
			wantOK: true,
		},
		{
			name:    "run_code empty language",
			tool:    "run_code",
			args:    map[string]any{"language": "  ", "code": "print(1)"},
			wantErr: true,
		},
		{
			name:   "read_file skipped",
			tool:   "read_file",
			args:   map[string]any{"path": "a.go"},
			wantOK: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok, err := CommandForTool(tt.tool, tt.args)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantOK, ok)
			if tt.wantOK {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestTruncateCommandForLog(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "short unchanged", in: "echo ok", want: "echo ok"},
		{name: "exact 200", in: stringsRepeat("a", 200), want: stringsRepeat("a", 200)},
		{name: "truncates over 200", in: stringsRepeat("b", 201), want: stringsRepeat("b", 200) + "..."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, TruncateCommandForLog(tt.in))
		})
	}
}

func stringsRepeat(s string, n int) string {
	b := make([]byte, 0, len(s)*n)
	for range n {
		b = append(b, s...)
	}
	return string(b)
}
