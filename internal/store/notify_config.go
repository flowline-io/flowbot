package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/notifychannel"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/notifyrule"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/notifytemplate"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
)

// NotifyConfigStore persists notification channels, rules, and templates.
type NotifyConfigStore struct {
	client *gen.Client
}

// NewNotifyConfigStore creates a NotifyConfigStore with the given ent client.
func NewNotifyConfigStore(client *gen.Client) *NotifyConfigStore {
	return &NotifyConfigStore{client: client}
}

// NotifyConfigStoreFromDB returns a NotifyConfigStore using the global database client.
func NotifyConfigStoreFromDB() *NotifyConfigStore {
	return NewNotifyConfigStore(ClientFromDB())
}

// Client returns the underlying ent client.
func (s *NotifyConfigStore) Client() *gen.Client {
	if s == nil {
		return nil
	}
	return s.client
}

// CreateNotifyChannel persists a new notify channel.
func (s *NotifyConfigStore) CreateNotifyChannel(ctx context.Context, name, protocol, uri string) (int64, error) {
	ch, err := s.client.NotifyChannel.Create().
		SetName(name).
		SetProtocol(protocol).
		SetURI(uri).
		SetEnabled(true).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return 0, fmt.Errorf("postgres: create notify channel: %w", err)
	}
	return ch.ID, nil
}

// GetNotifyChannel returns the notify channel.
func (s *NotifyConfigStore) GetNotifyChannel(ctx context.Context, id int64) (model.NotifyChannel, error) {
	ch, err := s.client.NotifyChannel.Query().Where(notifychannel.IDEQ(id)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return model.NotifyChannel{}, types.ErrNotFound
		}
		return model.NotifyChannel{}, fmt.Errorf("postgres: get notify channel: %w", err)
	}
	out := notifyChannelToModel(ch)
	out.URI = MaskNotifyURI(ch.Protocol, ch.URI)
	return out, nil
}

// GetNotifyChannelRaw returns the notify channel raw.
func (s *NotifyConfigStore) GetNotifyChannelRaw(ctx context.Context, id int64) (model.NotifyChannel, error) {
	ch, err := s.client.NotifyChannel.Query().Where(notifychannel.IDEQ(id)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return model.NotifyChannel{}, types.ErrNotFound
		}
		return model.NotifyChannel{}, fmt.Errorf("postgres: get notify channel raw: %w", err)
	}
	return notifyChannelToModel(ch), nil
}

// GetNotifyChannelByNameRaw returns the notify channel by name raw.
func (s *NotifyConfigStore) GetNotifyChannelByNameRaw(ctx context.Context, name string) (model.NotifyChannel, error) {
	ch, err := s.client.NotifyChannel.Query().Where(notifychannel.NameEQ(name)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return model.NotifyChannel{}, types.ErrNotFound
		}
		return model.NotifyChannel{}, fmt.Errorf("postgres: get notify channel by name raw: %w", err)
	}
	return notifyChannelToModel(ch), nil
}

// GetDefaultNotifyChannelRaw returns the default notify channel raw.
func (s *NotifyConfigStore) GetDefaultNotifyChannelRaw(ctx context.Context) (model.NotifyChannel, error) {
	ch, err := s.client.NotifyChannel.Query().
		Where(notifychannel.IsDefaultEQ(true), notifychannel.EnabledEQ(true)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return model.NotifyChannel{}, types.ErrNotFound
		}
		return model.NotifyChannel{}, fmt.Errorf("postgres: get default notify channel: %w", err)
	}
	return notifyChannelToModel(ch), nil
}

// ListNotifyChannels returns notify channels.
func (s *NotifyConfigStore) ListNotifyChannels(ctx context.Context, opts ListNotifyChannelOptions) ([]model.NotifyChannel, error) {
	q := s.client.NotifyChannel.Query()
	if opts.Protocol != "" {
		q = q.Where(notifychannel.Protocol(opts.Protocol))
	}
	if opts.Enabled != nil {
		q = q.Where(notifychannel.Enabled(*opts.Enabled))
	}
	chs, err := q.Order(gen.Asc(notifychannel.FieldName)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: list notify channels: %w", err)
	}
	result := make([]model.NotifyChannel, len(chs))
	for i, ch := range chs {
		out := notifyChannelToModel(ch)
		out.URI = MaskNotifyURI(ch.Protocol, ch.URI)
		result[i] = out
	}
	return result, nil
}

