package model

import (
	"slices"
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/msg"
)

// DefaultContextWindow is the fallback input token budget for unknown models.
const DefaultContextWindow = 128000

const deepseekV4ProDescription = "DeepSeek V4 Pro is a large-scale Mixture-of-Experts model from DeepSeek with 1.6T total parameters and 49B activated parameters, supporting a 1M-token context window. It is designed for advanced reasoning, coding, and long-context tasks."

const deepseekV4FlashDescription = "DeepSeek V4 Flash is an efficiency-optimized Mixture-of-Experts model from DeepSeek with 284B total parameters and 13B activated parameters, supporting a 1M-token context window. It is designed for fast inference and low-latency agent workloads."

const gpt53CodexDescription = "GPT-5.3-Codex is OpenAI's most advanced agentic coding model, combining the frontier software engineering performance of GPT-5.2-Codex with the broader reasoning and professional knowledge capabilities of GPT-5.2. It achieves state-of-the-art results on SWE-Bench Pro and strong performance on Terminal-Bench 2.0 and OSWorld-Verified, reflecting improved multi-language coding, terminal proficiency, and real-world computer-use skills. The model is optimized for long-running, tool-using workflows and supports interactive steering during execution, making it suitable for complex development tasks, debugging, deployment, and iterative product work.\n\nBeyond coding, GPT-5.3-Codex performs strongly on structured knowledge-work benchmarks such as GDPval, supporting tasks like document drafting, spreadsheet analysis, slide creation, and operational research across domains. It is trained with enhanced cybersecurity awareness, including vulnerability identification capabilities, and deployed with additional safeguards for high-risk use cases. Compared to prior Codex models, it is more token-efficient and approximately 25% faster, targeting professional end-to-end workflows that span reasoning, execution, and computer interaction."

const claudeOpus48Description = "Claude Opus 4.8 is Anthropic's most capable generally available model in the Opus family. It supports text, image, and file inputs with text output, with reasoning support and a 1M-token context window for advanced analysis, coding, and long-running agent workflows."

const claudeSonnet46Description = "Sonnet 4.6 is Anthropic's most capable Sonnet-class model yet, with frontier performance across coding, agents, and professional work. It excels at iterative development, complex codebase navigation, end-to-end project management with memory, polished document creation, and confident computer use for web QA and workflow automation."

const qwen37PlusDescription = "Qwen3.7-Plus is a cost-effective model in Alibaba's Qwen3.7 series. It supports text and image input with text output, building on the series' text capabilities with a comprehensive upgrade to its multimodal reasoning, tool use, and long-context performance."

const qwen37MaxDescription = "Qwen3.7-Max is the flagship model in Alibaba's Qwen3.7 series. It supports text input and output and is designed for agent-centric workloads, with particular strengths in coding, office and productivity tasks, complex reasoning, and long-context tool-using workflows."

const gpt55ProDescription = "GPT-5.5 Pro is OpenAI's high-capability model optimized for deep reasoning and accuracy on complex, high-stakes workloads. It features a 1M+ token context window (922K input, 128K output) with support for text, image, and file inputs, function calling, and structured JSON output for agent and enterprise workflows."

const grok45Description = "Grok 4.5 is a Cursor and SpaceXAI model for long-running software engineering and knowledge work. It is trained for multi-step agent workflows that use tools, inspect results, recover from errors, and adapt strategy, targeting coding and other computer-based tasks with fewer steps than comparable frontier models."

const mimoV25Description = "Xiaomi MiMo-V2.5 is an omni-modal agent model with a 1M-token context window. It natively understands images, video, audio, and text for cross-modal perception and long-range reasoning, and supports native agent execution for browse, comprehend, reason, and act workloads."

const mimoV25ProDescription = "Xiaomi MiMo-V2.5-Pro is a trillion-scale Mixture-of-Experts model (1T total parameters, 42B activated) with a 1M-token context window. It targets peak agent performance on demanding agentic workloads while retaining the V2.5 series' omni-modal understanding of text, image, audio, and video."

