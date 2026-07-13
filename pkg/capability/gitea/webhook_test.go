package gitea

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/types"
)

func TestGiteaWebhook_WebhookPath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		want string
	}{
		{name: "returns gitea/events path", want: "gitea/events"},
		{name: "consistent path", want: "gitea/events"},
		{name: "always the same", want: "gitea/events"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := NewGiteaWebhook()
			assert.Equal(t, tt.want, w.WebhookPath())
		})
	}
}

func TestGiteaWebhook_VerifySignature(t *testing.T) {
	t.Parallel()
	body := []byte(`{"ref":"refs/heads/main","before":"abc","after":"def","compare_url":"https://example.com","commits":[],"total_commits":1}`)
	secret := "test-secret"
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	validSig := hex.EncodeToString(mac.Sum(nil))

	tests := []struct {
		name    string
		secret  string
		headers map[string]string
		body    []byte
		wantErr bool
	}{
		{
			name:    "valid signature",
			secret:  secret,
			headers: map[string]string{"X-Gitea-Signature": validSig},
			body:    body,
			wantErr: false,
		},
		{
			name:    "empty secret returns error",
			secret:  "",
			headers: map[string]string{},
			body:    body,
			wantErr: true,
		},
		{
			name:    "missing signature header",
			secret:  secret,
			headers: map[string]string{},
			body:    body,
			wantErr: true,
		},
		{
			name:    "invalid signature",
			secret:  secret,
			headers: map[string]string{"X-Gitea-Signature": "bad-signature"},
			body:    body,
			wantErr: true,
		},
		{
			name:    "wrong header name with X-Hub-Signature",
			secret:  secret,
			headers: map[string]string{"X-Hub-Signature-256": "sha256=" + validSig},
			body:    body,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := &GiteaWebhook{getSecret: func() string { return tt.secret }}
			err := w.VerifySignature(tt.headers, tt.body)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGiteaWebhook_Convert_Push(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		body      string
		wantErr   bool
		wantType  string
		wantEntID string
	}{
		{
			name:      "valid push payload",
			body:      `{"ref":"refs/heads/main","before":"abc123","after":"def456","compare_url":"https://example.com/compare","commits":[],"total_commits":0,"repository":{"full_name":"owner/repo","name":"repo"},"pusher":{"login":"user"},"sender":{"login":"user2"}}`,
			wantErr:   false,
			wantType:  types.EventForgePush,
			wantEntID: "owner/repo@def456",
		},
		{
			name:      "push with commits",
			body:      `{"ref":"refs/heads/feature","before":"111","after":"222","compare_url":"https://example.com/compare","commits":[{"id":"a1","message":"feat: add x","url":"","author":{"login":"dev"},"committer":{"login":"dev"},"verification":null,"timestamp":"2026-01-01T00:00:00Z","added":["x.go"],"removed":[],"modified":[]}],"total_commits":1,"head_commit":{"id":"a1","message":"feat: add x","url":"","author":{"login":"dev"},"committer":{"login":"dev"},"verification":null,"timestamp":"2026-01-01T00:00:00Z","added":["x.go"],"removed":[],"modified":[]},"repository":{"full_name":"owner/repo","name":"repo"},"pusher":{"login":"dev"},"sender":{"login":"dev"}}`,
			wantErr:   false,
			wantType:  types.EventForgePush,
			wantEntID: "owner/repo@222",
		},
		{
			name:    "invalid push JSON",
			body:    `{invalid`,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := NewGiteaWebhook()
			headers := map[string]string{
				"X-Gitea-Event":    "push",
				"X-Gitea-Delivery": "delivery-001",
			}
			events, err := w.Convert([]byte(tt.body), headers)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, events, 1)
			assert.NotEmpty(t, events[0].EventID)
			assert.Equal(t, tt.wantType, events[0].EventType)
			assert.Equal(t, "gitea_webhook", events[0].Source)
			assert.Equal(t, "gitea", events[0].Capability)
			assert.Equal(t, tt.wantEntID, events[0].EntityID)
			assert.Equal(t, "delivery-001", events[0].IdempotencyKey)
			assert.Equal(t, "push", events[0].Operation)
		})
	}
}

