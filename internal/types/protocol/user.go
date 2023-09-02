package protocol

import (
	"github.com/flowline-io/flowbot/internal/types"
)

type User struct {
	RawData     types.KV
	Email       string
	Name        string
	FirstName   string
	LastName    string
	NickName    string
	Description string
	UserID      string
	AvatarURL   string
	Location    string
	Platform    string
	IsBot       bool
}
