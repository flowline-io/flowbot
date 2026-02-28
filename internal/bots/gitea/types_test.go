package gitea

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorVariables(t *testing.T) {
	assert.NotNil(t, ErrMissingCommitMessage)
	assert.NotNil(t, ErrMissingDiffContent)
	assert.NotNil(t, ErrEmptyPrompt)
}

func TestErrorMessages(t *testing.T) {
	assert.Equal(t, "missing commit message in metadata", ErrMissingCommitMessage.Error())
	assert.Equal(t, "missing diff content", ErrMissingDiffContent.Error())
	assert.Equal(t, "empty prompt after formatting", ErrEmptyPrompt.Error())
}

func TestCodeIssue_Fields(t *testing.T) {
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
}

func TestSecurityIssue_Fields(t *testing.T) {
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
}

func TestQualityMetric_Fields(t *testing.T) {
	metric := QualityMetric{
		SecurityScore:     9.0,
		PerformanceScore:  8.0,
		ReadabilityScore:  7.5,
		BestPracticeScore: 8.5,
	}
	assert.Equal(t, 9.0, metric.SecurityScore)
	assert.Equal(t, 8.0, metric.PerformanceScore)
	assert.Equal(t, 7.5, metric.ReadabilityScore)
	assert.Equal(t, 8.5, metric.BestPracticeScore)
}

func TestReviewResult_Fields(t *testing.T) {
	result := ReviewResult{
		Score:    8.5,
		Comments: []string{"good"},
	}
	assert.Equal(t, 8.5, result.Score)
	assert.Equal(t, []string{"good"}, result.Comments)
}

func TestCodeContext_Len(t *testing.T) {
	ctx := CodeContext{
		FilesContext: []map[string]any{
			{"file_path": "a.go"},
			{"file_path": "b.go"},
		},
	}
	assert.Equal(t, 2, ctx.Len())
}

func TestCodeContext_LenEmpty(t *testing.T) {
	ctx := CodeContext{}
	assert.Equal(t, 0, ctx.Len())
}

func TestReviewPrompt_NotEmpty(t *testing.T) {
	assert.NotEmpty(t, ReviewPrompt)
	assert.Contains(t, ReviewPrompt, "Security")
	assert.Contains(t, ReviewPrompt, "Performance")
	assert.Contains(t, ReviewPrompt, "Readability")
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	assert.NotNil(t, cfg)
	assert.Equal(t, 5, cfg.ContextWindow)
	assert.Equal(t, 4096, cfg.MaxTokens)
	assert.Equal(t, 8.5, cfg.QualityThreshold)
	assert.Equal(t, 5, cfg.MaxSecurityIssues)
	assert.NotEmpty(t, cfg.IgnorePatterns)
	assert.NotEmpty(t, cfg.ScoringRules)
	assert.NotEmpty(t, cfg.IgnoreCommitStrings)
}

func TestDefaultConfig_ScoringRules(t *testing.T) {
	cfg := DefaultConfig()
	assert.Equal(t, 0.3, cfg.ScoringRules["security"])
	assert.Equal(t, 0.2, cfg.ScoringRules["performance"])
	assert.Equal(t, 0.2, cfg.ScoringRules["readability"])
	assert.Equal(t, 0.3, cfg.ScoringRules["best_practice"])
}
