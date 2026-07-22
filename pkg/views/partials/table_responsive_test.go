package partials_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/pkg/types/model"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

func TestTablePinCols(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		render func() (string, error)
	}{
		{
			name: "token table pins columns",
			render: func() (string, error) {
				var buf bytes.Buffer
				err := partials.TokenTable(nil).Render(context.Background(), &buf)
				return buf.String(), err
			},
		},
		{
			name: "notify channels table pins columns",
			render: func() (string, error) {
				var buf bytes.Buffer
				err := partials.NotifyChannelsTable(nil, "").Render(context.Background(), &buf)
				return buf.String(), err
			},
		},
		{
			name: "agent subagent table pins columns",
			render: func() (string, error) {
				var buf bytes.Buffer
				err := partials.AgentSubagentTable(nil).Render(context.Background(), &buf)
				return buf.String(), err
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			html, err := tt.render()
			if err != nil {
				t.Fatalf("render: %v", err)
			}
			if !strings.Contains(html, "flowbot-table-pin") {
				t.Fatalf("want flowbot-table-pin in %q", html)
			}
			if strings.Contains(html, "table-pin-cols") {
				t.Fatalf("did not want DaisyUI table-pin-cols in %q", html)
			}
		})
	}
}

func TestTableCardStackMarkup(t *testing.T) {
	t.Parallel()
	channel := model.NotifyChannel{
		Name:     "email",
		Protocol: "smtp",
		URI:      "smtp://localhost",
		Enabled:  true,
	}
	subagent := model.AgentSubagent{
		Flag:        "researcher",
		Name:        "Researcher",
		Description: "Research specialist",
		Model:       "default",
		Enabled:     true,
	}
	tests := []struct {
		name string
		want []string
		html func() (string, error)
	}{
		{
			name: "notify channels uses card stack class and data labels",
			want: []string{
				"flowbot-table-cards",
				`data-label="Name"`,
				`data-label="Protocol"`,
				`data-label="URI"`,
				`data-label="Status"`,
				`data-label="Default"`,
				`data-label="Actions"`,
			},
			html: func() (string, error) {
				var buf bytes.Buffer
				err := partials.NotifyChannelsTable([]model.NotifyChannel{channel}, "").Render(context.Background(), &buf)
				return buf.String(), err
			},
		},
		{
			name: "agent subagents uses card stack class and data labels",
			want: []string{
				"flowbot-table-cards",
				`data-label="Flag"`,
				`data-label="Name"`,
				`data-label="Description"`,
				`data-label="Model"`,
				`data-label="Enabled"`,
				`data-label="Updated"`,
				`data-label="Actions"`,
			},
			html: func() (string, error) {
				var buf bytes.Buffer
				err := partials.AgentSubagentTable([]model.AgentSubagent{subagent}).Render(context.Background(), &buf)
				return buf.String(), err
			},
		},
		{
			name: "token table does not use card stack",
			want: []string{},
			html: func() (string, error) {
				var buf bytes.Buffer
				err := partials.TokenTable(nil).Render(context.Background(), &buf)
				return buf.String(), err
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			html, err := tt.html()
			if err != nil {
				t.Fatalf("render: %v", err)
			}
			if len(tt.want) == 0 {
				if strings.Contains(html, "flowbot-table-cards") {
					t.Fatalf("did not want flowbot-table-cards in %q", html)
				}
				return
			}
			for _, want := range tt.want {
				if !strings.Contains(html, want) {
					t.Fatalf("want %q in %q", want, html)
				}
			}
		})
	}
}
