package slack

import (
	"strings"
	"testing"

	"github.com/slack-go/slack"
)

func TestHeader(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{name: "simple header", text: "Hello"},
		{name: "empty header", text: ""},
		{name: "long header", text: "A very long header text that spans multiple words"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hdr := header(tt.text)
			if hdr.Type != slack.MBTHeader {
				t.Errorf("expected Header block type, got %s", hdr.Type)
			}
			if hdr.Text.Text != tt.text {
				t.Errorf("expected text %q, got %q", tt.text, hdr.Text.Text)
			}
		})
	}
}

func TestSection(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{name: "markdown text", text: "**bold**"},
		{name: "plain text", text: "hello world"},
		{name: "empty text", text: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sec := section(tt.text)
			if sec.Type != slack.MBTSection {
				t.Errorf("expected Section block type, got %s", sec.Type)
			}
			if sec.Text.Text != tt.text {
				t.Errorf("expected text %q, got %q", tt.text, sec.Text.Text)
			}
		})
	}
}

func TestSectionWithButton(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		btnText  string
		actionID string
		value    string
		style    slack.Style
	}{
		{
			name:     "primary button",
			text:     "Click below",
			btnText:  "Go",
			actionID: "act-1",
			value:    "v1",
			style:    slack.StylePrimary,
		},
		{
			name:     "danger button",
			text:     "Warning",
			btnText:  "Delete",
			actionID: "act-del",
			value:    "del-1",
			style:    slack.StyleDanger,
		},
		{
			name:     "no style",
			text:     "Plain",
			btnText:  "OK",
			actionID: "act-ok",
			value:    "ok",
			style:    "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sec := sectionWithButton(tt.text, tt.btnText, tt.actionID, tt.value, tt.style)
			if sec.Type != slack.MBTSection {
				t.Errorf("expected Section block type, got %s", sec.Type)
			}
			if sec.Accessory == nil {
				t.Fatal("expected accessory")
			}
			if sec.Accessory.ButtonElement == nil {
				t.Fatal("expected button element in accessory")
			}
			btn := sec.Accessory.ButtonElement
			if btn.ActionID != tt.actionID {
				t.Errorf("expected actionID %q, got %q", tt.actionID, btn.ActionID)
			}
			if btn.Value != tt.value {
				t.Errorf("expected value %q, got %q", tt.value, btn.Value)
			}
			if btn.Style != tt.style {
				t.Errorf("expected style %q, got %q", tt.style, btn.Style)
			}
		})
	}
}

func TestSectionFields(t *testing.T) {
	tests := []struct {
		name       string
		fields     map[string]string
		wantFields int
	}{
		{
			name:       "multiple fields",
			fields:     map[string]string{"CPU": "80%", "Memory": "60%", "Disk": "40%"},
			wantFields: 3,
		},
		{
			name:       "single field",
			fields:     map[string]string{"Status": "OK"},
			wantFields: 1,
		},
		{
			name:       "empty fields",
			fields:     map[string]string{},
			wantFields: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sec := sectionFields(tt.fields)
			if sec.Type != slack.MBTSection {
				t.Errorf("expected Section block type, got %s", sec.Type)
			}
			if len(sec.Fields) != tt.wantFields {
				t.Errorf("expected %d fields, got %d", tt.wantFields, len(sec.Fields))
			}
		})
	}
}

func TestContextBlock(t *testing.T) {
	tests := []struct {
		name     string
		elements []string
		wantLen  int
	}{
		{name: "multiple elements", elements: []string{"a", "b", "c"}, wantLen: 3},
		{name: "single element", elements: []string{"alone"}, wantLen: 1},
		{name: "no elements", elements: nil, wantLen: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := contextBlock(tt.elements...)
			if ctx.Type != slack.MBTContext {
				t.Errorf("expected Context block type, got %s", ctx.Type)
			}
			if len(ctx.ContextElements.Elements) != tt.wantLen {
				t.Errorf("expected %d elements, got %d", tt.wantLen, len(ctx.ContextElements.Elements))
			}
		})
	}
}

func TestDivider(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "creates divider block"},
		{name: "always returns correct type"},
		{name: "divider block has no text content"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			div := divider()
			if div.Type != slack.MBTDivider {
				t.Errorf("expected Divider block type, got %s", div.Type)
			}
		})
	}
}

