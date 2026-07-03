package chatagent

import "testing"

func TestActiveSubagentTools(t *testing.T) {
	tests := []struct {
		name   string
		tools  []string
		skills []string
		want   []string
	}{
		{
			name:   "empty returns read-only default",
			tools:  nil,
			skills: nil,
			want:   []string{"read_file", "web_search"},
		},
		{
			name:   "tools only",
			tools:  []string{"read_file", "run_terminal"},
			skills: nil,
			want:   []string{"read_file", "run_terminal"},
		},
		{
			name:   "skills append read_skill",
			tools:  []string{"read_file"},
			skills: []string{"demo-skill"},
			want:   []string{"read_file", "read_skill"},
		},
		{
			name:   "skills only enables read_skill",
			tools:  nil,
			skills: []string{"demo-skill"},
			want:   []string{"read_skill"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := activeSubagentTools(tt.tools, tt.skills)
			if tt.want == nil {
				if got != nil {
					t.Fatalf("got %v, want nil", got)
				}
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			seen := make(map[string]struct{}, len(got))
			for _, name := range got {
				seen[name] = struct{}{}
			}
			for _, name := range tt.want {
				if _, ok := seen[name]; !ok {
					t.Fatalf("missing %q in %v", name, got)
				}
			}
		})
	}
}

func TestAppendUniqueTool(t *testing.T) {
	tests := []struct {
		name  string
		tools []string
		add   string
		want  int
	}{
		{name: "appends new tool", tools: []string{"read_file"}, add: "read_skill", want: 2},
		{name: "skips duplicate", tools: []string{"read_skill"}, add: "read_skill", want: 1},
		{name: "empty slice", tools: nil, add: "read_skill", want: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := appendUniqueTool(tt.tools, tt.add)
			if len(got) != tt.want {
				t.Fatalf("len(got)=%d, want %d (%v)", len(got), tt.want, got)
			}
		})
	}
}
