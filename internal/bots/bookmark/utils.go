package bookmark

import (
	"context"
	"fmt"

	jsonrepair "github.com/RealAlexandreAI/json-repair"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
	"github.com/flowline-io/flowbot/internal/agents"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/providers/hoarder"
	"github.com/flowline-io/flowbot/pkg/utils"
	json "github.com/json-iterator/go"
)

var tagsPrompt = `
You are a bot in a read-it-later app and your responsibility is to help with automatic tagging.
Please analyze the text between the sentences "CONTENT START HERE" and "CONTENT END HERE" and suggest relevant tags that describe its key themes, topics, and main ideas. The rules are:
- Aim for a variety of tags, including broad categories, specific keywords, and potential sub-genres.
- The tags language must be in {{.language}}.
- If it's a famous website you may also include a tag for the website. If the tag is not generic enough, don't include it.
- The content can include text for cookie consent and privacy policy, ignore those while tagging.
- Aim for 3-5 tags.
- If there are no good tags, leave the array empty.

CONTENT START HERE
{{.content}}
<CONTENT_HERE>

CONTENT END HERE
You must respond in JSON with the key "tags" and the value is an array of string tags.
`

func extractTags(ctx context.Context, bookmark hoarder.Bookmark) ([]string, error) {
	var content string
	title := bookmark.Content.Title
	if title != nil {
		content = *title
	}
	if content == "" {
		return nil, nil
	}

	messages, err := prompt.FromMessages(schema.GoTemplate,
		schema.UserMessage(tagsPrompt),
	).Format(ctx, map[string]any{
		"content":  content,
		"language": config.App.Agent.Language,
	})
	if err != nil {
		return nil, fmt.Errorf("%s bot, prompt format failed, %w", Name, err)
	}

	llm, err := agents.ChatModel(ctx, agents.Model())
	if err != nil {
		return nil, fmt.Errorf("%s bot, chat model failed, %w", Name, err)
	}

	resp, err := agents.Generate(ctx, llm, messages)
	if err != nil {
		return nil, fmt.Errorf("%s bot, llm generate failed, %w", Name, err)
	}

	flog.Info("[%s] extract bookmark, llm generate result %s", Name, resp.Content)

	jsonText, err := jsonrepair.RepairJSON(resp.Content)
	if err != nil {
		return nil, fmt.Errorf("%s bot, json repair failed, %w", Name, err)
	}

	var t tagsList
	err = json.Unmarshal(utils.StringToBytes(jsonText), &t)
	if err != nil {
		return nil, fmt.Errorf("%s bot, json unmarshal failed, %w", Name, err)
	}

	return t.List, nil
}

type tagsList struct {
	List []string `json:"tags"`
}
