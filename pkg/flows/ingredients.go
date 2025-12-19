package flows

import (
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/tidwall/gjson"
)

// Ingredient defines how to extract a variable from an event payload.
//
// Path is a gjson path evaluated against a JSON object. For convenience, the
// extractor evaluates the path against a root object with these keys:
// - payload: the full payload
// - item: an optional object (for polling per-item extraction)
//
// Example Path: "payload.data.id" or "item.status".
type Ingredient struct {
	Name        string `json:"name" yaml:"name"`
	Path        string `json:"path" yaml:"path"`
	Required    bool   `json:"required,omitempty" yaml:"required,omitempty"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

// ExtractIngredients extracts ingredient values from payload.
func ExtractIngredients(payload types.KV, item any, ingredients []Ingredient) (types.KV, error) {
	out := make(types.KV)
	if len(ingredients) == 0 {
		return out, nil
	}

	root := map[string]any{
		"payload": payload,
	}
	if item != nil {
		root["item"] = item
	}

	b, err := sonic.Marshal(root)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}
	jsonStr := string(b)

	for _, ing := range ingredients {
		if ing.Name == "" {
			continue
		}
		if ing.Path == "" {
			if ing.Required {
				return nil, fmt.Errorf("ingredient '%s' path is required", ing.Name)
			}
			continue
		}

		val := gjson.Get(jsonStr, ing.Path)
		if !val.Exists() {
			if ing.Required {
				return nil, fmt.Errorf("ingredient '%s' not found at path '%s'", ing.Name, ing.Path)
			}
			continue
		}
		out[ing.Name] = val.Value()
	}

	return out, nil
}
