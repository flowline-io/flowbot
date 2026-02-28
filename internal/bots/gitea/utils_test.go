package gitea

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilterFiles_NoIgnore(t *testing.T) {
	files := []string{"main.go", "utils.go"}
	result := filterFiles(files, []string{})
	assert.Equal(t, files, result)
}

func TestFilterFiles_WithIgnore(t *testing.T) {
	files := []string{"main.go", "vendor/lib.go", "utils.go"}
	patterns := []string{"vendor/*"}
	result := filterFiles(files, patterns)
	assert.Equal(t, []string{"main.go", "utils.go"}, result)
}

func TestFilterFiles_AllIgnored(t *testing.T) {
	files := []string{"vendor/a.go", "vendor/b.go"}
	patterns := []string{"vendor/*"}
	result := filterFiles(files, patterns)
	assert.Empty(t, result)
}

func TestFilterCommitString_Match(t *testing.T) {
	assert.True(t, filterCommitString("build(deps): update", []string{"build(deps):"}))
	assert.True(t, filterCommitString("Merge pull request #1", []string{"Merge pull request"}))
}

func TestFilterCommitString_NoMatch(t *testing.T) {
	assert.False(t, filterCommitString("feat: add feature", []string{"build(deps):", "Merge pull request"}))
}

func TestFilterCommitString_EmptyPatterns(t *testing.T) {
	assert.False(t, filterCommitString("anything", []string{}))
}

func TestFilterFilesContext(t *testing.T) {
	filesContext := []map[string]any{
		{"file_path": "a.go", "context": "code1"},
		{"file_path": "b.go", "context": "code2"},
		{"file_path": "c.go", "context": "code3"},
	}
	result := filterFilesContext(filesContext, []string{"a.go", "c.go"})
	assert.Len(t, result, 2)
}

func TestFilterFilesContext_NoMatch(t *testing.T) {
	filesContext := []map[string]any{
		{"file_path": "a.go"},
	}
	result := filterFilesContext(filesContext, []string{"b.go"})
	assert.Empty(t, result)
}

func TestGetTotalDiffSize(t *testing.T) {
	chunks := []*CodeContext{
		{Diff: "abc"},
		{Diff: "de"},
	}
	assert.Equal(t, 5, getTotalDiffSize(chunks))
}

func TestGetTotalDiffSize_Empty(t *testing.T) {
	assert.Equal(t, 0, getTotalDiffSize([]*CodeContext{}))
}

func TestSplitCodeChunk_EmptyDiff(t *testing.T) {
	ctx := CodeContext{
		Diff:     "",
		Metadata: map[string]string{"commit_id": "abc12345"},
	}
	chunks := splitCodeChunk(ctx)
	assert.Empty(t, chunks)
}

func TestGenerateComments_NilInput(t *testing.T) {
	assert.Nil(t, generateComments(nil, nil))
	assert.Nil(t, generateComments(&ReviewResult{}, nil))
	assert.Nil(t, generateComments(nil, &CodeContext{}))
}

func TestGenerateComments_ValidInput(t *testing.T) {
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
}
