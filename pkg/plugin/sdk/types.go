// Package sdk provides the plugin SDK for building flowbot module plugins.
// It defines the Module interface, no-op ModuleBase, and go-plugin serve helpers.
package sdk

// Context wraps flowbot's types.Context with SDK-friendly types.
type Context struct {
	AuthContext string            `json:"auth_context"`
	UserID      string            `json:"user_id"`
	ChannelID   string            `json:"channel_id"`
	Platform    string            `json:"platform"`
	Metadata    map[string]string `json:"metadata"`
}

// MsgPayload is the message payload returned by plugin handlers.
type MsgPayload struct {
	Text string `json:"text"`
}

// Rules holds plugin-declared rulesets.
type Rules struct {
	Commands    []any `json:"commands"`
	Forms       []any `json:"forms"`
	Webservices []any `json:"webservices"`
	Webhooks    []any `json:"webhooks"`
}
