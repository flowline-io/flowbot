package eval_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/eval"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListStampedReports(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	early := sampleReport("capability", "case_a", true)
	late := sampleReport("capability", "case_b", false)
	require.NoError(t, eval.WriteReportJSON(filepath.Join(dir, "capability_20260101T120000Z.json"), early))
	require.NoError(t, eval.WriteReportJSON(filepath.Join(dir, "capability_20260102T120000Z.json"), late))
	require.NoError(t, eval.WriteReportJSON(filepath.Join(dir, "capability_latest.json"), late))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("skip"), 0o644))

	files, err := eval.ListStampedReports(dir)
	require.NoError(t, err)
	require.Len(t, files, 2)
	assert.Equal(t, "20260101T120000Z", files[0].Stamp)
	assert.Equal(t, "20260102T120000Z", files[1].Stamp)
	assert.Equal(t, "capability", files[0].Suite)
	assert.NotNil(t, files[0].Report.Scorecard)
}

func TestWriteDetailHTML(t *testing.T) {
	t.Parallel()
	report := sampleReport("capability", "tools_edit", false)
	report.Cases[0].Error = "missing <script>alert(1)</script>"
	report.Cases[0].TranscriptSummary = "trace <b>bold</b>"
	sc := eval.ScorecardFromReport(report)
	report.Scorecard = &sc

	path := filepath.Join(t.TempDir(), "detail.html")
	require.NoError(t, eval.WriteDetailHTML(path, report))
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	body := string(raw)
	assert.Contains(t, body, "tools_edit")
	assert.Contains(t, body, "FAIL")
	assert.Contains(t, body, ">total<")
	assert.Contains(t, body, "&lt;script&gt;")
	assert.NotContains(t, body, "missing <script>alert")
	assert.Contains(t, body, "&lt;b&gt;bold&lt;/b&gt;")
}

func TestWriteOverviewHTML(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	r1 := sampleReport("capability", "a", true)
	r2 := sampleReport("capability", "a", false)
	require.NoError(t, eval.WriteReportJSON(filepath.Join(dir, "capability_20260101T010000Z.json"), r1))
	require.NoError(t, eval.WriteReportJSON(filepath.Join(dir, "capability_20260102T010000Z.json"), r2))
	files, err := eval.ListStampedReports(dir)
	require.NoError(t, err)

	out := filepath.Join(dir, "index.html")
	require.NoError(t, eval.WriteOverviewHTML(out, files))
	raw, err := os.ReadFile(out)
	require.NoError(t, err)
	body := string(raw)
	assert.Contains(t, body, "chart.js")
	assert.Contains(t, body, ">total<")
	assert.Contains(t, body, "20260101T010000Z")
	assert.Contains(t, body, "20260102T010000Z")
	assert.Contains(t, body, "html/capability_20260101T010000Z.html")
	assert.Contains(t, body, "html/capability_20260102T010000Z.html")
}

func TestWriteHTMLReports(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	r := sampleReport("regression", "echo_happy", true)
	require.NoError(t, eval.WriteReportJSON(filepath.Join(dir, "regression_20260301T000000Z.json"), r))

	require.NoError(t, eval.WriteHTMLReports(dir))
	require.FileExists(t, filepath.Join(dir, "index.html"))
	detail := filepath.Join(dir, "html", "regression_20260301T000000Z.html")
	require.FileExists(t, detail)
	raw, err := os.ReadFile(detail)
	require.NoError(t, err)
	assert.Contains(t, string(raw), "echo_happy")
}

func sampleReport(suite, caseName string, passed bool) eval.EvalReport {
	passAt, passHat := 1.0, 0.5
	if !passed {
		passAt, passHat = 0.5, 0.0
	}
	report := eval.EvalReport{
		Suite:       suite,
		GeneratedAt: "2026-01-01T00:00:00Z",
		JudgeMode:   "fake",
		Trials:      3,
		PassAtK:     &passAt,
		PassHatK:    &passHat,
		Summary: eval.ReportSummary{
			Total:  1,
			Passed: boolToInt(passed),
			Failed: boolToInt(!passed),
		},
		Cases: []eval.CaseResult{{
			Name:        caseName,
			Difficulty:  eval.DifficultyMedium,
			Passed:      passed,
			Metrics:     eval.Metrics{DurationMs: 10, TotalTokens: 3, Passed: passed},
			TrialPasses: []bool{passed, passed, passed},
		}},
	}
	sc := eval.ScorecardFromReport(report)
	report.Scorecard = &sc
	return report
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
