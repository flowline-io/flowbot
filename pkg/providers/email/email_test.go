package email

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeDecodeMessageID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		mailbox     string
		uidValidity uint32
		uid         uint32
		legacy      string
		wantErr     bool
	}{
		{name: "opaque round trip", mailbox: "INBOX", uidValidity: 42, uid: 7},
		{name: "legacy form", legacy: "42:7", uidValidity: 42, uid: 7},
		{name: "invalid", legacy: "42", wantErr: true},
		{name: "invalid uid", legacy: "1:x", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.wantErr {
				_, err := DecodeMessageID(tt.legacy)
				require.Error(t, err)
				return
			}
			if tt.legacy != "" {
				ref, err := DecodeMessageID(tt.legacy)
				require.NoError(t, err)
				assert.Equal(t, tt.uidValidity, ref.UIDValidity)
				assert.Equal(t, tt.uid, ref.UID)
				return
			}
			id := EncodeMessageID(tt.mailbox, tt.uidValidity, tt.uid)
			ref, err := DecodeMessageID(id)
			require.NoError(t, err)
			assert.Equal(t, tt.mailbox, ref.Mailbox)
			assert.Equal(t, tt.uidValidity, ref.UIDValidity)
			assert.Equal(t, tt.uid, ref.UID)
			assert.NotContains(t, id, ":")
		})
	}
}

func TestResolveTLSMode(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		mode string
		port int
		want string
	}{
		{name: "explicit tls", mode: "tls", port: 587, want: TLSModeTLS},
		{name: "explicit starttls", mode: "STARTTLS", port: 465, want: TLSModeSTARTTLS},
		{name: "default 465 tls", mode: "", port: 465, want: TLSModeTLS},
		{name: "default 993 tls", mode: "", port: 993, want: TLSModeTLS},
		{name: "default 587 starttls", mode: "", port: 587, want: TLSModeSTARTTLS},
		{name: "default 143 starttls", mode: "", port: 143, want: TLSModeSTARTTLS},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ResolveTLSMode(tt.mode, tt.port, 465, 993)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildMIMEMessage(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		in         SendInput
		wantSubstr []string
		wantAbsent []string
		wantErr    bool
	}{
		{
			name: "html only",
			in:   SendInput{To: []string{"a@b.c"}, Subject: "Hi", HTML: "<p>x</p>"},
			wantSubstr: []string{
				"To: a@b.c",
				"Subject: Hi",
				"text/html",
				"quoted-printable",
				"<p>x</p>",
			},
			wantAbsent: []string{
				"&lt;p&gt;",
			},
		},
		{
			name: "multipart alternative",
			in:   SendInput{To: []string{"a@b.c"}, Cc: []string{"c@d.e"}, Subject: "Hi", Text: "plain", HTML: "<b>x</b>"},
			wantSubstr: []string{
				"Cc: c@d.e",
				"multipart/alternative",
				"--flowbot-",
				"quoted-printable",
				"plain",
				"<b>x</b>",
			},
		},
		{
			name: "utf-8 subject encoded",
			in:   SendInput{To: []string{"a@b.c"}, Subject: "你好", Text: "body"},
			wantSubstr: []string{
				"=?utf-8?q?",
			},
			wantAbsent: []string{
				"Subject: 你好",
			},
		},
		{
			name: "text equals-sign quoted-printable",
			in:   SendInput{To: []string{"a@b.c"}, Subject: "Hi", Text: "a=b"},
			wantSubstr: []string{
				"a=3Db",
			},
		},
		{
			name: "text control chars stripped",
			in:   SendInput{To: []string{"a@b.c"}, Subject: "Hi", Text: "ok\x01\x00there"},
			wantSubstr: []string{
				"okthere",
			},
			wantAbsent: []string{
				"\x01",
				"\x00",
			},
		},
		{
			name:    "subject CRLF rejected",
			in:      SendInput{To: []string{"a@b.c"}, Subject: "Hi\r\nBcc: evil@x.y", Text: "body"},
			wantErr: true,
		},
		{
			name:    "to CRLF rejected",
			in:      SendInput{To: []string{"a@b.c\r\nBcc: evil@x.y"}, Subject: "Hi", Text: "body"},
			wantErr: true,
		},
		{
			name:    "from name CRLF rejected",
			in:      SendInput{To: []string{"a@b.c"}, Subject: "Hi", Text: "body", FromName: "Bot\r\nBcc: evil@x.y"},
			wantErr: true,
		},
		{
			name:    "cc CRLF rejected",
			in:      SendInput{To: []string{"a@b.c"}, Cc: []string{"c@d.e\r\nBcc: evil@x.y"}, Subject: "Hi", Text: "body"},
			wantErr: true,
		},
		{
			name:    "bcc CRLF rejected",
			in:      SendInput{To: []string{"a@b.c"}, Bcc: []string{"h@x.y\r\nTo: evil@z.z"}, Subject: "Hi", Text: "body"},
			wantErr: true,
		},
		{
			name: "from display name quoted",
			in:   SendInput{To: []string{"a@b.c"}, Subject: "Hi", Text: "body", FromName: "Ops, Bot"},
			wantSubstr: []string{
				`"Ops, Bot"`,
			},
		},
		{
			name: "bcc omitted from mime",
			in:   SendInput{To: []string{"a@b.c"}, Bcc: []string{"hidden@x.y"}, Subject: "Hi", Text: "body"},
			wantSubstr: []string{
				"To: a@b.c",
			},
			wantAbsent: []string{
				"hidden@x.y",
				"Bcc:",
			},
		},
		{
			name: "html script stripped",
			in:   SendInput{To: []string{"a@b.c"}, Subject: "Hi", HTML: `<p>ok</p><script>alert(1)</script>`},
			wantSubstr: []string{
				"<p>ok</p>",
			},
			wantAbsent: []string{
				"<script>",
				"alert(1)",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			in, err := sanitizeSendInput(tt.in)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			fromHeader, err := formatFromHeader(in.FromName, "u@example.com")
			require.NoError(t, err)
			msg, err := buildMIMEMessage(fromHeader, in)
			require.NoError(t, err)
			s := string(msg)
			for _, sub := range tt.wantSubstr {
				assert.Contains(t, s, sub)
			}
			for _, absent := range tt.wantAbsent {
				assert.NotContains(t, s, absent)
			}
		})
	}
}

