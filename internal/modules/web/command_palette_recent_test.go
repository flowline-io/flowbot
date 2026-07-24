package web

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRecordCommandPaletteRecent(t *testing.T) {
	t.Parallel()
	itemA := commandPaletteItem{ID: "page:home", Title: "Home", Href: "/service/web/home", Group: "pages"}
	itemB := commandPaletteItem{ID: "pipeline:sync", Title: "sync", Href: "/service/web/pipelines/sync", Group: "pipelines"}
	itemC := commandPaletteItem{ID: "homelab:app", Title: "app", Href: "/service/web/homelab/app", Group: "homelab"}

	tests := []struct {
		name     string
		existing []commandPaletteItem
		next     commandPaletteItem
		wantIDs  []string
		wantLen  int
	}{
		{
			name:     "prepends onto empty list",
			existing: nil,
			next:     itemA,
			wantIDs:  []string{"page:home"},
		},
		{
			name:     "moves duplicate to front",
			existing: []commandPaletteItem{itemA, itemB},
			next:     itemB,
			wantIDs:  []string{"pipeline:sync", "page:home"},
		},
		{
			name:     "ignores empty href",
			existing: []commandPaletteItem{itemA},
			next:     commandPaletteItem{ID: "x", Title: "X", Href: "", Group: "pages"},
			wantIDs:  []string{"page:home"},
		},
		{
			name: "caps at eight entries",
			existing: func() []commandPaletteItem {
				out := make([]commandPaletteItem, 8)
				for i := range 8 {
					out[i] = commandPaletteItem{
						ID:    "page:" + itoaRecent(i),
						Title: "P",
						Href:  "/service/web/p/" + itoaRecent(i),
						Group: "pages",
					}
				}
				return out
			}(),
			next:    itemC,
			wantIDs: []string{"homelab:app", "page:0", "page:1", "page:2", "page:3", "page:4", "page:5", "page:6"},
			wantLen: 8,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := recordCommandPaletteRecent(tt.existing, tt.next)
			assert.Equal(t, tt.wantIDs, itemIDs(got))
			if tt.wantLen > 0 {
				assert.Len(t, got, tt.wantLen)
			}
		})
	}
}

func itoaRecent(n int) string {
	if n == 0 {
		return "0"
	}
	digits := []byte{}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
