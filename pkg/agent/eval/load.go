package eval

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/goccy/go-yaml"
	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
	"github.com/flowline-io/flowbot/pkg/agent/tools/coding"
	"github.com/flowline-io/flowbot/pkg/agent/tools/echo"
)

// caseFile is the on-disk YAML shape for regression/capability scenarios.
type caseFile struct {
	Name      string        `yaml:"name"`
	Suite     string        `yaml:"suite"`
	Prompt    string        `yaml:"prompt"`
	Toolset   string        `yaml:"toolset"`
	Workspace bool          `yaml:"workspace"`
	Fixtures  []caseFixture `yaml:"fixtures"`
	Scripts   []caseScript  `yaml:"scripts"`
	Expect    caseExpect    `yaml:"expect"`
}

// caseFixture seeds a file into the isolated workspace before the run.
type caseFixture struct {
	Path    string `yaml:"path"`
	Content string `yaml:"content"`
}

type caseScript struct {
	Type    string `yaml:"type"` // tool_call | text
	ID      string `yaml:"id"`
	Name    string `yaml:"name"`
	Args    string `yaml:"args"`
	Content string `yaml:"content"`
}

type caseExpect struct {
	RequiredTools     []string            `yaml:"required_tools"`
	ForbiddenTools    []string            `yaml:"forbidden_tools"`
	ExpectedTools     []string            `yaml:"expected_tools"`
	StrictToolOrder   bool                `yaml:"strict_tool_order"`
	RequiredArgs      map[string][]string `yaml:"required_args"`
	MaxSteps          int                 `yaml:"max_steps"`
	SoftMaxSteps      bool                `yaml:"soft_max_steps"`
	RequireCompletion bool                `yaml:"require_completion"`
	Outcome           caseOutcome         `yaml:"outcome"`
}

type caseOutcome struct {
	FinalTextContains    []string         `yaml:"final_text_contains"`
	FinalTextContainsAny []string         `yaml:"final_text_contains_any"`
	Files                []caseFileAssert `yaml:"files"`
}

type caseFileAssert struct {
	Path     string `yaml:"path"`
	Contains string `yaml:"contains"`
	Equals   string `yaml:"equals"`
}

// LoadScenariosFromDir loads *.yaml case files from dir into Scenarios.
// workspaceParent is used when a case sets workspace: true.
func LoadScenariosFromDir(dir, workspaceParent string) ([]Scenario, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("eval: read cases dir: %w", err)
	}
	var out []Scenario
	for _, ent := range entries {
		if ent.IsDir() || filepath.Ext(ent.Name()) != ".yaml" {
			continue
		}
		path := filepath.Join(dir, ent.Name())
		sc, err := LoadScenarioFile(path, workspaceParent)
		if err != nil {
			return nil, fmt.Errorf("eval: load %s: %w", path, err)
		}
		out = append(out, sc)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("eval: no yaml cases in %s", dir)
	}
	return out, nil
}

// LoadScenarioFile loads one YAML case file.
func LoadScenarioFile(path, workspaceParent string) (Scenario, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Scenario{}, err
	}
	var cf caseFile
	if err := yaml.Unmarshal(raw, &cf); err != nil {
		return Scenario{}, err
	}
	if strings.TrimSpace(cf.Name) == "" || strings.TrimSpace(cf.Prompt) == "" {
		return Scenario{}, fmt.Errorf("name and prompt are required")
	}
	scripts, err := scriptsFromCase(cf.Scripts)
	if err != nil {
		return Scenario{}, err
	}
	sc := scenarioFromCase(cf, scripts)
	if err := prepareScenarioWorkspace(&sc, cf, workspaceParent); err != nil {
		return Scenario{}, err
	}
	tools, err := toolsForToolset(cf.Toolset, sc.WorkspaceRoot)
	if err != nil {
		return Scenario{}, err
	}
	sc.Tools = tools
	return sc, nil
}

func scriptsFromCase(in []caseScript) ([]agentllm.ResponseScript, error) {
	scripts := make([]agentllm.ResponseScript, 0, len(in))
	for _, s := range in {
		switch s.Type {
		case "tool_call":
			scripts = append(scripts, ToolCallScript(s.ID, s.Name, s.Args))
		case "text":
			scripts = append(scripts, TextScript(s.Content))
		default:
			return nil, fmt.Errorf("unknown script type %q", s.Type)
		}
	}
	return scripts, nil
}

