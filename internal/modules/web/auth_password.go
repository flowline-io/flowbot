package web

import (
	"crypto/subtle"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// minPasswordLength is the minimum allowed length for plaintext auth.password.
const minPasswordLength = 12

// minBcryptCost is the minimum accepted bcrypt cost for auth.password_hash.
const minBcryptCost = 10

// knownWeakPasswords are rejected for plaintext password and password_hash configs.
var knownWeakPasswords = map[string]struct{}{
	"admin":        {},
	"password":     {},
	"password123":  {},
	"password1234": {},
	"123456":       {},
	"12345678":     {},
	"1234567890":   {},
	"123456789012": {},
	"qwerty":       {},
	"letmein":      {},
	"welcome":      {},
	"changeme":     {},
	"flowbot":      {},
	"adminadmin":   {},
	"adminadmin12": {},
	"root":         {},
	"toor":         {},
	"passw0rd":     {},
	"default":      {},
}

// knownWeakCredentialPairs are rejected username/password combinations.
var knownWeakCredentialPairs = []struct {
	username string
	password string
}{
	{"admin", "admin"},
	{"admin", "password"},
	{"admin", "123456"},
	{"root", "root"},
	{"root", "password"},
	{"user", "user"},
	{"test", "test"},
}

// validateAuthConfig checks modules.web.auth credentials at module Init.
func validateAuthConfig(cfg AuthConfig) error {
	if strings.TrimSpace(cfg.Username) == "" {
		return fmt.Errorf("web auth: username is required")
	}
	hasPassword := cfg.Password != ""
	hasHash := cfg.PasswordHash != ""
	if !hasPassword && !hasHash {
		return fmt.Errorf("web auth: password or password_hash is required")
	}
	if hasPassword && hasHash {
		return fmt.Errorf("web auth: set either password or password_hash, not both")
	}
	if hasHash {
		return validatePasswordHash(cfg.PasswordHash)
	}
	return validatePlaintextPassword(cfg.Username, cfg.Password)
}

func validatePlaintextPassword(username, password string) error {
	for _, pair := range knownWeakCredentialPairs {
		if username == pair.username && password == pair.password {
			return fmt.Errorf("web auth: known weak default credentials %q/%q are not allowed", pair.username, pair.password)
		}
	}
	if len(password) < minPasswordLength {
		return fmt.Errorf("web auth: password must be at least %d characters", minPasswordLength)
	}
	if _, weak := knownWeakPasswords[strings.ToLower(password)]; weak {
		return fmt.Errorf("web auth: known weak password is not allowed")
	}
	return nil
}

func validatePasswordHash(hash string) error {
	if !isBcryptHash(hash) {
		return fmt.Errorf("web auth: invalid password_hash (expected bcrypt)")
	}
	cost, err := bcrypt.Cost([]byte(hash))
	if err != nil {
		return fmt.Errorf("web auth: invalid password_hash (expected bcrypt)")
	}
	if cost < minBcryptCost {
		return fmt.Errorf("web auth: password_hash bcrypt cost must be at least %d", minBcryptCost)
	}
	// Hash length does not reveal plaintext length; empty is the only cheap length check.
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte("")) == nil {
		return fmt.Errorf("web auth: password_hash must not match an empty password")
	}
	for weak := range knownWeakPasswords {
		if bcrypt.CompareHashAndPassword([]byte(hash), []byte(weak)) == nil {
			return fmt.Errorf("web auth: password_hash matches a known weak password")
		}
	}
	for _, pair := range knownWeakCredentialPairs {
		if bcrypt.CompareHashAndPassword([]byte(hash), []byte(pair.password)) == nil {
			return fmt.Errorf("web auth: password_hash matches a known weak password")
		}
	}
	return nil
}

func isBcryptHash(hash string) bool {
	switch {
	case strings.HasPrefix(hash, "$2a$"),
		strings.HasPrefix(hash, "$2b$"),
		strings.HasPrefix(hash, "$2y$"):
		_, err := bcrypt.Cost([]byte(hash))
		return err == nil
	default:
		return false
	}
}

// verifyCredentials reports whether username and password match the configured credentials.
// Password verification always runs so failed username checks do not skip the expensive compare.
func (a AuthConfig) verifyCredentials(username, password string) bool {
	userOK := subtle.ConstantTimeCompare([]byte(username), []byte(a.Username)) == 1
	passOK := a.verifyPassword(password)
	return userOK && passOK
}

// verifyPassword reports whether password matches the configured credentials.
// When PasswordHash is set, bcrypt.CompareHashAndPassword is used.
// Otherwise a constant-time comparison against plaintext Password is used.
func (a AuthConfig) verifyPassword(password string) bool {
	if a.PasswordHash != "" {
		return bcrypt.CompareHashAndPassword([]byte(a.PasswordHash), []byte(password)) == nil
	}
	return subtle.ConstantTimeCompare([]byte(password), []byte(a.Password)) == 1
}
