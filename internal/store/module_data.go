package store

import (
	"context"
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/behavior"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/configdata"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/counter"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/data"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/form"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/instruct"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/oauth"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/page"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/parameter"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
	"github.com/flowline-io/flowbot/pkg/utils"
)

// ModuleDataStore persists module KV data, config, OAuth, forms, pages, behaviors, parameters, instructs, and counters.
type ModuleDataStore struct {
	client *gen.Client
}

// NewModuleDataStore creates a ModuleDataStore with the given ent client.
func NewModuleDataStore(client *gen.Client) *ModuleDataStore {
	return &ModuleDataStore{client: client}
}

// ModuleDataStoreFromDB returns a ModuleDataStore using the global database client.
func ModuleDataStoreFromDB() *ModuleDataStore {
	return NewModuleDataStore(ClientFromDB())
}

// Client returns the underlying ent client.
func (s *ModuleDataStore) Client() *gen.Client {
	if s == nil {
		return nil
	}
	return s.client
}

// DataSet set module data.
func (s *ModuleDataStore) DataSet(ctx context.Context, uid types.Uid, topic, key string, value types.KV) error {
	existing, err := s.client.Data.Query().
		Where(data.UID(uid.String()), data.Topic(topic), data.Key(key)).
		Only(ctx)
	if err != nil && !gen.IsNotFound(err) {
		return fmt.Errorf("postgres: dataset query: %w", err)
	}

	if existing != nil {
		_, err = s.client.Data.Update().Where(data.IDEQ(existing.ID)).
			SetValue(map[string]any(value)).
			SetUpdatedAt(time.Now()).
			Save(ctx)
	} else {
		_, err = s.client.Data.Create().
			SetUID(uid.String()).
			SetTopic(topic).
			SetKey(key).
			SetValue(map[string]any(value)).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(ctx)
	}
	if err != nil {
		return fmt.Errorf("postgres: dataset save: %w", err)
	}
	return nil
}

// DataGet get module data.
func (s *ModuleDataStore) DataGet(ctx context.Context, uid types.Uid, topic, key string) (types.KV, error) {
	d, err := s.client.Data.Query().
		Where(data.UID(uid.String()), data.Topic(topic), data.Key(key)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: dataget: %w", err)
	}
	return types.KV(d.Value), nil
}

// DataList list module data.
func (s *ModuleDataStore) DataList(ctx context.Context, uid types.Uid, topic string, filter types.DataFilter) ([]*gen.Data, error) {
	q := s.client.Data.Query().Where(data.UID(uid.String()), data.Topic(topic))
	if filter.Prefix != nil && *filter.Prefix != "" {
		q = q.Where(data.KeyHasPrefix(*filter.Prefix))
	}
	if filter.CreatedStart != nil {
		q = q.Where(data.CreatedAtGTE(*filter.CreatedStart))
	}
	if filter.CreatedEnd != nil {
		q = q.Where(data.CreatedAtLTE(*filter.CreatedEnd))
	}
	q = q.Order(gen.Asc(data.FieldCreatedAt)).Limit(queryMaxResults())
	items, err := q.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: datalist: %w", err)
	}
	result := make([]*gen.Data, len(items))
	for i, d := range items {
		result[i] = &gen.Data{
			ID:        d.ID,
			UID:       d.UID,
			Topic:     d.Topic,
			Key:       d.Key,
			Value:     schema.JSON(d.Value),
			CreatedAt: d.CreatedAt,
			UpdatedAt: d.UpdatedAt,
		}
	}
	return result, nil
}

// DataDelete delete module data.
func (s *ModuleDataStore) DataDelete(ctx context.Context, uid types.Uid, topic, key string) error {
	_, err := s.client.Data.Delete().
		Where(data.UID(uid.String()), data.Topic(topic), data.Key(key)).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("postgres: datadelete: %w", err)
	}
	return nil
}

// configTopic maps callers' empty topic to a non-empty store value (ent NotEmpty).
func configTopic(topic string) string {
	if topic == "" {
		return "-"
	}
	return topic
}

