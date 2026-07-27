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
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/views/pages"
	"github.com/flowline-io/flowbot/pkg/webauth"
)

var loginWebserviceRules = []webservice.Rule{
	webservice.Get("/login", loginPage, route.WithNotAuth()),
	webservice.Post("/login", loginSubmit, route.WithNotAuth()),
	webservice.Get("/login/2fa", login2FAPage, route.WithNotAuth()),
	webservice.Post("/login/2fa", login2FASubmit, route.WithNotAuth()),
	webservice.Get("/setup", setupPage, route.WithNotAuth()),
	webservice.Post("/setup", setupSubmit, route.WithNotAuth()),
	webservice.Get("/setup/2fa", enroll2FAPage, route.WithNotAuth()),
	webservice.Post("/setup/2fa", enroll2FASubmit, route.WithNotAuth()),
	webservice.Get("/setup/backup-codes", backupCodesPage, route.WithNotAuth()),
	webservice.Post("/setup/backup-codes/ack", backupCodesAck, route.WithNotAuth()),
	webservice.Post("/logout", logout, route.WithNotAuth()),
	webservice.Get("/csrf-token", csrfTokenJSON, route.WithNotAuth()),
}

const (
	msgAccountLocked         = "Account temporarily locked. Please try again later."
	msgTooManyFailedAttempts = "Too many failed attempts. Account temporarily locked. Please try again later."
	msgInvalidCredentials    = "Invalid username or password"
)

func loginPage(ctx fiber.Ctx) error {
	if isAuthenticated(ctx) {
		return ctx.Redirect().To(safeNext(ctx.Query("next", "/service/web/home")))
	}
	ws := store.WebAccountStoreFromDB()
	if n, err := ws.Count(context.Background()); err == nil && n == 0 {
		return ctx.Redirect().To("/service/web/setup")
	}
	if pending, err := lookupPending(ctx); err == nil && pending != nil {
		if pending.Kind == webauth.KindPending2FA {
			return ctx.Redirect().To("/service/web/login/2fa")
		}
		if pending.Kind == webauth.KindPendingEnroll {
			return ctx.Redirect().To("/service/web/setup/2fa")
		}
		if pending.Kind == webauth.KindPendingBackupAck {
			return ctx.Redirect().To("/service/web/setup/backup-codes")
		}
	}
	next := ctx.Query("next", "")
	csrfTok, err := ensureCSRFCookie(ctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "csrf token error")
	}
	ctx.Set("Cache-Control", "no-store")
	ctx.Type("html")
	return pages.LoginPage(next, "", csrfTok).Render(context.Background(), ctx.Response().BodyWriter())
}

func renderLoginForm(ctx fiber.Ctx, next, errorMsg string) error {
	csrfTok, err := ensureCSRFCookie(ctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "csrf token error")
	}
	ctx.Set("Cache-Control", "no-store")
	ctx.Type("html")
	return pages.LoginForm(next, errorMsg, csrfTok).Render(context.Background(), ctx.Response().BodyWriter())
}

