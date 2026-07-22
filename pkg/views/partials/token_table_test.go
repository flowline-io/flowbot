package partials_test

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

func TestTokenTableClientFilter(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 7, 22, 12, 0, 0, 0, time.UTC)
	item := model.TokenItem{
		Token:     "abcdefghijklmnopqrstuvwxyz",
		UID:       types.Uid("user:demo"),
		Scopes:    []string{"admin:*", "pipeline:read"},
		CreatedAt: now,
		ExpiredAt: now.Add(24 * time.Hour),
	}
	tests := []struct {
		name string
		want []string
	}{
		{
			name: "wraps table in tableFilter alpine data",
			want: []string{`x-data="tableFilter"`},
		},
		{
			name: "renders filter bar with token test id",
			want: []string{`data-testid="token-table-filter"`, `placeholder="Search tokens..."`},
		},
		{
			name: "rows expose filter text for client matching",
			want: []string{`data-filter-text=`, `x-show="rowMatches($el)"`},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			if err := partials.TokenTable([]model.TokenItem{item}).Render(context.Background(), &buf); err != nil {
				t.Fatalf("render: %v", err)
			}
			html := buf.String()
			for _, want := range tt.want {
				if !strings.Contains(html, want) {
					t.Fatalf("want %q in %q", want, html)
				}
			}
		})
	}
}

func TestTokenFilterText(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 7, 22, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		item model.TokenItem
		want string
	}{
		{
			name: "includes uid token prefix and scopes",
			item: model.TokenItem{
				Token:     "abcdefghijklmnop",
				UID:       types.Uid("user:a"),
				Scopes:    []string{"admin:*", "pipeline:read"},
				CreatedAt: now,
				ExpiredAt: now,
			},
			want: "user:a abcdefghijkl admin:* pipeline:read",
		},
		{
			name: "empty scopes still includes uid and prefix",
			item: model.TokenItem{
				Token:     "zzzzzzzzzzzzzzzz",
				UID:       types.Uid("user:b"),
				CreatedAt: now,
				ExpiredAt: now,
			},
			want: "user:b zzzzzzzzzzzz",
		},
		{
			name: "short token uses full value as prefix source",
			item: model.TokenItem{
				Token:     "short",
				UID:       types.Uid("uid"),
				CreatedAt: now,
				ExpiredAt: now,
			},
			want: "uid short",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := partials.TokenFilterText(tt.item)
			if got != tt.want {
				t.Fatalf("TokenFilterText() = %q, want %q", got, tt.want)
			}
		})
	}
}
