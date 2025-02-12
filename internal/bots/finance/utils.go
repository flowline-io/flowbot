package finance

import (
	"fmt"

	"github.com/tmc/langchaingo/prompts"
)

func classify() {
	prompt := prompts.NewPromptTemplate(
		`Given i want to categorize transactions on my bank account into this categories: {{.categories}}
In which category would a transaction from "{{.destination_name}}" with the subject "{{.description}}" fall into?
Just output the name of the category. Does not have to be a complete sentence.`,
		[]string{"categories", "destination_name", "description"},
	)
	_, _ = fmt.Println(prompt)
}
