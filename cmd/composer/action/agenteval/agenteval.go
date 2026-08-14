// Package agenteval provides composer commands for agent evaluation suites.
package agenteval

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/spf13/cobra"
	"github.com/tmc/langchaingo/llms"

	"github.com/flowline-io/flowbot/pkg/agent/eval"
	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/flowline-io/flowbot/pkg/config"
)

const (
	defaultOutDir     = "tmp/agent_eval"
	defaultTrials     = 3
	defaultConfigPath = "."
)

// EvalCommand returns the agenteval command group.
func EvalCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agenteval",
		Short: "run agent evaluation suites and write local reports",
	}
	cmd.AddCommand(runCommand())
	cmd.AddCommand(harnessCommand())
	cmd.AddCommand(liveCommand())
	cmd.AddCommand(compareCommand())
	cmd.AddCommand(exportCommand())
	cmd.AddCommand(reportCommand())
	return cmd
}

func harnessCommand() *cobra.Command {
	var (
		outDir         string
		casesDir       string
		runPat         string
		sandboxMode    string
		sandboxImage   string
		sandboxNetwork string
		sandboxMemory  string
	)
	cmd := &cobra.Command{
		Use:   "harness",
		Short: "run harness reliability suite with FakeModel",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runHarnessRegression(cmd.Context(), outDir, casesDir, runPat, liveFlags{
				sandboxMode: sandboxMode, sandboxImage: sandboxImage,
				sandboxNetwork: sandboxNetwork, sandboxMemory: sandboxMemory,
			})
		},
	}
	cmd.Flags().StringVar(&outDir, "out", defaultOutDir, "output directory for JSON/Markdown reports")
	cmd.Flags().StringVar(&casesDir, "cases", "", "harness cases directory (default: package testdata/harness)")
	cmd.Flags().StringVar(&runPat, "run", "", "regexp matching case names (like go test -run)")
	cmd.Flags().StringVar(&sandboxMode, "sandbox", "workspace", "workspace|docker")
	cmd.Flags().StringVar(&sandboxImage, "sandbox-image", "", "docker sandbox image override")
	cmd.Flags().StringVar(&sandboxNetwork, "sandbox-network", "", "docker sandbox network mode")
	cmd.Flags().StringVar(&sandboxMemory, "sandbox-memory", "", "docker sandbox memory limit, e.g. 512m")
	return cmd
}

func runCommand() *cobra.Command {
	var (
		outDir         string
		casesDir       string
		runPat         string
		sandboxMode    string
		sandboxImage   string
		sandboxNetwork string
		sandboxMemory  string
	)
	cmd := &cobra.Command{
		Use:   "run",
		Short: "run regression suite with FakeModel (CI-safe)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runRegression(cmd.Context(), outDir, casesDir, runPat, liveFlags{
				sandboxMode: sandboxMode, sandboxImage: sandboxImage,
				sandboxNetwork: sandboxNetwork, sandboxMemory: sandboxMemory,
			})
		},
	}
	cmd.Flags().StringVar(&outDir, "out", defaultOutDir, "output directory for JSON/Markdown reports")
	cmd.Flags().StringVar(&casesDir, "cases", "", "regression cases directory (default: package testdata/regression)")
	cmd.Flags().StringVar(&runPat, "run", "", "regexp matching case names (like go test -run)")
	cmd.Flags().StringVar(&sandboxMode, "sandbox", "workspace", "workspace|docker")
	cmd.Flags().StringVar(&sandboxImage, "sandbox-image", "", "docker sandbox image override")
	cmd.Flags().StringVar(&sandboxNetwork, "sandbox-network", "", "docker sandbox network mode")
	cmd.Flags().StringVar(&sandboxMemory, "sandbox-memory", "", "docker sandbox memory limit, e.g. 512m")
	return cmd
}

