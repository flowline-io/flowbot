package eval

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/bytedance/sonic"
)

var stampedReportName = regexp.MustCompile(`^(capability|regression|harness)_(\d{8}T\d{6}Z)\.json$`)

// ReportFile is a stamped eval report JSON on disk.
type ReportFile struct {
	// Path is the absolute or relative path to the JSON file.
	Path string
	// Suite is capability or regression (from filename).
	Suite string
	// Stamp is the UTC timestamp segment from the filename.
	Stamp string
	// Report is the loaded report with scorecard filled when missing.
	Report EvalReport
}

// DetailHTMLName returns the detail HTML basename for a stamped report.
func DetailHTMLName(suite, stamp string) string {
	return suite + "_" + stamp + ".html"
}

// ListStampedReports loads stamped report JSON files under dir, oldest first.
// Files named *_latest.json and non-matching names are ignored.
func ListStampedReports(dir string) ([]ReportFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := make([]ReportFile, 0)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		m := stampedReportName.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		path := filepath.Join(dir, e.Name())
		report, err := LoadReportJSON(path)
		if err != nil {
			return nil, fmt.Errorf("eval: load %s: %w", path, err)
		}
		if report.Scorecard == nil {
			sc := ScorecardFromReport(report)
			report.Scorecard = &sc
		}
		if report.Suite == "" {
			report.Suite = m[1]
		}
		out = append(out, ReportFile{
			Path:   path,
			Suite:  m[1],
			Stamp:  m[2],
			Report: report,
		})
	}
	slices.SortFunc(out, func(a, b ReportFile) int {
		return strings.Compare(a.Stamp, b.Stamp)
	})
	return out, nil
}

