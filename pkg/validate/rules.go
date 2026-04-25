// Package validate provides shared validation rules and helpers for the application.
package validate

import "github.com/go-playground/validator/v10"

// Common validation constants
const (
	// Text field limits
	TitleMaxLen = 200
	DescMaxLen  = 2000
	NameMaxLen  = 100

	// URL limits
	URLMaxLen = 2048

	// Tag limits
	TagMaxLen    = 50
	MaxTagsCount = 50
	MinTagLen    = 1

	// Query limits
	QueryMaxLen    = 100
	MaxSearchLimit = 100

	// File upload limits
	MaxFileSizeMB    = 10
	MaxFileSizeBytes = MaxFileSizeMB * 1024 * 1024
	MaxFileCount     = 10
)

// Validate is the global validator instance
var Validate = validator.New()

// ValidateVar validates a single variable against a tag
func ValidateVar(v any, tag string) (any, error) {
	return v, Validate.Var(v, tag)
}

// Common validation tags for reuse in struct definitions
const (
	// Required title: non-empty, max 200 chars
	TagTitle = "required,min=1,max=200"

	// Optional title: if provided, max 200 chars
	TagTitleOptional = "omitempty,min=1,max=200"

	// Required description: non-empty, max 2000 chars
	TagDescription = "required,min=1,max=2000"

	// Optional description: if provided, max 2000 chars
	TagDescriptionOptional = "omitempty,max=2000"

	// Required URL: valid URL format, max 2048 chars
	TagURL = "required,url,max=2048"

	// Optional URL: if provided, valid URL format
	TagURLOptional = "omitempty,url,max=2048"

	// Positive integer ID (for IDs that must be > 0)
	TagID = "required,gte=1"

	// Optional positive integer ID
	TagIDOptional = "omitempty,gte=1"

	// Non-negative integer
	TagNonNegative = "gte=0"

	// Required non-empty string
	TagRequired = "required,min=1"

	// Tag list: required, each tag 1-50 chars, max 50 tags
	TagList = "required,dive,min=1,max=50"
)
