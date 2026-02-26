package flows

import (
	"errors"
	"fmt"
	"regexp"
	"slices"

	"github.com/flowline-io/flowbot/pkg/types"
)

type ParamType string

const (
	ParamTypeString ParamType = "string"
	ParamTypeNumber ParamType = "number"
	ParamTypeBool   ParamType = "bool"
	ParamTypeObject ParamType = "object"
	ParamTypeArray  ParamType = "array"
)

// ParamSpec defines an input parameter for an action.
// It is intentionally minimal but practical.
type ParamSpec struct {
	Name        string    `json:"name" yaml:"name"`
	Type        ParamType `json:"type" yaml:"type"`
	Required    bool      `json:"required,omitempty" yaml:"required,omitempty"`
	Description string    `json:"description,omitempty" yaml:"description,omitempty"`

	Enum    []string `json:"enum,omitempty" yaml:"enum,omitempty"`
	Pattern string   `json:"pattern,omitempty" yaml:"pattern,omitempty"`
}

// ValidationError is returned when params do not match a ParamSpec set.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string {
	if e == nil || len(e.Fields) == 0 {
		return "validation failed"
	}
	return "validation failed"
}

func ValidateParams(params types.KV, specs []ParamSpec) error {
	if len(specs) == 0 {
		return nil
	}
	fields := make(map[string]string)
	for _, spec := range specs {
		if spec.Name == "" {
			continue
		}
		val, ok := params[spec.Name]
		if !ok || val == nil {
			if spec.Required {
				fields[spec.Name] = "is required"
			}
			continue
		}

		switch spec.Type {
		case ParamTypeString:
			if _, ok := val.(string); !ok {
				fields[spec.Name] = "must be a string"
				continue
			}
			if len(spec.Enum) > 0 {
				s := val.(string)
				allowed := slices.Contains(spec.Enum, s)
				if !allowed {
					fields[spec.Name] = fmt.Sprintf("must be one of %v", spec.Enum)
					continue
				}
			}
			if spec.Pattern != "" {
				re, err := regexp.Compile(spec.Pattern)
				if err != nil {
					return fmt.Errorf("invalid pattern for '%s': %w", spec.Name, err)
				}
				s := val.(string)
				if !re.MatchString(s) {
					fields[spec.Name] = "does not match pattern"
					continue
				}
			}
		case ParamTypeNumber:
			switch val.(type) {
			case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
				// ok
			default:
				fields[spec.Name] = "must be a number"
				continue
			}
		case ParamTypeBool:
			if _, ok := val.(bool); !ok {
				fields[spec.Name] = "must be a boolean"
				continue
			}
		case ParamTypeObject:
			if _, ok := val.(map[string]any); !ok {
				if _, ok2 := val.(types.KV); !ok2 {
					fields[spec.Name] = "must be an object"
					continue
				}
			}
		case ParamTypeArray:
			if _, ok := val.([]any); !ok {
				fields[spec.Name] = "must be an array"
				continue
			}
		default:
			// treat empty as any
		}
	}

	if len(fields) > 0 {
		return &ValidationError{Fields: fields}
	}
	return nil
}

func IsValidationError(err error) (*ValidationError, bool) {
	var ve *ValidationError
	if errors.As(err, &ve) {
		return ve, true
	}
	return nil, false
}
