package gitea

import "fmt"

var (
	ErrMissingCommitMessage = fmt.Errorf("missing commit message in metadata")
	ErrMissingDiffContent   = fmt.Errorf("missing diff content")
	ErrEmptyPrompt          = fmt.Errorf("empty prompt after formatting")
)

type CodeIssue struct {
	FilePath    string `json:"file_path"`
	StartLine   int    `json:"start_line"`
	EndLine     int    `json:"end_line"`
	Description string `json:"description"`
	Suggestion  string `json:"suggestion"`
}

type SecurityIssue struct {
	Severity    string `json:"severity"`
	FilePath    string `json:"file_path"`
	StartLine   int    `json:"start_line"`
	EndLine     int    `json:"end_line"`
	Description string `json:"description"`
	Suggestion  string `json:"suggestion"`
}

type QualityMetric struct {
	SecurityScore     float64 `json:"security_score"`
	PerformanceScore  float64 `json:"performance_score"`
	ReadabilityScore  float64 `json:"readability_score"`
	BestPracticeScore float64 `json:"best_practice_score"`
}

type ReviewResult struct {
	Comments       []string         `json:"comments"`
	Suggestions    []string         `json:"suggestions"`
	Score          float64          `json:"score"`
	Issues         []*CodeIssue     `json:"issues"`
	SecurityIssues []*SecurityIssue `json:"security_issues"`
	QualityMetrics *QualityMetric   `json:"quality_metrics"`
}

type CodeContext struct {
	Diff         string            `json:"diff"`
	FilesContext []map[string]any  `json:"files_context"`
	Metadata     map[string]string `json:"metadata"`
}

func (c *CodeContext) Len() int {
	return len(c.FilesContext)
}

const ReviewPrompt = `You are a professional code review expert. Please review the following code changes. During the review, please pay special attention to the following points:

1. Security (30% weight):
   - Check for SQL injection, XSS and other security vulnerabilities
   - Check for sensitive information exposure
   - Check for permission control issues

2. Performance (20% weight):
   - Check algorithm complexity
   - Check resource usage efficiency 
   - Check concurrency handling

3. Readability (20% weight):
   - Code formatting standards
   - Clear naming conventions
   - Sufficient comments

4. Best Practices (30% weight):
   - Following design patterns
   - Unit test coverage
   - Type hints
   - SOLID principles compliance

Scoring Rules:
- Security issues: High risk -3 points each, Medium risk -1 point each
- Performance issues: -2 points each
- Readability issues: -0.5 points each
- Best practices: Missing unit tests -2 points, No type hints -1 point

Commit Message:
{commit_message}

Code changes:
{diff}

Related file context:
{files_context}

Please provide a detailed review result, including:
1. Overall score (out of 10)
2. List of specific issues (with file paths and code locations)
3. List of security issues (with file paths and code locations)
4. Detailed scores for each dimension

Please return the result in JSON format as follows:
{{
    "score": float,
    "issues": [
        {{
            "file_path": string,
            "start_line": int,
            "end_line": int | null,
            "description": string,
            "suggestion": string
        }}
    ],
    "security_issues": [
        {{
            "severity": string,
            "file_path": string,
            "start_line": int,
            "end_line": int | null,
            "description": string,
            "suggestion": string
        }}
    ],
    "quality_metrics": {{
        "security_score": float,
        "performance_score": float,
        "readability_score": float,
        "best_practice_score": float
    }}
}}

Note:
1. Each issue must specify the exact file path and code location (line number)
2. If an issue involves multiple lines of code, provide start_line and end_line
3. If an issue only involves a single line of code, end_line can be null
4. All line numbers must be actual code line numbers
5. Please answer in {language}
`

type Config struct {
	QualityThreshold  float64            `json:"quality_threshold" title:"Quality Threshold Score"`
	MaxSecurityIssues int                `json:"max_security_issues" title:"Maximum Number of Security Issues"`
	IgnorePatterns    []string           `json:"ignore_patterns" title:"Ignored File Patterns"`
	ScoringRules      map[string]float64 `json:"scoring_rules" title:"Scoring Rule Weights"`
	ContextWindow     int                `json:"context_window" title:"Code Context Window Size"`
	MaxTokens         int                `json:"max_tokens" title:"Maximum Token Count"`
}

func DefaultConfig() *Config {
	return &Config{
		ContextWindow:     5,
		MaxTokens:         4096,
		QualityThreshold:  8.5,
		MaxSecurityIssues: 5,
		IgnorePatterns: []string{
			"**/node_modules/", "**/vendor/", "**/venv/", "**/.venv/",
			"**/bower_components/", "**/jspm_packages/", "**/packages/",
			"**/deps/", "**/dist/", "**/build/", "**/out/", "**/target/",
			"**/bin/", "**/obj/", "**/*.exe", "**/*.dll", "**/*.so",
			"**/*.a", "**/*.jar", "**/*.class", "**/*.pyc",
			"**/__pycache__/", "**/*.egg-info/", "**/.DS_Store",
			"**/Thumbs.db", "**/Desktop.ini", "**/.idea/", "**/.vscode/",
			"**/.vs/", "**/*.suo", "**/*.user", "**/*.sublime-project",
			"**/*.sublime-workspace", "**/*.log", "**/logs/", "**/tmp/",
			"**/*.tmp", "**/*.swp", "**/*.swo", "**/.sass-cache/",
			"**/coverage/", "**/.nyc_output/", "**/junit.xml",
			"**/test-results/", "**/*.min.js", "**/*.min.css", "**/*.map",
			"**/public/static/", "**/compiled/", "**/generated/", "**/.env",
			"**/.env.local", "**/.env.*.local", "**/docker-compose.override.yml",
			"**/*.key", "**/*.pem", "**/*.crt", "**/docs/_build/",
			"**/site/", "**/.vuepress/dist/", "**/package-lock.json",
			"**/yarn.lock", "**/Gemfile.lock", "**/Podfile.lock",
		},
		ScoringRules: map[string]float64{
			"security":      0.3,
			"performance":   0.2,
			"readability":   0.2,
			"best_practice": 0.3,
		},
	}
}
