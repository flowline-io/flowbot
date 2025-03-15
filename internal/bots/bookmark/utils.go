package bookmark

import (
	"context"
	"fmt"
	"strings"

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

const tagsPrompt = `
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

const similarTagsPrompt = `
You are a bot in a reading application, responsible for helping to analyze and merge similar tags.
Please analyze the list of tags provided below, identify similar or duplicate tags, and suggest how to merge them.
For each tag that needs to be merged, please use the format "Original Tag -> Target Tag" to indicate.
- For example, if "Programming" and "Program Design" are similar, you can suggest "Program Design -> Programming".
- The tag language must be {{.language}}.
- Do not merge technical terms.
- Do not explain, just give the result, and do not answer the content of the tag

Tag list:
{{.tags}}
`

// analyzeSimilarTags Analyze similar tags using a large model
func analyzeSimilarTags(ctx context.Context, tags []string) (map[string]string, error) {
	messages, err := prompt.FromMessages(schema.GoTemplate,
		schema.UserMessage(similarTagsPrompt),
	).Format(ctx, map[string]any{
		"tags":     strings.Join(tags, "\n"),
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

	result := make(map[string]string)
	lines := strings.Split(resp.Content, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Split(line, "->")
		if len(parts) != 2 {
			continue
		}

		originalTag := strings.TrimSpace(parts[0])
		similarTag := strings.TrimSpace(parts[1])
		result[originalTag] = similarTag
	}

	flog.Info("[%s] analyze similar tags, count %d", Name, len(result))

	return result, nil
}

// replaceSimilarTags Replace similar tags
func replaceSimilarTags(tags []string, similarTags map[string]string) []string {
	if len(tags) == 0 {
		return tags
	}

	newTags := make([]string, 0, len(tags))
	seen := make(map[string]bool)

	for _, tag := range tags {
		// If there is a similar tag mapping, use the mapped tag
		if target, ok := similarTags[tag]; ok {
			tag = target
		}

		// Remove duplicates
		if !seen[tag] {
			newTags = append(newTags, tag)
			seen[tag] = true
		}
	}

	return newTags
}

// sliceEqual Check if two string slices are equal
func sliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// convertTagsToStrings Convert hoarder.Tag slice to string slice
func convertTagsToStrings(tags []hoarder.Tag) []string {
	result := make([]string, len(tags))
	for i, tag := range tags {
		result[i] = tag.Name
	}
	return result
}

// convertBookmarkTagsToStrings Convert BookmarkTagsInner slice to string slice
func convertBookmarkTagsToStrings(tags []hoarder.BookmarkTagsInner) []string {
	result := make([]string, len(tags))
	for i, tag := range tags {
		result[i] = tag.Name
	}
	return result
}

// convertStringsToBookmarkTags Convert string slice to BookmarkTagsInner slice
func convertStringsToBookmarkTags(tags []string) []hoarder.BookmarkTagsInner {
	result := make([]hoarder.BookmarkTagsInner, len(tags))
	for i, tag := range tags {
		result[i] = hoarder.BookmarkTagsInner{
			Name: tag,
		}
	}
	return result
}
