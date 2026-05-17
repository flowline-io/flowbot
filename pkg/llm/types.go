package llm

// Role type
type Role = string

// Role constants
const (
	SystemRole    = "system"
	UserRole      = "user"
	AssistantRole = "model"
	ToolRole      = "tool"
)

// Agent name constants
const (
	AgentChat              = "chat"
	AgentBillClassify      = "bill_classify"
	AgentExtractTags       = "extract_tags"
	AgentSimilarTags       = "similar_tags"
	AgentNewsSummary       = "news_summary"
	AgentRepoReviewComment = "repo_review_comment"
)

// Model provider constants
const (
	ProviderOpenAI           = "openai"
	ProviderOpenAICompatible = "openai_compatible"
	ProviderGemini           = "gemini"
	ProviderAnthropic        = "anthropic"
)

// Message represents a chat message
type Message struct {
	Role    string
	Content string
	Name    string
}

// ToolCall represents a function call from the model
type ToolCall struct {
	ID       string
	Type     string
	Function ToolCallFunction
}

// ToolCallFunction represents function call details
type ToolCallFunction struct {
	Name      string
	Arguments string
}

// ParamsOneOf represents tool parameters schema
type ParamsOneOf struct {
	OneOf []Schema
}

// Schema represents JSON schema for tool parameters
type Schema struct {
	Type        string
	Description string
	Properties  map[string]Schema
	Required    []string
}

// ToJSONSchema converts ParamsOneOf to JSON schema map
func (p *ParamsOneOf) ToJSONSchema() (map[string]any, error) {
	if p == nil {
		return map[string]any{
			"type": "object",
		}, nil
	}

	result := map[string]any{
		"type": "object",
	}

	if len(p.OneOf) > 0 {
		properties := make(map[string]any)
		var required []string
		for _, s := range p.OneOf {
			if s.Properties != nil {
				for name, prop := range s.Properties {
					properties[name] = map[string]any{
						"type":        prop.Type,
						"description": prop.Description,
					}
				}
			}
			if len(s.Required) > 0 {
				required = append(required, s.Required...)
			}
		}
		if len(properties) > 0 {
			result["properties"] = properties
		}
		if len(required) > 0 {
			result["required"] = required
		}
	}

	return result, nil
}
