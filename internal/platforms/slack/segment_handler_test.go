package slack

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/types/protocol"
)

func TestHandleSegChart(t *testing.T) {
	tests := []struct {
		name       string
		segment    protocol.MessageSegment
		wantBlocks int
		wantText   string
		wantFiles  int
	}{
		{
			name: "bar chart with full data",
			segment: protocol.MessageSegment{
				Type: "chart",
				Data: map[string]any{
					"chart_type": "bar",
					"title":      "CPU",
					"subtitle":   "24h",
					"labels":     []string{"srv1", "srv2"},
					"values":     []float64{80, 45},
				},
			},
			wantBlocks: 3, // header + context + section
			wantText:   "",
			wantFiles:  0,
		},
		{
			name: "pie chart",
			segment: protocol.MessageSegment{
				Type: "chart",
				Data: map[string]any{
					"chart_type": "pie",
					"title":      "Disk",
					"labels":     []string{"Used", "Free"},
					"values":     []float64{75, 25},
				},
			},
			wantBlocks: 2, // header + section
			wantText:   "",
			wantFiles:  0,
		},
		{
			name: "unknown chart type defaults to bar",
			segment: protocol.MessageSegment{
				Type: "chart",
				Data: map[string]any{
					"chart_type": "line",
					"title":      "Trend",
					"labels":     []string{"A"},
					"values":     []float64{10},
				},
			},
			wantBlocks: 2, // header + section
			wantText:   "",
			wantFiles:  0,
		},
		{
			name: "chart with no labels or values",
			segment: protocol.MessageSegment{
				Type: "chart",
				Data: map[string]any{
					"chart_type": "bar",
					"title":      "Empty",
				},
			},
			wantBlocks: 1, // header only
			wantText:   "",
			wantFiles:  0,
		},
		{
			name: "chart with empty data map",
			segment: protocol.MessageSegment{
				Type: "chart",
				Data: map[string]any{},
			},
			wantBlocks: 0,
			wantText:   "",
			wantFiles:  0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, blocks, files := handleSegChart(tt.segment)
			if len(blocks) != tt.wantBlocks {
				t.Errorf("expected %d blocks, got %d", tt.wantBlocks, len(blocks))
			}
			if len(files) != tt.wantFiles {
				t.Errorf("expected %d files, got %d", tt.wantFiles, len(files))
			}
		})
	}
}

func TestHandleSegTable(t *testing.T) {
	tests := []struct {
		name       string
		segment    protocol.MessageSegment
		wantBlocks int
	}{
		{
			name: "table with title and data",
			segment: protocol.MessageSegment{
				Type: "table",
				Data: map[string]any{
					"title":   "Stats",
					"headers": []string{"Name", "Value"},
					"rows":    [][]any{{"CPU", "80%"}, {"Mem", "60%"}},
				},
			},
			wantBlocks: 2, // header + section
		},
		{
			name: "table without title",
			segment: protocol.MessageSegment{
				Type: "table",
				Data: map[string]any{
					"headers": []string{"Col"},
					"rows":    [][]any{{"val"}},
				},
			},
			wantBlocks: 1, // section only
		},
		{
			name: "table with empty data",
			segment: protocol.MessageSegment{
				Type: "table",
				Data: map[string]any{
					"title": "Empty",
				},
			},
			wantBlocks: 1, // header only
		},
		{
			name: "table with no data at all",
			segment: protocol.MessageSegment{
				Type: "table",
				Data: map[string]any{},
			},
			wantBlocks: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, blocks, _ := handleSegTable(tt.segment)
			if len(blocks) != tt.wantBlocks {
				t.Errorf("expected %d blocks, got %d", tt.wantBlocks, len(blocks))
			}
		})
	}
}

