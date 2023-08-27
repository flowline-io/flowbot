package dev

import "github.com/flowline-io/flowbot/internal/ruleset/instruct"

const (
	ExampleInstructID = "dev_example"
)

var instructRules = []instruct.Rule{
	{
		Id:   ExampleInstructID,
		Args: []string{"txt"},
	},
}
