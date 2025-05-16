package finance

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
	"github.com/flowline-io/flowbot/internal/agents"
)

func classify(ctx context.Context) {
	template := prompt.FromMessages(schema.GoTemplate,
		schema.UserMessage(`Given i want to categorize transactions on my bank account into this categories: {{.categories}}
In which category would a transaction from "{{.destination_name}}" with the subject "{{.description}}" fall into?
Just output the name of the category. Does not have to be a complete sentence.`),
	)
	_, _ = fmt.Println(template)
}

const billPrompt = `You are a bill parsing assistant. Please help me extract each transaction record from the following text. Requirements:
1. Extract information such as date, amount, merchant name, and transaction description.
2. Return in JSON format, containing an array of records, where each record includes fields: date (in YYYY-MM-DD HH:mm:ss format), amount (number), merchant, and description.
3. Fill null for any fields that cannot be parsed.
4. Use positive numbers for amounts, representing both expenses and income as positive numbers.

The bill text is as follows:
---
`

func billParser(ctx context.Context, billText string) (string, error) {
	template := billPrompt + billText

	llm, err := agents.ChatModel(ctx, agents.AgentModelName(agents.AgentBillClassify))
	if err != nil {
		return "", fmt.Errorf("chat model failed, %w", err)
	}

	messages, err := agents.BaseTemplate().Format(ctx, map[string]any{
		"content": template,
	})
	if err != nil {
		return "", fmt.Errorf("format prompt failed, %w", err)
	}

	resp, err := agents.Generate(ctx, llm, messages)
	if err != nil {
		return "", fmt.Errorf("generate response failed, %w", err)
	}

	if resp == nil || resp.Content == "" {
		return "", fmt.Errorf("empty response")
	}

	return resp.Content, nil
}