func TestHandleSegForm(t *testing.T) {
	tests := []struct {
		name       string
		segment    protocol.MessageSegment
		wantBlocks int
	}{
		{
			name: "form with title and fields",
			segment: protocol.MessageSegment{
				Type: "form",
				Data: map[string]any{
					"title": "Settings",
					"fields": []any{
						map[string]any{"label": "Name", "key": "name", "type": "text", "placeholder": "Enter name"},
						map[string]any{"label": "Email", "key": "email", "type": "email"},
					},
				},
			},
			wantBlocks: 4, // header + divider + 2 sections
		},
		{
			name: "form with no title",
			segment: protocol.MessageSegment{
				Type: "form",
				Data: map[string]any{
					"fields": []any{
						map[string]any{"label": "Option", "key": "opt"},
					},
				},
			},
			wantBlocks: 2, // divider + 1 section
		},
		{
			name: "form with initial value",
			segment: protocol.MessageSegment{
				Type: "form",
				Data: map[string]any{
					"title":  "Config",
					"fields": []any{map[string]any{"label": "Key", "key": "k", "initial_value": "default"}},
				},
			},
			wantBlocks: 3, // header + divider + section
		},
		{
			name: "form with no fields",
			segment: protocol.MessageSegment{
				Type: "form",
				Data: map[string]any{
					"title": "Title Only",
				},
			},
			wantBlocks: 2, // header + divider
		},
		{
			name: "form with nil fields",
			segment: protocol.MessageSegment{
				Type: "form",
				Data: map[string]any{},
			},
			wantBlocks: 1, // divider only
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, blocks, _ := handleSegForm(tt.segment)
			if len(blocks) != tt.wantBlocks {
				t.Errorf("expected %d blocks, got %d", tt.wantBlocks, len(blocks))
			}
		})
	}
}

func TestHandleSegActionCard(t *testing.T) {
	tests := []struct {
		name       string
		segment    protocol.MessageSegment
		wantBlocks int
	}{
		{
			name: "full action card",
			segment: protocol.MessageSegment{
				Type: "action_card",
				Data: map[string]any{
					"title":       "Deploy",
					"description": "Deploy to prod",
					"image_url":   "https://img.com/logo.png",
					"footer":      "by admin",
					"fields":      map[string]string{"env": "prod"},
					"buttons":     []any{map[string]any{"text": "Go", "action_id": "deploy", "value": "yes"}},
				},
			},
			wantBlocks: 8, // header + desc + image + divider + fields + divider + buttons + footer
		},
		{
			name: "action card with title only",
			segment: protocol.MessageSegment{
				Type: "action_card",
				Data: map[string]any{
					"title": "Info",
				},
			},
			wantBlocks: 1, // header only
		},
		{
			name: "action card with no fields or buttons",
			segment: protocol.MessageSegment{
				Type: "action_card",
				Data: map[string]any{
					"title":       "Notice",
					"description": "Something happened",
				},
			},
			wantBlocks: 2, // header + description
		},
		{
			name: "empty action card",
			segment: protocol.MessageSegment{
				Type: "action_card",
				Data: map[string]any{},
			},
			wantBlocks: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, blocks, _ := handleSegActionCard(tt.segment)
			if len(blocks) != tt.wantBlocks {
				t.Errorf("expected %d blocks, got %d", tt.wantBlocks, len(blocks))
			}
		})
	}
}

func TestHandleSegStatus(t *testing.T) {
	tests := []struct {
		name       string
		segment    protocol.MessageSegment
		wantBlocks int
	}{
		{
			name: "status with text",
			segment: protocol.MessageSegment{
				Type: "status",
				Data: map[string]any{"text": "Processing..."},
			},
			wantBlocks: 1,
		},
		{
			name: "status with empty text defaults to Processing",
			segment: protocol.MessageSegment{
				Type: "status",
				Data: map[string]any{"text": ""},
			},
			wantBlocks: 1,
		},
		{
			name: "status with missing text key",
			segment: protocol.MessageSegment{
				Type: "status",
				Data: map[string]any{},
			},
			wantBlocks: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, blocks, _ := handleSegStatus(tt.segment)
			if len(blocks) != tt.wantBlocks {
				t.Errorf("expected %d blocks, got %d", tt.wantBlocks, len(blocks))
			}
		})
	}
}

func TestHandleSegLink(t *testing.T) {
	tests := []struct {
		name       string
		segment    protocol.MessageSegment
		wantBlocks int
	}{
		{
			name: "link with title and cover",
			segment: protocol.MessageSegment{
				Type: "link",
				Data: map[string]any{
					"title": "GitHub",
					"url":   "https://github.com",
					"cover": "https://img.com/gh.png",
				},
			},
			wantBlocks: 1, // imageSection
		},
		{
			name: "link with title and url no cover",
			segment: protocol.MessageSegment{
				Type: "link",
				Data: map[string]any{
					"title": "GitHub",
					"url":   "https://github.com",
				},
			},
			wantBlocks: 1, // sectionWithButton
		},
		{
			name: "link without title uses url as title",
			segment: protocol.MessageSegment{
				Type: "link",
				Data: map[string]any{
					"url": "https://example.com",
				},
			},
			wantBlocks: 1,
		},
		{
			name: "link without url returns no blocks",
			segment: protocol.MessageSegment{
				Type: "link",
				Data: map[string]any{
					"title": "No URL",
				},
			},
			wantBlocks: 0,
		},
		{
			name: "link with empty data",
			segment: protocol.MessageSegment{
				Type: "link",
				Data: map[string]any{},
			},
			wantBlocks: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, blocks, _ := handleSegLink(tt.segment)
			if len(blocks) != tt.wantBlocks {
				t.Errorf("expected %d blocks, got %d", tt.wantBlocks, len(blocks))
			}
		})
	}
}

