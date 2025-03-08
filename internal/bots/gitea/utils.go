package gitea

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/flowline-io/flowbot/internal/agents"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/providers/gitea"
)

func splitCodeChunk(codeContext CodeContext) []*CodeContext {
	conf := DefaultConfig()
	maxTokens := conf.MaxTokens - 1000

	chunks := make([]*CodeContext, 0, 10)
	currentChunk := &CodeContext{
		Diff:         "",
		FilesContext: make([]map[string]any, 0, 10),
		Metadata:     codeContext.Metadata,
	}
	currentTokens := 0
	currentFiles := make([]string, 0)

	// Split diff content by file
	fileDiffs := strings.Split(codeContext.Diff, "diff --git ")
	if len(fileDiffs) > 0 && fileDiffs[0] == "" {
		fileDiffs = fileDiffs[1:] // Remove the empty first element
	}

	for _, fileDiff := range fileDiffs {
		if fileDiff == "" {
			continue
		}

		fileDiff = "diff --git " + fileDiff
		fileTokens := len(fileDiff) // Use character length as a simple token count

		// Extract file path from diff
		re := regexp.MustCompile(`a/(.*?) b/`)
		match := re.FindStringSubmatch(fileDiff)
		if len(match) < 2 {
			continue
		}
		filePath := match[1]

		if currentTokens+fileTokens > maxTokens && currentChunk.Diff != "" {
			// Add relevant file context
			currentChunk.FilesContext = filterFilesContext(codeContext.FilesContext, currentFiles)
			chunks = append(chunks, currentChunk)

			// Reset current chunk
			currentChunk = &CodeContext{
				Diff:         "",
				FilesContext: make([]map[string]any, 0, 10),
				Metadata:     codeContext.Metadata,
			}
			currentTokens = 0
			currentFiles = make([]string, 0)
		}

		// Add file diff to current chunk
		currentChunk.Diff += fileDiff
		currentTokens += fileTokens
		currentFiles = append(currentFiles, filePath)
	}

	// Process the last chunk
	if currentChunk.Diff != "" {
		currentChunk.FilesContext = filterFilesContext(codeContext.FilesContext, currentFiles)
		chunks = append(chunks, currentChunk)
	}

	flog.Info("Split code into %d chunks with total size: %d characters",
		len(chunks),
		getTotalDiffSize(chunks))

	return chunks
}

// filterFilesContext filters the file context related to the specified file paths
func filterFilesContext(filesContext []map[string]any, filePaths []string) []map[string]any {
	result := make([]map[string]any, 0, len(filePaths))

	// Create a mapping of file paths for quick lookup
	filePathMap := make(map[string]bool)
	for _, path := range filePaths {
		filePathMap[path] = true
	}

	// Filter out the matching file contexts
	for _, fileContext := range filesContext {
		if filePath, ok := fileContext["file_path"].(string); ok {
			if filePathMap[filePath] {
				result = append(result, fileContext)
			}
		}
	}

	return result
}

// getTotalDiffSize calculates the total size of all code chunks' diffs
func getTotalDiffSize(chunks []*CodeContext) int {
	totalSize := 0
	for _, chunk := range chunks {
		totalSize += len(chunk.Diff)
	}
	return totalSize
}

