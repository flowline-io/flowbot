package web

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestValidateAuthConfig(t *testing.T) {
	strongHash := mustBcryptHash(t, "correct-horse-battery")
	weakHash := mustBcryptHash(t, "admin")
	emptyHash := mustBcryptHash(t, "")
	lowCostHash := mustBcryptHashCost(t, "correct-horse-battery", bcrypt.MinCost)

	tests := []struct {
		name    string
		cfg     AuthConfig
		wantErr string
	}{
		{
			name: "empty auth allowed for setup path",
			cfg:  AuthConfig{},
		},
		{
			name: "plaintext password meets policy",
			cfg:  AuthConfig{Username: "admin", Password: "flowbot-dev-pass"},
		},
		{
			name: "password_hash meets policy",
			cfg:  AuthConfig{Username: "admin", PasswordHash: strongHash},
		},
		{
			name:    "username required when password set",
			cfg:     AuthConfig{Username: "", Password: "flowbot-dev-pass"},
			wantErr: "username is required",
		},
		{
			name:    "both password and password_hash rejected",
			cfg:     AuthConfig{Username: "admin", Password: "flowbot-dev-pass", PasswordHash: strongHash},
			wantErr: "set either password or password_hash, not both",
		},
		{
			name:    "admin/admin weak default rejected",
			cfg:     AuthConfig{Username: "admin", Password: "admin"},
			wantErr: "at least 12 characters",
		},
		{
			name:    "plaintext shorter than minimum rejected",
			cfg:     AuthConfig{Username: "alice", Password: "short-pass"},
			wantErr: "at least 12 characters",
		},
		{
			name:    "known weak plaintext rejected despite length",
			cfg:     AuthConfig{Username: "alice", Password: "password1234"},
			wantErr: "known weak password",
		},
		{
			name:    "invalid password_hash rejected",
			cfg:     AuthConfig{Username: "admin", PasswordHash: "not-a-bcrypt-hash"},
			wantErr: "invalid password_hash",
		},
		{
			name:    "password_hash of weak password rejected",
			cfg:     AuthConfig{Username: "admin", PasswordHash: weakHash},
			wantErr: "known weak password",
		},
		{
			name:    "password_hash of empty password rejected",
			cfg:     AuthConfig{Username: "admin", PasswordHash: emptyHash},
			wantErr: "empty password",
		},
		{
			name:    "password_hash with low bcrypt cost rejected",
			cfg:     AuthConfig{Username: "admin", PasswordHash: lowCostHash},
			wantErr: "cost must be at least",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAuthConfig(tt.cfg)
			if tt.wantErr == "" {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func mustBcryptHash(t *testing.T, password string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), webauthMinCost())
	require.NoError(t, err)
	return string(hash)
}

func mustBcryptHashCost(t *testing.T, password string, cost int) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	require.NoError(t, err)
	return string(hash)
}

func webauthMinCost() int {
	return 10
}