func liveCommand() *cobra.Command {
	var (
		outDir          string
		casesDir        string
		trials          int
		smoke           bool
		modelName       string
		judgeName       string
		judgeFake       bool
		configPath      string
		runPat          string
		difficulty      string
		tier            string
		latencyBudgetMs int64
		tokenBudget     int
		repeats         int
		sandboxMode     string
		sandboxImage    string
		sandboxNetwork  string
		sandboxMemory   string
	)
	cmd := &cobra.Command{
		Use:   "live",
		Short: "run capability suite (FakeModel offline, or --model / --judge-model from flowbot.yaml)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runCapability(cmd.Context(), liveFlags{
				outDir: outDir, casesDir: casesDir, trials: trials, smoke: smoke,
				modelName: modelName, judgeName: judgeName, judgeFake: judgeFake, configPath: configPath,
				runPattern: runPat, difficulty: difficulty, tier: tier,
				latencyBudgetMs: latencyBudgetMs, tokenBudget: tokenBudget, repeats: repeats,
				sandboxMode: sandboxMode, sandboxImage: sandboxImage,
				sandboxNetwork: sandboxNetwork, sandboxMemory: sandboxMemory,
			})
		},
	}
	cmd.Flags().StringVar(&outDir, "out", defaultOutDir, "output directory for JSON/Markdown reports")
	cmd.Flags().StringVar(&casesDir, "cases", "", "capability cases directory (default: testdata/capability/*)")
	cmd.Flags().IntVar(&trials, "trials", defaultTrials, "number of trials per task (k)")
	cmd.Flags().BoolVar(&smoke, "smoke", true, "run DefaultSmokeCaseNames subset; ignored when --run, --difficulty, or --tier is set")
	cmd.Flags().StringVar(&modelName, "model", "", "subject model name from flowbot.yaml (empty = FakeModel scripts)")
	cmd.Flags().StringVar(&judgeName, "judge-model", "", "judge model name from flowbot.yaml (requires --model unless --judge-fake)")
	cmd.Flags().BoolVar(&judgeFake, "judge-fake", true, "use scripted FakeModel judge (set false with --judge-model for real judge)")
	cmd.Flags().StringVar(&configPath, "config", defaultConfigPath, "directory containing flowbot.yaml (for --model/--judge-model)")
	cmd.Flags().StringVar(&runPat, "run", "", "regexp matching case names (like go test -run)")
	cmd.Flags().StringVar(&difficulty, "difficulty", "", "easy|medium|hard|medium+|hard+|easy,hard (skips --smoke when set)")
	cmd.Flags().StringVar(&tier, "tier", "", "basic|combo|system|repair (comma list; skips --smoke when set)")
	cmd.Flags().Int64Var(&latencyBudgetMs, "latency-budget-ms", eval.DefaultLatencyBudgetMs, "per-trial latency budget for L3 LatencyScore")
	cmd.Flags().IntVar(&tokenBudget, "token-budget", eval.DefaultTokenBudget, "per-trial token budget for L3 TokenScore")
	cmd.Flags().IntVar(&repeats, "repeats", 1, "suite-level repeats for Total/L1/L2/L3 mean+/-CI (N>=2)")
	cmd.Flags().StringVar(&sandboxMode, "sandbox", "workspace", "workspace|docker")
	cmd.Flags().StringVar(&sandboxImage, "sandbox-image", "", "docker sandbox image override")
	cmd.Flags().StringVar(&sandboxNetwork, "sandbox-network", "", "docker sandbox network mode")
	cmd.Flags().StringVar(&sandboxMemory, "sandbox-memory", "", "docker sandbox memory limit, e.g. 512m")
	return cmd
}

type liveFlags struct {
	outDir, casesDir, modelName, judgeName, configPath, runPattern, difficulty, tier string
	trials                                                                           int
	smoke, judgeFake                                                                 bool
	latencyBudgetMs                                                                  int64
	tokenBudget, repeats                                                             int
	sandboxMode, sandboxImage, sandboxNetwork, sandboxMemory                         string
}

