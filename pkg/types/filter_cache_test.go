package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilterCacheSetSource(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		sources   []string
		wantCount int
	}{
		{
			name:      "add single source",
			sources:   []string{"github"},
			wantCount: 1,
		},
		{
			name:      "add duplicate source ignored",
			sources:   []string{"github", "github"},
			wantCount: 1,
		},
		{
			name:      "add multiple distinct sources",
			sources:   []string{"github", "gitea", "reader"},
			wantCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fc := NewFilterCache()
			for _, s := range tt.sources {
				fc.SetSource(s)
			}
			assert.Len(t, fc.Sources(), tt.wantCount)
		})
	}
}

func TestFilterCacheHydrate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		initialSrcs   []string
		hydrateSrcs   []string
		wantSrcCount  int
	}{
		{
			name:         "hydrate into empty cache",
			initialSrcs:  nil,
			hydrateSrcs:  []string{"github", "gitea"},
			wantSrcCount: 2,
		},
		{
			name:         "hydrate with overlap deduplicates",
			initialSrcs:  []string{"github"},
			hydrateSrcs:  []string{"github", "reader"},
			wantSrcCount: 2,
		},
		{
			name:         "hydrate empty list preserves existing",
			initialSrcs:  []string{"github"},
			hydrateSrcs:  nil,
			wantSrcCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fc := NewFilterCache()
			for _, s := range tt.initialSrcs {
				fc.SetSource(s)
			}
			fc.Hydrate(tt.hydrateSrcs, nil)
			assert.Len(t, fc.Sources(), tt.wantSrcCount)
		})
	}
}

func TestFilterCacheEmptySource(t *testing.T) {
	t.Parallel()
	fc := NewFilterCache()
	fc.SetSource("")
	assert.Empty(t, fc.Sources())
}
