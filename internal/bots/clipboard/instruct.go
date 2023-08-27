package clipboard

import "github.com/sysatom/flowbot/internal/ruleset/instruct"

const (
	ShareInstruct = "clipboard_share"
)

var instructRules = []instruct.Rule{
	{
		Id:   ShareInstruct,
		Args: []string{"txt"},
	},
}
