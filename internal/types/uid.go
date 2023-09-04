package types

// Uid is a database-specific record id, suitable to be used as a primary key.
type Uid string

// ZeroUid is a constant representing uninitialized Uid.
const ZeroUid Uid = ""

// IsZero checks if Uid is uninitialized.
func (uid Uid) IsZero() bool {
	return uid == ZeroUid
}

// stringer implements fmt.Stringer interface.
func (uid Uid) String() string {
	return string(uid)
}
