package cloudflare

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormRules_Initialized(t *testing.T) {
	// formRules is populated at Bootstrap time, so it starts as nil/empty
	assert.Empty(t, formRules)
}
