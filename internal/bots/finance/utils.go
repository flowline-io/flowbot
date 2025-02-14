package finance

import (
	"context"
	"fmt"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
)

func classify(ctx context.Context) {
	template := prompt.FromMessages(schema.GoTemplate,
		schema.UserMessage(`Given i want to categorize transactions on my bank account into this categories: {{.categories}}
In which category would a transaction from "{{.destination_name}}" with the subject "{{.description}}" fall into?
Just output the name of the category. Does not have to be a complete sentence.`),
	)
	_, _ = fmt.Println(template)
}
