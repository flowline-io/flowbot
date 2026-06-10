package llm

// Agent name constants identify YAML config.agents entries for single-shot LLM tasks.
const (
	AgentChat              = "chat"
	AgentBillClassify      = "bill_classify"
	AgentExtractTags       = "extract_tags"
	AgentSimilarTags       = "similar_tags"
	AgentNewsSummary       = "news_summary"
	AgentRepoReviewComment = "repo_review_comment"
)

// Model provider constants match config.models provider values.
const (
	ProviderOpenAI           = "openai"
	ProviderOpenAICompatible = "openai_compatible"
	ProviderGemini           = "gemini"
	ProviderAnthropic        = "anthropic"
)
