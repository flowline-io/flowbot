package protocol

import (
	"github.com/flowline-io/flowbot/pkg/types"
)

type User struct {
	RawData     types.KV `json:"raw_data"`
	Email       string   `json:"email"`
	Name        string   `json:"name"`
	FirstName   string   `json:"first_name"`
	LastName    string   `json:"last_name"`
	NickName    string   `json:"nick_name"`
	Description string   `json:"description"`
	UserID      string   `json:"user_id"`
	AvatarURL   string   `json:"avatar_url"`
	Location    string   `json:"location"`
	Platform    string   `json:"platform"`
	IsBot       bool     `json:"is_bot"`
}
