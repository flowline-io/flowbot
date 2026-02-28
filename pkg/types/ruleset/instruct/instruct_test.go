package instruct

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestRule_ID(t *testing.T) {
	r := Rule{Id: "test_instruct"}
	assert.Equal(t, "test_instruct", r.ID())
}

func TestRule_TYPE(t *testing.T) {
	r := Rule{Id: "test_instruct"}
	assert.Equal(t, types.InstructRule, r.TYPE())
}

func TestRule_EmptyID(t *testing.T) {
	r := Rule{}
	assert.Equal(t, "", r.ID())
}

func TestRuleset_Creation(t *testing.T) {
	rules := Ruleset{
		{Id: "instruct1", Args: []string{"arg1", "arg2"}},
		{Id: "instruct2", Args: []string{"arg3"}},
	}
	assert.Len(t, rules, 2)
}

func TestRuleset_RuleArgs(t *testing.T) {
	r := Rule{
		Id:   "clipboard_share",
		Args: []string{"txt"},
	}
	assert.Equal(t, "clipboard_share", r.ID())
	assert.Equal(t, []string{"txt"}, r.Args)
}

func TestRuleset_Empty(t *testing.T) {
	rules := Ruleset{}
	assert.Len(t, rules, 0)
}

func TestRuleset_MultipleArgs(t *testing.T) {
	r := Rule{
		Id:   "multi_arg",
		Args: []string{"cpu", "memory", "info"},
	}
	assert.Equal(t, types.InstructRule, r.TYPE())
	assert.Len(t, r.Args, 3)
	assert.Contains(t, r.Args, "cpu")
	assert.Contains(t, r.Args, "memory")
	assert.Contains(t, r.Args, "info")
}
