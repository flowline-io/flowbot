package hub

import (
	"context"
	"fmt"
	"strings"

	jsonrepair "github.com/RealAlexandreAI/json-repair"
	"github.com/bytedance/sonic"

	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/utils"
)

var tagPrompt = `
You are an AI program for merging and simplifying tags. Your task is to merge and simplify the given list of tags based on semantic similarity and return a structured JSON format result.

Please follow the requirements below:
1. **Input**: You will receive a list of tags, each with a corresponding number of bookmarks.
2. **Task**: Analyze the semantics of these tags, merging tags that are semantically identical or highly similar into a more general tag, and indicate which tags have been replaced by the new tag.
3. **Output**: Return a JSON object containing the merged tags and their corresponding replaced tag lists. The JSON format is as follows:
json
{
    "tags": [
        {
            "new": "Name of the merged tag",
            "replaced": ["Replaced tag 1", "Replaced tag 2", ...]
        },
        ...
    ]
}

**Example Input:**
json
{
    "tags": [
        "Open Source Software",
        "Open Source Project",
        "Open Source Technology",
        "Artificial Intelligence",
        "Machine Learning",
        "Natural Language Processing"
    ]
}

**Example Output:**
json
{
    "tags": [
        {
            "new": "Open Source",
            "replaced": ["Open Source Software", "Open Source Project", "Open Source Technology"]
        },
        {
            "new": "Artificial Intelligence",
            "replaced": ["Machine Learning", "Natural Language Processing"]
        }
    ]
}

**Requirements:**
1. Ensure that the merged tags are semantically clear and general.
2. Clearly indicate which tags have been replaced by the new tag.
3. If some tags cannot be merged, keep them as they are.
4. The returned JSON must strictly follow the above format.
5. The tags language must be in {{.language}}.

**Input Data:**
json
{
    "tags": [
        "Docker",
        "Containerization",
        "Container Technology",
        "Container Image",
        "Programming",
        "Programming Language",
        "Programming Tools"
    ]
}
`

const tagsPrompt = `
You are a bot in a read-it-later app and your responsibility is to help with automatic tagging.
Please analyze the text between the sentences "CONTENT START HERE" and "CONTENT END HERE" and suggest relevant tags that describe its key themes, topics, and main ideas. The rules are:
- Aim for a variety of tags, including broad categories, specific keywords, and potential sub-genres.
- The tags language must be in %s.
- If it's a famous website you may also include a tag for the website. If the tag is not generic enough, don't include it.
- The content can include text for cookie consent and privacy policy, ignore those while tagging.
- Aim for 3-5 tags.
- If there are no good tags, leave the array empty.

CONTENT START HERE
%s
<CONTENT_HERE>

CONTENT END HERE
You must respond in JSON with the key "tags" and the value is an array of string tags.
`

type tagsList struct {
	List []string `json:"tags"`
}

func extractTags(ctx context.Context, _ string, title string) ([]string, error) {
	if title == "" {
		return nil, nil
	}

	prompt := fmt.Sprintf(tagsPrompt, config.App.Flowbot.Language, title)
	response, err := agentllm.LLMGenerate(ctx, agentllm.AgentModelName(agentllm.AgentExtractTags), prompt)
	if err != nil {
		return nil, fmt.Errorf("%s module, llm generate failed, %w", Name, err)
	}

	flog.Info("[%s] extract bookmark, llm generate result %s", Name, response)

	jsonText, err := jsonrepair.RepairJSON(response)
	if err != nil {
		return nil, fmt.Errorf("%s module, json repair failed, %w", Name, err)
	}

	var t tagsList
	err = sonic.Unmarshal(utils.StringToBytes(jsonText), &t)
	if err != nil {
		return nil, fmt.Errorf("%s module, json unmarshal failed, %w", Name, err)
	}

	return t.List, nil
}

const similarTagsPrompt = `
You are a bot in a reading application, responsible for helping to analyze and merge similar tags.
Please analyze the list of tags provided below, identify similar or duplicate tags, and suggest how to merge them.
For each tag that needs to be merged, please use the format "Original Tag -> Target Tag" to indicate.
- For example, if "Programming" and "Program Design" are similar, you can suggest "Program Design -> Programming".
- The tag language must be %s.
- Do not merge technical terms.
- Do not explain, just give the result, and do not answer the content of the tag

Tag list:
%s
`

func analyzeSimilarTags(ctx context.Context, tags []string) (map[string]string, error) {
	prompt := fmt.Sprintf(similarTagsPrompt, config.App.Flowbot.Language, strings.Join(tags, "\n"))
	response, err := agentllm.LLMGenerate(ctx, agentllm.AgentModelName(agentllm.AgentSimilarTags), prompt)
	if err != nil {
		return nil, fmt.Errorf("%s module, llm generate failed, %w", Name, err)
	}

	result := make(map[string]string)
	lines := strings.SplitSeq(response, "\n")
	for line := range lines {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "->", 2)
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

func replaceSimilarTags(tags []string, similarTags map[string]string) []string {
	if len(tags) == 0 {
		return tags
	}

	newTags := make([]string, 0, len(tags))
	seen := make(map[string]bool)

	for _, tag := range tags {
		if target, ok := similarTags[tag]; ok {
			tag = target
		}

		if !seen[tag] {
			newTags = append(newTags, tag)
			seen[tag] = true
		}
	}

	return newTags
}

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
