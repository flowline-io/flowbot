package reader

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCronRulesAllHaveWhen(t *testing.T) {
	for _, r := range cronRules {
		assert.NotEmpty(t, r.When, "cron rule %q should have a When expression", r.Name)
	}
}
