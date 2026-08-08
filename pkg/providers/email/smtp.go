package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
)

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

	fromAddr := c.cfg.Username
	fromHeader := fromAddr
	if name := strings.TrimSpace(in.FromName); name != "" {
		fromHeader = fmt.Sprintf("%s <%s>", name, fromAddr)
	}

	msg, err := buildMIMEMessage(fromHeader, in)
	if err != nil {
		return err
	}

	recipients := append([]string{}, in.To...)
	recipients = append(recipients, in.Cc...)
	recipients = append(recipients, in.Bcc...)

	addr := fmt.Sprintf("%s:%d", c.cfg.SMTPHost, c.cfg.SMTPPort)
	auth := smtp.PlainAuth("", c.cfg.Username, c.cfg.Password, c.cfg.SMTPHost)

	switch c.cfg.SMTPTLS {
	case TLSModeTLS:
		return sendSMTPTLS(ctx, addr, c.cfg.SMTPHost, auth, fromAddr, recipients, msg)
	default:
		return sendSMTPSTARTTLS(ctx, addr, c.cfg.SMTPHost, auth, fromAddr, recipients, msg)
	}
}

func buildMIMEMessage(fromHeader string, in SendInput) ([]byte, error) {
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
		boundary := "flowbot-email-boundary"
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
	addr := fmt.Sprintf("%s:%d", c.cfg.SMTPHost, c.cfg.SMTPPort)
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
