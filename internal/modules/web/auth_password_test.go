package web

import (
	"strings"
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
			name: "plaintext password meets policy",
			cfg:  AuthConfig{Username: "admin", Password: "flowbot-dev-pass"},
		},
		{
			name: "password_hash meets policy",
			cfg:  AuthConfig{Username: "admin", PasswordHash: strongHash},
		},
		{
			name:    "empty username rejected",
			cfg:     AuthConfig{Username: "", Password: "flowbot-dev-pass"},
			wantErr: "username is required",
		},
		{
			name:    "neither password nor password_hash rejected",
			cfg:     AuthConfig{Username: "admin"},
			wantErr: "password or password_hash is required",
		},
		{
			name:    "both password and password_hash rejected",
			cfg:     AuthConfig{Username: "admin", Password: "flowbot-dev-pass", PasswordHash: strongHash},
			wantErr: "set either password or password_hash, not both",
		},
		{
			name:    "empty password rejected",
			cfg:     AuthConfig{Username: "admin", Password: ""},
			wantErr: "password or password_hash is required",
		},
		{
			name:    "admin/admin weak default rejected",
			cfg:     AuthConfig{Username: "admin", Password: "admin"},
			wantErr: "known weak default credentials",
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

func TestVerifyPassword(t *testing.T) {
	hash := mustBcryptHash(t, "correct-horse-battery")

	tests := []struct {
		name     string
		cfg      AuthConfig
		password string
		want     bool
	}{
		{
			name:     "plaintext match succeeds",
			cfg:      AuthConfig{Password: "flowbot-dev-pass"},
			password: "flowbot-dev-pass",
			want:     true,
		},
		{
			name:     "plaintext mismatch fails",
			cfg:      AuthConfig{Password: "flowbot-dev-pass"},
			password: "wrong-password",
			want:     false,
		},
		{
			name:     "password_hash match succeeds",
			cfg:      AuthConfig{PasswordHash: hash},
			password: "correct-horse-battery",
			want:     true,
		},
		{
			name:     "password_hash mismatch fails",
			cfg:      AuthConfig{PasswordHash: hash},
			password: "wrong-password",
			want:     false,
		},
		{
			name:     "empty candidate fails against plaintext",
			cfg:      AuthConfig{Password: "flowbot-dev-pass"},
			password: "",
			want:     false,
		},
		{
			name:     "empty candidate fails against password_hash",
			cfg:      AuthConfig{PasswordHash: hash},
			password: "",
			want:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.cfg.verifyPassword(tt.password))
		})
	}
}

func TestVerifyPasswordConstantTimePlaintext(t *testing.T) {
	cfg := AuthConfig{Password: "flowbot-dev-pass"}
	// Same length as configured password to exercise ConstantTimeCompare path.
	mismatch := strings.Repeat("x", len(cfg.Password))
	assert.False(t, cfg.verifyPassword(mismatch))
	assert.True(t, cfg.verifyPassword(cfg.Password))
}

func TestVerifyCredentials(t *testing.T) {
	hash := mustBcryptHash(t, "correct-horse-battery")
	tests := []struct {
		name     string
		cfg      AuthConfig
		username string
		password string
		want     bool
	}{
		{
			name:     "matching username and plaintext password",
			cfg:      AuthConfig{Username: "admin", Password: "flowbot-dev-pass"},
			username: "admin",
			password: "flowbot-dev-pass",
			want:     true,
		},
		{
			name:     "wrong username fails even with correct password",
			cfg:      AuthConfig{Username: "admin", Password: "flowbot-dev-pass"},
			username: "other",
			password: "flowbot-dev-pass",
			want:     false,
		},
		{
			name:     "matching username and password_hash",
			cfg:      AuthConfig{Username: "admin", PasswordHash: hash},
			username: "admin",
			password: "correct-horse-battery",
			want:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.cfg.verifyCredentials(tt.username, tt.password))
		})
	}
}

func mustBcryptHash(t *testing.T, password string) string {
	t.Helper()
	return mustBcryptHashCost(t, password, bcrypt.DefaultCost)
}

func mustBcryptHashCost(t *testing.T, password string, cost int) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	require.NoError(t, err)
	return string(hash)
}