func TestBuildMIMEMessageRejectsUnsanitizedAddressCRLF(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   SendInput
	}{
		{name: "to", in: SendInput{To: []string{"a@b.c\r\nBcc: evil@x.y"}, Subject: "Hi", Text: "body"}},
		{name: "cc", in: SendInput{To: []string{"a@b.c"}, Cc: []string{"c@d.e\r\nBcc: evil@x.y"}, Subject: "Hi", Text: "body"}},
		{name: "bcc", in: SendInput{To: []string{"a@b.c"}, Bcc: []string{"h@x.y\r\nTo: evil@z"}, Subject: "Hi", Text: "body"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := buildMIMEMessage("u@example.com", tt.in)
			require.Error(t, err)
		})
	}
}

func TestApplyUIDCursor(t *testing.T) {
	t.Parallel()
	uids := []uint32{10, 9, 8, 7}
	assert.Equal(t, []uint32{10, 9, 8, 7}, applyUIDCursor(uids, ""))
	assert.Equal(t, []uint32{8, 7}, applyUIDCursor(uids, "9"))
}

func TestNewClientDefaults(t *testing.T) {
	t.Parallel()
	c, err := NewClient(Config{
		Username: "u@example.com",
		Password: "secret",
		SMTPHost: "smtp.example.com",
		IMAPHost: "imap.example.com",
	})
	require.NoError(t, err)
	cfg := c.Config()
	assert.Equal(t, "INBOX", cfg.Mailbox)
	assert.Equal(t, 465, cfg.SMTPPort)
	assert.Equal(t, 993, cfg.IMAPPort)
	assert.Equal(t, TLSModeTLS, cfg.SMTPTLS)
	assert.Equal(t, TLSModeTLS, cfg.IMAPTLS)
}
