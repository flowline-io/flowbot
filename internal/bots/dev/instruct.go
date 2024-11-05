package dev

import "github.com/flowline-io/flowbot/internal/types/ruleset/instruct"

const (
	ExampleInstructID = "dev_example"
)

var instructRules = []instruct.Rule{
	{
		Id:   ExampleInstructID,
		Args: []string{"txt"},
	},
}