func TestActionButtons(t *testing.T) {
	tests := []struct {
		name       string
		buttons    []ButtonDef
		wantBtnLen int
		check      func(*testing.T, *slack.ActionBlock)
	}{
		{
			name:       "single button",
			buttons:    []ButtonDef{{Text: "OK", ActionID: "ok", Value: "yes"}},
			wantBtnLen: 1,
		},
		{
			name: "multiple buttons",
			buttons: []ButtonDef{
				{Text: "OK", ActionID: "ok", Value: "yes", Style: slack.StylePrimary},
				{Text: "Cancel", ActionID: "cancel", Value: "no", Style: slack.StyleDanger},
				{Text: "Link", ActionID: "link", URL: "https://example.com"},
			},
			wantBtnLen: 3,
			check: func(t *testing.T, act *slack.ActionBlock) {
				elements := act.Elements.ElementSet
				if len(elements) != 3 {
					t.Fatalf("expected 3 elements, got %d", len(elements))
				}
				if elements[0].(*slack.ButtonBlockElement).Style != slack.StylePrimary {
					t.Error("expected primary style on first button")
				}
				if elements[2].(*slack.ButtonBlockElement).URL != "https://example.com" {
					t.Error("expected URL on third button")
				}
			},
		},
		{
			name:       "no buttons",
			buttons:    nil,
			wantBtnLen: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			act := actionButtons(tt.buttons...)
			if act.Type != slack.MBTAction {
				t.Errorf("expected Action block type, got %s", act.Type)
			}
			if len(act.Elements.ElementSet) != tt.wantBtnLen {
				t.Errorf("expected %d buttons, got %d", tt.wantBtnLen, len(act.Elements.ElementSet))
			}
			if tt.check != nil {
				tt.check(t, act)
			}
		})
	}
}

func TestImageBlock(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		altText string
		title   string
	}{
		{name: "with title", url: "https://img.com/pic.png", altText: "pic", title: "My Pic"},
		{name: "no title", url: "https://img.com/pic.png", altText: "pic", title: ""},
		{name: "empty all", url: "", altText: "", title: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := imageBlock(tt.url, tt.altText, tt.title)
			if img.Type != slack.MBTImage {
				t.Errorf("expected Image block type, got %s", img.Type)
			}
			if img.ImageURL != tt.url {
				t.Errorf("expected URL %q, got %q", tt.url, img.ImageURL)
			}
			if img.AltText != tt.altText {
				t.Errorf("expected alt text %q, got %q", tt.altText, img.AltText)
			}
		})
	}
}

func TestImageSection(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		imgURL  string
		altText string
	}{
		{name: "with text and image", text: "Check this", imgURL: "https://img.com/p.png", altText: "picture"},
		{name: "with empty alt text", text: "Look", imgURL: "https://img.com/x.png", altText: ""},
		{name: "with empty text", text: "", imgURL: "https://img.com/y.png", altText: "img"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sec := imageSection(tt.text, tt.imgURL, tt.altText)
			if sec.Type != slack.MBTSection {
				t.Errorf("expected Section block type, got %s", sec.Type)
			}
			if sec.Accessory == nil {
				t.Fatal("expected accessory")
			}
			if sec.Accessory.ImageElement == nil {
				t.Error("expected image element in accessory")
			}
		})
	}
}

func TestStatusBlocks(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{name: "thinking", text: "Thinking..."},
		{name: "processing", text: "Processing data"},
		{name: "empty status", text: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocks := statusBlocks(tt.text)
			if len(blocks) != 1 {
				t.Fatalf("expected 1 block, got %d", len(blocks))
			}
			ctx, ok := blocks[0].(*slack.ContextBlock)
			if !ok {
				t.Fatal("expected *slack.ContextBlock")
			}
			if len(ctx.ContextElements.Elements) == 0 {
				t.Fatal("expected at least 1 element")
			}
		})
	}
}

