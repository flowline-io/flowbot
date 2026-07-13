package github

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/flog"
	provider "github.com/flowline-io/flowbot/pkg/providers/github"
	"github.com/flowline-io/flowbot/pkg/types"
)

// GithubWebhook implements capability.WebhookConverter for the GitHub provider.
// It validates HMAC-SHA256 signature and converts GitHub webhook payloads
// into DataEvent records.
type GithubWebhook struct {
	getSecret func() string
}

// NewGithubWebhook creates a GithubWebhook that reads the HMAC secret
// from the github provider config at verification time.
func NewGithubWebhook() *GithubWebhook {
	return &GithubWebhook{
		getSecret: provider.GetWebhookSecret,
	}
}

// Compile-time interface check.
var _ capability.WebhookConverter = (*GithubWebhook)(nil)

// WebhookPath returns the URL path that receives webhook events from GitHub.
// The full URL is /webhook/provider/github/events.
func (*GithubWebhook) WebhookPath() string {
	return "github/events"
}

// VerifySignature validates the HMAC-SHA256 signature from the X-Hub-Signature-256 header.
// GitHub signs the raw body with the configured webhook secret using SHA-256,
// and sends it as "sha256=<hex digest>".
func (w *GithubWebhook) VerifySignature(headers map[string]string, body []byte) error {
	secret := w.getSecret()
	if secret == "" {
		return types.Errorf(types.ErrUnauthorized, "webhook secret not configured")
	}
	signature, ok := headers["X-Hub-Signature-256"]
	if !ok {
		return types.Errorf(types.ErrUnauthorized, "missing X-Hub-Signature-256 header")
	}
	const prefix = "sha256="
	if !strings.HasPrefix(signature, prefix) {
		return types.Errorf(types.ErrUnauthorized, "invalid signature format")
	}
	signature = signature[len(prefix):]
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return types.Errorf(types.ErrUnauthorized, "invalid signature")
	}
	return nil
}

// Convert transforms the raw GitHub webhook body into DataEvent records.
// Event type is determined by the X-GitHub-Event header:
//
//	"push"   — emitted as forge.push
//	"issues" — emitted as forge.issue.<action>
//
// Unsupported event types and actions are logged and yield no events.
func (*GithubWebhook) Convert(body []byte, headers map[string]string) ([]types.DataEvent, error) {
	eventHeader := headers["X-GitHub-Event"]
	delivery := headers["X-GitHub-Delivery"]
	if delivery == "" {
		delivery = types.Id()
		flog.Warn("github webhook: missing X-GitHub-Delivery header, idempotency may be affected")
	}

	switch eventHeader {
	case "push":
		var payload pushPayload
		if err := sonic.Unmarshal(body, &payload); err != nil {
			return nil, types.Errorf(types.ErrInvalidArgument, "invalid push payload: %v", err)
		}
		entityID := fmt.Sprintf("%s/%s@%s", payload.Repository.FullName, payload.Ref, payload.After)
		return []types.DataEvent{{
			EventID:        types.Id(),
			EventType:      types.EventForgePush,
			Source:         "github_webhook",
			Capability:     "github",
			Operation:      "push",
			EntityID:       entityID,
			IdempotencyKey: delivery,
			Data:           types.KV{"payload": payload},
		}}, nil

	case "issues":
		var payload issuePayload
		if err := sonic.Unmarshal(body, &payload); err != nil {
			return nil, types.Errorf(types.ErrInvalidArgument, "invalid issue payload: %v", err)
		}
		eventType := issueActionToEventType(payload.Action)
		if eventType == "" {
			flog.Warn("github webhook: unsupported issue action %s", payload.Action)
			return nil, nil
		}
		entityID := fmt.Sprintf("%d", payload.Issue.Number)
		return []types.DataEvent{{
			EventID:        types.Id(),
			EventType:      eventType,
			Source:         "github_webhook",
			Capability:     "github",
			Operation:      payload.Action,
			EntityID:       entityID,
			IdempotencyKey: delivery,
			Data: types.KV{
				"issue":      payload.Issue,
				"repository": payload.Repository,
				"sender":     payload.Sender,
				"event_type": eventHeader,
			},
		}}, nil

	default:
		flog.Warn("github webhook: unsupported event %s", eventHeader)
		return nil, nil
	}
}

// pushPayload is the subset of GitHub push webhook payload used by this converter.
type pushPayload struct {
	Ref        string        `json:"ref"`
	After      string        `json:"after"`
	Repository repositoryRef `json:"repository"`
}

// issuePayload is the subset of GitHub issues webhook payload used by this converter.
type issuePayload struct {
	Action     string        `json:"action"`
	Issue      issueRef      `json:"issue"`
	Repository repositoryRef `json:"repository"`
	Sender     userRef       `json:"sender"`
}

type repositoryRef struct {
	FullName string `json:"full_name"`
}

type issueRef struct {
	Number int64  `json:"number"`
	Title  string `json:"title"`
}

type userRef struct {
	Login string `json:"login"`
}

// issueActionToEventType maps a GitHub issue action to the corresponding event type constant.
// Returns an empty string for unsupported actions.
func issueActionToEventType(action string) string {
	switch action {
	case "opened":
		return types.EventForgeIssueOpened
	case "closed":
		return types.EventForgeIssueClosed
	case "reopened":
		return types.EventForgeIssueReopened
	case "edited":
		return types.EventForgeIssueEdited
	default:
		return ""
	}
}
