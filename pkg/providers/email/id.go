package email

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/bytedance/sonic"
)

// MessageRef identifies a message inside a mailbox.
type MessageRef struct {
	Mailbox     string `json:"m"`
	UIDValidity uint32 `json:"v"`
	UID         uint32 `json:"u"`
}

// EncodeMessageID builds an opaque external message id (mailbox + uidvalidity + uid).
func EncodeMessageID(mailbox string, uidValidity, uid uint32) string {
	raw, err := sonic.Marshal(MessageRef{Mailbox: mailbox, UIDValidity: uidValidity, UID: uid})
	if err != nil {
		return FormatLegacyMessageID(uidValidity, uid)
	}
	return base64.RawURLEncoding.EncodeToString(raw)
}

// DecodeMessageID parses an opaque id, or a legacy uidvalidity:uid form.
func DecodeMessageID(id string) (MessageRef, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return MessageRef{}, fmt.Errorf("email: message id is required")
	}
	if raw, err := base64.RawURLEncoding.DecodeString(id); err == nil {
		var ref MessageRef
		if err := sonic.Unmarshal(raw, &ref); err == nil && ref.UID != 0 {
			return ref, nil
		}
	}
	uidValidity, uid, err := ParseLegacyMessageID(id)
	if err != nil {
		return MessageRef{}, fmt.Errorf("email: invalid message id %q", id)
	}
	return MessageRef{UIDValidity: uidValidity, UID: uid}, nil
}

// FormatLegacyMessageID builds the legacy uidvalidity:uid form (tests / migration).
func FormatLegacyMessageID(uidValidity, uid uint32) string {
	return fmt.Sprintf("%d:%d", uidValidity, uid)
}

// ParseLegacyMessageID parses uidvalidity:uid.
func ParseLegacyMessageID(id string) (uidValidity, uid uint32, err error) {
	parts := strings.Split(id, ":")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("email: invalid message id %q", id)
	}
	v, err := strconv.ParseUint(parts[0], 10, 32)
	if err != nil {
		return 0, 0, fmt.Errorf("email: invalid uidvalidity in id %q: %w", id, err)
	}
	u, err := strconv.ParseUint(parts[1], 10, 32)
	if err != nil {
		return 0, 0, fmt.Errorf("email: invalid uid in id %q: %w", id, err)
	}
	return uint32(v), uint32(u), nil
}

// ResolveTLSMode returns tls|starttls. Empty mode uses port defaults (465/993 → tls, else starttls).
func ResolveTLSMode(mode string, port int, implicitPorts ...int) string {
	m := strings.ToLower(strings.TrimSpace(mode))
	switch m {
	case TLSModeTLS, TLSModeSTARTTLS:
		return m
	}
	for _, p := range implicitPorts {
		if port == p {
			return TLSModeTLS
		}
	}
	return TLSModeSTARTTLS
}