// ConfigSet set config data.
func (s *ModuleDataStore) ConfigSet(ctx context.Context, uid types.Uid, topic, key string, value types.KV) error {
	topic = configTopic(topic)
	existing, err := s.client.ConfigData.Query().
		Where(configdata.UID(uid.String()), configdata.Topic(topic), configdata.Key(key)).
		Only(ctx)
	if err != nil && !gen.IsNotFound(err) {
		return fmt.Errorf("postgres: configset query: %w", err)
	}

	if existing != nil {
		_, err = s.client.ConfigData.Update().Where(configdata.IDEQ(existing.ID)).
			SetValue(map[string]any(value)).
			SetUpdatedAt(time.Now()).
			Save(ctx)
	} else {
		_, err = s.client.ConfigData.Create().
			SetUID(uid.String()).
			SetTopic(topic).
			SetKey(key).
			SetValue(map[string]any(value)).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(ctx)
	}
	if err != nil {
		return fmt.Errorf("postgres: configset save: %w", err)
	}
	return nil
}

// ConfigGet get config data.
func (s *ModuleDataStore) ConfigGet(ctx context.Context, uid types.Uid, topic, key string) (types.KV, error) {
	topic = configTopic(topic)
	d, err := s.client.ConfigData.Query().
		Where(configdata.UID(uid.String()), configdata.Topic(topic), configdata.Key(key)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: configget: %w", err)
	}
	return types.KV(d.Value), nil
}

// ListConfigByPrefix returns config by prefix.
func (s *ModuleDataStore) ListConfigByPrefix(ctx context.Context, uid types.Uid, topic, prefix string) ([]*gen.ConfigData, error) {
	topic = configTopic(topic)
	q := s.client.ConfigData.Query().Where(configdata.UID(uid.String()), configdata.Topic(topic))
	if prefix != "" {
		q = q.Where(configdata.KeyHasPrefix(prefix))
	}
	q = q.Order(gen.Asc(configdata.FieldCreatedAt)).Limit(queryMaxResults())
	items, err := q.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: listconfigbyprefix: %w", err)
	}
	result := make([]*gen.ConfigData, len(items))
	for i, d := range items {
		result[i] = &gen.ConfigData{
			ID:        d.ID,
			UID:       d.UID,
			Topic:     d.Topic,
			Key:       d.Key,
			Value:     schema.JSON(d.Value),
			CreatedAt: d.CreatedAt,
			UpdatedAt: d.UpdatedAt,
		}
	}
	return result, nil
}

// ConfigDelete delete config data.
func (s *ModuleDataStore) ConfigDelete(ctx context.Context, uid types.Uid, topic, key string) error {
	topic = configTopic(topic)
	_, err := s.client.ConfigData.Delete().
		Where(configdata.UID(uid.String()), configdata.Topic(topic), configdata.Key(key)).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("postgres: configdelete: %w", err)
	}
	return nil
}

