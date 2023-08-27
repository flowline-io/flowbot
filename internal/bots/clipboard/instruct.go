package clipboard

import "github.com/flowline-io/flowbot/internal/ruleset/instruct"

const (
	ShareInstruct = "clipboard_share"
)

var instructRules = []instruct.Rule{
	{
		Id:   ShareInstruct,
		Args: []string{"txt"},
	},
}
