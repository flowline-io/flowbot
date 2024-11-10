package types

import "time"

// ObjHeader is the header shared by all stored objects.
type ObjHeader struct {
	// using string to get around rethinkdb's problems with uint64;
	// `bson:"_id"` tag is for mongodb to use as primary key '_id'.
	Id string `bson:"_id"`
	//id        Uid
	CreatedAt time.Time
	UpdatedAt time.Time
}

// FileDef is a stored record of a file upload
type FileDef struct {
	ObjHeader `bson:",inline"`
	// Status of upload
	Status int
	// User who created the file
	User string
	// Type of the file.
	MimeType string
	// Size of the file in bytes.
	Size int64
	// Internal file location, i.e. path on disk or an S3 blob address.
	Location string
}

func (i *FileDef) Uid() Uid {
	return Uid(i.User)
}
