package model

import (
	"time"
)


// Fileupload mapped from table <fileuploads>
type Fileupload struct {
	ID        int64     `json:"id"`
	UID       string    `json:"uid"`
	Fid       string    `json:"fid"`
	Name      string    `json:"name"`
	Mimetype  string    `json:"mimetype"`
	Size      int64     `json:"size"`
	Location  string    `json:"location"`
	State     FileState `json:"state"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