func llmAnalyzeCode(ctx context.Context, codeContext CodeContext) (*ReviewResult, error) {
	chunks := splitCodeChunk(codeContext)
	results := make([]ReviewResult, 0, len(chunks))

	for i, chunk := range chunks {
		flog.Info("Analyzing chunk %d/%d with size: %d characters",
			i+1, len(chunks),
			len(chunk.Diff))

		// Format file context
		var filesContextStr strings.Builder
		if len(chunk.FilesContext) > 0 {
			for _, f := range chunk.FilesContext {
				filePath, _ := f["file_path"].(string)
				fileType, _ := f["file_type"].(string)
				codeContext, _ := f["context"].(string)
				_, _ = filesContextStr.WriteString(
					"File: " + filePath + " (" + fileType + ")\n" + codeContext + "\n\n")
			}
		} else {
			_, _ = filesContextStr.WriteString("No file context")
		}

		// Validate required parameters
		if _, ok := codeContext.Metadata["commit_message"]; !ok {
			commitID := "unknown"
			if id, ok := codeContext.Metadata["commit_id"]; ok && len(id) >= 8 {
				commitID = id[:8]
			}
			flog.Error(fmt.Errorf("missing commit message in metadata for commit: %s", commitID))
			return nil, ErrMissingCommitMessage
		}

		if chunk.Diff == "" {
			commitID := "unknown"
			if id, ok := codeContext.Metadata["commit_id"]; ok && len(id) >= 8 {
				commitID = id[:8]
			}
			flog.Error(fmt.Errorf("missing diff content for commit: %s", commitID))
			return nil, ErrMissingDiffContent
		}

		// Format prompt text
		prompt := strings.ReplaceAll(
			strings.ReplaceAll(
				strings.ReplaceAll(ReviewPrompt,
					"{commit_message}", codeContext.Metadata["commit_message"]),
				"{diff}", chunk.Diff),
			"{files_context}", filesContextStr.String())

		if prompt == "" {
			flog.Error(fmt.Errorf("empty prompt after formatting"))
			return nil, ErrEmptyPrompt
		}

		// Call LLM model
		flog.Info("Sending request to LLM model with prompt size: %d characters", len(prompt))

		// Call LLM to get response
		responseText, err := agents.LLMGenerate(ctx, prompt)
		if err != nil {
			flog.Error(fmt.Errorf("error calling LLM model: %w", err))
			return nil, fmt.Errorf("error getting LLM response: %w", err)
		}

		flog.Info("Received response with size: %d characters, content: %s", len(responseText), responseText)

		responseText = strings.TrimSpace(responseText)

		// Find the start and end positions of JSON content
		jsonStart := strings.Index(responseText, "{")
		jsonEnd := strings.LastIndex(responseText, "}") + 1

		if jsonStart == -1 || jsonEnd <= jsonStart {
			flog.Error(fmt.Errorf("no valid JSON found in response of size: %d characters", len(responseText)))
			// Return default result instead of throwing an exception
			return &ReviewResult{
				Score:          0,
				Issues:         []*CodeIssue{},
				SecurityIssues: []*SecurityIssue{},
				QualityMetrics: &QualityMetric{
					SecurityScore:     0,
					PerformanceScore:  0,
					ReadabilityScore:  0,
					BestPracticeScore: 0,
				},
			}, nil
		}

		responseText = responseText[jsonStart:jsonEnd]

		// Process possible Markdown code blocks
		if strings.Contains(responseText, "```json") {
			parts := strings.Split(responseText, "```json")
			if len(parts) > 1 {
				subParts := strings.Split(parts[1], "```")
				if len(subParts) > 0 {
					responseText = subParts[0]
				}
			}
		} else if strings.Contains(responseText, "```") {
			parts := strings.Split(responseText, "```")
			if len(parts) > 2 {
				responseText = parts[1]
			}
		}

		// Clean and format JSON string
		responseText = strings.ReplaceAll(strings.ReplaceAll(responseText, "\n", " "), "\r", "")

		flog.Info("json string with size: %d characters, content: %s", len(responseText), responseText)

		var result ReviewResult
		if err := json.Unmarshal([]byte(responseText), &result); err != nil {
			flog.Error(fmt.Errorf("JSON parsing error: %w", err))
			// Return a default review result
			results = append(results, ReviewResult{
				Score:          0,
				Issues:         []*CodeIssue{},
				SecurityIssues: []*SecurityIssue{},
				QualityMetrics: &QualityMetric{
					SecurityScore:     0,
					PerformanceScore:  0,
					ReadabilityScore:  0,
					BestPracticeScore: 0,
				},
			})
		} else {
			// Validate and fix results
			if result.Issues == nil {
				result.Issues = []*CodeIssue{}
			}
			if result.SecurityIssues == nil {
				result.SecurityIssues = []*SecurityIssue{}
			}
			if result.QualityMetrics == nil {
				result.QualityMetrics = &QualityMetric{
					SecurityScore:     0,
					PerformanceScore:  0,
					ReadabilityScore:  0,
					BestPracticeScore: 0,
				}
			}
			results = append(results, result)
			commitID := "unknown"
			if id, ok := codeContext.Metadata["commit_id"]; ok && len(id) >= 8 {
				commitID = id[:8]
			}
			flog.Info("Successfully analyzed chunk %d/%d for commit: %s",
				i+1, len(chunks), commitID)
		}
	}

	// Merge all chunk results
	if len(results) == 0 {
		commitID := "unknown"
		if id, ok := codeContext.Metadata["commit_id"]; ok && len(id) >= 8 {
			commitID = id[:8]
		}
		flog.Info("No valid results for commit: %s", commitID)
		return &ReviewResult{
			Score:          0,
			Issues:         []*CodeIssue{},
			SecurityIssues: []*SecurityIssue{},
			QualityMetrics: &QualityMetric{
				SecurityScore:     0,
				PerformanceScore:  0,
				ReadabilityScore:  0,
				BestPracticeScore: 0,
			},
		}, nil
	}

	// Use the lowest score as the final score
	minScore := results[0].Score
	minSecurityScore := results[0].QualityMetrics.SecurityScore
	minPerformanceScore := results[0].QualityMetrics.PerformanceScore
	minReadabilityScore := results[0].QualityMetrics.ReadabilityScore
	minBestPracticeScore := results[0].QualityMetrics.BestPracticeScore

	for _, r := range results[1:] {
		if r.Score < minScore {
			minScore = r.Score
		}
		if r.QualityMetrics.SecurityScore < minSecurityScore {
			minSecurityScore = r.QualityMetrics.SecurityScore
		}
		if r.QualityMetrics.PerformanceScore < minPerformanceScore {
			minPerformanceScore = r.QualityMetrics.PerformanceScore
		}
		if r.QualityMetrics.ReadabilityScore < minReadabilityScore {
			minReadabilityScore = r.QualityMetrics.ReadabilityScore
		}
		if r.QualityMetrics.BestPracticeScore < minBestPracticeScore {
			minBestPracticeScore = r.QualityMetrics.BestPracticeScore
		}
	}

	flog.Info("review results: %+v", results)

	// Merge all issues
	allIssues := make([]*CodeIssue, 0)
	allSecurityIssues := make([]*SecurityIssue, 0)
	for _, r := range results {
		allIssues = append(allIssues, r.Issues...)
		allSecurityIssues = append(allSecurityIssues, r.SecurityIssues...)
	}

	finalResult := &ReviewResult{
		Score:          minScore,
		Issues:         allIssues,
		SecurityIssues: allSecurityIssues,
		QualityMetrics: &QualityMetric{
			SecurityScore:     minSecurityScore,
			PerformanceScore:  minPerformanceScore,
			ReadabilityScore:  minReadabilityScore,
			BestPracticeScore: minBestPracticeScore,
		},
	}

	commitID := "unknown"
	if id, ok := codeContext.Metadata["commit_id"]; ok && len(id) >= 8 {
		commitID = id[:8]
	}
	flog.Info("Analysis completed for commit %s with final score: %.2f",
		commitID, finalResult.Score)
	return finalResult, nil
}

