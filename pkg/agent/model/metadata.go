package model

// Metadata describes a known LLM and its operational limits.
type Metadata struct {
	// ID is the provider-facing model identifier.
	ID string
	// Name is a human-readable display name.
	Name string
	// Description summarizes model capabilities for UI and docs.
	Description string
	// ContextLength is the maximum input token budget.
	ContextLength int
	// MaxOutput is the maximum completion token budget.
	MaxOutput int
	// Features lists supported capabilities and modalities.
	Features []Feature
}