// ListConfigs returns configs.
func (s *ModuleDataStore) ListConfigs(ctx context.Context, opts ListConfigOptions) ([]model.ConfigItem, error) {
	q := s.client.ConfigData.Query()
	if opts.Search != "" {
		q = q.Where(
			configdata.Or(
				configdata.UIDContains(opts.Search),
				configdata.TopicContains(opts.Search),
				configdata.KeyContains(opts.Search),
			),
		)
	}
	limit := opts.Limit
	if limit <= 0 || limit > queryMaxResults() {
		limit = queryMaxResults()
	}
	items, err := q.
		Offset(opts.Offset).
		Limit(limit).
		Order(gen.Desc(configdata.FieldUpdatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: listconfigs: %w", err)
	}
	result := make([]model.ConfigItem, len(items))
	for i, d := range items {
		result[i] = model.ConfigItem{
			ID:        d.ID,
			UID:       d.UID,
			Topic:     d.Topic,
			Key:       d.Key,
			Value:     types.KV(d.Value),
			CreatedAt: d.CreatedAt,
			UpdatedAt: d.UpdatedAt,
		}
	}
	return result, nil
}

// OAuthSet stores OAuth credentials.
func (s *ModuleDataStore) OAuthSet(ctx context.Context, oauthModel gen.OAuth) error {
	existing, err := s.client.OAuth.Query().
		Where(
			oauth.UID(oauthModel.UID),
			oauth.Topic(oauthModel.Topic),
			oauth.Type(oauthModel.Type),
		).
		Only(ctx)
	if err != nil && !gen.IsNotFound(err) {
		return fmt.Errorf("postgres: oauthset query: %w", err)
	}

	if existing != nil {
		u := s.client.OAuth.Update().Where(oauth.IDEQ(existing.ID)).
			SetName(oauthModel.Name).
			SetToken(oauthModel.Token).
			SetUpdatedAt(time.Now())
		if oauthModel.Extra != nil {
			u = u.SetExtra(map[string]any(oauthModel.Extra))
		}
		if oauthModel.RefreshToken != "" {
			u = u.SetRefreshToken(oauthModel.RefreshToken)
		}
		if !oauthModel.ExpiresAt.IsZero() {
			u = u.SetExpiresAt(oauthModel.ExpiresAt)
		}
		if oauthModel.TokenType != "" {
			u = u.SetTokenType(oauthModel.TokenType)
		}
		if oauthModel.Scope != "" {
			u = u.SetScope(oauthModel.Scope)
		}
		_, err = u.Save(ctx)
	} else {
		extra := map[string]any(oauthModel.Extra)
		if extra == nil {
			extra = map[string]any{}
		}
		c := s.client.OAuth.Create().
			SetUID(oauthModel.UID).
			SetTopic(oauthModel.Topic).
			SetName(oauthModel.Name).
			SetType(oauthModel.Type).
			SetToken(oauthModel.Token).
			SetExtra(extra).
			SetCreatedAt(oauthModel.CreatedAt).
			SetUpdatedAt(oauthModel.UpdatedAt)
		if oauthModel.RefreshToken != "" {
			c = c.SetRefreshToken(oauthModel.RefreshToken)
		}
		if !oauthModel.ExpiresAt.IsZero() {
			c = c.SetExpiresAt(oauthModel.ExpiresAt)
		}
		if oauthModel.TokenType != "" {
			c = c.SetTokenType(oauthModel.TokenType)
		}
		if oauthModel.Scope != "" {
			c = c.SetScope(oauthModel.Scope)
		}
		_, err = c.Save(ctx)
	}
	if err != nil {
		return fmt.Errorf("postgres: oauthset save: %w", err)
	}
	return nil
}

// OAuthGet returns OAuth credentials for uid, topic, and type.
func (s *ModuleDataStore) OAuthGet(ctx context.Context, uid types.Uid, topic, t string) (gen.OAuth, error) {
	o, err := s.client.OAuth.Query().
		Where(oauth.UID(uid.String()), oauth.Topic(topic), oauth.Type(t)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return gen.OAuth{}, types.ErrNotFound
		}
		return gen.OAuth{}, fmt.Errorf("postgres: oauthget: %w", err)
	}
	return *o, nil
}

// OAuthGetAvailable lists OAuth credentials of the given type.
func (s *ModuleDataStore) OAuthGetAvailable(ctx context.Context, t string) ([]gen.OAuth, error) {
	oauths, err := s.client.OAuth.Query().Where(oauth.Type(t)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: oauthgetavailable: %w", err)
	}
	result := make([]gen.OAuth, len(oauths))
	for i, o := range oauths {
		result[i] = *o
	}
	return result, nil
}

