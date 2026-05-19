package model

import (
	"time"
)


// Parameter mapped from table <parameter>
type Parameter struct {
	ID        int64     `json:"id"`
	Flag      string    `json:"flag"`
	Params    JSON      `json:"params"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	ExpiredAt time.Time `json:"expired_at"`
}
