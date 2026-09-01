package partials

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/pkg/types/model"
)

func TestAgentSubagentTaskRow_expandDoesNotScroll(t *testing.T) {
	t.Parallel()
	item := model.AgentSubagentTask{
		ID:           3,
		SubagentName: "researcher",
		StartedAt:    time.Date(2026, 8, 31, 8, 6, 7, 0, time.UTC),
	}
	var buf bytes.Buffer
	if err := AgentSubagentTaskRow(context.Background(), item).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	assertExpandRowStaysInPlace(t, buf.String(), `hx-swap="innerHTML show:none"`)
}

func TestAgentSubagentTaskDetail_constrainsWidePrompt(t *testing.T) {
	t.Parallel()
	item := model.AgentSubagentTask{
		ID:           3,
		SubagentName: "researcher",
		SessionID:    "sess-1",
		Prompt:       strings.Repeat("https://example.com/token ", 40),
		StartedAt:    time.Date(2026, 8, 31, 8, 6, 7, 0, time.UTC),
	}
	var buf bytes.Buffer
	if err := AgentSubagentTaskDetail(context.Background(), item).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(buf.String(), `flowbot-table-expand-cell`) {
		t.Fatalf("want constrained expand cell in html\n%s", buf.String())
	}
}
