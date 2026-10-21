package mcp

import (
	"errors"
	"net/url"
	"strings"
)

const (
	// ServerCardSchema is the v1 Server Card JSON Schema URL (SEP-2127).
	ServerCardSchema = "https://static.modelcontextprotocol.io/schemas/v1/server-card.schema.json"
	// ServerCardName is the reverse-DNS identity advertised on the Server Card.
	ServerCardName = "flowline-io/flowbot"
	// ServerCardMediaType is the media type for a hosted Server Card.
	ServerCardMediaType = "application/mcp-server-card+json"
	// SupportedProtocolVersion is the MCP protocol version this server speaks.
	SupportedProtocolVersion = "2026-07-28"

	serverCardDescription = "Homelab data hub. Connect to list, inspect, and run automation."
)

// ServerCard is a public SEP-2127 discovery document.
// It identifies the server and one remote endpoint. It does not list tools.
type ServerCard struct {
	// Schema is the Server Card JSON Schema URL.
	Schema string `json:"$schema"`
	// Name is the reverse-DNS server identity.
	Name string `json:"name"`
	// Version is the server build version.
	Version string `json:"version"`
	// Description is the public summary. It does not list tools.
	Description string `json:"description"`
	// Remotes are the HTTP endpoints a client can open.
	Remotes []ServerCardRemote `json:"remotes"`
}

// ServerCardRemote is one HTTP transport on a Server Card.
type ServerCardRemote struct {
	// Type is the MCP transport. This server uses streamable-http.
	Type string `json:"type"`
	// URL is the Streamable HTTP endpoint.
	URL string `json:"url"`
	// SupportedProtocolVersions lists the protocol versions this endpoint speaks.
	SupportedProtocolVersions []string `json:"supportedProtocolVersions"`
}

// ServerCardURL is the discovery URL for a Streamable HTTP endpoint.
func ServerCardURL(endpoint string) (string, error) {
	u, err := parseHTTPEndpoint(endpoint)
	if err != nil {
		return "", err
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/server-card"
	return u.String(), nil
}

// ServerCardURLOrEmpty returns the discovery URL, or empty when endpoint is not HTTP.
func ServerCardURLOrEmpty(endpoint string) string {
	card, err := ServerCardURL(endpoint)
	if err != nil {
		return ""
	}
	return card
}

// NewServerCard returns the public card for one Streamable HTTP endpoint.
// The document has no tool list. An endpoint that is not HTTP, or a blank version, returns an error.
func NewServerCard(endpoint, version string) (ServerCard, error) {
	u, err := parseHTTPEndpoint(endpoint)
	if err != nil {
		return ServerCard{}, err
	}
	version = strings.TrimSpace(version)
	if version == "" || len(version) > 255 || strings.ContainsAny(version, " \t\r\n") {
		return ServerCard{}, errors.New("mcp version is invalid")
	}
	return ServerCard{
		Schema:      ServerCardSchema,
		Name:        ServerCardName,
		Version:     version,
		Description: serverCardDescription,
		Remotes: []ServerCardRemote{{
			Type:                      "streamable-http",
			URL:                       u.String(),
			SupportedProtocolVersions: []string{SupportedProtocolVersion},
		}},
	}, nil
}

func parseHTTPEndpoint(raw string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	u, err := url.ParseRequestURI(raw)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, errors.New("mcp url is invalid")
	}
	u.User = nil
	u.RawQuery = ""
	u.Fragment = ""
	u.Path = strings.TrimRight(u.Path, "/")
	if u.Path == "" {
		u.Path = "/"
	}
	return u, nil
}
