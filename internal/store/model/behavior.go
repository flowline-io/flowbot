package model

import (
	"time"
)


// Behavior mapped from table <behavior>
type Behavior struct {
	ID        int64     `json:"id"`
	UID       string    `json:"uid"`
	Flag      string    `json:"flag"`
	Count_    int32     `json:"count"`
	Extra     *JSON     `json:"extra"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
