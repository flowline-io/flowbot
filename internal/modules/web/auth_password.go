package web

import (
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/flowline-io/flowbot/pkg/webauth"
)

// knownWeakPasswords are rejected for both plaintext and password_hash configs.
var knownWeakPasswords = []string{
	"admin", "password", "password123", "password1234",
	"123456", "12345678", "1234567890", "123456789012",
	"qwerty", "letmein", "welcome", "changeme",
	"flowbot", "adminadmin", "adminadmin12", "root",
	"toor", "passw0rd", "default",
}

// validateAuthConfig checks modules.web.auth at module Init.
// YAML username/password are optional (used only for one-time migration); setup covers empty DB.
func validateAuthConfig(cfg AuthConfig) error {
	hasUser := strings.TrimSpace(cfg.Username) != ""
	hasPassword := cfg.Password != ""
	hasHash := cfg.PasswordHash != ""
	if hasPassword && hasHash {
		return fmt.Errorf("web auth: set either password or password_hash, not both")
	}
	if !hasPassword && !hasHash {
		return nil
	}
	if !hasUser {
		return fmt.Errorf("web auth: username is required when password or password_hash is set")
	}
	if hasHash {
		return validatePasswordHash(cfg.PasswordHash)
	}
	return webauth.ValidatePasswordStrength(cfg.Username, cfg.Password)
}

func validatePasswordHash(hash string) error {
	if err := validatePasswordHashFormat(hash); err != nil {
		return err
	}
	// Under -race, each bcrypt compare at MinBcryptCost is extremely expensive. Nightly
	// race jobs run this path many times (-count=10); keep empty-password rejection and
	// defer the full weak-list probe to rejectWeakPasswordHash unit tests (MinCost).
	if raceDetectorEnabled {
		if bcrypt.CompareHashAndPassword([]byte(hash), []byte("")) == nil {
			return fmt.Errorf("web auth: password_hash must not match an empty password")
		}
		return nil
	}
	return rejectWeakPasswordHash(hash)
}

func validatePasswordHashFormat(hash string) error {
	if !isBcryptHash(hash) {
		return fmt.Errorf("web auth: invalid password_hash (expected bcrypt)")
	}
	cost, err := bcrypt.Cost([]byte(hash))
	if err != nil {
		return fmt.Errorf("web auth: invalid password_hash (expected bcrypt)")
	}
	if cost < webauth.MinBcryptCost {
		return fmt.Errorf("web auth: password_hash bcrypt cost must be at least %d", webauth.MinBcryptCost)
	}
	return nil
}

func rejectWeakPasswordHash(hash string) error {
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte("")) == nil {
		return fmt.Errorf("web auth: password_hash must not match an empty password")
	}
	for _, w := range knownWeakPasswords {
		if bcrypt.CompareHashAndPassword([]byte(hash), []byte(w)) == nil {
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

// yamlMigrationHash returns a bcrypt hash for YAML migration credentials.
func yamlMigrationHash(cfg AuthConfig) (string, error) {
	if cfg.PasswordHash != "" {
		return cfg.PasswordHash, nil
	}
	if cfg.Password == "" {
		return "", fmt.Errorf("no yaml password")
	}
	return webauth.HashPassword(cfg.Password)
}
