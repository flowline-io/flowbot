// Package webauth provides password hashing, TOTP, and encryption helpers for web UI login.
package webauth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1" // #nosec G505 -- HMAC-SHA1 is the RFC 6238 TOTP default; authenticator apps expect it
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	// MinPasswordLength is the minimum allowed password length for web accounts.
	MinPasswordLength = 12
	// MinBcryptCost is the minimum accepted bcrypt cost.
	MinBcryptCost = 10
	// BackupCodeCount is how many one-time backup codes are issued at enroll time.
	BackupCodeCount = 10
	// BackupCodeBytes is the entropy used per backup code before encoding.
	BackupCodeBytes = 5
	// PendingSessionTTL is the max lifetime of pending_2fa / pending_enroll cookies.
	PendingSessionTTL = 5 * time.Minute
	// FullSessionTTL is the idle lifetime of a full web cookie session after login or slide.
	FullSessionTTL = 24 * time.Hour
	// SessionSlideThrottle skips persist and Set-Cookie when remaining TTL is still
	// greater than FullSessionTTL minus this duration.
	SessionSlideThrottle = 5 * time.Minute
	defaultKeyFile       = "web_auth_encryption.key"
)

// Session kinds stored in parameter params["kind"].
const (
	KindPending2FA       = "pending_2fa"
	KindPendingEnroll    = "pending_enroll"
	KindPendingBackupAck = "pending_backup_ack"
	KindFull             = "full"
)

// Cookie names for web auth.
const (
	CookieAccessToken = "accessToken"
	CookiePending     = "pendingAuth"
)

// IsSlidableFullSession reports whether a persisted token is a browser web UI session.
func IsSlidableFullSession(kind, topic string) bool {
	return kind == KindFull && topic == "web"
}

// ShouldSlideFullSession reports whether expiredAt is stale enough to push to now+FullSessionTTL.
func ShouldSlideFullSession(expiredAt, now time.Time) bool {
	remaining := expiredAt.Sub(now)
	if remaining <= 0 {
		return false
	}
	return remaining <= FullSessionTTL-SessionSlideThrottle
}

// Encryptor holds the AES-256-GCM key used for TOTP secrets and backup-code pepper.
type Encryptor struct {
	key []byte
}

// LoadEncryptor resolves the encryption key from explicit config or a persistent file.
// When encryptionKey is empty, a key file is created/read under keyDir (0600) and a warning
// should be logged by the caller (FromFile reports created=true).
func LoadEncryptor(encryptionKey, keyDir string) (enc *Encryptor, fromFile, created bool, err error) {
	if strings.TrimSpace(encryptionKey) != "" {
		key, err := parseKey(encryptionKey)
		if err != nil {
			return nil, false, false, err
		}
		return &Encryptor{key: key}, false, false, nil
	}
	if keyDir == "" {
		keyDir = "."
	}
	path := filepath.Join(keyDir, defaultKeyFile)
	data, readErr := os.ReadFile(path)
	if readErr == nil {
		key, err := parseKey(strings.TrimSpace(string(data)))
		if err != nil {
			return nil, true, false, fmt.Errorf("webauth: read key file: %w", err)
		}
		return &Encryptor{key: key}, true, false, nil
	}
	if !os.IsNotExist(readErr) {
		return nil, true, false, fmt.Errorf("webauth: read key file: %w", readErr)
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return nil, true, false, fmt.Errorf("webauth: generate key: %w", err)
	}
	encoded := base64.RawStdEncoding.EncodeToString(raw)
	if err := os.MkdirAll(keyDir, 0o700); err != nil {
		return nil, true, false, fmt.Errorf("webauth: mkdir key dir: %w", err)
	}
	if err := os.WriteFile(path, []byte(encoded+"\n"), 0o600); err != nil {
		return nil, true, false, fmt.Errorf("webauth: write key file: %w", err)
	}
	return &Encryptor{key: raw}, true, true, nil
}

func parseKey(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("webauth: empty encryption key")
	}
	if decoded, err := base64.RawStdEncoding.DecodeString(s); err == nil && len(decoded) == 32 {
		return decoded, nil
	}
	if decoded, err := base64.StdEncoding.DecodeString(s); err == nil && len(decoded) == 32 {
		return decoded, nil
	}
	sum := sha256.Sum256([]byte(s))
	return sum[:], nil
}

// Encrypt encrypts plaintext with AES-256-GCM. Returns ciphertext and nonce.
func (e *Encryptor) Encrypt(plaintext []byte) (ciphertext, nonce []byte, err error) {
	if e == nil || len(e.key) != 32 {
		return nil, nil, fmt.Errorf("webauth: encryptor not ready")
	}
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}
	nonce = make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, err
	}
	ciphertext = gcm.Seal(nil, nonce, plaintext, nil)
	return ciphertext, nonce, nil
}

// Decrypt decrypts ciphertext produced by Encrypt.
func (e *Encryptor) Decrypt(ciphertext, nonce []byte) ([]byte, error) {
	if e == nil || len(e.key) != 32 {
		return nil, fmt.Errorf("webauth: encryptor not ready")
	}
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return gcm.Open(nil, nonce, ciphertext, nil)
}