func compareCommand() *cobra.Command {
	var baseline, candidate, outPath string
	cmd := &cobra.Command{
		Use:   "compare",
		Short: "compare two eval report JSON files",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runCompare(baseline, candidate, outPath)
		},
	}
	cmd.Flags().StringVar(&baseline, "baseline", "", "baseline report.json path")
	cmd.Flags().StringVar(&candidate, "candidate", "", "candidate report.json path")
	cmd.Flags().StringVar(&outPath, "out", "", "optional markdown output path")
	_ = cmd.MarkFlagRequired("baseline")
	_ = cmd.MarkFlagRequired("candidate")
	return cmd
}

func exportCommand() *cobra.Command {
	var reportPath, outDir string
	var onlyFailed bool
	cmd := &cobra.Command{
		Use:   "export",
		Short: "export failed cases from a report as task draft YAML files",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runExport(reportPath, outDir, onlyFailed)
		},
	}
	cmd.Flags().StringVar(&reportPath, "report", "", "report.json path")
	cmd.Flags().StringVar(&outDir, "out", filepath.Join(defaultOutDir, "drafts"), "directory for draft YAML files")
	cmd.Flags().BoolVar(&onlyFailed, "failed-only", true, "export only failed cases")
	_ = cmd.MarkFlagRequired("report")
	return cmd
}

func reportCommand() *cobra.Command {
	var dir, reportPath, outDir string
	cmd := &cobra.Command{
		Use:   "report",
		Short: "generate HTML overview charts and per-run detail pages",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runReport(dir, reportPath, outDir)
		},
	}
	cmd.Flags().StringVar(&dir, "dir", defaultOutDir, "directory of stamped eval JSON reports")
	cmd.Flags().StringVar(&reportPath, "report", "", "optional single report.json for detail HTML only")
	cmd.Flags().StringVar(&outDir, "out", "", "detail HTML output dir when --report is set (default: <dir>/html)")
	return cmd
}

func runRegression(ctx context.Context, outDir, casesDir, runPattern string, flags liveFlags) error {
	workspace := filepath.Join(outDir, "workspaces", fmt.Sprintf("%d", time.Now().UnixNano()))
	loadOpts, err := resolveLoadOptions(flags)
	if err != nil {
		return err
	}
	var scenarios []eval.Scenario
	if casesDir != "" {
		scenarios, err = eval.LoadScenariosFromDirWithOptions(casesDir, workspace, loadOpts)
	} else {
		scenarios, err = eval.BuiltinRegressionScenariosWithOptions(workspace, loadOpts)
	}
	if err != nil {
		return err
	}
	scenarios, err = eval.FilterByRun(scenarios, runPattern)
	if err != nil {
		return err
	}
	cases := make([]eval.CaseResult, 0, len(scenarios))
	for idx, sc := range scenarios {
		printProgress(eval.ProgressEvent{
			Phase: "case_start", CaseName: sc.Name,
			CaseIndex: idx + 1, CaseTotal: len(scenarios),
		})
		start := time.Now()
		run, err := eval.RunFakeScenario(ctx, sc)
		if err != nil {
			return err
		}
		cr := eval.CaseResultFromRun(sc.Name, run)
		cr.Difficulty = eval.NormalizeDifficulty(sc.Difficulty)
		cr.Tier = eval.NormalizeTier(sc.Tier)
		cr.TrialMetrics = []eval.Metrics{run.Metrics}
		cr.TrialPasses = []bool{run.Metrics.Passed}
		if !cr.Passed && cr.Error == "" {
			cr.Error = eval.FailReason(run.Metrics, sc.Expect)
		}
		cases = append(cases, cr)
		printProgress(eval.ProgressEvent{
			Phase: "case_done", CaseName: sc.Name,
			CaseIndex: idx + 1, CaseTotal: len(scenarios),
			Passed: cr.Passed, Duration: time.Since(start), Detail: cr.Error,
		})
	}
	return writeReports(outDir, "regression", eval.NewReport("regression", cases))
}

