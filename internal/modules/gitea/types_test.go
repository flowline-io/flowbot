package gitea

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "error variables should be non-nil",
			test: func(t *testing.T) {
				t.Parallel()
		require.Error(t, ErrMissingCommitMessage)
		require.Error(t, ErrMissingDiffContent)
		require.Error(t, ErrEmptyPrompt)
			},
		},
		{
			name: "error messages should be correct",
			test: func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, "missing commit message in metadata", ErrMissingCommitMessage.Error())
				assert.Equal(t, "missing diff content", ErrMissingDiffContent.Error())
				assert.Equal(t, "empty prompt after formatting", ErrEmptyPrompt.Error())
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func TestCodeIssue(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "CodeIssue fields should be settable and readable"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			issue := CodeIssue{
				FilePath:    "main.go",
				StartLine:   10,
				EndLine:     20,
				Description: "test issue",
				Suggestion:  "fix it",
			}
			assert.Equal(t, "main.go", issue.FilePath)
			assert.Equal(t, 10, issue.StartLine)
			assert.Equal(t, 20, issue.EndLine)
			assert.Equal(t, "test issue", issue.Description)
			assert.Equal(t, "fix it", issue.Suggestion)
		})
	}
}

func TestSecurityIssue(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "SecurityIssue fields should be settable and readable"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			issue := SecurityIssue{
				Severity:    "high",
				FilePath:    "auth.go",
				StartLine:   5,
				EndLine:     15,
				Description: "SQL injection",
				Suggestion:  "use parameterized queries",
			}
			assert.Equal(t, "high", issue.Severity)
			assert.Equal(t, "auth.go", issue.FilePath)
		})
	}
}

func TestQualityMetric(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "QualityMetric fields should be settable and readable"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			metric := QualityMetric{
				SecurityScore:     9.0,
				PerformanceScore:  8.0,
				ReadabilityScore:  7.5,
				BestPracticeScore: 8.5,
			}
			assert.InEpsilon(t, 9.0, metric.SecurityScore, 0.001)
			assert.InEpsilon(t, 8.0, metric.PerformanceScore, 0.001)
			assert.InEpsilon(t, 7.5, metric.ReadabilityScore, 0.001)
			assert.InEpsilon(t, 8.5, metric.BestPracticeScore, 0.001)
		})
	}
}

func TestReviewResult(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "ReviewResult fields should be settable and readable"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := ReviewResult{
				Score:    8.5,
				Comments: []string{"good"},
			}
			assert.InEpsilon(t, 8.5, result.Score, 0.001)
			assert.Equal(t, []string{"good"}, result.Comments)
		})
	}
}

func TestCodeContext_Len(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		filesCtx []map[string]any
		wantLen  int
	}{
		{
			name: "with files returns count",
			filesCtx: []map[string]any{
				{"file_path": "a.go"},
				{"file_path": "b.go"},
			},
			wantLen: 2,
		},
		{
			name:     "empty returns 0",
			filesCtx: nil,
			wantLen:  0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := CodeContext{FilesContext: tt.filesCtx}
			assert.Equal(t, tt.wantLen, ctx.Len())
		})
	}
}

func TestReviewPrompt(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "ReviewPrompt should contain required sections"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.NotEmpty(t, ReviewPrompt)
			assert.Contains(t, ReviewPrompt, "Security")
			assert.Contains(t, ReviewPrompt, "Performance")
			assert.Contains(t, ReviewPrompt, "Readability")
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "should return non-nil config with correct defaults",
			test: func(t *testing.T) {
				t.Parallel()
				cfg := DefaultConfig()
				assert.NotNil(t, cfg)
				assert.Equal(t, 5, cfg.ContextWindow)
				assert.Equal(t, 4096, cfg.MaxTokens)
				assert.InEpsilon(t, 8.5, cfg.QualityThreshold, 0.001)
				assert.Equal(t, 5, cfg.MaxSecurityIssues)
				assert.NotEmpty(t, cfg.IgnorePatterns)
				assert.NotEmpty(t, cfg.ScoringRules)
				assert.NotEmpty(t, cfg.IgnoreCommitStrings)
			},
		},
		{
			name: "scoring rules should have correct weights",
			test: func(t *testing.T) {
				t.Parallel()
				cfg := DefaultConfig()
				assert.InEpsilon(t, 0.3, cfg.ScoringRules["security"], 0.001)
				assert.InEpsilon(t, 0.2, cfg.ScoringRules["performance"], 0.001)
				assert.InEpsilon(t, 0.2, cfg.ScoringRules["readability"], 0.001)
				assert.InEpsilon(t, 0.3, cfg.ScoringRules["best_practice"], 0.001)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}