func TestGiteaWebhook_Convert_Issues(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		action    string
		wantType  string
		wantOp    string
		wantEntID string
	}{
		{
			name:      "issue opened",
			action:    "opened",
			wantType:  types.EventForgeIssueOpened,
			wantOp:    "opened",
			wantEntID: "42",
		},
		{
			name:      "issue closed",
			action:    "closed",
			wantType:  types.EventForgeIssueClosed,
			wantOp:    "closed",
			wantEntID: "7",
		},
		{
			name:      "issue reopened",
			action:    "reopened",
			wantType:  types.EventForgeIssueReopened,
			wantOp:    "reopened",
			wantEntID: "1",
		},
		{
			name:      "issue edited",
			action:    "edited",
			wantType:  types.EventForgeIssueEdited,
			wantOp:    "edited",
			wantEntID: "99",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := NewGiteaWebhook()
			body := []byte(`{"action":"` + tt.action + `","number":` + tt.wantEntID + `,"issue":{"title":"test"},"repository":{"full_name":"owner/repo","name":"repo"},"sender":{"login":"user"}}`)
			headers := map[string]string{
				"X-Gitea-Event":    "issues",
				"X-Gitea-Delivery": "delivery-002",
			}
			events, err := w.Convert(body, headers)
			require.NoError(t, err)
			require.Len(t, events, 1)
			assert.NotEmpty(t, events[0].EventID)
			assert.Equal(t, tt.wantType, events[0].EventType)
			assert.Equal(t, "gitea_webhook", events[0].Source)
			assert.Equal(t, "gitea", events[0].Capability)
			assert.Equal(t, tt.wantOp, events[0].Operation)
			assert.Equal(t, tt.wantEntID, events[0].EntityID)
			assert.Equal(t, "delivery-002", events[0].IdempotencyKey)
		})
	}
}

func TestGiteaWebhook_Convert_Unsupported(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		eventHdr string
		body     string
	}{
		{
			name:     "unsupported event type pull_request",
			eventHdr: "pull_request",
			body:     `{"action":"opened","number":1}`,
		},
		{
			name:     "unsupported issue action assigned",
			eventHdr: "issues",
			body:     `{"action":"assigned","number":1,"issue":{"title":"test"},"repository":{"full_name":"owner/repo","name":"repo"},"sender":{"login":"user"}}`,
		},
		{
			name:     "unsupported issue action label_updated",
			eventHdr: "issues",
			body:     `{"action":"label_updated","number":1,"issue":{"title":"test"},"repository":{"full_name":"owner/repo","name":"repo"},"sender":{"login":"user"}}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := NewGiteaWebhook()
			headers := map[string]string{"X-Gitea-Event": tt.eventHdr}
			events, err := w.Convert([]byte(tt.body), headers)
			require.NoError(t, err)
			assert.Empty(t, events)
		})
	}
}

func TestGiteaWebhook_Convert_InvalidJSON(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		eventHdr string
		body     string
		wantErr  bool
	}{
		{
			name:     "malformed push payload",
			eventHdr: "push",
			body:     `{broken`,
			wantErr:  true,
		},
		{
			name:     "malformed issues payload",
			eventHdr: "issues",
			body:     `{broken`,
			wantErr:  true,
		},
		{
			name:     "empty event header is treated as unsupported",
			eventHdr: "",
			body:     `{}`,
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := NewGiteaWebhook()
			headers := map[string]string{"X-Gitea-Event": tt.eventHdr}
			events, err := w.Convert([]byte(tt.body), headers)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Empty(t, events)
		})
	}
}

func TestGiteaWebhook_ImplementsWebhookConverter(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "GiteaWebhook implements WebhookConverter"},
		{name: "pointer receiver satisfies interface"},
		{name: "compile-time check matches runtime behavior"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := NewGiteaWebhook()
			var v capability.WebhookConverter = w
			assert.NotNil(t, v)
		})
	}
}
