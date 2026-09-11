package store

import (
	"sync"

	"github.com/flowline-io/flowbot/pkg/webauth"
)

var (
	oauthEncMu sync.RWMutex
	oauthEnc   *webauth.Encryptor
)

// SetOAuthEncryptor wires AES-GCM sealing for OAuth token columns.
// Nil disables encryption (tests / before web module Init).
func SetOAuthEncryptor(e *webauth.Encryptor) {
	oauthEncMu.Lock()
	defer oauthEncMu.Unlock()
	oauthEnc = e
}

func oauthEncryptor() *webauth.Encryptor {
	oauthEncMu.RLock()
	defer oauthEncMu.RUnlock()
	return oauthEnc
}

func sealOAuthField(v string) (string, error) {
	enc := oauthEncryptor()
	if enc == nil || v == "" {
		return v, nil
	}
	return enc.SealString(v)
}

func openOAuthField(v string) (string, error) {
	if v == "" {
		return "", nil
	}
	return oauthEncryptor().OpenString(v)
}
