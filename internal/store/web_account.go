package store

import (
	"context"
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/parameter"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/user"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/webaccount"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/webauth"
)

// WebAccountStore persists web UI login accounts.
type WebAccountStore struct {
	client *gen.Client
}

// NewWebAccountStore creates a WebAccountStore with the given ent client.
func NewWebAccountStore(client *gen.Client) *WebAccountStore {
	return &WebAccountStore{client: client}
}

// WebAccountStoreFromDB returns a WebAccountStore using the global database client.
func WebAccountStoreFromDB() *WebAccountStore {
	return NewWebAccountStore(ClientFromDB())
}

func (s *WebAccountStore) ready() bool {
	return s != nil && s.client != nil
}

// Client returns the underlying ent client.
func (s *WebAccountStore) Client() *gen.Client {
	if s == nil {
		return nil
	}
	return s.client
}

// Count returns the number of web accounts.
func (s *WebAccountStore) Count(ctx context.Context) (int, error) {
	if !s.ready() {
		return 0, fmt.Errorf("web account store not available")
	}
	n, err := s.client.WebAccount.Query().Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("web account: count: %w", err)
	}
	return n, nil
}

// GetByUsername returns the account for username.
func (s *WebAccountStore) GetByUsername(ctx context.Context, username string) (*gen.WebAccount, error) {
	if !s.ready() {
		return nil, fmt.Errorf("web account store not available")
	}
	row, err := s.client.WebAccount.Query().Where(webaccount.UsernameEQ(username)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("web account: get by username: %w", err)
	}
	return row, nil
}

// GetByUID returns the account for uid.
func (s *WebAccountStore) GetByUID(ctx context.Context, uid string) (*gen.WebAccount, error) {
	if !s.ready() {
		return nil, fmt.Errorf("web account store not available")
	}
	row, err := s.client.WebAccount.Query().Where(webaccount.UIDEQ(uid)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("web account: get by uid: %w", err)
	}
	return row, nil
}

// CreateAccountInput holds fields for creating a web account and ensuring a users row.
type CreateAccountInput struct {
	Username     string
	PasswordHash string
}