// filterFiles filters files that do not need review
func filterFiles(files []string, ignorePatterns []string) []string {
	filtered := make([]string, 0, len(files))
	for _, filename := range files {
		shouldIgnore := false
		for _, pattern := range ignorePatterns {
			matched, err := filepath.Match(pattern, filename)
			if err == nil && matched {
				shouldIgnore = true
				break
			}
		}

		if !shouldIgnore {
			filtered = append(filtered, filename)
		}
	}
	return filtered
}

func collectContext(owner, repo string, commitDiff *gitea.CommitDiff) (*CodeContext, error) {
	conf := DefaultConfig()
	// Filter files
	filteredFiles := filterFiles(commitDiff.Files, conf.IgnorePatterns)
	if len(filteredFiles) == 0 {
		flog.Info("No files to review after filtering for commit %s (changed files: %d)",
			commitDiff.CommitID[:8], len(commitDiff.Files))
		return nil, nil
	}

	// Collect context for all files
	var filesContext []map[string]any
	windowSize := conf.ContextWindow

	endpoint, _ := providers.GetConfig(gitea.ID, gitea.EndpointKey)
	token, _ := providers.GetConfig(gitea.ID, gitea.TokenKey)
	client, err := gitea.NewGitea(endpoint.String(), token.String())
	if err != nil {
		flog.Error(fmt.Errorf("failed to create gitea client: %w", err))
		return nil, fmt.Errorf("failed to create gitea client: %w", err)
	}

	for _, filename := range filteredFiles {
		fileType := "unknown"
		if ext := filepath.Ext(filename); ext != "" {
			fileType = ext[1:]
		}

		// Get file context
		codeContext, err := client.GetFileContent(
			owner,
			repo,
			commitDiff.CommitID,
			filename,
			1,
			windowSize*2,
		)
		if err != nil {
			flog.Error(fmt.Errorf("error getting context for file %s in commit %s: %v",
				filename, commitDiff.CommitID[:8], err))
			continue
		}

		if len(codeContext) == 0 {
			flog.Warn("No context returned for file: %s in commit: %s",
				filename, commitDiff.CommitID[:8])
			continue
		}

		filesContext = append(filesContext, map[string]any{
			"file_path": filename,
			"file_type": fileType,
			"context":   codeContext,
		})
	}

	if len(filesContext) == 0 {
		flog.Warn("No valid file contexts collected for commit %s (total files: %d)",
			commitDiff.CommitID[:8], len(commitDiff.Files))
		return nil, nil
	}

	// Create context object
	codeContext := &CodeContext{
		Diff:         commitDiff.DiffContent,
		FilesContext: filesContext,
		Metadata: map[string]string{
			"commit_id":      commitDiff.CommitID,
			"commit_message": commitDiff.CommitMessage,
		},
	}

	return codeContext, nil
}

