package eval

import (
	"path/filepath"
)

// BuiltinRegressionScenarios loads YAML cases from testdata/regression.
func BuiltinRegressionScenarios(workspaceParent string) ([]Scenario, error) {
	dir, err := ResolveCasesDir(
		filepath.Join("testdata", "regression"),
		filepath.Join("pkg", "agent", "eval", "testdata", "regression"),
	)
	if err != nil {
		return nil, err
	}
	return LoadScenariosFromDir(dir, workspaceParent)
}

// BuiltinOpenQASmoke loads capability openqa YAML cases (smoke-sized by file set).
func BuiltinOpenQASmoke() ([]Scenario, error) {
	dir, err := ResolveCasesDir(
		filepath.Join("testdata", "capability", "openqa"),
		filepath.Join("pkg", "agent", "eval", "testdata", "capability", "openqa"),
	)
	if err != nil {
		return nil, err
	}
	return LoadScenariosFromDir(dir, "")
}

// OpenQAGoldDir resolves the openqa gold directory.
func OpenQAGoldDir() (string, error) {
	return ResolveCasesDir(
		filepath.Join("testdata", "capability", "openqa"),
		filepath.Join("pkg", "agent", "eval", "testdata", "capability", "openqa"),
	)
}

// LimitSmoke truncates scenarios to at most n when smoke is requested.
func LimitSmoke(scenarios []Scenario, smoke bool, n int) []Scenario {
	if !smoke || n <= 0 || len(scenarios) <= n {
		return scenarios
	}
	return scenarios[:n]
}
