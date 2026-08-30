package webauth_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/webauth"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	enc, _, _, err := webauth.LoadEncryptor("dGVzdC1rZXktMzItYnl0ZXMtbG9uZyEhISE", t.TempDir())
	require.NoError(t, err)
	ct, nonce, err := enc.Encrypt([]byte("totp-secret"))
	require.NoError(t, err)
	pt, err := enc.Decrypt(ct, nonce)
	require.NoError(t, err)
	assert.Equal(t, "totp-secret", string(pt))
}

func TestLoadEncryptorCreatesKeyFile(t *testing.T) {
	dir := t.TempDir()
	enc, fromFile, created, err := webauth.LoadEncryptor("", dir)
	require.NoError(t, err)
	assert.True(t, fromFile)
	assert.True(t, created)
	require.NotNil(t, enc)

	enc2, fromFile2, created2, err := webauth.LoadEncryptor("", dir)
	require.NoError(t, err)
	assert.True(t, fromFile2)
	assert.False(t, created2)

	ct, nonce, err := enc.Encrypt([]byte("x"))
	require.NoError(t, err)
	pt, err := enc2.Decrypt(ct, nonce)
	require.NoError(t, err)
	assert.Equal(t, "x", string(pt))

	_, err = os.Stat(filepath.Join(dir, "web_auth_encryption.key"))
	require.NoError(t, err)
}

func TestPasswordAndBackupCodes(t *testing.T) {
	enc, _, _, err := webauth.LoadEncryptor("another-dev-key-for-tests-only", t.TempDir())
	require.NoError(t, err)

	hash, err := webauth.HashPassword("flowbot-dev-pass")
	require.NoError(t, err)
	assert.True(t, webauth.CheckPassword(hash, "flowbot-dev-pass"))
	assert.False(t, webauth.CheckPassword(hash, "wrong-password"))

	codes, hashes, err := enc.GenerateBackupCodes(3)
	require.NoError(t, err)
	require.Len(t, codes, 3)
	require.Len(t, hashes, 3)

	remaining, ok := enc.ConsumeBackupCode(hashes, codes[1])
	assert.True(t, ok)
	assert.Len(t, remaining, 2)
	_, ok = enc.ConsumeBackupCode(remaining, codes[1])
	assert.False(t, ok)
}

func TestVerifyTOTP(t *testing.T) {
	secret, err := webauth.GenerateTOTPSecret()
	require.NoError(t, err)
	now := time.Unix(1_700_000_000, 0)
	_, ok := webauth.VerifyTOTP(secret, "000000", now)
	assert.False(t, ok)

	code, err := webauth.CodeAt(secret, now)
	require.NoError(t, err)
	step, ok := webauth.VerifyTOTP(secret, code, now)
	assert.True(t, ok)
	assert.Equal(t, now.Unix()/30, step)

	uri := webauth.TOTPProvisioningURI(secret, "admin", "Flowbot")
	assert.Contains(t, uri, "otpauth://totp/")
	assert.Contains(t, uri, secret)
}

func TestValidatePasswordStrength(t *testing.T) {
	require.Error(t, webauth.ValidatePasswordStrength("admin", "short"))
	require.Error(t, webauth.ValidatePasswordStrength("admin", "password1234"))
	require.NoError(t, webauth.ValidatePasswordStrength("admin", "flowbot-dev-pass"))
}

func TestIsSlidableFullSession(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		kind  string
		topic string
		want  bool
	}{
		{name: "full web session", kind: webauth.KindFull, topic: "web", want: true},
		{name: "pending 2fa is not slidable", kind: webauth.KindPending2FA, topic: "web", want: false},
		{name: "full session on other topic", kind: webauth.KindFull, topic: "cli", want: false},
		{name: "full session missing topic", kind: webauth.KindFull, topic: "", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, webauth.IsSlidableFullSession(tt.kind, tt.topic))
		})
	}
}

func TestShouldSlideFullSession(t *testing.T) {
	t.Parallel()
	now := time.Unix(1_700_000_000, 0)
	tests := []struct {
		name      string
		expiredAt time.Time
		want      bool
	}{
		{name: "fresh 24h remaining skips", expiredAt: now.Add(webauth.FullSessionTTL), want: false},
		{name: "remaining just inside throttle slides", expiredAt: now.Add(webauth.FullSessionTTL - webauth.SessionSlideThrottle - time.Second), want: true},
		{name: "one hour remaining slides", expiredAt: now.Add(time.Hour), want: true},
		{name: "already expired does not resurrect", expiredAt: now.Add(-time.Second), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, webauth.ShouldSlideFullSession(tt.expiredAt, now))
		})
	}
}