func runHarnessRegression(ctx context.Context, outDir, casesDir, runPattern string, flags liveFlags) error {
	workspace := filepath.Join(outDir, "workspaces", fmt.Sprintf("%d", time.Now().UnixNano()))
	loadOpts, err := resolveLoadOptions(flags)
	if err != nil {
		return err
	}
	var scenarios []eval.Scenario
	if casesDir != "" {
		scenarios, err = eval.LoadScenariosFromDirWithOptions(casesDir, workspace, loadOpts)
	} else {
		scenarios, err = eval.BuiltinHarnessScenariosWithOptions(workspace, loadOpts)
	}
	if err != nil {
		return err
	}
	scenarios, err = eval.FilterByRun(scenarios, runPattern)
	if err != nil {
		return err
	}
	cases := make([]eval.CaseResult, 0, len(scenarios))
	for idx, sc := range scenarios {
		printProgress(eval.ProgressEvent{
			Phase: "case_start", CaseName: sc.Name,
			CaseIndex: idx + 1, CaseTotal: len(scenarios),
		})
		start := time.Now()
		run, err := eval.RunFakeScenarioWithHarness(ctx, sc)
		if err != nil {
			return err
		}
		cr := eval.CaseResultFromRun(sc.Name, run)
		cr.Difficulty = eval.NormalizeDifficulty(sc.Difficulty)
		cr.Tier = eval.NormalizeTier(sc.Tier)
		cr.TrialMetrics = []eval.Metrics{run.Metrics}
		cr.TrialPasses = []bool{run.Metrics.Passed}
		if !cr.Passed && cr.Error == "" {
			cr.Error = eval.FailReason(run.Metrics, sc.Expect)
		}
		cases = append(cases, cr)
		printProgress(eval.ProgressEvent{
			Phase: "case_done", CaseName: sc.Name,
			CaseIndex: idx + 1, CaseTotal: len(scenarios),
			Passed: cr.Passed, Duration: time.Since(start), Detail: cr.Error,
		})
	}
	return writeReports(outDir, "harness", eval.NewReport("harness", cases))
}

func runCapability(ctx context.Context, f liveFlags) error {
	workspace := filepath.Join(f.outDir, "workspaces", fmt.Sprintf("%d", time.Now().UnixNano()))
	scenarios, goldDirs, err := loadCapabilityScenarios(f, workspace)
	if err != nil {
		return err
	}
	names := make([]string, 0, len(scenarios))
	for _, sc := range scenarios {
		names = append(names, sc.Name)
	}
	goldByCase, err := eval.DefaultGoldByCaseFromDirs(goldDirs, names)
	if err != nil {
		return err
	}
	subject, err := resolveSubjectModel(ctx, f, scenarios)
	if err != nil {
		return err
	}
	judgeModel, err := resolveJudgeModel(ctx, f, len(scenarios))
	if err != nil {
		return err
	}
	repeats := f.repeats
	if repeats <= 0 {
		repeats = 1
	}
	cards := make([]eval.CapabilityScorecard, 0, repeats)
	var report eval.EvalReport
	for i := 0; i < repeats; i++ {
		one, err := eval.RunLiveScenarios(ctx, scenarios, subject, eval.LiveOptions{
			Trials:          f.trials,
			ModelName:       f.modelName,
			JudgeModel:      judgeModel,
			GoldByCase:      goldByCase,
			OnProgress:      printProgress,
			JudgeMode:       judgeModeFromFlags(f, judgeModel != nil),
			LatencyBudgetMs: f.latencyBudgetMs,
			TokenBudget:     f.tokenBudget,
		})
		if err != nil {
			return err
		}
		report = one
		if one.Scorecard != nil {
			cards = append(cards, *one.Scorecard)
		}
	}
	if repeats >= 2 && len(cards) >= 2 {
		merged := eval.MergeRepeatScorecards(cards)
		report.Scorecard = &merged
	}
	return writeReports(f.outDir, "capability", report)
}

