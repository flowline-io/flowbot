package clipboard

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInstructRules_Count(t *testing.T) {
	assert.Len(t, instructRules, 1)
}

func TestInstructRules_IDs(t *testing.T) {
	assert.Equal(t, ShareInstruct, instructRules[0].Id)
}

func TestShareInstruct_Constant(t *testing.T) {
	assert.Equal(t, "clipboard_share", ShareInstruct)
}

func TestInstructRules_ArgsContent(t *testing.T) {
	assert.Equal(t, []string{"txt"}, instructRules[0].Args)
}