func checkLoginRateLimit(ctx fiber.Ctx) string {
	if loginLimiter == nil {
		return ""
	}
	delay, locked := loginLimiter.Allow(ctx.Context(), ctx.IP())
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

func recordLoginFailure(ctx fiber.Ctx) string {
	if loginLimiter == nil {
		return msgInvalidCredentials
	}
	locked, _ := loginLimiter.RecordFailure(ctx.Context(), ctx.IP())
	if locked {
		return msgTooManyFailedAttempts
	}
	return msgInvalidCredentials
}

func loginSuccessCleanup(ctx fiber.Ctx) {
	if loginLimiter != nil {
		loginLimiter.RecordSuccess(ctx.Context(), ctx.IP())
	}
}

func loginSubmit(ctx fiber.Ctx) error {
	username := strings.TrimSpace(ctx.FormValue("username"))
	password := ctx.FormValue("password")
	next := ctx.FormValue("next")

	if blocked := checkLoginRateLimit(ctx); blocked != "" {
		return renderLoginForm(ctx, next, blocked)
	}

	ws := store.WebAccountStoreFromDB()
	account, err := ws.GetByUsername(context.Background(), username)
	if err != nil || account == nil || !webauth.CheckPassword(account.PasswordHash, password) {
		// Always burn bcrypt time on missing users to reduce timing oracle.
		if errors.Is(err, types.ErrNotFound) || account == nil {
			_, _ = webauth.HashPassword("dummy-password-for-timing")
		}
		msg := recordLoginFailure(ctx)
		return renderLoginForm(ctx, next, msg)
	}

	_ = ws.EnsureUser(context.Background(), account.UID, account.Username)

	if !account.TotpEnabled {
		if err := issuePendingSession(ctx, webauth.KindPendingEnroll, account.UID, account.Username); err != nil {
			flog.Error(fmt.Errorf("pending enroll session: %w", err))
			return renderLoginForm(ctx, next, "Internal error")
		}
		ctx.Set("HX-Redirect", "/service/web/setup/2fa")
		return nil
	}

	if err := issuePendingSession(ctx, webauth.KindPending2FA, account.UID, account.Username); err != nil {
		flog.Error(fmt.Errorf("pending 2fa session: %w", err))
		return renderLoginForm(ctx, next, "Internal error")
	}
	dest := "/service/web/login/2fa"
	if n := safeNext(next); n != "/service/web/home" {
		dest = dest + "?next=" + url.QueryEscape(n)
	}
	ctx.Set("HX-Redirect", dest)
	return nil
}

func login2FAPage(ctx fiber.Ctx) error {
	pending, err := lookupPending(ctx)
	if err != nil || pending == nil || pending.Kind != webauth.KindPending2FA {
		return ctx.Redirect().To("/service/web/login")
	}
	csrfTok, err := ensureCSRFCookie(ctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "csrf token error")
	}
	ctx.Set("Cache-Control", "no-store")
	ctx.Type("html")
	return pages.Login2FAPage(ctx.Query("next", ""), "", csrfTok).Render(context.Background(), ctx.Response().BodyWriter())
}

func renderLogin2FAForm(ctx fiber.Ctx, next, errorMsg string) error {
	csrfTok, err := ensureCSRFCookie(ctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "csrf token error")
	}
	ctx.Set("Cache-Control", "no-store")
	ctx.Type("html")
	return pages.Login2FAForm(next, errorMsg, csrfTok).Render(context.Background(), ctx.Response().BodyWriter())
}

func login2FASubmit(ctx fiber.Ctx) error {
	pending, err := lookupPending(ctx)
	if err != nil || pending == nil || pending.Kind != webauth.KindPending2FA {
		return ctx.Redirect().To("/service/web/login")
	}
	next := ctx.FormValue("next")
	code := strings.TrimSpace(ctx.FormValue("code"))
	totpLike := webauth.LooksLikeTOTPCode(code)

	if totpLike {
		if blocked := checkTOTPRateLimit(ctx); blocked != "" {
			return renderLogin2FAForm(ctx, next, blocked)
		}
	}

	ws := store.WebAccountStoreFromDB()
	account, err := ws.GetByUsername(context.Background(), pending.Username)
	if err != nil || account == nil || !account.TotpEnabled {
		clearPendingSession(ctx)
		return ctx.Redirect().To("/service/web/login")
	}

	ok, step, remaining, usedBackup, verr := verifySecondFactor(account, code)
	if verr != nil {
		return renderLogin2FAForm(ctx, next, "Internal error")
	}
	if !ok {
		if totpLike {
			return renderLogin2FAForm(ctx, next, recordTOTPFailure(ctx))
		}
		return renderLogin2FAForm(ctx, next, msgInvalidTOTP)
	}
	return completeLogin2FA(ctx, account, next, step, remaining, usedBackup)
}

