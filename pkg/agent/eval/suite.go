package eval

import (
	"cmp"
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// DefaultSmokeCaseNames are capability cases kept when --smoke=true.
// Covers greet, reasoning, safety refuse (2), and honesty — τ-bench style reliability smoke.
var DefaultSmokeCaseNames = []string{
	"openqa_greet",
	"openqa_explain_panic",
	"openqa_refuse_secrets",
	"openqa_refuse_shell",
	"openqa_admit_unknown",
}

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

// BuiltinOpenQASmoke loads capability openqa YAML cases (full openqa set; apply FilterSmoke separately).
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

// BuiltinCapabilityScenarios loads openqa + tools capability cases.
func BuiltinCapabilityScenarios(workspaceParent string) ([]Scenario, error) {
	dirs, err := CapabilityCaseDirs()
	if err != nil {
		return nil, err
	}
	return LoadScenariosFromDirs(dirs, workspaceParent)
}

// CapabilityCaseDirs resolves openqa, tools, and repair case directories.
func CapabilityCaseDirs() ([]string, error) {
	openqa, err := ResolveCasesDir(
		filepath.Join("testdata", "capability", "openqa"),
		filepath.Join("pkg", "agent", "eval", "testdata", "capability", "openqa"),
	)
	if err != nil {
		return nil, err
	}
	tools, err := ResolveCasesDir(
		filepath.Join("testdata", "capability", "tools"),
		filepath.Join("pkg", "agent", "eval", "testdata", "capability", "tools"),
	)
	if err != nil {
		return nil, err
	}
	dirs := []string{openqa, tools}
	repair, err := ResolveCasesDir(
		filepath.Join("testdata", "capability", "repair"),
		filepath.Join("pkg", "agent", "eval", "testdata", "capability", "repair"),
	)
	if err == nil {
		dirs = append(dirs, repair)
	}
	return dirs, nil
}

// FilterByTier keeps scenarios whose Tier is in the allowlist (comma-separated or repeated).
// Empty spec returns scenarios unchanged.
func FilterByTier(scenarios []Scenario, spec string) ([]Scenario, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return scenarios, nil
	}
	want := make(map[string]struct{})
	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(strings.ToLower(part))
		if part == "" {
			continue
		}
		switch part {
		case TierBasic, TierCombo, TierSystem, TierRepair:
			want[part] = struct{}{}
		default:
			return nil, fmt.Errorf("eval: invalid --tier %q (use basic|combo|system|repair)", part)
		}
	}
	if len(want) == 0 {
		return nil, fmt.Errorf("eval: empty --tier spec")
	}
	out := make([]Scenario, 0, len(scenarios))
	for _, sc := range scenarios {
		if _, ok := want[NormalizeTier(sc.Tier)]; ok {
			out = append(out, sc)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("eval: no cases matched --tier %q", spec)
	}
	return out, nil
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

// FilterSmoke keeps only named cases when smoke is requested (stable allowlist).
func FilterSmoke(scenarios []Scenario, smoke bool, names []string) []Scenario {
	if !smoke || len(names) == 0 {
		return scenarios
	}
	want := make(map[string]struct{}, len(names))
	for _, n := range names {
		want[n] = struct{}{}
	}
	out := make([]Scenario, 0, len(names))
	for _, sc := range scenarios {
		if _, ok := want[sc.Name]; ok {
			out = append(out, sc)
		}
	}
	slices.SortStableFunc(out, func(a, b Scenario) int { return cmp.Compare(a.Name, b.Name) })
	return out
}

// FilterByRun keeps scenarios whose Name matches pattern (Go regexp), like go test -run.
// Empty pattern returns scenarios unchanged. No matches is an error.
func FilterByRun(scenarios []Scenario, pattern string) ([]Scenario, error) {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return scenarios, nil
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("eval: invalid -run regexp: %w", err)
	}
	out := make([]Scenario, 0, len(scenarios))
	for _, sc := range scenarios {
		if re.MatchString(sc.Name) {
			out = append(out, sc)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("eval: no cases matched -run %q", pattern)
	}
	slices.SortStableFunc(out, func(a, b Scenario) int { return cmp.Compare(a.Name, b.Name) })
	return out, nil
}

// LoadScenariosFromDirs loads YAML cases from multiple directories (order preserved, then sorted by name).
func LoadScenariosFromDirs(dirs []string, workspaceParent string) ([]Scenario, error) {
	var out []Scenario
	for _, dir := range dirs {
		part, err := LoadScenariosFromDir(dir, workspaceParent)
		if err != nil {
			return nil, err
		}
		out = append(out, part...)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("eval: no yaml cases in %v", dirs)
	}
	slices.SortStableFunc(out, func(a, b Scenario) int { return cmp.Compare(a.Name, b.Name) })
	return out, nil
}

// DefaultGoldByCaseFromDirs merges gold files from multiple case directories.
func DefaultGoldByCaseFromDirs(dirs []string, names []string) (map[string]GoldScores, error) {
	out := make(map[string]GoldScores, len(names))
	for _, dir := range dirs {
		part, err := DefaultGoldByCase(dir, names)
		if err != nil {
			return nil, err
		}
		for k, v := range part {
			out[k] = v
		}
	}
	return out, nil
}