// UpdateNotifyChannel updates the notify channel.
func (s *NotifyConfigStore) UpdateNotifyChannel(ctx context.Context, id int64, name, protocol, uri string, enabled bool) error {
	upd := s.client.NotifyChannel.Update().Where(notifychannel.IDEQ(id)).
		SetName(name).
		SetProtocol(protocol).
		SetEnabled(enabled).
		SetUpdatedAt(time.Now())
	if uri != "" {
		upd = upd.SetURI(uri)
	}
	if !enabled {
		upd = upd.SetIsDefault(false)
	}
	n, err := upd.Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: update notify channel: %w", err)
	}
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// SetDefaultNotifyChannel sets the default notify channel.
func (s *NotifyConfigStore) SetDefaultNotifyChannel(ctx context.Context, id int64) error {
	tx, err := s.client.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("postgres: set default notify channel begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	ch, err := tx.NotifyChannel.Query().Where(notifychannel.IDEQ(id)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return types.ErrNotFound
		}
		return fmt.Errorf("postgres: set default notify channel get: %w", err)
	}
	if !ch.Enabled {
		return types.Errorf(types.ErrInvalidArgument, "default notify channel must be enabled")
	}
	if _, err := tx.NotifyChannel.Update().Where(notifychannel.IsDefaultEQ(true)).SetIsDefault(false).Save(ctx); err != nil {
		return fmt.Errorf("postgres: clear default notify channels: %w", err)
	}
	n, err := tx.NotifyChannel.Update().Where(notifychannel.IDEQ(id)).
		SetIsDefault(true).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: set default notify channel: %w", err)
	}
	if n == 0 {
		return types.ErrNotFound
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("postgres: set default notify channel commit: %w", err)
	}
	return nil
}

// DeleteNotifyChannel deletes the notify channel.
func (s *NotifyConfigStore) DeleteNotifyChannel(ctx context.Context, id int64) error {
	_, err := s.client.NotifyChannel.Delete().Where(notifychannel.IDEQ(id)).Exec(ctx)
	if err != nil {
		return fmt.Errorf("postgres: delete notify channel: %w", err)
	}
	return nil
}

func notifyChannelToModel(ch *gen.NotifyChannel) model.NotifyChannel {
	return model.NotifyChannel{
		ID:        ch.ID,
		Name:      ch.Name,
		Protocol:  ch.Protocol,
		URI:       ch.URI,
		Enabled:   ch.Enabled,
		IsDefault: ch.IsDefault,
		CreatedAt: ch.CreatedAt,
		UpdatedAt: ch.UpdatedAt,
	}
}

// MaskNotifyURI produces a display-safe masked form of a notification URI.
func MaskNotifyURI(protocol, uri string) string {
	switch protocol {
	case "slack":
		return maskSlackURI(uri)
	case "ntfy":
		return maskNtfyURI(uri)
	case "pushover":
		return maskPushoverURI(uri)
	case "message-pusher":
		return maskMessagePusherURI(uri)
	default:
		if len(uri) > 30 {
			return uri[:27] + "..."
		}
		return uri
	}
}

func maskSlackURI(uri string) string {
	parts := strings.SplitN(uri, "://", 2)
	if len(parts) < 2 {
		return "slack://******"
	}
	pathParts := strings.Split(parts[1], "/")
	if len(pathParts) > 3 {
		pathParts[len(pathParts)-3] = "T******"
	}
	if len(pathParts) > 2 {
		pathParts[len(pathParts)-2] = "B******"
	}
	if len(pathParts) > 1 {
		pathParts[len(pathParts)-1] = "C******"
	}
	return parts[0] + "://" + strings.Join(pathParts, "/")
}