// FormSet set a form.
func (s *ModuleDataStore) FormSet(ctx context.Context, formId string, formModel gen.Form) error {
	existing, err := s.client.Form.Query().Where(form.FormIDEQ(formId)).Only(ctx)
	if err != nil && !gen.IsNotFound(err) {
		return fmt.Errorf("postgres: formset query: %w", err)
	}

	if existing != nil {
		u := s.client.Form.Update().Where(form.IDEQ(existing.ID)).
			SetUID(formModel.UID).
			SetTopic(formModel.Topic).
			SetState(int(formModel.State)).
			SetUpdatedAt(time.Now())
		if formModel.Schema != nil {
			u = u.SetSchema(map[string]any(formModel.Schema))
		}
		if formModel.Values != nil {
			u = u.SetValues(map[string]any(formModel.Values))
		}
		if formModel.Extra != nil {
			u = u.SetExtra(map[string]any(formModel.Extra))
		}
		_, err = u.Save(ctx)
	} else {
		c := s.client.Form.Create().
			SetFormID(formModel.FormID).
			SetUID(formModel.UID).
			SetTopic(formModel.Topic).
			SetState(int(formModel.State)).
			SetCreatedAt(formModel.CreatedAt).
			SetUpdatedAt(formModel.UpdatedAt)
		if formModel.Schema != nil {
			c = c.SetSchema(map[string]any(formModel.Schema))
		}
		if formModel.Values != nil {
			c = c.SetValues(map[string]any(formModel.Values))
		}
		if formModel.Extra != nil {
			c = c.SetExtra(map[string]any(formModel.Extra))
		}
		_, err = c.Save(ctx)
	}
	if err != nil {
		return fmt.Errorf("postgres: formset save: %w", err)
	}
	return nil
}

// FormGet get a form.
func (s *ModuleDataStore) FormGet(ctx context.Context, formId string) (gen.Form, error) {
	f, err := s.client.Form.Query().Where(form.FormIDEQ(formId)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return gen.Form{}, types.ErrNotFound
		}
		return gen.Form{}, fmt.Errorf("postgres: formget: %w", err)
	}
	return *f, nil
}

// PageSet set a page.
func (s *ModuleDataStore) PageSet(ctx context.Context, pageId string, pageModel gen.Page) error {
	existing, err := s.client.Page.Query().Where(page.PageIDEQ(pageId)).Only(ctx)
	if err != nil && !gen.IsNotFound(err) {
		return fmt.Errorf("postgres: pageset query: %w", err)
	}

	if existing != nil {
		u := s.client.Page.Update().Where(page.IDEQ(existing.ID)).
			SetUID(pageModel.UID).
			SetTopic(pageModel.Topic).
			SetType(string(pageModel.Type)).
			SetState(int(pageModel.State)).
			SetUpdatedAt(time.Now())
		if pageModel.Schema != nil {
			u = u.SetSchema(map[string]any(pageModel.Schema))
		}
		_, err = u.Save(ctx)
	} else {
		c := s.client.Page.Create().
			SetPageID(pageModel.PageID).
			SetUID(pageModel.UID).
			SetTopic(pageModel.Topic).
			SetType(string(pageModel.Type)).
			SetState(int(pageModel.State)).
			SetCreatedAt(pageModel.CreatedAt).
			SetUpdatedAt(pageModel.UpdatedAt)
		if pageModel.Schema != nil {
			c = c.SetSchema(map[string]any(pageModel.Schema))
		}
		_, err = c.Save(ctx)
	}
	if err != nil {
		return fmt.Errorf("postgres: pageset save: %w", err)
	}
	return nil
}

// PageGet get a page.
func (s *ModuleDataStore) PageGet(ctx context.Context, pageId string) (gen.Page, error) {
	p, err := s.client.Page.Query().Where(page.PageIDEQ(pageId)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return gen.Page{}, types.ErrNotFound
		}
		return gen.Page{}, fmt.Errorf("postgres: pageget: %w", err)
	}
	return *p, nil
}

// BehaviorSet set behavior records.
func (s *ModuleDataStore) BehaviorSet(ctx context.Context, behaviorModel gen.Behavior) error {
	existing, err := s.client.Behavior.Query().
		Where(behavior.UID(behaviorModel.UID), behavior.Flag(behaviorModel.Flag)).
		Only(ctx)
	if err != nil && !gen.IsNotFound(err) {
		return fmt.Errorf("postgres: behaviorset query: %w", err)
	}

	if existing != nil {
		u := s.client.Behavior.Update().Where(behavior.IDEQ(existing.ID)).
			SetCount(behaviorModel.Count).
			SetUpdatedAt(time.Now())
		if behaviorModel.Extra != nil {
			u = u.SetExtra(behaviorModel.Extra)
		}
		_, err = u.Save(ctx)
	} else {
		c := s.client.Behavior.Create().
			SetUID(behaviorModel.UID).
			SetFlag(behaviorModel.Flag).
			SetCount(behaviorModel.Count).
			SetCreatedAt(behaviorModel.CreatedAt).
			SetUpdatedAt(behaviorModel.UpdatedAt)
		if behaviorModel.Extra != nil {
			c = c.SetExtra(behaviorModel.Extra)
		}
		_, err = c.Save(ctx)
	}
	if err != nil {
		return fmt.Errorf("postgres: behaviorset save: %w", err)
	}
	return nil
}

