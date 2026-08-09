package email

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"

	"github.com/microcosm-cc/bluemonday"
)

// emailHTMLPolicy allows common safe markup in HTML bodies and strips scripts/handlers.
var emailHTMLPolicy = bluemonday.UGCPolicy()

// Send delivers a message via SMTP.
func (c *Client) Send(ctx context.Context, in SendInput) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if c.cfg.SMTPHost == "" {
		return fmt.Errorf("email: smtp_host is not configured")
	}
	if len(in.To) == 0 {
		return fmt.Errorf("email: to is required")
	}
	if strings.TrimSpace(in.Subject) == "" {
		return fmt.Errorf("email: subject is required")
	}
	if strings.TrimSpace(in.Text) == "" && strings.TrimSpace(in.HTML) == "" {
		return fmt.Errorf("email: text or html body is required")
	}

	in, err := sanitizeSendInput(in)
	if err != nil {
		return err
	}

	fromAddr := c.cfg.Username
	fromHeader, err := formatFromHeader(in.FromName, fromAddr)
	if err != nil {
		return err
	}

	msg, err := buildMIMEMessage(fromHeader, in)
	if err != nil {
		return err
	}

	recipients := append([]string{}, in.To...)
	recipients = append(recipients, in.Cc...)
	recipients = append(recipients, in.Bcc...)

	addr := net.JoinHostPort(c.cfg.SMTPHost, strconv.Itoa(c.cfg.SMTPPort))
	auth := smtp.PlainAuth("", c.cfg.Username, c.cfg.Password, c.cfg.SMTPHost)

	switch c.cfg.SMTPTLS {
	case TLSModeTLS:
		return sendSMTPTLS(ctx, addr, c.cfg.SMTPHost, auth, fromAddr, recipients, msg)
	default:
		return sendSMTPSTARTTLS(ctx, addr, c.cfg.SMTPHost, auth, fromAddr, recipients, msg)
	}
}

// sanitizeSendInput validates addresses/headers against CRLF injection and
// sanitizes body content before it is incorporated into an SMTP message.
func sanitizeSendInput(in SendInput) (SendInput, error) {
	out := SendInput{
		FromName: strings.TrimSpace(in.FromName),
		Subject:  strings.TrimSpace(in.Subject),
		Text:     sanitizeTextBody(in.Text),
		HTML:     sanitizeHTMLBody(in.HTML),
	}
	if err := rejectHeaderValue(out.FromName, "from_name"); err != nil {
		return SendInput{}, err
	}
	if err := rejectHeaderValue(out.Subject, "subject"); err != nil {
		return SendInput{}, err
	}
	if out.Subject == "" {
		return SendInput{}, fmt.Errorf("email: subject is required")
	}

	var err error
	if out.To, err = sanitizeAddresses(in.To, "to"); err != nil {
		return SendInput{}, err
	}
	if out.Cc, err = sanitizeAddresses(in.Cc, "cc"); err != nil {
		return SendInput{}, err
	}
	if out.Bcc, err = sanitizeAddresses(in.Bcc, "bcc"); err != nil {
		return SendInput{}, err
	}
	if len(out.To) == 0 {
		return SendInput{}, fmt.Errorf("email: to is required")
	}
	if strings.TrimSpace(out.Text) == "" && strings.TrimSpace(out.HTML) == "" {
		return SendInput{}, fmt.Errorf("email: text or html body is required")
	}
	return out, nil
}

func sanitizeAddresses(addrs []string, field string) ([]string, error) {
	if len(addrs) == 0 {
		return nil, nil
	}
	out := make([]string, 0, len(addrs))
	for _, addr := range addrs {
		parsed, err := sanitizeAddress(addr)
		if err != nil {
			return nil, fmt.Errorf("email: invalid %s: %w", field, err)
		}
		out = append(out, parsed)
	}
	return out, nil
}