// analyzeCode analyzes the code changes in the entire commit
func analyzeCode(ctx context.Context, codeContext *CodeContext) (*ReviewResult, error) {
	flog.Debug("Analyzing commit: %s - %s (files: %d)",
		codeContext.Metadata["commit_id"][:8],
		strings.Split(codeContext.Metadata["commit_message"], "\n")[0][:50],
		len(codeContext.FilesContext))

	conf := DefaultConfig()

	result, err := llmAnalyzeCode(ctx, *codeContext)
	if err != nil {
		flog.Error(fmt.Errorf("Error analyzing code for commit %+v: %v\nFull error: %v",
			codeContext, err, err))
		// Return a default review result
		return &ReviewResult{
			Score:          0,
			Comments:       []string{"An error occurred during the code review process"},
			Suggestions:    []string{},
			Issues:         []*CodeIssue{},
			SecurityIssues: []*SecurityIssue{},
			QualityMetrics: &QualityMetric{
				SecurityScore:     0,
				PerformanceScore:  0,
				ReadabilityScore:  0,
				BestPracticeScore: 0,
			},
		}, fmt.Errorf("error analyzing code for commit %+v: %v", codeContext, err)
	}

	// Record review results
	flog.Info("Code analysis completed for commit %s with scores and %d files:",
		codeContext.Metadata["commit_id"][:8], len(codeContext.FilesContext))
	flog.Info("- Overall Score: %.1f/10 (weight: %f)", result.Score, conf.QualityThreshold)
	flog.Info("- Security: %.1f/10 (weight: %f)", result.QualityMetrics.SecurityScore, conf.ScoringRules["security"])
	flog.Info("- Performance: %.1f/10 (weight: %f)", result.QualityMetrics.PerformanceScore, conf.ScoringRules["performance"])
	flog.Info("- Readability: %.1f/10 (weight: %f)", result.QualityMetrics.ReadabilityScore, conf.ScoringRules["readability"])
	flog.Info("- Best Practices: %.1f/10 (weight: %f)", result.QualityMetrics.BestPracticeScore, conf.ScoringRules["best_practice"])

	if len(result.SecurityIssues) > 0 {
		flog.Warn("Found %d security issues in commit %s (threshold: %d)",
			len(result.SecurityIssues), codeContext.Metadata["commit_id"][:8], conf.MaxSecurityIssues)
	}

	return result, nil
}

type ReviewComment struct {
	Path     string `json:"path"`
	Line     int    `json:"line"`
	Body     string `json:"body"`
	CommitID string `json:"commit_id"`
}

