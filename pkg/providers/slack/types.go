package slack

// OAuthV2AccessResponse represents the Slack oauth.v2.access API response.
type OAuthV2AccessResponse struct {
	OK         bool   `json:"ok"`
	Error      string `json:"error,omitempty"`
	AuthedUser struct {
		ID          string `json:"id"`
		Scope       string `json:"scope"`
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
	} `json:"authed_user"`
	Team struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"team"`
}

// IdentityResponse represents the Slack users.identity API response.
type IdentityResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
	User  struct {
		Name    string `json:"name"`
		ID      string `json:"id"`
		Image48 string `json:"image_48"`
	} `json:"user"`
	Team struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"team"`
}