func sanitizeAddress(addr string) (string, error) {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return "", fmt.Errorf("empty address")
	}
	if err := rejectHeaderValue(addr, "address"); err != nil {
		return "", err
	}
	parsed, err := mail.ParseAddress(addr)
	if err != nil {
		return "", err
	}
	if err := rejectHeaderValue(parsed.Address, "address"); err != nil {
		return "", err
	}
	if parsed.Name != "" {
		if err := rejectHeaderValue(parsed.Name, "address name"); err != nil {
			return "", err
		}
		return (&mail.Address{Name: parsed.Name, Address: parsed.Address}).String(), nil
	}
	return parsed.Address, nil
}

func formatFromHeader(name, addr string) (string, error) {
	if err := rejectHeaderValue(addr, "from"); err != nil {
		return "", err
	}
	name = strings.TrimSpace(name)
	if err := rejectHeaderValue(name, "from_name"); err != nil {
		return "", err
	}
	return (&mail.Address{Name: name, Address: addr}).String(), nil
}

func rejectHeaderValue(v, field string) error {
	if strings.ContainsAny(v, "\r\n\x00") {
		return fmt.Errorf("email: %s contains invalid control characters", field)
	}
	return nil
}

func sanitizeTextBody(s string) string {
	// Plain-text bodies may contain newlines; strip NUL only.
	return strings.ReplaceAll(s, "\x00", "")
}

func sanitizeHTMLBody(s string) string {
	if strings.TrimSpace(s) == "" {
		return ""
	}
	return emailHTMLPolicy.Sanitize(strings.ReplaceAll(s, "\x00", ""))
}

func buildMIMEMessage(fromHeader string, in SendInput) ([]byte, error) {
	if err := rejectHeaderValue(fromHeader, "from"); err != nil {
		return nil, err
	}
	if err := rejectHeaderValue(in.Subject, "subject"); err != nil {
		return nil, err
	}
	for _, addr := range in.To {
		if err := rejectHeaderValue(addr, "to"); err != nil {
			return nil, err
		}
	}
	for _, addr := range in.Cc {
		if err := rejectHeaderValue(addr, "cc"); err != nil {
			return nil, err
		}
	}

	var b strings.Builder
	writeHeader := func(k, v string) {
		_, _ = b.WriteString(k)
		_, _ = b.WriteString(": ")
		_, _ = b.WriteString(v)
		_, _ = b.WriteString("\r\n")
	}
	writeHeader("From", fromHeader)
	writeHeader("To", strings.Join(in.To, ", "))
	if len(in.Cc) > 0 {
		writeHeader("Cc", strings.Join(in.Cc, ", "))
	}
	writeHeader("Subject", in.Subject)
	writeHeader("MIME-Version", "1.0")

	hasText := strings.TrimSpace(in.Text) != ""
	hasHTML := strings.TrimSpace(in.HTML) != ""
	switch {
	case hasText && hasHTML:
		boundary, err := randomMIMEBoundary()
		if err != nil {
			return nil, err
		}
		writeHeader("Content-Type", `multipart/alternative; boundary="`+boundary+`"`)
		_, _ = b.WriteString("\r\n")
		_, _ = b.WriteString("--" + boundary + "\r\n")
		_, _ = b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
		_, _ = b.WriteString(in.Text)
		_, _ = b.WriteString("\r\n--" + boundary + "\r\n")
		_, _ = b.WriteString("Content-Type: text/html; charset=UTF-8\r\n\r\n")
		_, _ = b.WriteString(in.HTML)
		_, _ = b.WriteString("\r\n--" + boundary + "--\r\n")
	case hasHTML:
		writeHeader("Content-Type", "text/html; charset=UTF-8")
		_, _ = b.WriteString("\r\n")
		_, _ = b.WriteString(in.HTML)
	default:
		writeHeader("Content-Type", "text/plain; charset=UTF-8")
		_, _ = b.WriteString("\r\n")
		_, _ = b.WriteString(in.Text)
	}
	return []byte(b.String()), nil
}

func randomMIMEBoundary() (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", fmt.Errorf("email: mime boundary: %w", err)
	}
	return "flowbot-" + hex.EncodeToString(buf[:]), nil
}

