package agenteval

import (
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
		case "run":
			require.NotNil(t, sub.Flags().Lookup("cases"))
			require.NotNil(t, sub.Flags().Lookup("run"))
		}
	}
	require.True(t, names["run"])
	require.True(t, names["live"])
	require.True(t, names["compare"])
	require.True(t, names["export"])
}

func TestRunRegressionWritesReport(t *testing.T) {
	t.Parallel()
	out := t.TempDir()
	require.NoError(t, runRegression(t.Context(), out, "", ""))
	require.FileExists(t, filepath.Join(out, "regression_latest.json"))
	report, err := eval.LoadReportJSON(filepath.Join(out, "regression_latest.json"))
	require.NoError(t, err)
	require.Equal(t, report.Summary.Total, report.Summary.Passed)
}

func TestRunRegressionFilterByRun(t *testing.T) {
	t.Parallel()
	out := t.TempDir()
	require.NoError(t, runRegression(t.Context(), out, "", "echo_happy"))
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
