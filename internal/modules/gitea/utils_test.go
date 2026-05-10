package gitea

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilterFiles(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		files    []string
		patterns []string
		want     []string
	}{
		{
			name:     "no ignore patterns returns all files",
			files:    []string{"main.go", "utils.go"},
			patterns: []string{},
			want:     []string{"main.go", "utils.go"},
		},
		{
			name:     "vendor pattern filters vendor files",
			files:    []string{"main.go", "vendor/lib.go", "utils.go"},
			patterns: []string{"vendor/*"},
			want:     []string{"main.go", "utils.go"},
		},
		{
			name:     "all files ignored returns empty",
			files:    []string{"vendor/a.go", "vendor/b.go"},
			patterns: []string{"vendor/*"},
			want:     []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := filterFiles(tt.files, tt.patterns)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestFilterCommitString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		msg      string
		patterns []string
		want     bool
	}{
		{
			name:     "build(deps) matches pattern",
			msg:      "build(deps): update",
			patterns: []string{"build(deps):"},
			want:     true,
		},
		{
			name:     "merge pull request matches pattern",
			msg:      "Merge pull request #1",
			patterns: []string{"Merge pull request"},
			want:     true,
		},
		{
			name:     "feat message does not match deps patterns",
			msg:      "feat: add feature",
			patterns: []string{"build(deps):", "Merge pull request"},
			want:     false,
		},
		{
			name:     "empty patterns always returns false",
			msg:      "anything",
			patterns: []string{},
			want:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, filterCommitString(tt.msg, tt.patterns))
		})
	}
}

func TestFilterFilesContext(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		ctx     []map[string]any
		files   []string
		wantLen int
	}{
		{
			name: "filters to matching file paths",
			ctx: []map[string]any{
				{"file_path": "a.go", "context": "code1"},
				{"file_path": "b.go", "context": "code2"},
				{"file_path": "c.go", "context": "code3"},
			},
			files:   []string{"a.go", "c.go"},
			wantLen: 2,
		},
		{
			name: "no match returns empty",
			ctx: []map[string]any{
				{"file_path": "a.go"},
			},
			files:   []string{"b.go"},
			wantLen: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := filterFilesContext(tt.ctx, tt.files)
			assert.Len(t, result, tt.wantLen)
		})
	}
}

func TestGetTotalDiffSize(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		chunks []*CodeContext
		want   int
	}{
		{
			name: "sums diff lengths",
			chunks: []*CodeContext{
				{Diff: "abc"},
				{Diff: "de"},
			},
			want: 5,
		},
		{
			name:   "empty list returns 0",
			chunks: []*CodeContext{},
			want:   0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, getTotalDiffSize(tt.chunks))
		})
	}
}

func TestSplitCodeChunk_EmptyDiff(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "empty diff returns empty chunks"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := CodeContext{
				Diff:     "",
				Metadata: map[string]string{"commit_id": "abc12345"},
			}
			chunks := splitCodeChunk(ctx)
			assert.Empty(t, chunks)
		})
	}
}

func TestGenerateComments_NilInput(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		result *ReviewResult
		ctx    *CodeContext
	}{
		{name: "nil result and nil ctx", result: nil, ctx: nil},
		{name: "valid result but nil ctx", result: &ReviewResult{}, ctx: nil},
		{name: "nil result with valid ctx", result: nil, ctx: &CodeContext{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, generateComments(tt.result, tt.ctx))
		})
	}
}

func TestGenerateComments_ValidInput(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "generates review comment with score and commit ID"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := &ReviewResult{
				Score: 8.5,
				QualityMetrics: &QualityMetric{
					SecurityScore:     9.0,
					PerformanceScore:  8.0,
					ReadabilityScore:  7.5,
					BestPracticeScore: 8.5,
				},
				Issues:         []*CodeIssue{},
				SecurityIssues: []*SecurityIssue{},
			}
			ctx := &CodeContext{
				Metadata: map[string]string{
					"commit_id":      "abc12345678",
					"commit_message": "test commit",
				},
			}
			comment := generateComments(result, ctx)
			assert.NotNil(t, comment)
			assert.Contains(t, comment.Body, "Code Review Report")
			assert.Contains(t, comment.Body, "8.5")
			assert.Equal(t, "abc12345678", comment.CommitID)
		})
	}
}
