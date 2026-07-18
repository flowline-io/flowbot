package beszel

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync"

	"github.com/bytedance/sonic"
	"resty.dev/v3"

	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/utils"
)

const (
	ID          = "beszel"
	EndpointKey = "endpoint"
	TokenKey    = "token"
	EmailKey    = "email"
	PasswordKey = "password"
)

// Beszel is an HTTP client for the Beszel hub (PocketBase).
type Beszel struct {
	c        *resty.Client
	email    string
	password string
	mu       sync.Mutex
	authed   bool
}

// GetClient builds a Beszel client from vendors.beszel config.
// Returns nil when endpoint is not configured.
func GetClient() *Beszel {
	endpoint, _ := providers.GetConfig(ID, EndpointKey)
	token, _ := providers.GetConfig(ID, TokenKey)
	email, _ := providers.GetConfig(ID, EmailKey)
	password, _ := providers.GetConfig(ID, PasswordKey)
	if endpoint.String() == "" {
		return nil
	}
	return NewBeszel(endpoint.String(), token.String(), email.String(), password.String())
}

// NewBeszel creates a Beszel client. Prefer token; otherwise email+password are used
// to authenticate on demand. Returns nil when endpoint is empty.
func NewBeszel(endpoint, token, email, password string) *Beszel {
	if endpoint == "" {
		return nil
	}
	v := &Beszel{email: email, password: password}
	v.c = utils.DefaultRestyClient()
	v.c.SetBaseURL(endpoint)
	if token != "" {
		v.c.SetHeader("Authorization", token)
		v.authed = true
	}
	return v
}

func (b *Beszel) ensureAuth(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.authed {
		return nil
	}
	if b.email == "" || b.password == "" {
		return fmt.Errorf("beszel: token or email/password required")
	}
	resp, err := b.c.R().
		SetContext(ctx).
		SetBody(map[string]string{"identity": b.email, "password": b.password}).
		Post("/api/collections/users/auth-with-password")
	if err != nil {
		return fmt.Errorf("beszel auth: %w", err)
	}
	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		return fmt.Errorf("beszel auth: status %d", resp.StatusCode())
	}
	var result AuthResponse
	if err := sonic.Unmarshal(resp.Bytes(), &result); err != nil {
		return fmt.Errorf("beszel auth decode: %w", err)
	}
	if result.Token == "" {
		return fmt.Errorf("beszel auth: empty token")
	}
	b.c.SetHeader("Authorization", result.Token)
	b.authed = true
	return nil
}

// ListSystems returns monitored systems (first page).
func (b *Beszel) ListSystems(ctx context.Context) (*SystemList, error) {
	if err := b.ensureAuth(ctx); err != nil {
		return nil, err
	}
	resp, err := b.c.R().SetContext(ctx).Get("/api/collections/systems/records")
	if err != nil {
		return nil, fmt.Errorf("beszel list systems: %w", err)
	}
	if err := checkStatus(resp, "list systems"); err != nil {
		return nil, err
	}
	var result SystemList
	if err := sonic.Unmarshal(resp.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("beszel list systems decode: %w", err)
	}
	return &result, nil
}

// GetSystem returns a single system by ID.
func (b *Beszel) GetSystem(ctx context.Context, id string) (*System, error) {
	if id == "" {
		return nil, fmt.Errorf("beszel: system id required")
	}
	if err := b.ensureAuth(ctx); err != nil {
		return nil, err
	}
	path := "/api/collections/systems/records/" + url.PathEscape(id)
	resp, err := b.c.R().SetContext(ctx).Get(path)
	if err != nil {
		return nil, fmt.Errorf("beszel get system: %w", err)
	}
	if err := checkStatus(resp, "get system"); err != nil {
		return nil, err
	}
	var result System
	if err := sonic.Unmarshal(resp.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("beszel get system decode: %w", err)
	}
	return &result, nil
}

func checkStatus(resp *resty.Response, op string) error {
	if resp == nil {
		return fmt.Errorf("beszel %s: nil response", op)
	}
	code := resp.StatusCode()
	if code >= http.StatusOK && code < http.StatusMultipleChoices {
		return nil
	}
	return fmt.Errorf("beszel %s: status %d", op, code)
}
