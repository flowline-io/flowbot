package web

import (
	"context"
	"errors"
	"fmt"
	"net/url"
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
		return webAuthMsg(ctx, "auth.account_locked")
	}
	if delay > 0 {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-ctx.Context().Done():
			return webAuthMsg(ctx, "auth.account_locked")
		case <-timer.C:
		}
	}
	return ""
}

func recordTOTPFailure(ctx fiber.Ctx) string {
	if totpLimiter == nil {
		return webAuthMsg(ctx, "auth.invalid_totp")
	}
	locked, _ := totpLimiter.RecordFailure(ctx.Context(), totpAttemptKey(ctx.IP()))
	if locked {
		return webAuthMsg(ctx, "auth.too_many_attempts")
	}
	return webAuthMsg(ctx, "auth.invalid_totp")
}

func totpSuccessCleanup(ctx fiber.Ctx) {
	if totpLimiter != nil {
		totpLimiter.RecordSuccess(ctx.Context(), totpAttemptKey(ctx.IP()))
	}
}

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
	if err != nil || p.ID <= 0 || route.AccessTokenIsExpired(p) {
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
	if err := store.ModuleDataStoreFromDB().ParameterSet(context.Background(), auth.HashToken(token), params, expiredAt); err != nil {
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
	if err := store.ModuleDataStoreFromDB().ParameterSet(context.Background(), auth.HashToken(token), params, expiredAt); err != nil {
		return err
	}
	setAccessTokenCookie(ctx, token, int(webauth.FullSessionTTL.Seconds()), time.Time{})
	return nil
}

func slideFullSession(ctx fiber.Ctx) {
	token := ctx.Cookies(webauth.CookieAccessToken)
	if token == "" {
		return
	}
	p, err := route.LookupAccessToken(ctx.Context(), token)
	if err != nil || p.ID <= 0 {
		return
	}
	paramKV := types.KV(p.Params)
	kind, _ := paramKV.String("kind")
	topic, _ := paramKV.String("topic")
	if !webauth.IsSlidableFullSession(kind, topic) {
		return
	}
	now := time.Now()
	if !webauth.ShouldSlideFullSession(p.ExpiredAt, now) {
		return
	}
	mds := store.ModuleDataStoreFromDB()
	if mds.Client() == nil {
		return
	}
	expiredAt := now.Add(webauth.FullSessionTTL)
	if err := mds.ParameterSet(ctx.Context(), p.Flag, paramKV, expiredAt); err != nil {
		flog.Error(fmt.Errorf("web: slide full session: %w", err))
		return
	}
	setAccessTokenCookie(ctx, token, int(webauth.FullSessionTTL.Seconds()), time.Time{})
}

func slideFullSessionMiddleware() fiber.Handler {
	return func(ctx fiber.Ctx) error {
		slideFullSession(ctx)
		return ctx.Next()
	}
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
	if safe, ok := safeServiceWebRedirectURL(next, true); ok {
		return safe
	}
	return "/service/web/home"
}

// safeServiceWebRedirectURL allows only relative /service/web paths (no scheme/host,
// protocol-relative URL, userinfo, opaque, or backslash open-redirect tricks).
// When allowExactRoot is true, path "/service/web" is accepted in addition to
// paths under "/service/web/".
func safeServiceWebRedirectURL(raw string, allowExactRoot bool) (string, bool) {
	if raw == "" || strings.ContainsAny(raw, "\\") || strings.HasPrefix(raw, "//") {
		return "", false
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", false
	}
	if u.Scheme != "" || u.Host != "" || u.User != nil || u.Opaque != "" {
		return "", false
	}
	if allowExactRoot && u.Path == "/service/web" {
		return u.RequestURI(), true
	}
	if !strings.HasPrefix(u.Path, "/service/web/") {
		return "", false
	}
	return u.RequestURI(), true
}

func accountTOTPSecret(ciphertext, nonce *[]byte) (string, error) {
	enc := getEncryptor()
	if enc == nil {
		return "", errors.New("encryptor not ready")
	}
	if ciphertext == nil || nonce == nil || len(*ciphertext) == 0 || len(*nonce) == 0 {
		return "", errors.New("totp secret missing")
	}
	pt, err := enc.Decrypt(*ciphertext, *nonce)
	if err != nil {
		return "", err
	}
	return string(pt), nil
}