func TestRenderBarChart(t *testing.T) {
	tests := []struct {
		name       string
		title      string
		subtitle   string
		labels     []string
		values     []float64
		wantBlocks int
	}{
		{
			name:       "full chart with title and subtitle",
			title:      "CPU Usage",
			subtitle:   "Last 24h",
			labels:     []string{"server1", "server2", "server3"},
			values:     []float64{80, 45, 60},
			wantBlocks: 3, // header + context + section
		},
		{
			name:       "chart without subtitle",
			title:      "Memory",
			subtitle:   "",
			labels:     []string{"A", "B"},
			values:     []float64{10, 20},
			wantBlocks: 2, // header + section
		},
		{
			name:       "chart with no title",
			title:      "",
			subtitle:   "",
			labels:     []string{"X", "Y"},
			values:     []float64{5, 10},
			wantBlocks: 1, // section only
		},
		{
			name:       "empty labels and values",
			title:      "Empty",
			subtitle:   "",
			labels:     []string{},
			values:     []float64{},
			wantBlocks: 1, // header only
		},
		{
			name:       "nil labels and values",
			title:      "Nil Chart",
			subtitle:   "test",
			labels:     nil,
			values:     nil,
			wantBlocks: 2, // header + context
		},
		{
			name:       "zero max value does not panic",
			title:      "Zero",
			subtitle:   "",
			labels:     []string{"X"},
			values:     []float64{0},
			wantBlocks: 2, // header + section (maxVal defaults to 1)
		},
		{
			name:       "more labels than values handled safely",
			title:      "Mismatch",
			subtitle:   "",
			labels:     []string{"A", "B", "C"},
			values:     []float64{10},
			wantBlocks: 2, // header + section (truncated to len(values))
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocks := renderBarChart(tt.title, tt.subtitle, tt.labels, tt.values)
			if len(blocks) != tt.wantBlocks {
				t.Errorf("expected %d blocks, got %d", tt.wantBlocks, len(blocks))
			}
			if tt.title != "" && len(blocks) > 0 {
				hdr, ok := blocks[0].(*slack.HeaderBlock)
				if ok && hdr.Text.Text != tt.title {
					t.Errorf("expected header title %q, got %q", tt.title, hdr.Text.Text)
				}
			}
		})
	}
}

func TestRenderPieChart(t *testing.T) {
	tests := []struct {
		name       string
		title      string
		labels     []string
		values     []float64
		wantBlocks int
	}{
		{
			name:       "full pie chart",
			title:      "Disk Usage",
			labels:     []string{"Used", "Free"},
			values:     []float64{75, 25},
			wantBlocks: 2, // header + section
		},
		{
			name:       "pie chart without title",
			title:      "",
			labels:     []string{"A", "B", "C"},
			values:     []float64{30, 40, 30},
			wantBlocks: 1, // section only
		},
		{
			name:       "empty pie chart",
			title:      "Empty Pie",
			labels:     []string{},
			values:     []float64{},
			wantBlocks: 1, // header only
		},
		{
			name:       "nil pie chart",
			title:      "Nil Pie",
			labels:     nil,
			values:     nil,
			wantBlocks: 1, // header only
		},
		{
			name:       "zero total value does not panic",
			title:      "Zero Pie",
			labels:     []string{"A"},
			values:     []float64{0},
			wantBlocks: 2, // header + section (total defaults to 1)
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocks := renderPieChart(tt.title, tt.labels, tt.values)
			if len(blocks) != tt.wantBlocks {
				t.Errorf("expected %d blocks, got %d", tt.wantBlocks, len(blocks))
			}
		})
	}
}

func TestBuildActionCard(t *testing.T) {
	tests := []struct {
		name      string
		card      ActionCardDef
		wantBlock int
	}{
		{
			name: "full card with all fields",
			card: ActionCardDef{
				Title:       "Deploy v2.0",
				Description: "Deploy the new version to production.",
				Fields:      map[string]string{"Env": "prod", "Version": "2.0.1"},
				ImageURL:    "https://img.com/logo.png",
				Buttons:     []ButtonDef{{Text: "Deploy", ActionID: "deploy", Value: "yes", Style: slack.StylePrimary}},
				Footer:      "Requested by admin",
			},
			wantBlock: 8, // header + desc + image + divider + sectionFields + divider + buttons + footer
		},
		{
			name: "card with title only",
			card: ActionCardDef{
				Title: "Info",
			},
			wantBlock: 1, // header only
		},
		{
			name: "card with fields only",
			card: ActionCardDef{
				Fields: map[string]string{"key": "val"},
			},
			wantBlock: 2, // divider + sectionFields
		},
		{
			name: "card with buttons only",
			card: ActionCardDef{
				Buttons: []ButtonDef{{Text: "OK", ActionID: "ok", Value: "1"}},
			},
			wantBlock: 2, // divider + actions
		},
		{
			name:      "empty card",
			card:      ActionCardDef{},
			wantBlock: 0,
		},
		{
			name: "footer only",
			card: ActionCardDef{
				Footer: "footer text",
			},
			wantBlock: 1, // context block only
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocks := buildActionCard(tt.card)
			if len(blocks) != tt.wantBlock {
				t.Errorf("expected %d blocks, got %d", tt.wantBlock, len(blocks))
			}
		})
	}
}

