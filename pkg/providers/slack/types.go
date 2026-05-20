package slack

// OAuthAuthedUser represents the authed_user in the OAuth response.
type OAuthAuthedUser struct {
	ID          string `json:"id"`
	Scope       string `json:"scope"`
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

// OAuthTeam represents the team in the OAuth response.
type OAuthTeam struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// OAuthV2AccessResponse represents the Slack oauth.v2.access API response.
type OAuthV2AccessResponse struct {
	OK         bool           `json:"ok"`
	Error      string         `json:"error,omitempty"`
	AuthedUser OAuthAuthedUser `json:"authed_user"`
	Team       OAuthTeam       `json:"team"`
}

// IdentityUser represents the user in the identity response.
type IdentityUser struct {
	Name    string `json:"name"`
	ID      string `json:"id"`
	Image48 string `json:"image_48"`
}

// IdentityTeam represents the team in the identity response.
type IdentityTeam struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// IdentityResponse represents the Slack users.identity API response.
type IdentityResponse struct {
	OK    bool         `json:"ok"`
	Error string       `json:"error,omitempty"`
	User  IdentityUser `json:"user"`
	Team  IdentityTeam `json:"team"`
}