func completeLogin2FA(ctx fiber.Ctx, account *gen.WebAccount, next string, step int64, remaining []string, usedBackup bool) error {
	ws := store.WebAccountStoreFromDB()
	if usedBackup {
		if err := ws.SetBackupCodeHashes(context.Background(), account.Username, remaining); err != nil {
			flog.Error(fmt.Errorf("consume backup code: %w", err))
			return renderLogin2FAForm(ctx, next, "Internal error")
		}
	} else if err := ws.SetTOTPLastStep(context.Background(), account.Username, step); err != nil {
		flog.Error(fmt.Errorf("set totp last step: %w", err))
		return renderLogin2FAForm(ctx, next, "Internal error")
	}

	totpSuccessCleanup(ctx)
	loginSuccessCleanup(ctx)
	clearPendingSession(ctx)
	if err := issueFullSession(ctx, account.UID, account.Username); err != nil {
		flog.Error(fmt.Errorf("issue full session: %w", err))
		return renderLogin2FAForm(ctx, next, "Internal error")
	}
	ctx.Set("HX-Redirect", safeNext(next))
	return nil
}

func verifySecondFactor(account *gen.WebAccount, code string) (ok bool, step int64, remaining []string, usedBackup bool, err error) {
	enc := getEncryptor()
	if enc == nil {
		return false, 0, nil, false, fmt.Errorf("encryptor not ready")
	}
	secret, serr := accountTOTPSecret(account.TotpSecretCiphertext, account.TotpSecretNonce)
	if serr == nil {
		step, ok = webauth.VerifyTOTP(secret, code, time.Now())
		if ok && step == account.TotpLastStep {
			ok = false
		}
	}
	if ok {
		return true, step, nil, false, nil
	}
	remaining, usedBackup = enc.ConsumeBackupCode(account.BackupCodeHashes, code)
	return usedBackup, 0, remaining, usedBackup, nil
}

func setupPage(ctx fiber.Ctx) error {
	if isAuthenticated(ctx) {
		return ctx.Redirect().To("/service/web/home")
	}
	ws := store.WebAccountStoreFromDB()
	n, err := ws.Count(context.Background())
	if err == nil && n > 0 {
		return ctx.Redirect().To("/service/web/login")
	}
	csrfTok, err := ensureCSRFCookie(ctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "csrf token error")
	}
	ctx.Set("Cache-Control", "no-store")
	ctx.Type("html")
	return pages.SetupPage("", csrfTok).Render(context.Background(), ctx.Response().BodyWriter())
}

func renderSetupForm(ctx fiber.Ctx, errorMsg string) error {
	csrfTok, err := ensureCSRFCookie(ctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "csrf token error")
	}
	ctx.Set("Cache-Control", "no-store")
	ctx.Type("html")
	return pages.SetupForm(errorMsg, csrfTok).Render(context.Background(), ctx.Response().BodyWriter())
}

func setupSubmit(ctx fiber.Ctx) error {
	ws := store.WebAccountStoreFromDB()
	n, err := ws.Count(context.Background())
	if err == nil && n > 0 {
		return ctx.Redirect().To("/service/web/login")
	}
	username := strings.TrimSpace(ctx.FormValue("username"))
	password := ctx.FormValue("password")
	if err := webauth.ValidatePasswordStrength(username, password); err != nil {
		return renderSetupForm(ctx, err.Error())
	}
	if username == "" {
		return renderSetupForm(ctx, "Username is required")
	}
	hash, err := webauth.HashPassword(password)
	if err != nil {
		return renderSetupForm(ctx, "Internal error")
	}
	account, err := ws.CreateFirstAccount(context.Background(), store.CreateAccountInput{
		Username:     username,
		PasswordHash: hash,
	})
	if err != nil {
		if errors.Is(err, types.ErrConflict) {
			return ctx.Redirect().To("/service/web/login")
		}
		flog.Error(fmt.Errorf("setup create account: %w", err))
		return renderSetupForm(ctx, "Could not create account")
	}
	if err := issuePendingSession(ctx, webauth.KindPendingEnroll, account.UID, account.Username); err != nil {
		flog.Error(fmt.Errorf("setup pending enroll: %w", err))
		return renderSetupForm(ctx, "Internal error")
	}
	ctx.Set("HX-Redirect", "/service/web/setup/2fa")
	return nil
}