// HashPassword returns a bcrypt hash of password.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), MinBcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword reports whether password matches hash.
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// HashBackupCode returns a HMAC-SHA256 hex digest of code using the encryptor key as pepper.
func (e *Encryptor) HashBackupCode(code string) string {
	mac := hmac.New(sha256.New, e.key)
	_, _ = mac.Write([]byte(strings.TrimSpace(code)))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// GenerateBackupCodes returns plaintext codes and their hashes.
func (e *Encryptor) GenerateBackupCodes(n int) (codes, hashes []string, err error) {
	if n <= 0 {
		n = BackupCodeCount
	}
	codes = make([]string, 0, n)
	hashes = make([]string, 0, n)
	for range n {
		raw := make([]byte, BackupCodeBytes)
		if _, err := rand.Read(raw); err != nil {
			return nil, nil, err
		}
		code := strings.ToUpper(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw))
		codes = append(codes, code)
		hashes = append(hashes, e.HashBackupCode(code))
	}
	return codes, hashes, nil
}

// ConsumeBackupCode returns updated hashes with matching code removed, or ok=false.
func (e *Encryptor) ConsumeBackupCode(hashes []string, code string) (remaining []string, ok bool) {
	want := e.HashBackupCode(code)
	for i, h := range hashes {
		if hmac.Equal([]byte(h), []byte(want)) {
			out := make([]string, 0, len(hashes)-1)
			out = append(out, hashes[:i]...)
			out = append(out, hashes[i+1:]...)
			return out, true
		}
	}
	return hashes, false
}

// GenerateTOTPSecret returns a new base32-encoded TOTP secret (20 bytes).
func GenerateTOTPSecret() (string, error) {
	raw := make([]byte, 20)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw), nil
}

// CodeAt returns the 6-digit TOTP for secret at the given time (exact step, no window).
func CodeAt(secret string, now time.Time) (string, error) {
	secret = strings.TrimSpace(strings.ToUpper(secret))
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil {
		return "", fmt.Errorf("webauth: invalid totp secret: %w", err)
	}
	return hotp(key, uint64(now.Unix()/30)), nil
}

// LooksLikeTOTPCode reports whether code is a 6-digit TOTP shape (not a backup code).
func LooksLikeTOTPCode(code string) bool {
	code = strings.TrimSpace(code)
	if len(code) != 6 {
		return false
	}
	for _, r := range code {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// VerifyTOTP validates a 6-digit code for secret at the current time (±1 step).
// Returns the matched time step (unix/30) so callers can reject replays.
func VerifyTOTP(secret, code string, now time.Time) (step int64, ok bool) {
	secret = strings.TrimSpace(strings.ToUpper(secret))
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil {
		return 0, false
	}
	code = strings.TrimSpace(code)
	if len(code) != 6 {
		return 0, false
	}
	cur := now.Unix() / 30
	for _, delta := range []int64{-1, 0, 1} {
		step = cur + delta
		if hotp(key, uint64(step)) == code {
			return step, true
		}
	}
	return 0, false
}

func hotp(key []byte, counter uint64) string {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], counter)
	mac := hmac.New(sha1.New, key)
	_, _ = mac.Write(buf[:])
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	truncated := binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7fffffff
	return fmt.Sprintf("%06d", truncated%1000000)
}

// TOTPProvisioningURI builds an otpauth URI for authenticator apps.
func TOTPProvisioningURI(secret, username, issuer string) string {
	if issuer == "" {
		issuer = "Flowbot"
	}
	label := url.PathEscape(issuer) + ":" + url.PathEscape(username)
	q := url.Values{}
	q.Set("secret", secret)
	q.Set("issuer", issuer)
	q.Set("algorithm", "SHA1")
	q.Set("digits", "6")
	q.Set("period", "30")
	return "otpauth://totp/" + label + "?" + q.Encode()
}

// UIDForUsername returns the stable web uid for a username.
func UIDForUsername(username string) string {
	return "user-" + username
}

// ValidatePasswordStrength rejects short or known-weak passwords.
func ValidatePasswordStrength(username, password string) error {
	if len(password) < MinPasswordLength {
		return fmt.Errorf("password must be at least %d characters", MinPasswordLength)
	}
	weak := map[string]struct{}{
		"admin": {}, "password": {}, "password123": {}, "password1234": {},
		"123456": {}, "12345678": {}, "1234567890": {}, "123456789012": {},
		"qwerty": {}, "letmein": {}, "welcome": {}, "changeme": {},
		"flowbot": {}, "adminadmin": {}, "adminadmin12": {}, "root": {},
		"toor": {}, "passw0rd": {}, "default": {},
	}
	if _, ok := weak[strings.ToLower(password)]; ok {
		return fmt.Errorf("known weak password is not allowed")
	}
	pairs := [][2]string{
		{"admin", "admin"}, {"admin", "password"}, {"admin", "123456"},
		{"root", "root"}, {"root", "password"}, {"user", "user"}, {"test", "test"},
	}
	for _, p := range pairs {
		if username == p[0] && password == p[1] {
			return fmt.Errorf("known weak default credentials are not allowed")
		}
	}
	return nil
}
