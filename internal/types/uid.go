package types

import (
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"strings"
)

// Uid is a database-specific record id, suitable to be used as a primary key.
type Uid uint64

// ZeroUid is a constant representing uninitialized Uid.
const ZeroUid Uid = 0

// NullValue is a Unicode DEL character which indicated that the value is being deleted.
const NullValue = "\u2421"

// Lengths of various Uid representations.
const (
	uidBase64Unpadded = 11
	p2pBase64Unpadded = 22
)

// IsZero checks if Uid is uninitialized.
func (uid Uid) IsZero() bool {
	return uid == ZeroUid
}

// Compare returns 0 if uid is equal to u2, 1 if u2 is greater than uid, -1 if u2 is smaller.
func (uid Uid) Compare(u2 Uid) int {
	if uid < u2 {
		return -1
	} else if uid > u2 {
		return 1
	}
	return 0
}

// MarshalBinary converts Uid to byte slice.
func (uid Uid) MarshalBinary() ([]byte, error) {
	dst := make([]byte, 8)
	binary.LittleEndian.PutUint64(dst, uint64(uid))
	return dst, nil
}

// UnmarshalBinary reads Uid from byte slice.
func (uid *Uid) UnmarshalBinary(b []byte) error {
	if len(b) < 8 {
		return errors.New("Uid.UnmarshalBinary: invalid length")
	}
	*uid = Uid(binary.LittleEndian.Uint64(b))
	return nil
}

// UnmarshalText reads Uid from string represented as byte slice.
func (uid *Uid) UnmarshalText(src []byte) error {
	if len(src) != uidBase64Unpadded {
		return errors.New("Uid.UnmarshalText: invalid length")
	}
	dec := make([]byte, base64.URLEncoding.WithPadding(base64.NoPadding).DecodedLen(uidBase64Unpadded))
	count, err := base64.URLEncoding.WithPadding(base64.NoPadding).Decode(dec, src)
	if count < 8 {
		if err != nil {
			return errors.New("Uid.UnmarshalText: failed to decode " + err.Error())
		}
		return errors.New("Uid.UnmarshalText: failed to decode")
	}
	*uid = Uid(binary.LittleEndian.Uint64(dec))
	return nil
}

// MarshalText converts Uid to string represented as byte slice.
func (uid *Uid) MarshalText() ([]byte, error) {
	if *uid == ZeroUid {
		return []byte{}, nil
	}
	src := make([]byte, 8)
	dst := make([]byte, base64.URLEncoding.WithPadding(base64.NoPadding).EncodedLen(8))
	binary.LittleEndian.PutUint64(src, uint64(*uid))
	base64.URLEncoding.WithPadding(base64.NoPadding).Encode(dst, src)
	return dst, nil
}

// MarshalJSON converts Uid to double quoted ("ajjj") string.
func (uid *Uid) MarshalJSON() ([]byte, error) {
	dst, _ := uid.MarshalText()
	return append(append([]byte{'"'}, dst...), '"'), nil
}

// UnmarshalJSON reads Uid from a double quoted string.
func (uid *Uid) UnmarshalJSON(b []byte) error {
	size := len(b)
	if size != (uidBase64Unpadded + 2) {
		return errors.New("Uid.UnmarshalJSON: invalid length")
	} else if b[0] != '"' || b[size-1] != '"' {
		return errors.New("Uid.UnmarshalJSON: unrecognized")
	}
	return uid.UnmarshalText(b[1 : size-1])
}

// String converts Uid to base64 string.
func (uid Uid) String() string {
	buf, _ := uid.MarshalText()
	return string(buf)
}

// String32 converts Uid to lowercase base32 string (suitable for file names on Windows).
func (uid Uid) String32() string {
	data, _ := uid.MarshalBinary()
	return strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(data))
}

// ParseUid parses string NOT prefixed with anything.
func ParseUid(s string) Uid {
	var uid Uid
	uid.UnmarshalText([]byte(s))
	return uid
}

// ParseUid32 parses base32-encoded string into Uid.
func ParseUid32(s string) Uid {
	var uid Uid
	if data, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(s); err == nil {
		uid.UnmarshalBinary(data)
	}
	return uid
}

// UserId converts Uid to string prefixed with 'usr', like usrXXXXX.
func (uid Uid) UserId() string {
	return uid.PrefixId("usr")
}

// FndName generates 'fnd' topic name for the given Uid.
func (uid Uid) FndName() string {
	return uid.PrefixId("fnd")
}

// PrefixId converts Uid to string prefixed with the given prefix.
func (uid Uid) PrefixId(prefix string) string {
	if uid.IsZero() {
		return ""
	}
	return prefix + uid.String()
}

// ParseUserId parses user ID of the form "usrXXXXXX".
func ParseUserId(s string) Uid {
	var uid Uid
	if strings.HasPrefix(s, "usr") {
		(&uid).UnmarshalText([]byte(s)[3:])
	}
	return uid
}