func maskNtfyURI(uri string) string {
	parts := strings.SplitN(uri, "://", 2)
	if len(parts) < 2 {
		return "ntfy://******"
	}
	hostParts := strings.SplitN(parts[1], "/", 2)
	if len(hostParts) < 2 {
		return parts[0] + "://" + hostParts[0] + "/******"
	}
	return parts[0] + "://" + hostParts[0] + "/******"
}

func maskPushoverURI(uri string) string {
	parts := strings.SplitN(uri, "://", 2)
	if len(parts) < 2 {
		return "pushover://******"
	}
	userIdx := strings.Index(parts[1], "@")
	if userIdx < 0 {
		return parts[0] + "://U******@" + maskEnd(parts[1])
	}
	return parts[0] + "://U******@A******"
}

func maskMessagePusherURI(uri string) string {
	parts := strings.SplitN(uri, "://", 2)
	if len(parts) < 2 {
		return "message-pusher://******"
	}
	atIdx := strings.Index(parts[1], "@")
	if atIdx < 0 {
		return parts[0] + "://******"
	}
	finalSlash := strings.LastIndex(parts[1], "/")
	if finalSlash < 0 {
		return parts[0] + "://" + parts[1][:atIdx+1] + "domain/******/******"
	}
	secondLast := strings.LastIndex(parts[1][:finalSlash], "/")
	if secondLast < 0 {
		return parts[0] + "://" + parts[1][:finalSlash+1] + "******"
	}
	return parts[0] + "://" + parts[1][:secondLast+1] + "******/******"
}

func maskEnd(s string) string {
	if len(s) > 8 {
		return s[:4] + "******"
	}
	return "******"
}

// CreateNotifyRule persists a new notify rule.
func (s *NotifyConfigStore) CreateNotifyRule(ctx context.Context, rule model.NotifyRule) (int64, error) {
	var params map[string]any
	if rule.ParamsJSON != "" {
		if err := sonic.Unmarshal([]byte(rule.ParamsJSON), &params); err != nil {
			return 0, fmt.Errorf("postgres: create notify rule params parse: %w", err)
		}
	} else {
		params = map[string]any{}
	}
	r, err := s.client.NotifyRule.Create().
		SetRuleID(rule.RuleID).
		SetName(rule.Name).
		SetAction(notifyrule.Action(rule.Action)).
		SetEventPattern(rule.EventPattern).
		SetChannelPattern(rule.ChannelPattern).
		SetNillableCondition(nilString(rule.Condition)).
		SetPriority(rule.Priority).
		SetParams(params).
		SetEnabled(rule.Enabled).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return 0, fmt.Errorf("postgres: create notify rule: %w", err)
	}
	return r.ID, nil
}

// GetNotifyRule returns the notify rule.
func (s *NotifyConfigStore) GetNotifyRule(ctx context.Context, id int64) (model.NotifyRule, error) {
	r, err := s.client.NotifyRule.Query().Where(notifyrule.IDEQ(id)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return model.NotifyRule{}, types.ErrNotFound
		}
		return model.NotifyRule{}, fmt.Errorf("postgres: get notify rule: %w", err)
	}
	return notifyRuleToModel(r)
}

// ListNotifyRules returns notify rules.
func (s *NotifyConfigStore) ListNotifyRules(ctx context.Context, opts ListNotifyRuleOptions) ([]model.NotifyRule, error) {
	q := s.client.NotifyRule.Query()
	if opts.Enabled != nil {
		q = q.Where(notifyrule.Enabled(*opts.Enabled))
	}
	rules, err := q.Order(gen.Desc(notifyrule.FieldPriority)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: list notify rules: %w", err)
	}
	result := make([]model.NotifyRule, len(rules))
	for i, r := range rules {
		m, err := notifyRuleToModel(r)
		if err != nil {
			return nil, err
		}
		result[i] = m
	}
	return result, nil
}

