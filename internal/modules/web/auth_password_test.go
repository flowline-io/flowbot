package web

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// Precomputed bcrypt cost=10 hashes (avoids GenerateFromPassword under -race -count=10).
const (
	testHashStrongCost10 = "$2a$10$U0yBQeBf7ekML/GF9JlcZOEiHXQn/w078f.ovTBLseu7o9XS6X2tC" // correct-horse-battery
	testHashAdminCost10  = "$2a$10$g9mNWvq6BIYIJ018ZB5YPeohn.u/kDT2JkyeLeQdLp.Evg3NZk4Ma" // admin
	testHashEmptyCost10  = "$2a$10$Q9UolPMVV1FgSf2mCEKln.J5NSnzJP.DqBAFoKBf4dlQg5POoJ5/W" // empty
)

func TestValidateAuthConfig(t *testing.T) {
	lowCostHash := mustBcryptHashCost(t, "correct-horse-battery", bcrypt.MinCost)

	tests := []struct {
		name     string
		cfg      AuthConfig
		wantErr  string
		skipRace bool
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
			cfg:  AuthConfig{Username: "admin", PasswordHash: testHashStrongCost10},
		},
		{
			name:    "username required when password set",
			cfg:     AuthConfig{Username: "", Password: "flowbot-dev-pass"},
			wantErr: "username is required",
		},
		{
			name:    "both password and password_hash rejected",
			cfg:     AuthConfig{Username: "admin", Password: "flowbot-dev-pass", PasswordHash: testHashStrongCost10},
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
			name:     "password_hash of weak password rejected",
			cfg:      AuthConfig{Username: "admin", PasswordHash: testHashAdminCost10},
			wantErr:  "known weak password",
			skipRace: true, // covered by TestRejectWeakPasswordHash at MinCost
		},
		{
			name:    "password_hash of empty password rejected",
			cfg:     AuthConfig{Username: "admin", PasswordHash: testHashEmptyCost10},
			wantErr: "empty password",
		},
		{
			name:    "password_hash with low bcrypt cost rejected",
			cfg:     AuthConfig{Username: "admin", PasswordHash: lowCostHash},
			wantErr: "cost must be at least",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipRace && raceDetectorEnabled {
				t.Skip("full weak-password bcrypt probe skipped under -race")
			}
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

func TestRejectWeakPasswordHash(t *testing.T) {
	t.Parallel()
	// MinCost keeps the probe loop affordable under -race while covering rejectWeakPasswordHash.
	weakHash := mustBcryptHashCost(t, "admin", bcrypt.MinCost)
	emptyHash := mustBcryptHashCost(t, "", bcrypt.MinCost)
	strongHash := mustBcryptHashCost(t, "correct-horse-battery", bcrypt.MinCost)

	tests := []struct {
		name    string
		hash    string
		wantErr string
	}{
		{name: "weak password rejected", hash: weakHash, wantErr: "known weak password"},
		{name: "empty password rejected", hash: emptyHash, wantErr: "empty password"},
		{name: "strong password accepted", hash: strongHash},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := rejectWeakPasswordHash(tt.hash)
			if tt.wantErr == "" {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func mustBcryptHashCost(t *testing.T, password string, cost int) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	require.NoError(t, err)
	return string(hash)
}
