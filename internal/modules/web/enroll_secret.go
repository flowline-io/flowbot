package web

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
)

const (
	paramEnrollSecretCT    = "enroll_secret_ct"
	paramEnrollSecretNonce = "enroll_secret_nonce"
)

func stashEnrollSecret(_ fiber.Ctx, pending *pendingSession, secret string) error {
	enc := getEncryptor()
	if enc == nil {
		return fmt.Errorf("encryptor not ready")
	}
	ct, nonce, err := enc.Encrypt([]byte(secret))
	if err != nil {
		return err
	}
	p, err := route.LookupAccessToken(context.Background(), pending.Token)
	if err != nil {
		return err
	}
	params := types.KV(p.Params)
	params[paramEnrollSecretCT] = base64.RawStdEncoding.EncodeToString(ct)
	params[paramEnrollSecretNonce] = base64.RawStdEncoding.EncodeToString(nonce)
	delete(params, "enroll_secret")
	return store.ModuleDataStoreFromDB().ParameterSet(context.Background(), auth.HashToken(pending.Token), params, p.ExpiredAt)
}

func readEnrollSecret(params types.KV) (string, error) {
	enc := getEncryptor()
	if enc == nil {
		return "", fmt.Errorf("encryptor not ready")
	}
	ctB64, _ := params.String(paramEnrollSecretCT)
	nonceB64, _ := params.String(paramEnrollSecretNonce)
	if ctB64 == "" || nonceB64 == "" {
		return "", nil
	}
	ct, err := base64.RawStdEncoding.DecodeString(ctB64)
	if err != nil {
		return "", fmt.Errorf("decode enroll secret ct: %w", err)
	}
	nonce, err := base64.RawStdEncoding.DecodeString(nonceB64)
	if err != nil {
		return "", fmt.Errorf("decode enroll secret nonce: %w", err)
	}
	pt, err := enc.Decrypt(ct, nonce)
	if err != nil {
		return "", err
	}
	return string(pt), nil
}
