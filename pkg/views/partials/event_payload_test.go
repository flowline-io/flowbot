package partials

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
)

func TestEventPayloadDetail_highlightAndCopy(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		payload string
		want    []string
		absent  []string
	}{
		{
			name:    "json object has highlight and copy",
			payload: `{"url":"https://example.com","n":1}`,
			want: []string{
				`data-testid="event-payload-copy"`,
				`data-clip-copy`,
				`flowbot-json-key`,
				`flowbot-json-string`,
				`flowbot-json-number`,
				`data-testid="event-payload-detail"`,
			},
			absent: []string{`<td`},
		},
		{
			name:    "escapes script in payload",
			payload: `{"x":"<script>"}`,
			want:    []string{`&lt;script&gt;`},
			absent:  []string{`<script>`},
		},
		{
			name:    "empty payload renders object",
			payload: "",
			want:    []string{`data-testid="event-payload-json"`, `{}`},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := EventPayloadDetail(tt.payload).Render(context.Background(), &buf); err != nil {
				t.Fatalf("render: %v", err)
			}
			html := buf.String()
			for _, sub := range tt.want {
				if !strings.Contains(html, sub) {
					t.Fatalf("want %q in html\n%s", sub, html)
				}
			}
			for _, sub := range tt.absent {
				if strings.Contains(html, sub) {
					t.Fatalf("did not want %q in html\n%s", sub, html)
				}
			}
		})
	}
}

func TestDataEventsTable_expandTarget(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		events []*gen.DataEvent
		want   []string
		absent []string
	}{
		{
			name:   "empty table",
			events: nil,
			want:   []string{"Actions", "No events yet", `data-testid="data-events-table"`},
		},
		{
			name: "row targets detail td and is not hidden",
			events: []*gen.DataEvent{{
				EventID:   "evt-1",
				EventType: "bookmark.created",
				Source:    "karakeep",
				CreatedAt: time.Date(2026, 7, 21, 19, 9, 33, 0, time.UTC),
			}},
			want: []string{
				`hx-target="#detail-evt-1 td"`,
				`id="detail-evt-1"`,
				`class="event-detail-row"`,
				`data-testid="event-similar-link"`,
			},
			absent: []string{`class="hidden"`, `class="event-detail-row hidden"`},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := DataEventsTable(nil, nil, tt.events, PageInfo{}, nil).Render(context.Background(), &buf)
			if err != nil {
				t.Fatalf("render: %v", err)
			}
			html := buf.String()
			for _, sub := range tt.want {
				if !strings.Contains(html, sub) {
					t.Fatalf("want %q in html\n%s", sub, html)
				}
			}
			for _, sub := range tt.absent {
				if strings.Contains(html, sub) {
					t.Fatalf("did not want %q in html\n%s", sub, html)
				}
			}
		})
	}
}