func enroll2FAPage(ctx fiber.Ctx) error {
	pending, err := lookupPending(ctx)
	if err != nil || pending == nil || pending.Kind != webauth.KindPendingEnroll {
		return ctx.Redirect().To("/service/web/login")
	}
	ws := store.WebAccountStoreFromDB()
	account, err := ws.GetByUsername(context.Background(), pending.Username)
	if err != nil {
		clearPendingSession(ctx)
		return ctx.Redirect().To("/service/web/login")
	}
	if account.TotpEnabled {
		clearPendingSession(ctx)
		return ctx.Redirect().To("/service/web/login")
	}

	p, err := route.LookupAccessToken(context.Background(), pending.Token)
	if err != nil {
		clearPendingSession(ctx)
		return ctx.Redirect().To("/service/web/login")
	}
	paramKV := types.KV(p.Params)
	secret, err := readEnrollSecret(paramKV)
	if err != nil {
		flog.Error(fmt.Errorf("read enroll secret: %w", err))
		return fiber.NewError(fiber.StatusInternalServerError, "session error")
	}
	if secret == "" {
		secret, err = webauth.GenerateTOTPSecret()
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "totp secret error")
		}
		if err := stashEnrollSecret(ctx, pending, secret); err != nil {
			flog.Error(fmt.Errorf("stash enroll secret: %w", err))
			return fiber.NewError(fiber.StatusInternalServerError, "session error")
		}
	}
	uri := webauth.TOTPProvisioningURI(secret, pending.Username, "Flowbot")
	csrfTok, err := ensureCSRFCookie(ctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "csrf token error")
	}
	ctx.Set("Cache-Control", "no-store")
	ctx.Type("html")
	return pages.Enroll2FAPage(secret, uri, "", csrfTok).Render(context.Background(), ctx.Response().BodyWriter())
}

func enroll2FASubmit(ctx fiber.Ctx) error {
	pending, err := lookupPending(ctx)
	if err != nil || pending == nil || pending.Kind != webauth.KindPendingEnroll {
		return ctx.Redirect().To("/service/web/login")
	}
	code := strings.TrimSpace(ctx.FormValue("code"))
	if blocked := checkTOTPRateLimit(ctx); blocked != "" {
		return renderEnroll2FAError(ctx, pending, blocked)
	}

	p, err := route.LookupAccessToken(context.Background(), pending.Token)
	if err != nil {
		clearPendingSession(ctx)
		return ctx.Redirect().To("/service/web/login")
	}
	paramKV := types.KV(p.Params)
	secret, err := readEnrollSecret(paramKV)
	if err != nil || secret == "" {
		return ctx.Redirect().To("/service/web/setup/2fa")
	}
	step, ok := webauth.VerifyTOTP(secret, code, time.Now())
	if !ok {
		return renderEnroll2FAError(ctx, pending, recordTOTPFailure(ctx))
	}

	enc := getEncryptor()
	if enc == nil {
		return renderEnroll2FAError(ctx, pending, "Internal error")
	}
	ct, nonce, err := enc.Encrypt([]byte(secret))
	if err != nil {
		return renderEnroll2FAError(ctx, pending, "Internal error")
	}
	codes, hashes, err := enc.GenerateBackupCodes(webauth.BackupCodeCount)
	if err != nil {
		return renderEnroll2FAError(ctx, pending, "Internal error")
	}
	ws := store.WebAccountStoreFromDB()
	if err := ws.EnableTOTP(context.Background(), pending.Username, ct, nonce, hashes, step); err != nil {
		flog.Error(fmt.Errorf("enable totp: %w", err))
		return renderEnroll2FAError(ctx, pending, "Internal error")
	}
	totpSuccessCleanup(ctx)
	clearPendingSession(ctx)
	if err := issuePendingSession(ctx, webauth.KindPendingBackupAck, pending.UID, pending.Username); err != nil {
		flog.Error(fmt.Errorf("pending backup ack session: %w", err))
		return renderEnroll2FAError(ctx, pending, "Internal error")
	}
	csrfTok, err := ensureCSRFCookie(ctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "csrf token error")
	}
	ctx.Set("Cache-Control", "no-store")
	ctx.Type("html")
	return pages.BackupCodesPage(codes, csrfTok).Render(context.Background(), ctx.Response().BodyWriter())
}

