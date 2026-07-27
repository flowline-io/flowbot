package web

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/views/pages"
	"github.com/flowline-io/flowbot/pkg/webauth"
)

var accountWebserviceRules = []webservice.Rule{
	webservice.Get("/account", accountSecurityPage, route.WithNotAuth()),
	webservice.Post("/account/password", accountChangePassword, route.WithNotAuth()),
	webservice.Post("/account/backup-codes", accountRegenBackupCodes, route.WithNotAuth()),
}

func accountSecurityPage(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	return renderAccountSecurity(ctx, ctx.Query("flash", ""), "")
}

func renderAccountSecurity(ctx fiber.Ctx, flash, errorMsg string) error {
	rc := route.GetRequestContext(ctx)
	username := accountUsername(rc)
	backupRemaining := 0
	totpEnabled := false
	ws := store.WebAccountStoreFromDB()
	if account, err := ws.GetByUsername(context.Background(), username); err == nil && account != nil {
		backupRemaining = len(account.BackupCodeHashes)
		totpEnabled = account.TotpEnabled
	}
	csrfTok, err := ensureCSRFCookie(ctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "csrf token error")
	}
	ctx.Set("Cache-Control", "no-store")
	ctx.Type("html")
	return pages.AccountSecurityPage(username, totpEnabled, backupRemaining, flash, errorMsg, csrfTok).
		Render(context.Background(), ctx.Response().BodyWriter())
}

func accountChangePassword(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	rc := route.GetRequestContext(ctx)
	username := accountUsername(rc)
	current := ctx.FormValue("current_password")
	newPass := ctx.FormValue("new_password")
	if err := webauth.ValidatePasswordStrength(username, newPass); err != nil {
		return renderAccountSecurity(ctx, "", err.Error())
	}
	ws := store.WebAccountStoreFromDB()
	account, err := ws.GetByUsername(context.Background(), username)
	if err != nil || account == nil || !webauth.CheckPassword(account.PasswordHash, current) {
		return renderAccountSecurity(ctx, "", "Current password is incorrect")
	}
	hash, err := webauth.HashPassword(newPass)
	if err != nil {
		return renderAccountSecurity(ctx, "", "Internal error")
	}
	if err := ws.UpdatePasswordHash(context.Background(), username, hash); err != nil {
		flog.Error(fmt.Errorf("update password: %w", err))
		return renderAccountSecurity(ctx, "", "Internal error")
	}
	if _, err := ws.DeleteWebSessionsForUID(context.Background(), account.UID); err != nil {
		flog.Error(fmt.Errorf("revoke sessions after password change: %w", err))
	}
	setAccessTokenCookie(ctx, "deleted", 0, time.Unix(0, 0))
	clearPendingSession(ctx)
	ctx.Redirect().To("/service/web/login")
	return nil
}

func accountRegenBackupCodes(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	rc := route.GetRequestContext(ctx)
	username := accountUsername(rc)
	password := ctx.FormValue("password")
	ws := store.WebAccountStoreFromDB()
	account, err := ws.GetByUsername(context.Background(), username)
	if err != nil || account == nil || !webauth.CheckPassword(account.PasswordHash, password) {
		return renderAccountSecurity(ctx, "", "Password is incorrect")
	}
	if !account.TotpEnabled {
		return renderAccountSecurity(ctx, "", "Enable two-factor authentication first")
	}
	enc := getEncryptor()
	if enc == nil {
		return renderAccountSecurity(ctx, "", "Internal error")
	}
	codes, hashes, err := enc.GenerateBackupCodes(webauth.BackupCodeCount)
	if err != nil {
		return renderAccountSecurity(ctx, "", "Internal error")
	}
	if err := ws.SetBackupCodeHashes(context.Background(), username, hashes); err != nil {
		flog.Error(fmt.Errorf("regen backup codes: %w", err))
		return renderAccountSecurity(ctx, "", "Internal error")
	}
	if _, err := ws.DeleteWebSessionsForUID(context.Background(), account.UID); err != nil {
		flog.Error(fmt.Errorf("revoke sessions after backup regen: %w", err))
	}
	if err := issueFullSession(ctx, account.UID, account.Username); err != nil {
		flog.Error(fmt.Errorf("reissue session after backup regen: %w", err))
		ctx.Redirect().To("/service/web/login")
		return nil
	}
	ctx.Set("Cache-Control", "no-store")
	ctx.Type("html")
	return pages.RegeneratedBackupCodesPage(codes).Render(context.Background(), ctx.Response().BodyWriter())
}

func accountUsername(rc *route.RequestContext) string {
	if rc == nil {
		return ""
	}
	if u, ok := rc.Param.String("username"); ok && u != "" {
		return u
	}
	return strings.TrimPrefix(rc.UID.String(), "user-")
}