func loadCapabilityScenarios(f liveFlags, workspace string) ([]eval.Scenario, []string, error) {
	loadOpts, err := resolveLoadOptions(f)
	if err != nil {
		return nil, nil, err
	}
	var scenarios []eval.Scenario
	var goldDirs []string
	if f.casesDir != "" {
		scenarios, err = eval.LoadScenariosFromDirWithOptions(f.casesDir, workspace, loadOpts)
		goldDirs = []string{f.casesDir}
	} else {
		scenarios, err = eval.BuiltinCapabilityScenariosWithOptions(workspace, loadOpts)
		if err == nil {
			goldDirs, err = eval.CapabilityCaseDirs()
		}
	}
	if err != nil {
		return nil, nil, err
	}
	scenarios, err = applyCapabilityFilters(scenarios, f)
	if err != nil {
		return nil, nil, err
	}
	if len(scenarios) == 0 {
		return nil, nil, fmt.Errorf("agenteval: no capability cases selected")
	}
	return scenarios, goldDirs, nil
}

func applyCapabilityFilters(scenarios []eval.Scenario, f liveFlags) ([]eval.Scenario, error) {
	useSmoke := f.smoke && f.runPattern == "" && strings.TrimSpace(f.difficulty) == "" && strings.TrimSpace(f.tier) == ""
	if useSmoke {
		return eval.FilterSmoke(scenarios, true, eval.DefaultSmokeCaseNames), nil
	}
	var err error
	if f.difficulty != "" {
		scenarios, err = eval.FilterByDifficulty(scenarios, f.difficulty)
		if err != nil {
			return nil, err
		}
	}
	if f.tier != "" {
		scenarios, err = eval.FilterByTier(scenarios, f.tier)
		if err != nil {
			return nil, err
		}
	}
	if f.runPattern != "" {
		scenarios, err = eval.FilterByRun(scenarios, f.runPattern)
		if err != nil {
			return nil, err
		}
	}
	return scenarios, nil
}

func judgeModeFromFlags(f liveFlags, hasJudge bool) string {
	if !hasJudge {
		return "none"
	}
	if f.judgeFake && f.judgeName == "" {
		return "fake"
	}
	if f.judgeName != "" {
		return "model:" + f.judgeName
	}
	return "model"
}

func resolveLoadOptions(f liveFlags) (eval.LoadOptions, error) {
	mode := strings.ToLower(strings.TrimSpace(f.sandboxMode))
	switch mode {
	case "", "workspace":
		return eval.LoadOptions{Sandbox: eval.NewWorkspaceSandbox()}, nil
	case "docker":
		return eval.LoadOptions{Sandbox: eval.NewDockerSandbox(eval.DockerSandboxConfig{
			Image:   strings.TrimSpace(f.sandboxImage),
			Network: strings.TrimSpace(f.sandboxNetwork),
			Memory:  strings.TrimSpace(f.sandboxMemory),
		})}, nil
	default:
		return eval.LoadOptions{}, fmt.Errorf("agenteval: invalid --sandbox %q (use workspace|docker)", f.sandboxMode)
	}
}

func printProgress(ev eval.ProgressEvent) {
	switch ev.Phase {
	case "case_start":
		_, _ = fmt.Fprintf(os.Stdout, "=== RUN   %s\n", ev.CaseName)
	case "trial":
		status := "PASS"
		if !ev.Passed {
			status = "FAIL"
		}
		_, _ = fmt.Fprintf(os.Stdout, "    --- %s: trial %d/%d (%.3fs)\n",
			status, ev.Trial, ev.Trials, ev.Duration.Seconds())
		if !ev.Passed && ev.Detail != "" {
			_, _ = fmt.Fprintf(os.Stdout, "        %s\n", ev.Detail)
		}
	case "case_done":
		status := "PASS"
		if !ev.Passed {
			status = "FAIL"
		}
		_, _ = fmt.Fprintf(os.Stdout, "--- %s: %s (%.3fs)\n", status, ev.CaseName, ev.Duration.Seconds())
		if !ev.Passed && ev.Detail != "" {
			_, _ = fmt.Fprintf(os.Stdout, "    %s\n", ev.Detail)
		}
	}
}