// BehaviorGet get behavior records.
func (s *ModuleDataStore) BehaviorGet(ctx context.Context, uid types.Uid, flag string) (gen.Behavior, error) {
	b, err := s.client.Behavior.Query().
		Where(behavior.UID(uid.String()), behavior.FlagEQ(flag)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return gen.Behavior{}, types.ErrNotFound
		}
		return gen.Behavior{}, fmt.Errorf("postgres: behaviorget: %w", err)
	}
	return *b, nil
}

// BehaviorList list behavior records.
func (s *ModuleDataStore) BehaviorList(ctx context.Context, uid types.Uid) ([]*gen.Behavior, error) {
	behaviors, err := s.client.Behavior.Query().
		Where(behavior.UID(uid.String())).
		Order(gen.Asc(behavior.FieldCreatedAt)).
		Limit(queryMaxResults()).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: behaviorlist: %w", err)
	}
	return behaviors, nil
}

// BehaviorIncrease increase behavior records.
func (s *ModuleDataStore) BehaviorIncrease(ctx context.Context, uid types.Uid, flag string, number int) error {
	delta, ok := utils.IntToInt32(number)
	if !ok {
		return fmt.Errorf("postgres: behaviorincrease: count delta out of range")
	}
	u := s.client.Behavior.Update().Where(behavior.UID(uid.String()), behavior.FlagEQ(flag))
	u = u.AddCount(delta).SetUpdatedAt(time.Now())
	_, err := u.Save(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return types.ErrNotFound
		}
		return fmt.Errorf("postgres: behaviorincrease: %w", err)
	}
	return nil
}

// ParameterSet set a parameter.
func (s *ModuleDataStore) ParameterSet(ctx context.Context, flag string, params types.KV, expiredAt time.Time) error {
	existing, err := s.client.Parameter.Query().Where(parameter.FlagEQ(flag)).Only(ctx)
	if err != nil && !gen.IsNotFound(err) {
		return fmt.Errorf("postgres: parameterset query: %w", err)
	}

	if existing != nil {
		_, err = s.client.Parameter.Update().Where(parameter.IDEQ(existing.ID)).
			SetParams(map[string]any(params)).
			SetExpiredAt(expiredAt).
			SetUpdatedAt(time.Now()).
			Save(ctx)
	} else {
		_, err = s.client.Parameter.Create().
			SetFlag(flag).
			SetParams(map[string]any(params)).
			SetExpiredAt(expiredAt).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(ctx)
	}
	if err != nil {
		return fmt.Errorf("postgres: parameterset save: %w", err)
	}
	return nil
}

// ParameterGet get a parameter.
func (s *ModuleDataStore) ParameterGet(ctx context.Context, flag string) (gen.Parameter, error) {
	p, err := s.client.Parameter.Query().Where(parameter.FlagEQ(flag)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return gen.Parameter{}, types.ErrNotFound
		}
		return gen.Parameter{}, fmt.Errorf("postgres: parameterget: %w", err)
	}
	return gen.Parameter{
		ID:        p.ID,
		Flag:      p.Flag,
		Params:    schema.JSON(p.Params),
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
		ExpiredAt: p.ExpiredAt,
	}, nil
}

// ParameterDelete delete a parameter.
func (s *ModuleDataStore) ParameterDelete(ctx context.Context, flag string) error {
	_, err := s.client.Parameter.Delete().Where(parameter.FlagEQ(flag)).Exec(ctx)
	if err != nil {
		return fmt.Errorf("postgres: parameterdelete: %w", err)
	}
	return nil
}