var catalog = map[string]Metadata{
	"deepseek-v4-pro": {
		ID:            "deepseek-v4-pro",
		Name:          "DeepSeek V4 Pro",
		Description:   deepseekV4ProDescription,
		ContextLength: 1_048_576,
		MaxOutput:     384_000,
		Features: []Feature{
			CapChat,
			CapFunctionCall,
			CapJsonMode,
			ModalityTextIn,
			ModalityTextOut,
		},
	},
	"deepseek-v4-flash": {
		ID:            "deepseek-v4-flash",
		Name:          "DeepSeek V4 Flash",
		Description:   deepseekV4FlashDescription,
		ContextLength: 1_048_576,
		MaxOutput:     384_000,
		Features: []Feature{
			CapChat,
			CapFunctionCall,
			CapJsonMode,
			ModalityTextIn,
			ModalityTextOut,
		},
	},
	"gpt-5.3-codex": {
		ID:            "gpt-5.3-codex",
		Name:          "GPT-5.3-Codex",
		Description:   gpt53CodexDescription,
		ContextLength: 400_000,
		MaxOutput:     128_000,
		Features: []Feature{
			CapChat,
			CapFunctionCall,
			CapJsonMode,
			ModalityImageIn,
			ModalityTextIn,
			ModalityTextOut,
		},
	},
	"claude-opus-4.8": {
		ID:            "claude-opus-4.8",
		Name:          "Claude Opus 4.8",
		Description:   claudeOpus48Description,
		ContextLength: 1_000_000,
		MaxOutput:     128_000,
		Features: []Feature{
			CapChat,
			CapFunctionCall,
			CapJsonMode,
			ModalityFileIn,
			ModalityImageIn,
			ModalityTextIn,
			ModalityTextOut,
		},
	},
	"claude-sonnet-4.6": {
		ID:            "claude-sonnet-4.6",
		Name:          "Claude Sonnet 4.6",
		Description:   claudeSonnet46Description,
		ContextLength: 1_000_000,
		MaxOutput:     128_000,
		Features: []Feature{
			CapChat,
			CapFunctionCall,
			CapJsonMode,
			ModalityImageIn,
			ModalityTextIn,
			ModalityTextOut,
		},
	},
	"qwen3.7-plus": {
		ID:            "qwen3.7-plus",
		Name:          "Qwen3.7 Plus",
		Description:   qwen37PlusDescription,
		ContextLength: 1_000_000,
		MaxOutput:     65_536,
		Features: []Feature{
			CapChat,
			CapFunctionCall,
			CapJsonMode,
			ModalityImageIn,
			ModalityTextIn,
			ModalityTextOut,
		},
	},
	"qwen3.7-max": {
		ID:            "qwen3.7-max",
		Name:          "Qwen3.7 Max",
		Description:   qwen37MaxDescription,
		ContextLength: 1_000_000,
		MaxOutput:     65_536,
		Features: []Feature{
			CapChat,
			CapFunctionCall,
			CapJsonMode,
			ModalityTextIn,
			ModalityTextOut,
		},
	},
	"gpt-5.5-pro": {
		ID:            "gpt-5.5-pro",
		Name:          "GPT-5.5 Pro",
		Description:   gpt55ProDescription,
		ContextLength: 1_050_000,
		MaxOutput:     128_000,
		Features: []Feature{
			CapChat,
			CapFunctionCall,
			CapJsonMode,
			ModalityFileIn,
			ModalityImageIn,
			ModalityTextIn,
			ModalityTextOut,
		},
	},
	"grok-4.5": {
		ID:            "grok-4.5",
		Name:          "Grok 4.5",
		Description:   grok45Description,
		ContextLength: 256_000,
		MaxOutput:     128_000,
		Features: []Feature{
			CapChat,
			CapFunctionCall,
			CapJsonMode,
			ModalityTextIn,
			ModalityTextOut,
		},
	},
	"mimo-v2.5": {
		ID:            "mimo-v2.5",
		Name:          "MiMo V2.5",
		Description:   mimoV25Description,
		ContextLength: 1_048_576,
		MaxOutput:     128_000,
		Features: []Feature{
			CapChat,
			CapFunctionCall,
			CapJsonMode,
			ModalityAudioIn,
			ModalityImageIn,
			ModalityTextIn,
			ModalityTextOut,
			ModalityVideoIn,
		},
	},
	"mimo-v2.5-pro": {
		ID:            "mimo-v2.5-pro",
		Name:          "MiMo V2.5 Pro",
		Description:   mimoV25ProDescription,
		ContextLength: 1_048_576,
		MaxOutput:     128_000,
		Features: []Feature{
			CapChat,
			CapFunctionCall,
			CapJsonMode,
			ModalityAudioIn,
			ModalityImageIn,
			ModalityTextIn,
			ModalityTextOut,
			ModalityVideoIn,
		},
	},
}

// Lookup returns catalog metadata for a model ID when known.
func Lookup(id string) (Metadata, bool) {
	meta, ok := catalog[id]
	return meta, ok
}

// ContextWindowFor returns the input token budget for a model name.
func ContextWindowFor(modelName string) int {
	if meta, ok := Lookup(modelName); ok && meta.ContextLength > 0 {
		return meta.ContextLength
	}
	return DefaultContextWindow
}

// MaxContextWindow returns the largest input token budget among the given model names.
func MaxContextWindow(modelNames ...string) int {
	maxWindow := 0
	for _, name := range modelNames {
		if name == "" {
			continue
		}
		window := ContextWindowFor(name)
		if window > maxWindow {
			maxWindow = window
		}
	}
	if maxWindow == 0 {
		return DefaultContextWindow
	}
	return maxWindow
}

// HasFeature reports whether a known model supports the given feature.
func HasFeature(modelName string, feature Feature) bool {
	meta, ok := Lookup(modelName)
	if !ok {
		return false
	}
	return slices.Contains(meta.Features, feature)
}

// SupportsModality reports whether modelName may accept the given input modality.
// Unknown (uncatalogued) models allow image input and rely on the provider to fail;
// audio and video require an explicit catalog feature.
func SupportsModality(modelName string, kind msg.MediaKind) bool {
	switch kind {
	case msg.MediaKindImage:
		if _, ok := Lookup(modelName); !ok {
			return true
		}
		return HasFeature(modelName, ModalityImageIn)
	case msg.MediaKindAudio:
		return HasFeature(modelName, ModalityAudioIn)
	case msg.MediaKindVideo:
		return HasFeature(modelName, ModalityVideoIn)
	default:
		return false
	}
}

// AcceptsMediaInput reports whether modelName accepts image, audio, or video input.
func AcceptsMediaInput(modelName string) bool {
	return SupportsModality(modelName, msg.MediaKindImage) ||
		SupportsModality(modelName, msg.MediaKindAudio) ||
		SupportsModality(modelName, msg.MediaKindVideo)
}

// RegisterTestMetadata adds or overrides a catalog entry for the duration of a test.
func RegisterTestMetadata(t *testing.T, meta Metadata) {
	t.Helper()
	if meta.ID == "" {
		t.Fatal("RegisterTestMetadata: ID is required")
	}
	prev, hadPrev := catalog[meta.ID]
	catalog[meta.ID] = meta
	t.Cleanup(func() {
		if hadPrev {
			catalog[meta.ID] = prev
		} else {
			delete(catalog, meta.ID)
		}
	})
}