func resolveSubjectModel(ctx context.Context, f liveFlags, scenarios []eval.Scenario) (llms.Model, error) {
	if f.modelName == "" {
		trials := f.trials
		if trials <= 0 {
			trials = defaultTrials
		}
		repeats := f.repeats
		if repeats <= 0 {
			repeats = 1
		}
		scripts := make([]agentllm.ResponseScript, 0)
		for r := 0; r < repeats; r++ {
			for _, sc := range scenarios {
				for i := 0; i < trials; i++ {
					scripts = append(scripts, sc.Scripts...)
				}
			}
		}
		return agentllm.NewFakeModel(scripts...), nil
	}
	if err := config.Load(f.configPath); err != nil {
		return nil, fmt.Errorf("load config for --model: %w", err)
	}
	model, _, err := agentllm.NewModel(ctx, f.modelName)
	if err != nil {
		return nil, err
	}
	return model, nil
}

func resolveJudgeModel(ctx context.Context, f liveFlags, caseCount int) (llms.Model, error) {
	if f.judgeFake && f.judgeName == "" {
		// JudgeAll scores 4 dimensions per case (last trial only in live path).
		repeats := f.repeats
		if repeats <= 0 {
			repeats = 1
		}
		n := caseCount*4*repeats + 8
		return agentllm.NewFakeModel(fakeJudgeScripts(n)...), nil
	}
	if f.judgeName == "" {
		return nil, fmt.Errorf("agenteval: --judge-model is required when --judge-fake=false")
	}
	if f.judgeName == f.modelName && f.modelName != "" {
		return nil, fmt.Errorf("agenteval: --judge-model must differ from --model")
	}
	if err := config.Load(f.configPath); err != nil {
		return nil, fmt.Errorf("load config for --judge-model: %w", err)
	}
	model, _, err := agentllm.NewModel(ctx, f.judgeName)
	if err != nil {
		return nil, err
	}
	return model, nil
}

func fakeJudgeScripts(n int) []agentllm.ResponseScript {
	scripts := make([]agentllm.ResponseScript, 0, n)
	body := `{"score":5,"unknown":false,"reasoning":"ok"}`
	for range n {
		scripts = append(scripts, agentllm.ResponseScript{Content: body})
	}
	return scripts
}

func runCompare(baselinePath, candidatePath, outPath string) error {
	baseline, err := eval.LoadReportJSON(baselinePath)
	if err != nil {
		return fmt.Errorf("baseline: %w", err)
	}
	candidate, err := eval.LoadReportJSON(candidatePath)
	if err != nil {
		return fmt.Errorf("candidate: %w", err)
	}
	diff := eval.CompareReports(baseline, candidate)
	md := eval.FormatCompareMarkdown(&diff)
	_, _ = fmt.Fprint(os.Stdout, md)
	if outPath != "" {
		if err := os.MkdirAll(filepath.Dir(outPath), 0o750); err != nil {
			return err
		}
		if err := os.WriteFile(outPath, []byte(md), 0o644); err != nil {
			return err
		}
	}
	data, err := sonic.MarshalIndent(diff, "", "  ")
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintln(os.Stdout, string(data))
	return nil
}

func runExport(reportPath, outDir string, onlyFailed bool) error {
	report, err := eval.LoadReportJSON(reportPath)
	if err != nil {
		return err
	}
	n := 0
	for _, c := range report.Cases {
		if onlyFailed && c.Passed {
			continue
		}
		path, err := eval.WriteTaskDraftYAML(outDir, eval.ExportTaskDraft(c, ""))
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintln(os.Stdout, path)
		n++
	}
	_, _ = fmt.Fprintf(os.Stdout, "exported %d draft(s) to %s\n", n, outDir)
	return nil
}

func runReport(dir, reportPath, outDir string) error {
	if reportPath != "" {
		report, err := eval.LoadReportJSON(reportPath)
		if err != nil {
			return err
		}
		if report.Scorecard == nil {
			sc := eval.ScorecardFromReport(report)
			report.Scorecard = &sc
		}
		suite := report.Suite
		if suite == "" {
			suite = "report"
		}
		stamp := stampFromReportPath(reportPath, report.GeneratedAt)
		if outDir == "" {
			outDir = filepath.Join(dir, "html")
		}
		detailPath := filepath.Join(outDir, eval.DetailHTMLName(suite, stamp))
		if err := eval.WriteDetailHTML(detailPath, report); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(os.Stdout, "wrote %s\n", detailPath)
		return nil
	}
	if err := eval.WriteHTMLReports(dir); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(os.Stdout, "wrote %s\n", filepath.Join(dir, "index.html"))
	_, _ = fmt.Fprintf(os.Stdout, "wrote %s\n", filepath.Join(dir, "html"))
	return nil
}