// CreateFirstAccount creates the first web account when none exist (setup / migration).
// It runs in a transaction: COUNT must be 0, then insert web_accounts + users.
func (s *WebAccountStore) CreateFirstAccount(ctx context.Context, in CreateAccountInput) (*gen.WebAccount, error) {
	if !s.ready() {
		return nil, fmt.Errorf("web account store not available")
	}
	if in.Username == "" || in.PasswordHash == "" {
		return nil, fmt.Errorf("web account: username and password hash required")
	}
	uid := webauth.UIDForUsername(in.Username)
	var created *gen.WebAccount
	err := withTx(ctx, s.client, func(tx *gen.Tx) error {
		n, err := tx.WebAccount.Query().Count(ctx)
		if err != nil {
			return fmt.Errorf("count: %w", err)
		}
		if n > 0 {
			return fmt.Errorf("%w: web account already exists", types.ErrConflict)
		}
		row, err := tx.WebAccount.Create().
			SetUsername(in.Username).
			SetUID(uid).
			SetPasswordHash(in.PasswordHash).
			SetTotpEnabled(false).
			SetBackupCodeHashes([]string{}).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("create web account: %w", err)
		}
		if err := ensureUserTx(ctx, tx, uid, in.Username); err != nil {
			return err
		}
		created = row
		return nil
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

// EnsureUser creates a users row for uid if missing.
func (s *WebAccountStore) EnsureUser(ctx context.Context, uid, username string) error {
	if !s.ready() {
		return fmt.Errorf("web account store not available")
	}
	return ensureUser(ctx, s.client, uid, username)
}

func ensureUser(ctx context.Context, client *gen.Client, uid, username string) error {
	_, err := client.User.Query().Where(user.FlagEQ(uid)).Only(ctx)
	if err == nil {
		return nil
	}
	if !gen.IsNotFound(err) {
		return fmt.Errorf("web account: lookup user: %w", err)
	}
	_, err = client.User.Create().
		SetFlag(uid).
		SetName(username).
		SetTags("").
		SetState(int(schema.UserActive)).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("web account: create user: %w", err)
	}
	return nil
}

func ensureUserTx(ctx context.Context, tx *gen.Tx, uid, username string) error {
	_, err := tx.User.Query().Where(user.FlagEQ(uid)).Only(ctx)
	if err == nil {
		return nil
	}
	if !gen.IsNotFound(err) {
		return fmt.Errorf("lookup user: %w", err)
	}
	_, err = tx.User.Create().
		SetFlag(uid).
		SetName(username).
		SetTags("").
		SetState(int(schema.UserActive)).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

// UpdatePasswordHash sets a new password hash.
func (s *WebAccountStore) UpdatePasswordHash(ctx context.Context, username, hash string) error {
	if !s.ready() {
		return fmt.Errorf("web account store not available")
	}
	n, err := s.client.WebAccount.Update().
		Where(webaccount.UsernameEQ(username)).
		SetPasswordHash(hash).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("web account: update password: %w", err)
	}
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// EnableTOTP stores encrypted secret, backup hashes, marks enabled, and records the enroll step.
func (s *WebAccountStore) EnableTOTP(ctx context.Context, username string, ciphertext, nonce []byte, backupHashes []string, lastStep int64) error {
	if !s.ready() {
		return fmt.Errorf("web account store not available")
	}
	n, err := s.client.WebAccount.Update().
		Where(webaccount.UsernameEQ(username)).
		SetTotpSecretCiphertext(ciphertext).
		SetTotpSecretNonce(nonce).
		SetTotpEnabled(true).
		SetBackupCodeHashes(backupHashes).
		SetTotpLastStep(lastStep).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("web account: enable totp: %w", err)
	}
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// SetTOTPLastStep records the last accepted TOTP time step (replay protection).
func (s *WebAccountStore) SetTOTPLastStep(ctx context.Context, username string, step int64) error {
	if !s.ready() {
		return fmt.Errorf("web account store not available")
	}
	n, err := s.client.WebAccount.Update().
		Where(webaccount.UsernameEQ(username)).
		SetTotpLastStep(step).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("web account: set totp last step: %w", err)
	}
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// SetBackupCodeHashes replaces backup code hashes.
func (s *WebAccountStore) SetBackupCodeHashes(ctx context.Context, username string, hashes []string) error {
	if !s.ready() {
		return fmt.Errorf("web account store not available")
	}
	if hashes == nil {
		hashes = []string{}
	}
	n, err := s.client.WebAccount.Update().
		Where(webaccount.UsernameEQ(username)).
		SetBackupCodeHashes(hashes).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("web account: set backup codes: %w", err)
	}
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// ResetTOTP clears TOTP secret, disables 2FA, and clears backup codes.
func (s *WebAccountStore) ResetTOTP(ctx context.Context, username string) error {
	if !s.ready() {
		return fmt.Errorf("web account store not available")
	}
	n, err := s.client.WebAccount.Update().
		Where(webaccount.UsernameEQ(username)).
		ClearTotpSecretCiphertext().
		ClearTotpSecretNonce().
		SetTotpEnabled(false).
		SetTotpLastStep(0).
		SetBackupCodeHashes([]string{}).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("web account: reset totp: %w", err)
	}
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// RevokeLegacyWebSessions deletes web sessions that are not full authenticated sessions.
// This clears pre-2FA legacy cookies (missing kind) and any pending sessions on startup.
func (s *WebAccountStore) RevokeLegacyWebSessions(ctx context.Context) (int, error) {
	if !s.ready() {
		return 0, fmt.Errorf("web account store not available")
	}
	rows, err := s.client.Parameter.Query().All(ctx)
	if err != nil {
		return 0, fmt.Errorf("web account: list parameters: %w", err)
	}
	deleted := 0
	for _, p := range rows {
		params := types.KV(p.Params)
		topic, _ := params.String("topic")
		if topic != "web" {
			continue
		}
		kind, _ := params.String("kind")
		if kind == webauth.KindFull {
			continue
		}
		if _, err := s.client.Parameter.Delete().Where(parameter.IDEQ(p.ID)).Exec(ctx); err != nil {
			return deleted, fmt.Errorf("web account: delete legacy session: %w", err)
		}
		deleted++
	}
	return deleted, nil
}

// DeleteWebSessionsForUID removes parameter rows for web sessions belonging to uid.
func (s *WebAccountStore) DeleteWebSessionsForUID(ctx context.Context, uid string) (int, error) {
	if !s.ready() {
		return 0, fmt.Errorf("web account store not available")
	}
	rows, err := s.client.Parameter.Query().All(ctx)
	if err != nil {
		return 0, fmt.Errorf("web account: list parameters: %w", err)
	}
	deleted := 0
	for _, p := range rows {
		params := types.KV(p.Params)
		uidStr, _ := params.String("uid")
		if uidStr != uid {
			continue
		}
		topic, _ := params.String("topic")
		kind, _ := params.String("kind")
		if topic != "web" && kind == "" {
			continue
		}
		if _, err := s.client.Parameter.Delete().Where(parameter.IDEQ(p.ID)).Exec(ctx); err != nil {
			return deleted, fmt.Errorf("web account: delete session: %w", err)
		}
		deleted++
	}
	return deleted, nil
}

func withTx(ctx context.Context, client *gen.Client, fn func(tx *gen.Tx) error) error {
	tx, err := client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("web account: begin tx: %w", err)
	}
	if err := fn(tx); err != nil {
		if rerr := tx.Rollback(); rerr != nil {
			return fmt.Errorf("%w (rollback: %v)", err, rerr)
		}
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("web account: commit: %w", err)
	}
	return nil
}
