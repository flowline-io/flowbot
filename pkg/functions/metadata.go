package functions

import (
	"strings"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/goccy/go-yaml"
)

// HTTPAuth holds function HTTP authentication secrets.
type HTTPAuth struct {
	Token      string `yaml:"token" json:"token,omitempty"`
	HMACSecret string `yaml:"hmac_secret" json:"hmac_secret,omitempty"`
}

// HTTPConfig holds function HTTP settings.
type HTTPConfig struct {
	Auth HTTPAuth `yaml:"auth" json:"auth"`
}

// Metadata is the function directory metadata.yaml document.
type Metadata struct {
	Name string            `yaml:"name" json:"name"`
	HTTP HTTPConfig        `yaml:"http" json:"http"`
	Env  map[string]string `yaml:"env" json:"env,omitempty"`
}

// ParseMetadataYAML unmarshals and validates function metadata YAML.
func ParseMetadataYAML(data string) (*Metadata, error) {
	var meta Metadata
	if err := yaml.Unmarshal([]byte(data), &meta); err != nil {
		return nil, types.WrapError(types.ErrInvalidArgument, "invalid metadata YAML", err)
	}
	if err := ValidateMetadata(&meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

// ValidateMetadata requires a valid name and token and/or hmac_secret.
func ValidateMetadata(meta *Metadata) error {
	if meta == nil {
		return types.Errorf(types.ErrInvalidArgument, "metadata is required")
	}
	name := strings.TrimSpace(meta.Name)
	if err := ValidateName(name); err != nil {
		return types.WrapError(types.ErrInvalidArgument, "invalid function name", err)
	}
	meta.Name = name
	token := strings.TrimSpace(meta.HTTP.Auth.Token)
	hmacSecret := strings.TrimSpace(meta.HTTP.Auth.HMACSecret)
	meta.HTTP.Auth.Token = token
	meta.HTTP.Auth.HMACSecret = hmacSecret
	if token == "" && hmacSecret == "" {
		return types.Errorf(types.ErrInvalidArgument, "http.auth.token and/or http.auth.hmac_secret is required")
	}
	return nil
}

// ValidateName reports whether name is a valid function identifier.
func ValidateName(name string) error {
	return types.ValidatePipelineName(name)
}