func TestHandleSegMarkdownVarious(t *testing.T) {
	tests := []struct {
		name       string
		segment    protocol.MessageSegment
		wantBlocks int
	}{
		{
			name: "markdown with title and text",
			segment: protocol.MessageSegment{
				Type: "markdown",
				Data: map[string]any{"title": "Title", "text": "content"},
			},
			wantBlocks: 2,
		},
		{
			name: "markdown text only",
			segment: protocol.MessageSegment{
				Type: "markdown",
				Data: map[string]any{"text": "content only"},
			},
			wantBlocks: 1,
		},
		{
			name: "markdown title only",
			segment: protocol.MessageSegment{
				Type: "markdown",
				Data: map[string]any{"title": "Title Only"},
			},
			wantBlocks: 1,
		},
		{
			name: "markdown empty",
			segment: protocol.MessageSegment{
				Type: "markdown",
				Data: map[string]any{},
			},
			wantBlocks: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, blocks, _ := handleSegMarkdown(tt.segment)
			if len(blocks) != tt.wantBlocks {
				t.Errorf("expected %d blocks, got %d", tt.wantBlocks, len(blocks))
			}
		})
	}
}

func TestHandleSegKVVarious(t *testing.T) {
	tests := []struct {
		name       string
		segment    protocol.MessageSegment
		wantBlocks int
	}{
		{
			name: "kv with multiple fields",
			segment: protocol.MessageSegment{
				Type: "kv",
				Data: map[string]any{"fields": map[string]string{"k1": "v1", "k2": "v2"}},
			},
			wantBlocks: 1,
		},
		{
			name: "kv with single field",
			segment: protocol.MessageSegment{
				Type: "kv",
				Data: map[string]any{"fields": map[string]string{"key": "value"}},
			},
			wantBlocks: 1,
		},
		{
			name: "kv with empty fields",
			segment: protocol.MessageSegment{
				Type: "kv",
				Data: map[string]any{"fields": map[string]string{}},
			},
			wantBlocks: 0,
		},
		{
			name: "kv with missing fields key",
			segment: protocol.MessageSegment{
				Type: "kv",
				Data: map[string]any{},
			},
			wantBlocks: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, blocks, _ := handleSegKV(tt.segment)
			if len(blocks) != tt.wantBlocks {
				t.Errorf("expected %d blocks, got %d", tt.wantBlocks, len(blocks))
			}
		})
	}
}

func TestHandleSegFileTypes(t *testing.T) {
	tests := []struct {
		name      string
		segType   string
		wantFiles int
	}{
		{name: "file segment", segType: "file", wantFiles: 1},
		{name: "video segment", segType: "video", wantFiles: 1},
		{name: "audio segment", segType: "audio", wantFiles: 1},
		{name: "voice segment", segType: "voice", wantFiles: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, ok := segHandlers[tt.segType]
			if !ok {
				t.Fatalf("handler not found for %s", tt.segType)
			}
			_, _, files := handler(protocol.MessageSegment{Type: tt.segType, Data: map[string]any{"file_id": "/tmp/test"}})
			if len(files) != tt.wantFiles {
				t.Errorf("expected %d files, got %d", tt.wantFiles, len(files))
			}
		})
	}
}

func TestHandleSegMentionNoUserID(t *testing.T) {
	text, _, _ := handleSegMention(protocol.MessageSegment{Type: "mention", Data: map[string]any{}})
	if text != "" {
		t.Errorf("expected empty text for no user_id, got %q", text)
	}
}

func TestHandleSegURLEmpty(t *testing.T) {
	text, _, _ := handleSegURL(protocol.MessageSegment{Type: "url", Data: map[string]any{}})
	if text != "" {
		t.Errorf("expected empty text for no url, got %q", text)
	}
}