// ListTokens returns tokens.
func (s *ModuleDataStore) ListTokens(ctx context.Context) ([]model.TokenItem, error) {
	rows, err := s.client.Parameter.Query().
		Order(gen.Desc(parameter.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: list tokens: %w", err)
	}

	cutoff := time.Now().Add(-30 * 24 * time.Hour)
	result := make([]model.TokenItem, 0, len(rows))
	for _, r := range rows {
		paramsKV := types.KV(r.Params)
		if _, hasScopes := paramsKV["scopes"]; !hasScopes {
			continue
		}
		if r.ExpiredAt.Before(cutoff) {
			if _, hasUsed := paramsKV["last_used_at"]; !hasUsed {
				continue
			}
		}
		uidStr, ok := paramsKV.String("uid")
		if !ok {
			continue
		}
		var scopes []string
		if raw, ok := paramsKV["scopes"]; ok {
			switch v := raw.(type) {
			case []any:
				for _, item := range v {
					if s, ok := item.(string); ok {
						scopes = append(scopes, s)
					}
				}
			case []string:
				scopes = v
			}
		}
		var lastUsedAt *time.Time
		if usedStr, ok := paramsKV.String("last_used_at"); ok && usedStr != "" {
			if t, err := time.Parse(time.RFC3339Nano, usedStr); err == nil {
				lastUsedAt = &t
			}
		}
		result = append(result, model.TokenItem{
			Token:      r.Flag,
			UID:        types.Uid(uidStr),
			Scopes:     scopes,
			CreatedAt:  r.CreatedAt,
			LastUsedAt: lastUsedAt,
			ExpiredAt:  r.ExpiredAt,
		})
	}
	return result, nil
}

// CreateToken persists a new token.
func (s *ModuleDataStore) CreateToken(ctx context.Context, uid types.Uid, expiresAt time.Time, scopes []string) (string, error) {
	if len(scopes) == 0 {
		return "", fmt.Errorf("postgres: create token: %w", types.Errorf(types.ErrInvalidArgument, "at least one scope is required"))
	}
	token, err := auth.NewToken()
	if err != nil {
		return "", fmt.Errorf("postgres: create token: %w", err)
	}
	params := types.KV{
		"uid":    string(uid),
		"scopes": scopes,
	}
	now := time.Now()
	_, err = s.client.Parameter.Create().
		SetFlag(auth.HashToken(token)).
		SetParams(map[string]any(params)).
		SetExpiredAt(expiresAt).
		SetCreatedAt(now).
		SetUpdatedAt(now).
		Save(ctx)
	if err != nil {
		return "", fmt.Errorf("postgres: create token: %w", err)
	}
	return token, nil
}

// RevokeToken revokes the token.
func (s *ModuleDataStore) RevokeToken(ctx context.Context, flag string) error {
	n, err := s.client.Parameter.Delete().Where(parameter.FlagEQ(flag)).Exec(ctx)
	if err != nil {
		return fmt.Errorf("postgres: revoke token: %w", err)
	}
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// CreateInstruct persists a new instruct.
func (s *ModuleDataStore) CreateInstruct(ctx context.Context, instructModel *gen.Instruct) (int64, error) {
	c := s.client.Instruct.Create().
		SetNo(instructModel.No).
		SetUID(instructModel.UID).
		SetObject(string(instructModel.Object)).
		SetBot(instructModel.Bot).
		SetFlag(instructModel.Flag).
		SetPriority(int(instructModel.Priority)).
		SetState(int(instructModel.State)).
		SetExpireAt(instructModel.ExpireAt).
		SetCreatedAt(instructModel.CreatedAt).
		SetUpdatedAt(instructModel.UpdatedAt)
	if instructModel.Content != nil {
		c = c.SetContent(map[string]any(instructModel.Content))
	}
	u, err := c.Save(ctx)
	if err != nil {
		return 0, fmt.Errorf("postgres: createinstruct: %w", err)
	}
	return u.ID, nil
}

// ListInstruct returns instruct.
func (s *ModuleDataStore) ListInstruct(ctx context.Context, uid types.Uid, isExpire bool, limit int) ([]*gen.Instruct, error) {
	q := s.client.Instruct.Query().Where(instruct.UID(uid.String()))
	if isExpire {
		q = q.Where(instruct.ExpireAtLTE(time.Now()))
	}
	q = q.Order(gen.Asc(instruct.FieldCreatedAt))
	if limit > 0 {
		q = q.Limit(limit)
	} else {
		q = q.Limit(queryMaxResults())
	}
	items, err := q.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: listinstruct: %w", err)
	}
	return items, nil
}

// UpdateInstruct updates the instruct.
func (s *ModuleDataStore) UpdateInstruct(ctx context.Context, instructModel *gen.Instruct) error {
	_, err := s.client.Instruct.Update().
		Where(instruct.NoEQ(instructModel.No)).
		SetState(int(instructModel.State)).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: updateinstruct: %w", err)
	}
	return nil
}