func stampFromReportPath(path, generatedAt string) string {
	base := filepath.Base(path)
	if m := stampedReportRe.FindStringSubmatch(base); m != nil {
		return m[2]
	}
	if t, err := time.Parse(time.RFC3339, generatedAt); err == nil {
		return t.UTC().Format("20060102T150405Z")
	}
	return time.Now().UTC().Format("20060102T150405Z")
}

var stampedReportRe = regexp.MustCompile(`^(capability|regression|harness)_(\d{8}T\d{6}Z)\.json$`)

func writeReports(outDir, prefix string, report eval.EvalReport) error {
	if report.Scorecard == nil {
		sc := eval.ScorecardFromReport(report)
		report.Scorecard = &sc
	}
	stamp := time.Now().UTC().Format("20060102T150405Z")
	jsonPath := filepath.Join(outDir, prefix+"_"+stamp+".json")
	mdPath := filepath.Join(outDir, prefix+"_"+stamp+".md")
	latestJSON := filepath.Join(outDir, prefix+"_latest.json")
	latestMD := filepath.Join(outDir, prefix+"_latest.md")
	if err := eval.WriteReportJSON(jsonPath, report); err != nil {
		return err
	}
	if err := eval.WriteReportMarkdown(mdPath, report); err != nil {
		return err
	}
	if err := eval.WriteReportJSON(latestJSON, report); err != nil {
		return err
	}
	if err := eval.WriteReportMarkdown(latestMD, report); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(os.Stdout, "wrote %s\n", jsonPath)
	_, _ = fmt.Fprintf(os.Stdout, "wrote %s\n", mdPath)
	if err := eval.WriteHTMLReports(outDir); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(os.Stdout, "wrote %s\n", filepath.Join(outDir, "index.html"))
	printScorecard(report)
	if report.Summary.Failed > 0 {
		for _, c := range report.Cases {
			if !c.Passed {
				_, _ = fmt.Fprintf(os.Stdout, "FAIL\t%s\n", c.Name)
			}
		}
		return fmt.Errorf("agenteval: %d case(s) failed", report.Summary.Failed)
	}
	return nil
}

func printScorecard(report eval.EvalReport) {
	sc := report.Scorecard
	if sc == nil {
		return
	}
	_, _ = fmt.Fprintf(os.Stdout, "total: %.2f (L1=%.4f L2=%.4f L3=%.4f)\n", sc.Total, sc.L1Score, sc.L2Score, sc.L3Score)
	_, _ = fmt.Fprintf(os.Stdout, "pass@1: %.4f  reliability: %.4f  hard_pass: %d/%d\n",
		sc.PassAt1, sc.Reliability, report.Summary.Passed, report.Summary.Total)
	if sc.PassAtK != nil && sc.PassHatK != nil {
		_, _ = fmt.Fprintf(os.Stdout, "pass@k: %.4f  pass^k: %.4f\n", *sc.PassAtK, *sc.PassHatK)
	}
	if sc.QualityEnabled && sc.QualityAvg != nil {
		_, _ = fmt.Fprintf(os.Stdout, "quality_avg: %.2f (C=%.2f F=%.2f H=%.2f S=%.2f)\n",
			*sc.QualityAvg, *sc.CorrectnessAvg, *sc.FaithfulnessAvg, *sc.HelpfulnessAvg, *sc.SafetyAvg)
	} else if report.JudgeMode == "fake" || !sc.QualityEnabled {
		_, _ = fmt.Fprintln(os.Stdout, "quality: n/a — use --judge-model NAME --judge-fake=false for dimension scores")
	}
}