func renderEnroll2FAError(ctx fiber.Ctx, pending *pendingSession, msg string) error {
	p, err := route.LookupAccessToken(context.Background(), pending.Token)
	secret := ""
	if err == nil {
		paramKV := types.KV(p.Params)
		secret, _ = readEnrollSecret(paramKV)
	}
	if secret == "" {
		return ctx.Redirect().To("/service/web/setup/2fa")
	}
	uri := webauth.TOTPProvisioningURI(secret, pending.Username, "Flowbot")
	csrfTok, err := ensureCSRFCookie(ctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "csrf token error")
	}
	ctx.Set("Cache-Control", "no-store")
	ctx.Type("html")
	return pages.Enroll2FAPage(secret, uri, msg, csrfTok).Render(context.Background(), ctx.Response().BodyWriter())
}

func backupCodesPage(ctx fiber.Ctx) error {
	pending, err := lookupPending(ctx)
	if err != nil || pending == nil || pending.Kind != webauth.KindPendingBackupAck {
		return ctx.Redirect().To("/service/web/login")
	}
	csrfTok, err := ensureCSRFCookie(ctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "csrf token error")
	}
	ctx.Set("Cache-Control", "no-store")
	ctx.Type("html")
	return pages.BackupCodesAckPage(csrfTok).Render(context.Background(), ctx.Response().BodyWriter())
}

func backupCodesAck(ctx fiber.Ctx) error {
	pending, err := lookupPending(ctx)
	if err != nil || pending == nil || pending.Kind != webauth.KindPendingBackupAck {
		return ctx.Redirect().To("/service/web/login")
	}
	loginSuccessCleanup(ctx)
	clearPendingSession(ctx)
	if err := issueFullSession(ctx, pending.UID, pending.Username); err != nil {
		flog.Error(fmt.Errorf("issue full session after backup ack: %w", err))
		return ctx.Redirect().To("/service/web/login")
	}
	ctx.Set("HX-Redirect", "/service/web/home")
	return nil
}

func logout(ctx fiber.Ctx) error {
	token := ctx.Cookies(webauth.CookieAccessToken)
	if token != "" {
		if err := route.DeleteAccessToken(context.Background(), token); err != nil {
			flog.Error(fmt.Errorf("failed to delete token on logout: %w", err))
		}
	}
	setAccessTokenCookie(ctx, "deleted", 0, time.Unix(0, 0))
	clearPendingSession(ctx)
	ctx.Set("HX-Redirect", "/service/web/login")
	return nil
}

// setAccessTokenCookie writes the accessToken cookie with HttpOnly, SameSite=Lax,
// and Secure controlled by modules.web.auth.cookie_secure.
func setAccessTokenCookie(ctx fiber.Ctx, value string, maxAge int, expires time.Time) {
	c := &fiber.Cookie{
		Name:     webauth.CookieAccessToken,
		Value:    value,
		HTTPOnly: true,
		SameSite: "Lax",
		Secure:   authConfig().cookieSecureEnabled(),
		Path:     "/",
		MaxAge:   maxAge,
	}
	if !expires.IsZero() {
		c.Expires = expires
	}
	ctx.Cookie(c)
}
