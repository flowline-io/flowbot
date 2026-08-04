package agenteval

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/agent/eval"
)

func TestEvalCommandShape(t *testing.T) {
	t.Parallel()
	cmd := EvalCommand()
	require.Equal(t, "agenteval", cmd.Use)
	names := map[string]bool{}
	for _, sub := range cmd.Commands() {
		names[sub.Name()] = true
		switch sub.Name() {
		case "live":
			require.NotNil(t, sub.Flags().Lookup("smoke"))
			require.NotNil(t, sub.Flags().Lookup("model"))
			require.NotNil(t, sub.Flags().Lookup("judge-model"))
			require.NotNil(t, sub.Flags().Lookup("cases"))
			require.NotNil(t, sub.Flags().Lookup("run"))
			require.NotNil(t, sub.Flags().Lookup("difficulty"))
		case "run":
			require.NotNil(t, sub.Flags().Lookup("cases"))
			require.NotNil(t, sub.Flags().Lookup("run"))
		case "harness":
			require.NotNil(t, sub.Flags().Lookup("cases"))
			require.NotNil(t, sub.Flags().Lookup("run"))
		}
	}
	require.True(t, names["run"])
	require.True(t, names["harness"])
	require.True(t, names["live"])
	require.True(t, names["compare"])
	require.True(t, names["export"])
	require.True(t, names["report"])
}

func TestRunHarnessWritesReport(t *testing.T) {
	t.Parallel()
	out := t.TempDir()
	require.NoError(t, runHarnessRegression(t.Context(), out, "", "", liveFlags{}))
	require.FileExists(t, filepath.Join(out, "harness_latest.json"))
	report, err := eval.LoadReportJSON(filepath.Join(out, "harness_latest.json"))
	require.NoError(t, err)
	require.Equal(t, "harness", report.Suite)
	require.Equal(t, report.Summary.Total, report.Summary.Passed)
}

func TestRunRegressionWritesReport(t *testing.T) {
	t.Parallel()
	out := t.TempDir()
	require.NoError(t, runRegression(t.Context(), out, "", "", liveFlags{}))
	require.FileExists(t, filepath.Join(out, "regression_latest.json"))
	require.FileExists(t, filepath.Join(out, "index.html"))
	report, err := eval.LoadReportJSON(filepath.Join(out, "regression_latest.json"))
	require.NoError(t, err)
	require.Equal(t, report.Summary.Total, report.Summary.Passed)
}

func TestRunReportSingleDetail(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "capability_20260101T000000Z.json")
	report := eval.NewReport("capability", []eval.CaseResult{{Name: "x", Passed: true}})
	require.NoError(t, eval.WriteReportJSON(jsonPath, report))
	htmlOut := filepath.Join(dir, "custom_html")
	require.NoError(t, runReport(dir, jsonPath, htmlOut))
	require.FileExists(t, filepath.Join(htmlOut, "capability_20260101T000000Z.html"))
}

func TestRunRegressionFilterByRun(t *testing.T) {
	t.Parallel()
	out := t.TempDir()
	require.NoError(t, runRegression(t.Context(), out, "", "echo_happy", liveFlags{}))
	report, err := eval.LoadReportJSON(filepath.Join(out, "regression_latest.json"))
	require.NoError(t, err)
	require.Equal(t, 1, report.Summary.Total)
	require.Equal(t, "echo_happy", report.Cases[0].Name)
	require.Equal(t, 1, report.Summary.Passed)
}

func TestLimitSmokeLogic(t *testing.T) {
	t.Parallel()
	scenarios := make([]eval.Scenario, 7)
	require.Len(t, eval.LimitSmoke(scenarios, true, 5), 5)
	require.Len(t, eval.LimitSmoke(scenarios, false, 5), 7)
}

func TestRunCapabilityFakeRepeatsDoesNotUnderflowScripts(t *testing.T) {
	t.Parallel()
	out := t.TempDir()
	err := runCapability(t.Context(), liveFlags{
		outDir:    out,
		smoke:     true,
		trials:    1,
		repeats:   2,
		judgeFake: true,
	})
	require.NoError(t, err)
	report, loadErr := eval.LoadReportJSON(filepath.Join(out, "capability_latest.json"))
	require.NoError(t, loadErr)
	require.NotNil(t, report.Scorecard)
}

func TestFilterSmokeLogic(t *testing.T) {
	t.Parallel()
	scenarios := []eval.Scenario{
		{Name: "openqa_greet"},
		{Name: "other"},
		{Name: "openqa_admit_unknown"},
	}
	got := eval.FilterSmoke(scenarios, true, eval.DefaultSmokeCaseNames)
	require.Len(t, got, 2)
}

func TestResolveLoadOptions(t *testing.T) {
	t.Parallel()

	t.Run("invalid sandbox mode", func(t *testing.T) {
		t.Parallel()
		_, err := resolveLoadOptions(liveFlags{sandboxMode: "bad"})
		require.Error(t, err)
	})

	t.Run("docker mode accepts empty image with default", func(t *testing.T) {
		t.Parallel()
		opts, err := resolveLoadOptions(liveFlags{sandboxMode: "docker"})
		require.NoError(t, err)
		require.NotNil(t, opts.Sandbox)
		require.Equal(t, fmt.Sprintf("%T", eval.NewDockerSandbox(eval.DockerSandboxConfig{})), fmt.Sprintf("%T", opts.Sandbox))
	})
}
