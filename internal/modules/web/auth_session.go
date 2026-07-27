package web

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/webauth"
)

var (
	webEncryptor *webauth.Encryptor
	totpLimiter  *loginRateLimiter
)

func setWebEncryptor(e *webauth.Encryptor) {
	webEncryptor = e
}

func getEncryptor() *webauth.Encryptor {
	return webEncryptor
}

func wireTOTPRateLimiter() {
	if loginLimiterStore == nil || !handler.initialized {
		totpLimiter = nil
		return
	}
	if !config.Auth.BruteForce.bruteForceEnabled() {
		totpLimiter = nil
		return
	}
	bf := config.Auth.BruteForce
	bf.applyDefaults()
	lockoutTTL, err := time.ParseDuration(bf.LockoutDuration)
	if err != nil || lockoutTTL <= 0 {
		lockoutTTL = 15 * time.Minute
	}
	windowTTL, err := time.ParseDuration(bf.WindowDuration)
	if err != nil || windowTTL <= 0 {
		windowTTL = 15 * time.Minute
	}
	totpLimiter = newLoginRateLimiter(loginLimiterStore, 3, bf.LockoutAttempts, cache.TTL(windowTTL), cache.TTL(lockoutTTL))
}

func totpAttemptKey(ip string) string { return ip + ":totp" }

func checkTOTPRateLimit(ctx fiber.Ctx) string {
	if totpLimiter == nil {
		return ""
	}
	delay, locked := totpLimiter.Allow(ctx.Context(), totpAttemptKey(ctx.IP()))
	if locked {
		return msgAccountLocked
	}
	if delay > 0 {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-ctx.Context().Done():
			return msgAccountLocked
		case <-timer.C:
		}
	}
	return ""
}

func recordTOTPFailure(ctx fiber.Ctx) string {
	if totpLimiter == nil {
		return msgInvalidTOTP
	}
	locked, _ := totpLimiter.RecordFailure(ctx.Context(), totpAttemptKey(ctx.IP()))
	if locked {
		return msgTooManyFailedAttempts
	}
	return msgInvalidTOTP
}

func totpSuccessCleanup(ctx fiber.Ctx) {
	if totpLimiter != nil {
		totpLimiter.RecordSuccess(ctx.Context(), totpAttemptKey(ctx.IP()))
	}
}

const msgInvalidTOTP = "Invalid verification code"

// pendingSession holds parsed pending auth cookie state.
type pendingSession struct {
	Kind     string
	UID      string
	Username string
	Token    string
}

func lookupPending(ctx fiber.Ctx) (*pendingSession, error) {
	token := ctx.Cookies(webauth.CookiePending)
	if token == "" {
		return nil, types.ErrNotFound
	}
	p, err := route.LookupAccessToken(context.Background(), token)
	if err != nil || p.ID <= 0 || store.ParameterIsExpired(p) {
		return nil, types.ErrNotFound
	}
	paramKV := types.KV(p.Params)
	kind, _ := paramKV.String("kind")
	if kind != webauth.KindPending2FA && kind != webauth.KindPendingEnroll && kind != webauth.KindPendingBackupAck {
		return nil, types.ErrNotFound
	}
	uidStr, _ := paramKV.String("uid")
	username, _ := paramKV.String("username")
	if uidStr == "" || username == "" {
		return nil, types.ErrNotFound
	}
	return &pendingSession{Kind: kind, UID: uidStr, Username: username, Token: token}, nil
}

func issuePendingSession(ctx fiber.Ctx, kind, uid, username string) error {
	token, err := auth.NewToken()
	if err != nil {
		return err
	}
	params := types.KV{
		"uid":      uid,
		"username": username,
		"topic":    "web",
		"kind":     kind,
	}
	expiredAt := time.Now().Add(webauth.PendingSessionTTL)
	if err := store.Database.ParameterSet(context.Background(), auth.HashToken(token), params, expiredAt); err != nil {
		return err
	}
	setPendingCookie(ctx, token, int(webauth.PendingSessionTTL.Seconds()))
	return nil
}

func clearPendingSession(ctx fiber.Ctx) {
	token := ctx.Cookies(webauth.CookiePending)
	if token != "" {
		if err := route.DeleteAccessToken(context.Background(), token); err != nil {
			flog.Error(fmt.Errorf("failed to delete pending token: %w", err))
		}
	}
	setPendingCookie(ctx, "deleted", 0)
}

func issueFullSession(ctx fiber.Ctx, uid, username string) error {
	token, err := auth.NewToken()
	if err != nil {
		return err
	}
	params := types.KV{
		"uid":      uid,
		"username": username,
		"topic":    "web",
		"kind":     webauth.KindFull,
		"scopes":   []string{"admin:*"},
	}
	expiredAt := time.Now().Add(webauth.FullSessionTTL)
	if err := store.Database.ParameterSet(context.Background(), auth.HashToken(token), params, expiredAt); err != nil {
		return err
	}
	setAccessTokenCookie(ctx, token, int(webauth.FullSessionTTL.Seconds()), time.Time{})
	return nil
}

func setPendingCookie(ctx fiber.Ctx, value string, maxAge int) {
	c := &fiber.Cookie{
		Name:     webauth.CookiePending,
		Value:    value,
		HTTPOnly: true,
		SameSite: "Lax",
		Secure:   authConfig().cookieSecureEnabled(),
		Path:     "/",
		MaxAge:   maxAge,
	}
	if maxAge == 0 {
		c.Expires = time.Unix(0, 0)
	}
	ctx.Cookie(c)
}

func safeNext(next string) string {
	if next == "" || !strings.HasPrefix(next, "/") || strings.Contains(next, "//") || strings.Contains(next, ":") {
		return "/service/web/home"
	}
	return next
}

func accountTOTPSecret(ciphertext, nonce *[]byte) (string, error) {
	enc := getEncryptor()
	if enc == nil {
		return "", fmt.Errorf("encryptor not ready")
	}
	if ciphertext == nil || nonce == nil || len(*ciphertext) == 0 || len(*nonce) == 0 {
		return "", fmt.Errorf("totp secret missing")
	}
	pt, err := enc.Decrypt(*ciphertext, *nonce)
	if err != nil {
		return "", err
	}
	return string(pt), nil
}