// generateComments generates review comments
func generateComments(result *ReviewResult, codeContext *CodeContext) *ReviewComment {
	if result == nil || codeContext == nil {
		return nil
	}

	flog.Debug("Generating comments for commit: %s with %d issues",
		codeContext.Metadata["commit_id"][:8], len(result.Issues))

	// Add overall score comment
	overallComment := []string{
		"# 🔍 Code Review Report",
		"",
		fmt.Sprintf("## 📊 Score Overview (%.1f/10)", result.Score),
		"",
		"| Review Dimension | Score | Weight |",
		"|------------------|-------|--------|",
	}

	conf := DefaultConfig()
	overallComment = append(overallComment,
		fmt.Sprintf("| 🛡️ Security | %.1f/10 | %.0f |", result.QualityMetrics.SecurityScore, conf.ScoringRules["security"]*100),
		fmt.Sprintf("| ⚡ Performance | %.1f/10 | %.0f |", result.QualityMetrics.PerformanceScore, conf.ScoringRules["performance"]*100),
		fmt.Sprintf("| 📖 Readability | %.1f/10 | %.0f |", result.QualityMetrics.ReadabilityScore, conf.ScoringRules["readability"]*100),
		fmt.Sprintf("| ✨ Best Practices | %.1f/10 | %.0f |", result.QualityMetrics.BestPracticeScore, conf.ScoringRules["best_practice"]*100),
		"")

	// If there are issues
	if len(result.Issues) > 0 {
		overallComment = append(overallComment,
			"## 💡 Areas for Improvement",
			"")

		for _, issue := range result.Issues {
			endLineStr := ""
			if issue.EndLine > 0 {
				endLineStr = fmt.Sprintf("-%d行", issue.EndLine)
			}
			overallComment = append(overallComment,
				fmt.Sprintf("### %s", issue.FilePath),
				fmt.Sprintf("- Position: Line %d%s", issue.StartLine, endLineStr),
				fmt.Sprintf("- Issue: %s", issue.Description),
				fmt.Sprintf("- Suggestion: %s", issue.Suggestion),
				"")
		}
	}

	if len(result.SecurityIssues) > 0 {
		overallComment = append(overallComment,
			"## ⚠️ Security Issues",
			"")

		for _, issue := range result.SecurityIssues {
			severityIcon := "🟡"
			if strings.ToLower(issue.Severity) == "high" {
				severityIcon = "🔴"
			}

			endLineStr := ""
			if issue.EndLine > 0 {
				endLineStr = fmt.Sprintf("-%d行", issue.EndLine)
			}

			overallComment = append(overallComment,
				fmt.Sprintf("### %s %s", severityIcon, issue.FilePath),
				fmt.Sprintf("- Severity: %s", issue.Severity),
				fmt.Sprintf("- Position: Line %d%s", issue.StartLine, endLineStr),
				fmt.Sprintf("- Issue: %s", issue.Description),
				fmt.Sprintf("- Suggestion: %s", issue.Suggestion),
				"")
		}
	}

	comment := &ReviewComment{
		Path:     codeContext.Metadata["commit_message"],
		Line:     1,
		Body:     strings.Join(overallComment, "\n"),
		CommitID: codeContext.Metadata["commit_id"],
	}

	flog.Info("Generated 1 review comments for commit %s", codeContext.Metadata["commit_id"][:8])

	return comment
}

func reviewCommit(ctx context.Context, owner, repo, commitID string) (*ReviewComment, error) {
	flog.Info("Starting PR review for %s/%s #%s", owner, repo, commitID)

	endpoint, _ := providers.GetConfig(gitea.ID, gitea.EndpointKey)
	token, _ := providers.GetConfig(gitea.ID, gitea.TokenKey)
	client, err := gitea.NewGitea(endpoint.String(), token.String())
	if err != nil {
		return nil, fmt.Errorf("failed to create gitea client: %w", err)
	}

	commitDiffs, err := client.GetDiff(owner, repo, commitID)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit diff: %w", err)
	}
	flog.Info("commit diffs: %v %s", commitDiffs.CommitID, commitDiffs.CommitMessage)

	// Collect context for the entire commit
	codeContext, err := collectContext(owner, repo, commitDiffs)
	if err != nil {
		return nil, fmt.Errorf("failed to collect context: %w", err)
	}
	if codeContext == nil {
		flog.Warn("Skipping commit %s due to no reviewable files (total files: %d)",
			commitDiffs.CommitID[:8], len(commitDiffs.Files))
		return nil, nil
	}

	// Analyze the entire commit's code
	result, err := analyzeCode(ctx, codeContext)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze code: %w", err)
	}

	// Generate comments
	comment := generateComments(result, codeContext)

	conf := DefaultConfig()
	minScore := result.Score
	flog.Info("Commit review completed with minimum score: %v (threshold: %v)",
		minScore, conf.QualityThreshold)

	if minScore >= conf.QualityThreshold {
		flog.Info("Commit quality meets threshold (%v >= %v)",
			minScore, conf.QualityThreshold)
	} else {
		flog.Info("Commit quality below threshold (%v < %v)",
			minScore, conf.QualityThreshold)
	}

	return comment, nil
}
