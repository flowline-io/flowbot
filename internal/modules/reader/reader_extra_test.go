package reader

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCronRulesAllHaveWhen(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "all cron rules have when expression"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, r := range cronRules {
				assert.NotEmpty(t, r.When, "cron rule %q should have a When expression", r.Name)
			}
		})
	}
}
