package bookmark

import (
	"context"
	"fmt"
	jsonrepair "github.com/RealAlexandreAI/json-repair"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/providers"
	openaiProvider "github.com/flowline-io/flowbot/pkg/providers/openai"
	"github.com/flowline-io/flowbot/pkg/utils"
	json "github.com/json-iterator/go"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/prompts"
)

var tagsPrompt = `
You are a bot in a read-it-later app and your responsibility is to help with automatic tagging.
Please analyze the text between the sentences "CONTENT START HERE" and "CONTENT END HERE" and suggest relevant tags that describe its key themes, topics, and main ideas. The rules are:
- Aim for a variety of tags, including broad categories, specific keywords, and potential sub-genres.
- The tags language must be in chinese.
- If it's a famous website you may also include a tag for the website. If the tag is not generic enough, don't include it.
- The content can include text for cookie consent and privacy policy, ignore those while tagging.
- Aim for 3-5 tags.
- If there are no good tags, leave the array empty.

CONTENT START HERE
{{.content}}
<CONTENT_HERE>

CONTENT END HERE
You must respond in JSON with the key "tags" and the value is an array of string tags, please answer in {{.language}}.
`

func extractTags(ctx context.Context, content string) ([]string, error) {
	if content == "" {
		return nil, nil
	}
	tokenVal, _ := providers.GetConfig(openaiProvider.ID, openaiProvider.TokenKey)
	baseUrlVal, _ := providers.GetConfig(openaiProvider.ID, openaiProvider.BaseUrlKey)
	modelVal, _ := providers.GetConfig(openaiProvider.ID, openaiProvider.ModelKey)
	languageVal, _ := providers.GetConfig(openaiProvider.ID, openaiProvider.LanguageKey)

	llm, err := openai.New(
		openai.WithToken(tokenVal.String()),
		openai.WithBaseURL(baseUrlVal.String()),
		openai.WithModel(modelVal.String()),
	)
	if err != nil {
		return nil, fmt.Errorf("%s bot, openai new failed, %w", Name, err)
	}

	prompt := prompts.NewPromptTemplate(
		tagsPrompt, []string{"content", "language"},
	)
	result, err := prompt.Format(map[string]any{
		"content":  content,
		"language": languageVal.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("%s bot, prompt format failed, %w", Name, err)
	}

	text, err := llms.GenerateFromSinglePrompt(ctx, llm, result, llms.WithTemperature(0.8))
	if err != nil {
		return nil, fmt.Errorf("%s bot, llm generate failed, %w", Name, err)
	}

	flog.Info("[%s] extract bookmark, llm generate result %s", Name, text)

	jsonText, err := jsonrepair.RepairJSON(text)
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
