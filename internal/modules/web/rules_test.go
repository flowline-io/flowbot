package web

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllWebserviceRuleGroups(t *testing.T) {
	tests := []struct {
		name      string
		wantLen   int
		wantEmpty bool
	}{
		{name: "registers twenty-six route groups", wantLen: 26},
		{name: "every group has at least one route", wantEmpty: false},
		{name: "Rules matches allWebserviceRules length", wantLen: 26},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantLen > 0 {
				assert.Len(t, allWebserviceRules, tt.wantLen)
				var h moduleHandler
				assert.Len(t, h.Rules(), tt.wantLen)
			}
			if !tt.wantEmpty {
				for i, rules := range allWebserviceRules {
					require.NotEmpty(t, rules, "rule group index %d", i)
				}
			}
		})
	}
}