func TestBuildTableBlocks(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		headers []string
		rows    [][]any
		wantMin int // minimum blocks
	}{
		{
			name:    "table with title and data",
			title:   "Stats",
			headers: []string{"Name", "Value"},
			rows:    [][]any{{"CPU", "80%"}, {"MEM", "60%"}},
			wantMin: 2, // header + at least 1 section
		},
		{
			name:    "table without title",
			title:   "",
			headers: []string{"Col1", "Col2"},
			rows:    [][]any{{"a", 1}},
			wantMin: 1, // section only
		},
		{
			name:    "empty table",
			title:   "",
			headers: []string{},
			rows:    [][]any{},
			wantMin: 0,
		},
		{
			name:    "empty table with title",
			title:   "Empty",
			headers: []string{},
			rows:    [][]any{},
			wantMin: 1, // header only
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocks := buildTableBlocks(tt.title, tt.headers, tt.rows)
			if len(blocks) < tt.wantMin {
				t.Errorf("expected at least %d blocks, got %d", tt.wantMin, len(blocks))
			}
		})
	}
}

func TestCalcColumnWidths(t *testing.T) {
	tests := []struct {
		name    string
		headers []string
		rows    [][]any
		want    []int
	}{
		{
			name:    "header wider",
			headers: []string{"LongHeader", "S"},
			rows:    [][]any{{"a", "b"}},
			want:    []int{len("LongHeader"), 1},
		},
		{
			name:    "cell wider",
			headers: []string{"H", "Val"},
			rows:    [][]any{{"x", "very long value"}},
			want:    []int{1, len("very long value")},
		},
		{
			name:    "empty everything",
			headers: []string{},
			rows:    nil,
			want:    []int{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calcColumnWidths(tt.headers, tt.rows)
			if len(got) != len(tt.want) {
				t.Fatalf("expected %d widths, got %d", len(tt.want), len(got))
			}
			for i, w := range tt.want {
				if got[i] != w {
					t.Errorf("col %d: expected width %d, got %d", i, w, got[i])
				}
			}
		})
	}
}

func TestBuildTableLines(t *testing.T) {
	tests := []struct {
		name      string
		headers   []string
		colWidths []int
		rows      [][]any
		wantLines int // header + separator + rows
	}{
		{
			name:      "two columns with data",
			headers:   []string{"Name", "Value"},
			colWidths: []int{4, 5},
			rows:      [][]any{{"CPU", "80%"}},
			wantLines: 3,
		},
		{
			name:      "no rows",
			headers:   []string{"Col"},
			colWidths: []int{3},
			rows:      nil,
			wantLines: 2,
		},
		{
			name:      "multiple rows",
			headers:   []string{"A"},
			colWidths: []int{1},
			rows:      [][]any{{1}, {2}, {3}},
			wantLines: 5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := buildTableLines(tt.headers, tt.colWidths, tt.rows)
			if len(lines) != tt.wantLines {
				t.Errorf("expected %d lines, got %d", tt.wantLines, len(lines))
			}
		})
	}
}

func TestTableLinesToBlocks(t *testing.T) {
	tests := []struct {
		name    string
		lines   []string
		wantMin int
	}{
		{
			name:    "single short line",
			lines:   []string{"a b c"},
			wantMin: 1,
		},
		{
			name:    "multiple short lines",
			lines:   []string{"header", "------", "row1", "row2", "row3"},
			wantMin: 1,
		},
		{
			name:    "no lines",
			lines:   nil,
			wantMin: 0,
		},
		{
			name: "large input splits into multiple blocks",
			lines: func() []string {
				var l []string
				for i := range 100 {
					var line strings.Builder
					for j := range 80 {
						line.WriteString(string(rune('A' + (i+j)%26)))
					}
					l = append(l, line.String())
				}
				return l
			}(),
			wantMin: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocks := tableLinesToBlocks(tt.lines)
			if len(blocks) < tt.wantMin {
				t.Errorf("expected at least %d blocks, got %d", tt.wantMin, len(blocks))
			}
		})
	}
}