// UpdateNotifyRule updates the notify rule.
func (s *NotifyConfigStore) UpdateNotifyRule(ctx context.Context, id int64, rule model.NotifyRule) error {
	var params map[string]any
	if rule.ParamsJSON != "" {
		if err := sonic.Unmarshal([]byte(rule.ParamsJSON), &params); err != nil {
			return fmt.Errorf("postgres: update notify rule params parse: %w", err)
		}
	} else {
		params = map[string]any{}
	}
	n, err := s.client.NotifyRule.Update().Where(notifyrule.IDEQ(id)).
		SetRuleID(rule.RuleID).
		SetName(rule.Name).
		SetAction(notifyrule.Action(rule.Action)).
		SetEventPattern(rule.EventPattern).
		SetChannelPattern(rule.ChannelPattern).
		SetNillableCondition(nilString(rule.Condition)).
		SetPriority(rule.Priority).
		SetParams(params).
		SetEnabled(rule.Enabled).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: update notify rule: %w", err)
	}
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// DeleteNotifyRule deletes the notify rule.
func (s *NotifyConfigStore) DeleteNotifyRule(ctx context.Context, id int64) error {
	_, err := s.client.NotifyRule.Delete().Where(notifyrule.IDEQ(id)).Exec(ctx)
	if err != nil {
		return fmt.Errorf("postgres: delete notify rule: %w", err)
	}
	return nil
}

func notifyRuleToModel(r *gen.NotifyRule) (model.NotifyRule, error) {
	paramsJSON, err := sonic.MarshalString(r.Params)
	if err != nil {
		return model.NotifyRule{}, fmt.Errorf("postgres: marshal notify rule params: %w", err)
	}
	return model.NotifyRule{
		ID:             r.ID,
		RuleID:         r.RuleID,
		Name:           r.Name,
		Action:         string(r.Action),
		EventPattern:   r.EventPattern,
		ChannelPattern: r.ChannelPattern,
		Condition:      r.Condition,
		Priority:       r.Priority,
		ParamsJSON:     paramsJSON,
		Enabled:        r.Enabled,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
	}, nil
}

func nilString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// CreateNotifyTemplate persists a new notify template.
func (s *NotifyConfigStore) CreateNotifyTemplate(ctx context.Context, tmpl model.NotifyTemplate) (int64, error) {
	overrides, err := parseNotifyTemplateOverrides(tmpl.OverridesJSON)
	if err != nil {
		return 0, fmt.Errorf("postgres: create notify template overrides parse: %w", err)
	}
	row, err := s.client.NotifyTemplate.Create().
		SetTemplateID(tmpl.TemplateID).
		SetName(tmpl.Name).
		SetDescription(tmpl.Description).
		SetDefaultFormat(tmpl.DefaultFormat).
		SetDefaultTemplate(tmpl.DefaultTemplate).
		SetOverrides(overrides).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return 0, fmt.Errorf("postgres: create notify template: %w", err)
	}
	return row.ID, nil
}

// GetNotifyTemplate returns the notify template.
func (s *NotifyConfigStore) GetNotifyTemplate(ctx context.Context, id int64) (model.NotifyTemplate, error) {
	row, err := s.client.NotifyTemplate.Query().Where(notifytemplate.IDEQ(id)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return model.NotifyTemplate{}, types.ErrNotFound
		}
		return model.NotifyTemplate{}, fmt.Errorf("postgres: get notify template: %w", err)
	}
	return notifyTemplateToModel(row)
}

// GetNotifyTemplateByTemplateID returns the notify template by template id.
func (s *NotifyConfigStore) GetNotifyTemplateByTemplateID(ctx context.Context, templateID string) (model.NotifyTemplate, error) {
	row, err := s.client.NotifyTemplate.Query().Where(notifytemplate.TemplateIDEQ(templateID)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return model.NotifyTemplate{}, types.ErrNotFound
		}
		return model.NotifyTemplate{}, fmt.Errorf("postgres: get notify template by template_id: %w", err)
	}
	return notifyTemplateToModel(row)
}

// GetDefaultNotifyTemplate returns the default notify template.
func (s *NotifyConfigStore) GetDefaultNotifyTemplate(ctx context.Context) (model.NotifyTemplate, error) {
	row, err := s.client.NotifyTemplate.Query().Where(notifytemplate.IsDefaultEQ(true)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return model.NotifyTemplate{}, types.ErrNotFound
		}
		return model.NotifyTemplate{}, fmt.Errorf("postgres: get default notify template: %w", err)
	}
	return notifyTemplateToModel(row)
}

