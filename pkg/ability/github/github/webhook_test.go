package github

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/types"
)

func TestGithubWebhook_ImplementsWebhookConverter(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "GithubWebhook implements WebhookConverter"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := NewGithubWebhook()
			var v ability.WebhookConverter = w
			assert.NotNil(t, v)
		})
	}
}

func TestGithubWebhook_WebhookPath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		want string
	}{
		{name: "returns github events path", want: "github/events"},
		{name: "consistent path", want: "github/events"},
		{name: "path does not change", want: "github/events"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := NewGithubWebhook()
			assert.Equal(t, tt.want, w.WebhookPath())
		})
	}
}

func TestGithubWebhook_VerifySignature(t *testing.T) {
	t.Parallel()
	body := []byte(`{"action":"opened","issue":{"number":1}}`)
	secret := "test-secret"
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	validSig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

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
			headers: map[string]string{"X-Hub-Signature-256": validSig},
			body:    body,
			wantErr: false,
		},
		{
			name:    "missing header",
			secret:  secret,
			headers: map[string]string{},
			body:    body,
			wantErr: true,
		},
		{
			name:    "invalid signature",
			secret:  secret,
			headers: map[string]string{"X-Hub-Signature-256": "sha256=bad-signature"},
			body:    body,
			wantErr: true,
		},
		{
			name:    "empty secret returns error",
			secret:  "",
			headers: map[string]string{},
			body:    body,
			wantErr: true,
		},
		{
			name:    "wrong header name X-Hub-Signature",
			secret:  secret,
			headers: map[string]string{"X-Hub-Signature": validSig},
			body:    body,
			wantErr: true,
		},
		{
			name:    "signature without sha256 prefix",
			secret:  secret,
			headers: map[string]string{"X-Hub-Signature-256": hex.EncodeToString(mac.Sum(nil))},
			body:    body,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := &GithubWebhook{getSecret: func() string { return tt.secret }}
			err := w.VerifySignature(tt.headers, tt.body)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGithubWebhook_Convert(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		headers   map[string]string
		body      []byte
		wantErr   bool
		wantCount int
		check     func(t *testing.T, events []types.DataEvent)
	}{
		{
			name: "push event",
			headers: map[string]string{
				"X-GitHub-Event":    "push",
				"X-GitHub-Delivery": "delivery-001",
			},
			body:      []byte(`{"ref":"refs/heads/main","repository":{"full_name":"octocat/Hello-World"},"after":"abc123"}`),
			wantErr:   false,
			wantCount: 1,
			check: func(t *testing.T, events []types.DataEvent) {
				assert.Equal(t, types.EventForgePush, events[0].EventType)
				assert.Equal(t, "github_webhook", events[0].Source)
				assert.Equal(t, "github", events[0].Capability)
				assert.Equal(t, "github", events[0].Backend)
				assert.Equal(t, "push", events[0].Operation)
				assert.Equal(t, "delivery-001", events[0].IdempotencyKey)
				assert.NotEmpty(t, events[0].EventID)
			},
		},
		{
			name: "issues opened",
			headers: map[string]string{
				"X-GitHub-Event":    "issues",
				"X-GitHub-Delivery": "delivery-002",
			},
			body:      []byte(`{"action":"opened","issue":{"number":1,"title":"test"},"repository":{"full_name":"octocat/Hello-World"}}`),
			wantErr:   false,
			wantCount: 1,
			check: func(t *testing.T, events []types.DataEvent) {
				assert.Equal(t, types.EventForgeIssueOpened, events[0].EventType)
				assert.Equal(t, "github_webhook", events[0].Source)
				assert.Equal(t, "github", events[0].Capability)
				assert.Equal(t, "github", events[0].Backend)
				assert.Equal(t, "opened", events[0].Operation)
				assert.Equal(t, "delivery-002", events[0].IdempotencyKey)
			},
		},
		{
			name: "issues closed",
			headers: map[string]string{
				"X-GitHub-Event":    "issues",
				"X-GitHub-Delivery": "delivery-003",
			},
			body:      []byte(`{"action":"closed","issue":{"number":2,"title":"done"},"repository":{"full_name":"octocat/Hello-World"}}`),
			wantErr:   false,
			wantCount: 1,
			check: func(t *testing.T, events []types.DataEvent) {
				assert.Equal(t, types.EventForgeIssueClosed, events[0].EventType)
				assert.Equal(t, "closed", events[0].Operation)
			},
		},
		{
			name: "issues reopened",
			headers: map[string]string{
				"X-GitHub-Event":    "issues",
				"X-GitHub-Delivery": "delivery-004",
			},
			body:      []byte(`{"action":"reopened","issue":{"number":3},"repository":{"full_name":"octocat/Hello-World"}}`),
			wantErr:   false,
			wantCount: 1,
			check: func(t *testing.T, events []types.DataEvent) {
				assert.Equal(t, types.EventForgeIssueReopened, events[0].EventType)
				assert.Equal(t, "reopened", events[0].Operation)
			},
		},
		{
			name: "issues edited",
			headers: map[string]string{
				"X-GitHub-Event":    "issues",
				"X-GitHub-Delivery": "delivery-005",
			},
			body:      []byte(`{"action":"edited","issue":{"number":4},"repository":{"full_name":"octocat/Hello-World"}}`),
			wantErr:   false,
			wantCount: 1,
			check: func(t *testing.T, events []types.DataEvent) {
				assert.Equal(t, types.EventForgeIssueEdited, events[0].EventType)
				assert.Equal(t, "edited", events[0].Operation)
			},
		},
		{
			name: "unsupported event returns empty",
			headers: map[string]string{
				"X-GitHub-Event":    "star",
				"X-GitHub-Delivery": "delivery-006",
			},
			body:      []byte(`{"action":"created"}`),
			wantErr:   false,
			wantCount: 0,
			check:     nil,
		},
		{
			name: "unsupported issue action returns empty",
			headers: map[string]string{
				"X-GitHub-Event":    "issues",
				"X-GitHub-Delivery": "delivery-007",
			},
			body:      []byte(`{"action":"assigned","issue":{"number":5},"repository":{"full_name":"octocat/Hello-World"}}`),
			wantErr:   false,
			wantCount: 0,
			check:     nil,
		},
		{
			name: "invalid JSON",
			headers: map[string]string{
				"X-GitHub-Event":    "push",
				"X-GitHub-Delivery": "delivery-008",
			},
			body:      []byte(`{invalid`),
			wantErr:   true,
			wantCount: 0,
			check:     nil,
		},
		{
			name: "missing delivery header",
			headers: map[string]string{
				"X-GitHub-Event": "issues",
			},
			body:      []byte(`{"action":"opened","issue":{"number":1},"repository":{"full_name":"octocat/Hello-World"}}`),
			wantErr:   false,
			wantCount: 1,
			check: func(t *testing.T, events []types.DataEvent) {
				assert.Equal(t, types.EventForgeIssueOpened, events[0].EventType)
				assert.NotEmpty(t, events[0].IdempotencyKey)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := NewGithubWebhook()
			events, err := w.Convert(tt.body, tt.headers)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, events, tt.wantCount)
			if tt.check != nil {
				tt.check(t, events)
			}
		})
	}
}

func TestGithubWebhook_Convert_PushEntityID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		fullName string
		after    string
		wantID   string
	}{
		{name: "standard push", fullName: "octocat/Hello-World", after: "abc123", wantID: "octocat/Hello-World/refs/heads/main@abc123"},
		{name: "org repo push", fullName: "myorg/repo-name", after: "def456", wantID: "myorg/repo-name/refs/heads/main@def456"},
		{name: "long sha push", fullName: "a/b", after: "0123456789abcdef0123456789abcdef01234567", wantID: "a/b/refs/heads/main@0123456789abcdef0123456789abcdef01234567"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := NewGithubWebhook()
			body := fmt.Appendf(nil, `{"ref":"refs/heads/main","repository":{"full_name":"%s"},"after":"%s"}`, tt.fullName, tt.after)
			headers := map[string]string{
				"X-GitHub-Event":    "push",
				"X-GitHub-Delivery": "delivery",
			}
			events, err := w.Convert(body, headers)
			require.NoError(t, err)
			require.Len(t, events, 1)
			assert.Equal(t, tt.wantID, events[0].EntityID)
		})
	}
}
