package dropbox

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	UID         string `json:"uid"`
	AccountID   string `json:"account_id"`
	Scope       string `json:"scope"`
}