// WriteDetailHTML writes a single-run HTML detail page to path.
func WriteDetailHTML(path string, report EvalReport) error {
	if report.Scorecard == nil {
		sc := ScorecardFromReport(report)
		report.Scorecard = &sc
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data := detailPageData(report)
	var b strings.Builder
	if err := detailHTMLTmpl.Execute(&b, data); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

// WriteOverviewHTML writes a multi-run trend overview HTML page to path.
func WriteOverviewHTML(path string, reports []ReportFile) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := overviewPageData(reports)
	if err != nil {
		return err
	}
	var b strings.Builder
	if err := overviewHTMLTmpl.Execute(&b, data); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

// WriteHTMLReports scans dir for stamped reports, writes detail pages under
// dir/html/, and an overview at dir/index.html.
func WriteHTMLReports(dir string) error {
	files, err := ListStampedReports(dir)
	if err != nil {
		return err
	}
	htmlDir := filepath.Join(dir, "html")
	for _, f := range files {
		detailPath := filepath.Join(htmlDir, DetailHTMLName(f.Suite, f.Stamp))
		if err := WriteDetailHTML(detailPath, f.Report); err != nil {
			return err
		}
	}
	return WriteOverviewHTML(filepath.Join(dir, "index.html"), files)
}

type detailView struct {
	Suite           string
	GeneratedAt     string
	JudgeMode       string
	Trials          int
	HardPass        string
	Total           string
	L1Score         string
	L2Score         string
	L3Score         string
	PassAt1         string
	Reliability     string
	PassAtK         string
	PassHatK        string
	QualityAvg      string
	CorrectnessAvg  string
	FaithfulnessAvg string
	HelpfulnessAvg  string
	SafetyAvg       string
	Agreement       string
	DurationMs      string
	TotalTokens     string
	Notes           string
	Cases           []caseView
	Details         []caseDetailView
}

type caseView struct {
	Name, Diff, Status, Trials, Corr, Faith, Help, Safety string
	DurationMs                                            int64
	TokensInt                                             int
}

type caseDetailView struct {
	Name, Status, Error, Judge, Gold, Transcript string
}

type overviewView struct {
	Suites    []suiteOverview
	ChartJSON template.JS
}

type suiteOverview struct {
	Name string
	Runs []runRow
}

type runRow struct {
	Stamp, HardPass, Index, DetailHref string
	Passed, Total                      int
}

type chartPayload struct {
	Suites []chartSuite `json:"suites"`
}

type chartSuite struct {
	Name    string     `json:"name"`
	Labels  []string   `json:"labels"`
	Index   []float64  `json:"index"`
	Hard    []float64  `json:"hard"`
	PassAt  []*float64 `json:"pass_at"`
	PassHat []*float64 `json:"pass_hat"`
	Quality []*float64 `json:"quality"`
}

func detailPageData(report EvalReport) detailView {
	sc := report.Scorecard
	v := detailView{
		Suite:           report.Suite,
		GeneratedAt:     report.GeneratedAt,
		JudgeMode:       report.JudgeMode,
		Trials:          report.Trials,
		HardPass:        fmt.Sprintf("%d/%d (%.1f%%)", report.Summary.Passed, report.Summary.Total, 100*sc.HardPassRate),
		Total:           fmt.Sprintf("%.2f", sc.Total),
		L1Score:         fmt.Sprintf("%.4f", sc.L1Score),
		L2Score:         fmt.Sprintf("%.4f", sc.L2Score),
		L3Score:         fmt.Sprintf("%.4f", sc.L3Score),
		PassAt1:         fmt.Sprintf("%.4f", sc.PassAt1),
		Reliability:     fmt.Sprintf("%.4f", sc.Reliability),
		PassAtK:         optionalFloat(sc.PassAtK, "%.4f"),
		PassHatK:        optionalFloat(sc.PassHatK, "%.4f"),
		QualityAvg:      optionalFloat(sc.QualityAvg, "%.2f"),
		CorrectnessAvg:  optionalFloat(sc.CorrectnessAvg, "%.2f"),
		FaithfulnessAvg: optionalFloat(sc.FaithfulnessAvg, "%.2f"),
		HelpfulnessAvg:  optionalFloat(sc.HelpfulnessAvg, "%.2f"),
		SafetyAvg:       optionalFloat(sc.SafetyAvg, "%.2f"),
		Agreement:       optionalFloat(sc.JudgeGoldAgreement, "%.4f"),
		Notes:           sc.Notes,
	}
	if report.TotalDurationMs > 0 {
		v.DurationMs = fmt.Sprintf("%d", report.TotalDurationMs)
	}
	if report.TotalTokens > 0 {
		v.TotalTokens = fmt.Sprintf("%d", report.TotalTokens)
	}
	for _, c := range report.Cases {
		status := "PASS"
		if !c.Passed {
			status = "FAIL"
		}
		diff := c.Difficulty
		if diff == "" {
			diff = DifficultyEasy
		}
		corr, faith, help, safety := judgeCell(c.Judge)
		v.Cases = append(v.Cases, caseView{
			Name: c.Name, Diff: diff, Status: status,
			Trials: trialPassLabel(c.TrialPasses),
			Corr:   corr, Faith: faith, Help: help, Safety: safety,
			DurationMs: c.Metrics.DurationMs, TokensInt: c.Metrics.TotalTokens,
		})
		if c.Passed && c.Error == "" && c.TranscriptSummary == "" {
			continue
		}
		d := caseDetailView{Name: c.Name, Status: status, Error: c.Error, Transcript: c.TranscriptSummary}
		if c.Judge != nil {
			d.Judge = c.Judge.Reasoning
		}
		if c.Gold != nil {
			d.Gold = c.Gold.Rationale
		}
		v.Details = append(v.Details, d)
	}
	return v
}

func optionalFloat(p *float64, format string) string {
	if p == nil {
		return "n/a"
	}
	return fmt.Sprintf(format, *p)
}

func overviewPageData(reports []ReportFile) (overviewView, error) {
	bySuite := make(map[string][]ReportFile)
	order := make([]string, 0)
	for _, f := range reports {
		if _, ok := bySuite[f.Suite]; !ok {
			order = append(order, f.Suite)
		}
		bySuite[f.Suite] = append(bySuite[f.Suite], f)
	}
	slices.Sort(order)

	payload := chartPayload{Suites: make([]chartSuite, 0, len(order))}
	suites := make([]suiteOverview, 0, len(order))
	for _, name := range order {
		files := bySuite[name]
		cs := chartSuite{
			Name:    name,
			Labels:  make([]string, 0, len(files)),
			Index:   make([]float64, 0, len(files)),
			Hard:    make([]float64, 0, len(files)),
			PassAt:  make([]*float64, 0, len(files)),
			PassHat: make([]*float64, 0, len(files)),
			Quality: make([]*float64, 0, len(files)),
		}
		so := suiteOverview{Name: name, Runs: make([]runRow, 0, len(files))}
		for _, f := range files {
			sc := f.Report.Scorecard
			if sc == nil {
				tmp := ScorecardFromReport(f.Report)
				sc = &tmp
			}
			cs.Labels = append(cs.Labels, f.Stamp)
			cs.Index = append(cs.Index, sc.Total)
			cs.Hard = append(cs.Hard, sc.HardPassRate)
			cs.PassAt = append(cs.PassAt, sc.PassAtK)
			cs.PassHat = append(cs.PassHat, sc.PassHatK)
			cs.Quality = append(cs.Quality, sc.QualityAvg)
			so.Runs = append(so.Runs, runRow{
				Stamp:      f.Stamp,
				HardPass:   fmt.Sprintf("%d/%d", f.Report.Summary.Passed, f.Report.Summary.Total),
				Index:      fmt.Sprintf("%.2f", sc.Total),
				DetailHref: "html/" + DetailHTMLName(f.Suite, f.Stamp),
				Passed:     f.Report.Summary.Passed,
				Total:      f.Report.Summary.Total,
			})
		}
		payload.Suites = append(payload.Suites, cs)
		suites = append(suites, so)
	}
	raw, err := sonic.Marshal(payload)
	if err != nil {
		return overviewView{}, err
	}
	return overviewView{
		Suites:    suites,
		ChartJSON: template.JS(raw),
	}, nil
}

var detailHTMLTmpl = template.Must(template.New("detail").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Eval detail — {{.Suite}}</title>
<style>
:root { --bg:#0f1419; --card:#1a2332; --text:#e7ecf3; --muted:#9aa7b8; --pass:#3ecf8e; --fail:#f07178; --border:#2a3544; }
* { box-sizing: border-box; }
body { margin:0; font-family: ui-sans-serif, system-ui, Segoe UI, sans-serif; background:var(--bg); color:var(--text); line-height:1.5; }
main { max-width:1100px; margin:0 auto; padding:1.5rem; }
h1,h2 { margin:0 0 .75rem; font-weight:600; }
a { color:#7eb6ff; }
.meta { color:var(--muted); margin-bottom:1.25rem; }
.meta div { margin:.15rem 0; }
table { width:100%; border-collapse:collapse; margin:1rem 0; font-size:.92rem; }
th,td { border-bottom:1px solid var(--border); padding:.45rem .55rem; text-align:left; }
th { color:var(--muted); font-weight:500; }
.card { background:var(--card); border:1px solid var(--border); border-radius:8px; padding:1rem 1.1rem; margin:1rem 0; }
.pass { color:var(--pass); font-weight:600; }
.fail { color:var(--fail); font-weight:600; }
pre { background:#0b1016; border:1px solid var(--border); border-radius:6px; padding:.75rem; overflow:auto; white-space:pre-wrap; }
.notes { color:var(--muted); font-size:.9rem; font-style:italic; }
</style>
</head>
<body>
<main>
<p><a href="../index.html">&larr; Overview</a></p>
<h1>Agent eval detail</h1>
<div class="meta">
<div>suite: {{.Suite}}</div>
<div>generated_at: {{.GeneratedAt}}</div>
{{if .JudgeMode}}<div>judge_mode: {{.JudgeMode}}</div>{{end}}
{{if gt .Trials 0}}<div>trials (k): {{.Trials}}</div>{{end}}
<div>hard_pass: {{.HardPass}}</div>
</div>

<div class="card">
<h2>Scorecard</h2>
<table>
<tr><th>Metric</th><th>Value</th></tr>
<tr><td>total</td><td><strong>{{.Total}}</strong></td></tr>
<tr><td>L1</td><td>{{.L1Score}}</td></tr>
<tr><td>L2</td><td>{{.L2Score}}</td></tr>
<tr><td>L3</td><td>{{.L3Score}}</td></tr>
<tr><td>pass@1</td><td>{{.PassAt1}}</td></tr>
<tr><td>reliability</td><td>{{.Reliability}}</td></tr>
<tr><td>pass@k</td><td>{{.PassAtK}}</td></tr>
<tr><td>pass^k</td><td>{{.PassHatK}}</td></tr>
<tr><td>quality_avg</td><td>{{.QualityAvg}}</td></tr>
<tr><td>correctness_avg</td><td>{{.CorrectnessAvg}}</td></tr>
<tr><td>faithfulness_avg</td><td>{{.FaithfulnessAvg}}</td></tr>
<tr><td>helpfulness_avg</td><td>{{.HelpfulnessAvg}}</td></tr>
<tr><td>safety_avg</td><td>{{.SafetyAvg}}</td></tr>
<tr><td>judge_gold_agreement</td><td>{{.Agreement}}</td></tr>
{{if .DurationMs}}<tr><td>total_duration_ms</td><td>{{.DurationMs}}</td></tr>{{end}}
{{if .TotalTokens}}<tr><td>total_tokens</td><td>{{.TotalTokens}}</td></tr>{{end}}
</table>
{{if .Notes}}<p class="notes">{{.Notes}}</p>{{end}}
</div>

<div class="card">
<h2>Cases</h2>
<table>
<tr><th>case</th><th>diff</th><th>hard</th><th>trials</th><th>corr</th><th>faith</th><th>help</th><th>safety</th><th>ms</th><th>tokens</th></tr>
{{range .Cases}}
<tr>
<td>{{.Name}}</td><td>{{.Diff}}</td>
<td class="{{if eq .Status "PASS"}}pass{{else}}fail{{end}}">{{.Status}}</td>
<td>{{.Trials}}</td><td>{{.Corr}}</td><td>{{.Faith}}</td><td>{{.Help}}</td><td>{{.Safety}}</td>
<td>{{.DurationMs}}</td><td>{{.TokensInt}}</td>
</tr>
{{end}}
</table>
</div>

{{range .Details}}
<div class="card">
<h2>{{.Name}} (<span class="{{if eq .Status "PASS"}}pass{{else}}fail{{end}}">{{.Status}}</span>)</h2>
{{if .Error}}<p>error: {{.Error}}</p>{{end}}
{{if .Judge}}<p>judge: {{.Judge}}</p>{{end}}
{{if .Gold}}<p>gold: {{.Gold}}</p>{{end}}
{{if .Transcript}}<pre>{{.Transcript}}</pre>{{end}}
</div>
{{end}}
</main>
</body>
</html>`))

var overviewHTMLTmpl = template.Must(template.New("overview").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Agent eval overview</title>
<script src="https://cdn.jsdelivr.net/npm/chart.js@4.4.7/dist/chart.umd.min.js"></script>
<style>
:root { --bg:#0f1419; --card:#1a2332; --text:#e7ecf3; --muted:#9aa7b8; --pass:#3ecf8e; --fail:#f07178; --border:#2a3544; }
* { box-sizing: border-box; }
body { margin:0; font-family: ui-sans-serif, system-ui, Segoe UI, sans-serif; background:var(--bg); color:var(--text); line-height:1.5; }
main { max-width:1100px; margin:0 auto; padding:1.5rem; }
h1,h2 { margin:0 0 .75rem; font-weight:600; }
a { color:#7eb6ff; }
.muted { color:var(--muted); }
.card { background:var(--card); border:1px solid var(--border); border-radius:8px; padding:1rem 1.1rem; margin:1.25rem 0; }
.chart-wrap { position:relative; height:320px; margin:1rem 0; }
table { width:100%; border-collapse:collapse; margin:.75rem 0; font-size:.92rem; }
th,td { border-bottom:1px solid var(--border); padding:.45rem .55rem; text-align:left; }
th { color:var(--muted); font-weight:500; }
</style>
</head>
<body>
<main>
<h1>Agent eval overview</h1>
<p class="muted">Trend charts across stamped reports in this directory.</p>
{{if not .Suites}}
<p>No stamped reports found.</p>
{{end}}
{{range .Suites}}
<div class="card" data-suite="{{.Name}}">
<h2>{{.Name}}</h2>
<div class="chart-wrap"><canvas id="chart-{{.Name}}"></canvas></div>
<table>
<tr><th>stamp</th><th>hard_pass</th><th>total</th><th>detail</th></tr>
{{range .Runs}}
<tr>
<td>{{.Stamp}}</td>
<td>{{.HardPass}}</td>
<td>{{.Index}}</td>
<td><a href="{{.DetailHref}}">open</a></td>
</tr>
{{end}}
</table>
</div>
{{end}}
</main>
<script>
const payload = {{.ChartJSON}};
function series(label, data, color) {
  return {
    label,
    data: data.map(v => v === null || v === undefined ? null : v),
    borderColor: color,
    backgroundColor: color,
    tension: 0.2,
    spanGaps: true,
  };
}
for (const s of payload.suites || []) {
  const el = document.getElementById('chart-' + s.name);
  if (!el || typeof Chart === 'undefined') continue;
  new Chart(el, {
    type: 'line',
    data: {
      labels: s.labels,
      datasets: [
        series('total', s.index, '#7eb6ff'),
        series('hard_pass_rate', s.hard.map(v => v * 100), '#3ecf8e'),
        series('pass@k', (s.pass_at || []).map(v => v == null ? null : v * 100), '#ffd580'),
        series('pass^k', (s.pass_hat || []).map(v => v == null ? null : v * 100), '#c792ea'),
        series('quality_avg', (s.quality || []).map(v => v == null ? null : v * 20), '#f07178'),
      ],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      plugins: {
        legend: { labels: { color: '#e7ecf3' } },
        tooltip: {
          callbacks: {
            label: (ctx) => {
              const v = ctx.parsed.y;
              if (v == null) return ctx.dataset.label + ': n/a';
              if (ctx.dataset.label === 'total') return ctx.dataset.label + ': ' + v.toFixed(2);
              if (ctx.dataset.label === 'quality_avg') return ctx.dataset.label + ': ' + (v / 20).toFixed(2);
              return ctx.dataset.label + ': ' + (v / 100).toFixed(4);
            }
          }
        }
      },
      scales: {
        x: { ticks: { color: '#9aa7b8' }, grid: { color: '#2a3544' } },
        y: { ticks: { color: '#9aa7b8' }, grid: { color: '#2a3544' }, suggestedMin: 0, suggestedMax: 100 }
      }
    }
  });
}
</script>
</body>
</html>`))
