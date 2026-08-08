// Package email implements SMTP send and IMAP read for the email provider.
package email

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/flowline-io/flowbot/pkg/providers"
)

// Client is an SMTP/IMAP email client backed by YAML config.
type Client struct {
	cfg Config
}

// GetClient returns an email client from YAML config.
// Returns nil, nil when username or hosts are not configured.
func GetClient() (*Client, error) {
	cfg, ok := loadConfig()
	if !ok {
		return nil, nil
	}
	return NewClient(cfg)
}

// NewClient validates config and returns a client.
func NewClient(cfg Config) (*Client, error) {
	if cfg.Username == "" || cfg.Password == "" {
		return nil, fmt.Errorf("email: username and password are required")
	}
	if cfg.SMTPHost == "" && cfg.IMAPHost == "" {
		return nil, fmt.Errorf("email: smtp_host or imap_host is required")
	}
	if cfg.Mailbox == "" {
		cfg.Mailbox = "INBOX"
	}
	if cfg.SMTPPort == 0 {
		cfg.SMTPPort = 465
	}
	if cfg.IMAPPort == 0 {
		cfg.IMAPPort = 993
	}
	cfg.SMTPTLS = ResolveTLSMode(cfg.SMTPTLS, cfg.SMTPPort, 465)
	cfg.IMAPTLS = ResolveTLSMode(cfg.IMAPTLS, cfg.IMAPPort, 993)
	return &Client{cfg: cfg}, nil
}

// Config returns a copy of the client configuration.
func (c *Client) Config() Config {
	return c.cfg
}

func loadConfig() (Config, bool) {
	username, _ := providers.GetConfig(ID, UsernameKey)
	password, _ := providers.GetConfig(ID, PasswordKey)
	smtpHost, _ := providers.GetConfig(ID, SMTPHostKey)
	imapHost, _ := providers.GetConfig(ID, IMAPHostKey)
	if username.String() == "" || password.String() == "" {
		return Config{}, false
	}
	if smtpHost.String() == "" && imapHost.String() == "" {
		return Config{}, false
	}

	cfg := Config{
		Username: username.String(),
		Password: password.String(),
		SMTPHost: smtpHost.String(),
		IMAPHost: imapHost.String(),
	}

	if v, _ := providers.GetConfig(ID, SMTPPortKey); v.String() != "" {
		if n, err := strconv.Atoi(v.String()); err == nil {
			cfg.SMTPPort = n
		}
	}
	if v, _ := providers.GetConfig(ID, IMAPPortKey); v.String() != "" {
		if n, err := strconv.Atoi(v.String()); err == nil {
			cfg.IMAPPort = n
		}
	}
	if v, _ := providers.GetConfig(ID, SMTPTLSKey); v.String() != "" {
		cfg.SMTPTLS = v.String()
	}
	if v, _ := providers.GetConfig(ID, IMAPTLSKey); v.String() != "" {
		cfg.IMAPTLS = v.String()
	}
	if v, _ := providers.GetConfig(ID, MailboxKey); v.String() != "" {
		cfg.Mailbox = v.String()
	}
	cfg.UnseenOnly = true
	if v, _ := providers.GetConfig(ID, UnseenOnlyKey); v.String() != "" {
		cfg.UnseenOnly = parseBoolDefault(v.String(), true)
	}
	if v, _ := providers.GetConfig(ID, MarkSeenAfterEmitKey); v.String() != "" {
		cfg.MarkSeenAfterEmit = parseBoolDefault(v.String(), false)
	}
	return cfg, true
}

func parseBoolDefault(s string, def bool) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	default:
		return def
	}
}