func scenarioFromCase(cf caseFile, scripts []agentllm.ResponseScript) Scenario {
	sc := Scenario{
		Name:    cf.Name,
		Suite:   cf.Suite,
		Prompt:  cf.Prompt,
		Scripts: scripts,
		Expect: Expectation{
			RequiredTools:     cf.Expect.RequiredTools,
			ForbiddenTools:    cf.Expect.ForbiddenTools,
			ExpectedTools:     cf.Expect.ExpectedTools,
			StrictToolOrder:   cf.Expect.StrictToolOrder,
			RequiredArgs:      cf.Expect.RequiredArgs,
			MaxSteps:          cf.Expect.MaxSteps,
			SoftMaxSteps:      cf.Expect.SoftMaxSteps,
			RequireCompletion: cf.Expect.RequireCompletion,
			Outcome: OutcomeAsserts{
				FinalTextContains:    cf.Expect.Outcome.FinalTextContains,
				FinalTextContainsAny: cf.Expect.Outcome.FinalTextContainsAny,
			},
		},
	}
	for _, f := range cf.Expect.Outcome.Files {
		sc.Expect.Outcome.Files = append(sc.Expect.Outcome.Files, FileAssert{
			Path: f.Path, Contains: f.Contains, Equals: f.Equals,
		})
	}
	return sc
}

func prepareScenarioWorkspace(sc *Scenario, cf caseFile, workspaceParent string) error {
	needsWorkspace := cf.Workspace || len(cf.Fixtures) > 0 || toolsetNeedsWorkspace(cf.Toolset)
	if !needsWorkspace {
		return nil
	}
	root := filepath.Join(workspaceParent, cf.Name)
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	sc.WorkspaceRoot = root
	return writeFixtures(root, cf.Fixtures)
}

func toolsetNeedsWorkspace(toolset string) bool {
	switch strings.TrimSpace(toolset) {
	case "write_file", "read_file", "fs":
		return true
	default:
		return false
	}
}

func writeFixtures(root string, fixtures []caseFixture) error {
	for _, f := range fixtures {
		rel := filepath.FromSlash(strings.TrimSpace(f.Path))
		if rel == "" || strings.Contains(rel, "..") {
			return fmt.Errorf("invalid fixture path %q", f.Path)
		}
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(full, []byte(f.Content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func toolsForToolset(toolset, workspaceRoot string) ([]tool.Tool, error) {
	ws := coding.Workspace{Root: workspaceRoot}
	switch strings.TrimSpace(toolset) {
	case "", "echo":
		return []tool.Tool{echo.Tool{}}, nil
	case "write_file":
		if workspaceRoot == "" {
			return nil, fmt.Errorf("write_file toolset requires workspace")
		}
		return []tool.Tool{coding.WriteFileTool{Workspace: ws}}, nil
	case "read_file":
		if workspaceRoot == "" {
			return nil, fmt.Errorf("read_file toolset requires workspace")
		}
		return []tool.Tool{coding.ReadFileTool{Workspace: ws}}, nil
	case "fs":
		if workspaceRoot == "" {
			return nil, fmt.Errorf("fs toolset requires workspace")
		}
		return []tool.Tool{
			coding.ReadFileTool{Workspace: ws},
			coding.WriteFileTool{Workspace: ws},
			echo.Tool{},
		}, nil
	default:
		return nil, fmt.Errorf("unknown toolset %q", toolset)
	}
}

// ResolveCasesDir finds the first existing directory among candidates.
func ResolveCasesDir(candidates ...string) (string, error) {
	var tried []string
	for _, c := range candidates {
		tried = append(tried, c)
		st, err := os.Stat(c)
		if err == nil && st.IsDir() {
			return c, nil
		}
	}
	if root := findModuleRoot(); root != "" {
		for _, c := range candidates {
			if filepath.IsAbs(c) {
				continue
			}
			abs := filepath.Join(root, c)
			tried = append(tried, abs)
			st, err := os.Stat(abs)
			if err == nil && st.IsDir() {
				return abs, nil
			}
		}
	}
	return "", fmt.Errorf("eval: cases dir not found among %v", tried)
}

func findModuleRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// LoadGoldFile reads gold.json from path.
func LoadGoldFile(path string) (GoldScores, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return GoldScores{}, err
	}
	var gold GoldScores
	if err := sonic.Unmarshal(data, &gold); err != nil {
		return GoldScores{}, fmt.Errorf("parse gold json: %w", err)
	}
	return gold, nil
}

// DefaultGoldByCase loads gold files named <case>.gold.json (or nested gold.json).
// Missing files are skipped; corrupt existing files return an error.
func DefaultGoldByCase(dir string, names []string) (map[string]GoldScores, error) {
	out := make(map[string]GoldScores, len(names))
	for _, name := range names {
		candidates := []string{
			filepath.Join(dir, name+".gold.json"),
			filepath.Join(dir, name, "gold.json"),
			filepath.Join(dir, "gold", name+".json"),
		}
		loaded := false
		for _, p := range candidates {
			_, statErr := os.Stat(p)
			if errors.Is(statErr, os.ErrNotExist) {
				continue
			}
			if statErr != nil {
				return nil, fmt.Errorf("eval: stat gold %s: %w", p, statErr)
			}
			gold, err := LoadGoldFile(p)
			if err != nil {
				return nil, fmt.Errorf("eval: gold %s: %w", p, err)
			}
			out[name] = gold
			loaded = true
			break
		}
		_ = loaded
	}
	return out, nil
}
