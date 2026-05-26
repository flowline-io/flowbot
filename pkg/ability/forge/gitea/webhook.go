package gitea

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/flog"
	provider "github.com/flowline-io/flowbot/pkg/providers/gitea"
	"github.com/flowline-io/flowbot/pkg/types"
)

// GiteaWebhook implements ability.WebhookConverter for the Gitea provider.
// It validates HMAC-SHA256 signature and converts Gitea webhook payloads
// into DataEvent records.
type GiteaWebhook struct {
	getSecret func() string
}

// NewGiteaWebhook creates a GiteaWebhook that reads the HMAC secret
// from the gitea provider config at verification time.
func NewGiteaWebhook() *GiteaWebhook {
	return &GiteaWebhook{
		getSecret: provider.GetWebhookSecret,
	}
}

// Compile-time interface check.
var _ ability.WebhookConverter = (*GiteaWebhook)(nil)

// WebhookPath returns the URL path that receives webhook events from Gitea.
// The full URL is /webhook/provider/gitea/events.
func (*GiteaWebhook) WebhookPath() string {
	return "gitea/events"
}

// VerifySignature validates the HMAC-SHA256 signature from the X-Gitea-Signature header.
// The raw body is signed with the configured webhook secret.
func (w *GiteaWebhook) VerifySignature(headers map[string]string, body []byte) error {
	secret := w.getSecret()
	if secret == "" {
		return types.Errorf(types.ErrUnauthorized, "webhook secret not configured")
	}
	signature, ok := headers["X-Gitea-Signature"]
	if !ok {
		return types.Errorf(types.ErrUnauthorized, "missing X-Gitea-Signature header")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return types.Errorf(types.ErrUnauthorized, "invalid signature")
	}
	return nil
}

// Convert transforms the raw Gitea webhook body into DataEvent records.
// Event type is determined by the X-Gitea-Event header:
//
//	"push"   — parsed as RepoPayload, emitted as forge.push
//	"issues" — parsed as IssuePayload, emitted as forge.issue.<action>
//
// Unsupported event types are logged and yield no events.
func (*GiteaWebhook) Convert(body []byte, headers map[string]string) ([]types.DataEvent, error) {
	eventHeader := headers["X-Gitea-Event"]
	delivery := headers["X-Gitea-Delivery"]
	if delivery == "" {
		flog.Warn("gitea webhook: missing X-Gitea-Delivery header, idempotency may be affected")
	}

	switch eventHeader {
	case "push":
		var payload provider.RepoPayload
		if err := sonic.Unmarshal(body, &payload); err != nil {
			return nil, types.Errorf(types.ErrInvalidArgument, "invalid push payload: %v", err)
		}
		entityID := fmt.Sprintf("%s@%s", payload.Repository.FullName, payload.After)
		return []types.DataEvent{{
			EventID:        types.Id(),
			EventType:      types.EventForgePush,
			Source:         "gitea_webhook",
			Capability:     "forge",
			Operation:      "push",
			Backend:        "gitea",
			EntityID:       entityID,
			IdempotencyKey: delivery,
			Data:           types.KV{"payload": payload},
		}}, nil

	case "issues":
		var payload provider.IssuePayload
		if err := sonic.Unmarshal(body, &payload); err != nil {
			return nil, types.Errorf(types.ErrInvalidArgument, "invalid issue payload: %v", err)
		}
		eventType := issueActionToEventType(payload.Action)
		if eventType == "" {
			flog.Warn("gitea webhook: unsupported issue action %s", payload.Action)
			return nil, nil
		}
		entityID := fmt.Sprintf("%d", payload.Index)
		return []types.DataEvent{{
			EventID:        types.Id(),
			EventType:      eventType,
			Source:         "gitea_webhook",
			Capability:     "forge",
			Operation:      string(payload.Action),
			Backend:        "gitea",
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
		flog.Warn("gitea webhook: unsupported event %s", eventHeader)
		return nil, nil
	}
}

// issueActionToEventType maps a Gitea issue action to the corresponding event type constant.
// Returns an empty string for unsupported actions.
func issueActionToEventType(action provider.HookIssueAction) string {
	switch action {
	case provider.HookIssueOpened:
		return types.EventForgeIssueOpened
	case provider.HookIssueClosed:
		return types.EventForgeIssueClosed
	case provider.HookIssueReOpened:
		return types.EventForgeIssueReopened
	case provider.HookIssueEdited:
		return types.EventForgeIssueEdited
	default:
		return ""
	}
}
