package notify

import (
	"context"
	"errors"
	"slices"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/flog"
	notifytmpl "github.com/flowline-io/flowbot/pkg/notify/template"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
)

const (
	// AgentNotifyTemplateID is the seeded template for chatagent send_notification.
	AgentNotifyTemplateID = "agent.notify"
	// AgentNotifyTemplateBody is the default body for AgentNotifyTemplateID.
	AgentNotifyTemplateBody = "{{ .summary }}"
)

var (
	// ErrNoDefaultChannel reports that no enabled default notify channel is configured.
	ErrNoDefaultChannel = errors.New("no default notify channel configured")
	// ErrNoDefaultTemplate reports that no default notify template is configured.
	ErrNoDefaultTemplate = errors.New("no default notify template configured")
)

// TemplateReferencesSummary reports whether a template body (and overrides JSON)
// references the summary payload field used by GatewaySendDefaults.
func TemplateReferencesSummary(defaultTemplate, overridesJSON string) bool {
	if fieldListContains(notifytmpl.ExtractTemplateFields(defaultTemplate), PayloadKeySummary) {
		return true
	}
	if overridesJSON == "" || overridesJSON == "[]" {
		return false
	}
	var overrides []schema.NotifyTemplateOverride
	if err := sonic.Unmarshal([]byte(overridesJSON), &overrides); err != nil {
		return false
	}
	for _, o := range overrides {
		if fieldListContains(notifytmpl.ExtractTemplateFields(o.Template), PayloadKeySummary) {
			return true
		}
	}
	return false
}

func fieldListContains(fields []string, want string) bool {
	return slices.Contains(fields, want)
}

// ResolveDefaultChannelName returns the name of the global default enabled channel.
func ResolveDefaultChannelName(ctx context.Context) (string, error) {
	db := loadDatabase()
	if db == nil {
		return "", types.ErrUnavailable
	}
	ch, err := db.GetDefaultNotifyChannelRaw(ctx)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return "", ErrNoDefaultChannel
		}
		return "", err
	}
	if !ch.Enabled || ch.Name == "" {
		return "", ErrNoDefaultChannel
	}
	return ch.Name, nil
}

// ResolveDefaultTemplateID returns the template_id of the global default template.
func ResolveDefaultTemplateID(ctx context.Context) (string, error) {
	db := loadDatabase()
	if db == nil {
		return "", types.ErrUnavailable
	}
	tmpl, err := db.GetDefaultNotifyTemplate(ctx)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return "", ErrNoDefaultTemplate
		}
		return "", err
	}
	if tmpl.TemplateID == "" {
		return "", ErrNoDefaultTemplate
	}
	return tmpl.TemplateID, nil
}

// GatewaySendDefaultChannel sends using an explicit template and the global default channel.
func GatewaySendDefaultChannel(ctx context.Context, uid types.Uid, templateID string, payload map[string]any) error {
	channel, err := ResolveDefaultChannelName(ctx)
	if err != nil {
		return err
	}
	return GatewaySend(ctx, uid, templateID, []string{channel}, payload)
}

// GatewaySendDefaults sends using the global default template and default channel.
func GatewaySendDefaults(ctx context.Context, uid types.Uid, payload map[string]any) error {
	templateID, err := ResolveDefaultTemplateID(ctx)
	if err != nil {
		return err
	}
	return GatewaySendDefaultChannel(ctx, uid, templateID, payload)
}

// WarnSkipNoDefault logs and skips when err is a missing-default sentinel.
// Returns true when the caller should treat the send as a soft skip.
func WarnSkipNoDefault(err error, what string) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrNoDefaultChannel) || errors.Is(err, ErrNoDefaultTemplate) {
		flog.Warn("[notify] skip %s: %v", what, err)
		return true
	}
	return false
}

// SeedAgentNotifyTemplate ensures the agent.notify template exists (not marked default).
func SeedAgentNotifyTemplate(ctx context.Context) error {
	db := loadDatabase()
	if db == nil {
		db = store.Database
	}
	if db == nil {
		return nil
	}
	_, err := db.GetNotifyTemplateByTemplateID(ctx, AgentNotifyTemplateID)
	if err == nil {
		return nil
	}
	if !errors.Is(err, types.ErrNotFound) {
		return err
	}
	_, err = db.CreateNotifyTemplate(ctx, model.NotifyTemplate{
		TemplateID:      AgentNotifyTemplateID,
		Name:            "Agent notify",
		Description:     "Chatagent send_notification default-ready template ({{ .summary }})",
		DefaultFormat:   "markdown",
		DefaultTemplate: AgentNotifyTemplateBody,
		OverridesJSON:   "[]",
	})
	if err != nil {
		return err
	}
	flog.Info("[notify] seeded template %s", AgentNotifyTemplateID)
	return nil
}