// ListNotifyTemplates returns notify templates.
func (s *NotifyConfigStore) ListNotifyTemplates(ctx context.Context, _ ListNotifyTemplateOptions) ([]model.NotifyTemplate, error) {
	rows, err := s.client.NotifyTemplate.Query().Order(gen.Asc(notifytemplate.FieldTemplateID)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: list notify templates: %w", err)
	}
	result := make([]model.NotifyTemplate, len(rows))
	for i, row := range rows {
		m, err := notifyTemplateToModel(row)
		if err != nil {
			return nil, err
		}
		result[i] = m
	}
	return result, nil
}

// UpdateNotifyTemplate updates the notify template.
func (s *NotifyConfigStore) UpdateNotifyTemplate(ctx context.Context, id int64, tmpl model.NotifyTemplate) error {
	overrides, err := parseNotifyTemplateOverrides(tmpl.OverridesJSON)
	if err != nil {
		return fmt.Errorf("postgres: update notify template overrides parse: %w", err)
	}
	n, err := s.client.NotifyTemplate.Update().Where(notifytemplate.IDEQ(id)).
		SetTemplateID(tmpl.TemplateID).
		SetName(tmpl.Name).
		SetDescription(tmpl.Description).
		SetDefaultFormat(tmpl.DefaultFormat).
		SetDefaultTemplate(tmpl.DefaultTemplate).
		SetOverrides(overrides).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: update notify template: %w", err)
	}
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// SetDefaultNotifyTemplate sets the default notify template.
func (s *NotifyConfigStore) SetDefaultNotifyTemplate(ctx context.Context, id int64) error {
	tx, err := s.client.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("postgres: set default notify template begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.NotifyTemplate.Query().Where(notifytemplate.IDEQ(id)).Only(ctx); err != nil {
		if gen.IsNotFound(err) {
			return types.ErrNotFound
		}
		return fmt.Errorf("postgres: set default notify template get: %w", err)
	}
	if _, err := tx.NotifyTemplate.Update().Where(notifytemplate.IsDefaultEQ(true)).SetIsDefault(false).Save(ctx); err != nil {
		return fmt.Errorf("postgres: clear default notify templates: %w", err)
	}
	n, err := tx.NotifyTemplate.Update().Where(notifytemplate.IDEQ(id)).
		SetIsDefault(true).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: set default notify template: %w", err)
	}
	if n == 0 {
		return types.ErrNotFound
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("postgres: set default notify template commit: %w", err)
	}
	return nil
}

// DeleteNotifyTemplate deletes the notify template.
func (s *NotifyConfigStore) DeleteNotifyTemplate(ctx context.Context, id int64) error {
	_, err := s.client.NotifyTemplate.Delete().Where(notifytemplate.IDEQ(id)).Exec(ctx)
	if err != nil {
		return fmt.Errorf("postgres: delete notify template: %w", err)
	}
	return nil
}

func parseNotifyTemplateOverrides(overridesJSON string) ([]schema.NotifyTemplateOverride, error) {
	if overridesJSON == "" {
		return []schema.NotifyTemplateOverride{}, nil
	}
	var overrides []schema.NotifyTemplateOverride
	if err := sonic.Unmarshal([]byte(overridesJSON), &overrides); err != nil {
		return nil, err
	}
	if overrides == nil {
		overrides = []schema.NotifyTemplateOverride{}
	}
	return overrides, nil
}

func notifyTemplateToModel(row *gen.NotifyTemplate) (model.NotifyTemplate, error) {
	overridesJSON := "[]"
	if len(row.Overrides) > 0 {
		s, err := sonic.MarshalString(row.Overrides)
		if err != nil {
			return model.NotifyTemplate{}, fmt.Errorf("postgres: marshal notify template overrides: %w", err)
		}
		overridesJSON = s
	}
	return model.NotifyTemplate{
		ID:              row.ID,
		TemplateID:      row.TemplateID,
		Name:            row.Name,
		Description:     row.Description,
		DefaultFormat:   row.DefaultFormat,
		DefaultTemplate: row.DefaultTemplate,
		OverridesJSON:   overridesJSON,
		IsDefault:       row.IsDefault,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}, nil
}