func sendSMTPTLS(ctx context.Context, addr, host string, auth smtp.Auth, from string, to []string, msg []byte) error {
	dialer := &net.Dialer{}
	conn, err := tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12})
	if err != nil {
		return fmt.Errorf("email: smtp tls dial: %w", err)
	}
	defer func() { _ = conn.Close() }()
	if err := ctx.Err(); err != nil {
		return err
	}
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("email: smtp client: %w", err)
	}
	defer func() { _ = client.Close() }()
	return smtpSend(client, auth, from, to, msg)
}

func sendSMTPSTARTTLS(ctx context.Context, addr, host string, auth smtp.Auth, from string, to []string, msg []byte) error {
	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("email: smtp dial: %w", err)
	}
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("email: smtp client: %w", err)
	}
	defer func() { _ = client.Close() }()
	if ok, _ := client.Extension("STARTTLS"); !ok {
		return fmt.Errorf("email: smtp server does not support STARTTLS")
	}
	if err := client.StartTLS(&tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}); err != nil {
		return fmt.Errorf("email: smtp starttls: %w", err)
	}
	return smtpSend(client, auth, from, to, msg)
}

func (c *Client) checkSMTP(ctx context.Context) error {
	addr := net.JoinHostPort(c.cfg.SMTPHost, strconv.Itoa(c.cfg.SMTPPort))
	auth := smtp.PlainAuth("", c.cfg.Username, c.cfg.Password, c.cfg.SMTPHost)
	var client *smtp.Client
	var err error
	switch c.cfg.SMTPTLS {
	case TLSModeTLS:
		dialer := &net.Dialer{}
		conn, dialErr := tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{ServerName: c.cfg.SMTPHost, MinVersion: tls.VersionTLS12})
		if dialErr != nil {
			return fmt.Errorf("email: smtp tls dial: %w", dialErr)
		}
		client, err = smtp.NewClient(conn, c.cfg.SMTPHost)
	default:
		dialer := &net.Dialer{}
		conn, dialErr := dialer.DialContext(ctx, "tcp", addr)
		if dialErr != nil {
			return fmt.Errorf("email: smtp dial: %w", dialErr)
		}
		client, err = smtp.NewClient(conn, c.cfg.SMTPHost)
		if err == nil {
			if ok, _ := client.Extension("STARTTLS"); !ok {
				_ = client.Close()
				return fmt.Errorf("email: smtp server does not support STARTTLS")
			}
			if err = client.StartTLS(&tls.Config{ServerName: c.cfg.SMTPHost, MinVersion: tls.VersionTLS12}); err != nil {
				_ = client.Close()
				return fmt.Errorf("email: smtp starttls: %w", err)
			}
		}
	}
	if err != nil {
		return fmt.Errorf("email: smtp client: %w", err)
	}
	defer func() { _ = client.Close() }()
	if err := client.Hello("localhost"); err != nil {
		return fmt.Errorf("email: smtp hello: %w", err)
	}
	if ok, _ := client.Extension("AUTH"); ok {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("email: smtp auth: %w", err)
		}
	}
	if err := client.Quit(); err != nil {
		return fmt.Errorf("email: smtp quit: %w", err)
	}
	return nil
}

func smtpSend(client *smtp.Client, auth smtp.Auth, from string, to []string, msg []byte) error {
	if auth != nil {
		if ok, _ := client.Extension("AUTH"); ok {
			if err := client.Auth(auth); err != nil {
				return fmt.Errorf("email: smtp auth: %w", err)
			}
		}
	}
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("email: smtp mail: %w", err)
	}
	for _, addr := range to {
		if err := client.Rcpt(addr); err != nil {
			return fmt.Errorf("email: smtp rcpt %s: %w", addr, err)
		}
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("email: smtp data: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		_ = w.Close()
		return fmt.Errorf("email: smtp write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("email: smtp close: %w", err)
	}
	if err := client.Quit(); err != nil {
		return fmt.Errorf("email: smtp quit: %w", err)
	}
	return nil
}