// CreateCounter persists a new counter.
func (s *ModuleDataStore) CreateCounter(ctx context.Context, counterModel *gen.Counter) (int64, error) {
	c, err := s.client.Counter.Create().
		SetUID(counterModel.UID).
		SetTopic(counterModel.Topic).
		SetFlag(counterModel.Flag).
		SetDigit(counterModel.Digit).
		SetStatus(counterModel.Status).
		SetCreatedAt(counterModel.CreatedAt).
		SetUpdatedAt(counterModel.UpdatedAt).
		Save(ctx)
	if err != nil {
		return 0, fmt.Errorf("postgres: createcounter: %w", err)
	}
	return c.ID, nil
}

// IncreaseCounter increases the counter.
func (s *ModuleDataStore) IncreaseCounter(ctx context.Context, id, amount int64) error {
	_, err := s.client.Counter.Update().Where(counter.IDEQ(id)).
		AddDigit(amount).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return types.ErrNotFound
		}
		return fmt.Errorf("postgres: increasecounter: %w", err)
	}
	return s.record(ctx, id, amount)
}

// DecreaseCounter decreases the counter.
func (s *ModuleDataStore) DecreaseCounter(ctx context.Context, id, amount int64) error {
	_, err := s.client.Counter.Update().Where(counter.IDEQ(id)).
		AddDigit(-amount).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return types.ErrNotFound
		}
		return fmt.Errorf("postgres: decreasecounter: %w", err)
	}
	return s.record(ctx, id, -amount)
}

// ListCounter returns counter.
func (s *ModuleDataStore) ListCounter(ctx context.Context, uid types.Uid, topic string) ([]*gen.Counter, error) {
	counters, err := s.client.Counter.Query().
		Where(counter.UID(uid.String()), counter.Topic(topic)).
		Order(gen.Asc(counter.FieldCreatedAt)).
		Limit(queryMaxResults()).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: listcounter: %w", err)
	}
	return counters, nil
}

// GetCounter returns the counter.
func (s *ModuleDataStore) GetCounter(ctx context.Context, id int64) (gen.Counter, error) {
	c, err := s.client.Counter.Query().Where(counter.IDEQ(id)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return gen.Counter{}, types.ErrNotFound
		}
		return gen.Counter{}, fmt.Errorf("postgres: getcounter: %w", err)
	}
	return *c, nil
}

// GetCounterByFlag returns the counter by flag.
func (s *ModuleDataStore) GetCounterByFlag(ctx context.Context, uid types.Uid, topic, flag string) (gen.Counter, error) {
	c, err := s.client.Counter.Query().
		Where(counter.UID(uid.String()), counter.Topic(topic), counter.FlagEQ(flag)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return gen.Counter{}, types.ErrNotFound
		}
		return gen.Counter{}, fmt.Errorf("postgres: getcounterbyflag: %w", err)
	}
	return *c, nil
}

func (s *ModuleDataStore) record(ctx context.Context, id, digit int64) error {
	d, ok := utils.Int64ToInt32(digit)
	if !ok {
		return fmt.Errorf("postgres: counterrecord: digit out of range")
	}
	_, err := s.client.CounterRecord.Create().
		SetCounterID(id).
		SetDigit(d).
		SetCreatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: counterrecord: %w", err)
	}
	return nil
}
