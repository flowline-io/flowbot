package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	entsql "entgo.io/ent/dialect/sql"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/agentknowledge"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/agentmemoryfact"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/agentplan"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/agentsessionsummary"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/agentskill"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/agentskillfile"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/agentsubagent"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/agentsubagenttask"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/agenttodo"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/types"
)

// AgentStore persists chat-agent plans, skills, knowledge, memory, and subagents.
type AgentStore struct {
	client *gen.Client
}

// NewAgentStore creates an AgentStore with the given ent client.
func NewAgentStore(client *gen.Client) *AgentStore {
	return &AgentStore{client: client}
}

// AgentStoreFromDB returns an AgentStore using the global database client.
func AgentStoreFromDB() *AgentStore {
	return NewAgentStore(ClientFromDB())
}

// Client returns the underlying ent client.
func (s *AgentStore) Client() *gen.Client {
	if s == nil {
		return nil
	}
	return s.client
}

// CreateAgentPlan persists a new agent plan.
func (s *AgentStore) CreateAgentPlan(ctx context.Context, plan *gen.AgentPlan) error {
	if plan == nil {
		return errors.New("postgres: nil agent plan")
	}
	builder := s.client.AgentPlan.Create().
		SetFlag(plan.Flag).
		SetSessionID(plan.SessionID).
		SetTitle(plan.Title).
		SetContent(plan.Content).
		SetSourceEntryID(plan.SourceEntryID)
	if !plan.CreatedAt.IsZero() {
		builder = builder.SetCreatedAt(plan.CreatedAt)
	}
	if !plan.UpdatedAt.IsZero() {
		builder = builder.SetUpdatedAt(plan.UpdatedAt)
	}
	_, err := builder.Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: create agent plan: %w", err)
	}
	return nil
}

// GetAgentPlan returns the agent plan.
func (s *AgentStore) GetAgentPlan(ctx context.Context, flag string) (*gen.AgentPlan, error) {
	row, err := s.client.AgentPlan.Query().
		Where(agentplan.FlagEQ(flag)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: get agent plan: %w", err)
	}
	return row, nil
}

// GetAgentPlanInSession returns the agent plan in session.
func (s *AgentStore) GetAgentPlanInSession(ctx context.Context, sessionID, flag string) (*gen.AgentPlan, error) {
	row, err := s.client.AgentPlan.Query().
		Where(agentplan.FlagEQ(flag), agentplan.SessionIDEQ(sessionID)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: get agent plan in session: %w", err)
	}
	return row, nil
}

// ListAgentPlansBySession returns agent plans by session.
func (s *AgentStore) ListAgentPlansBySession(ctx context.Context, sessionID string) ([]*gen.AgentPlan, error) {
	rows, err := s.client.AgentPlan.Query().
		Where(agentplan.SessionIDEQ(sessionID)).
		Order(gen.Desc(agentplan.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: list agent plans: %w", err)
	}
	return rows, nil
}

// ListAgentTodosBySession returns agent todos by session.
func (s *AgentStore) ListAgentTodosBySession(ctx context.Context, sessionID string) ([]*gen.AgentTodo, error) {
	rows, err := s.client.AgentTodo.Query().
		Where(agenttodo.SessionIDEQ(sessionID)).
		Order(gen.Asc(agenttodo.FieldSortOrder), gen.Asc(agenttodo.FieldItemID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: list agent todos: %w", err)
	}
	return rows, nil
}

// ListAgentTodosBySessions returns agent todos by sessions.
func (s *AgentStore) ListAgentTodosBySessions(ctx context.Context, sessionIDs []string) ([]*gen.AgentTodo, error) {
	if len(sessionIDs) == 0 {
		return nil, nil
	}
	rows, err := s.client.AgentTodo.Query().
		Where(agenttodo.SessionIDIn(sessionIDs...)).
		Order(
			gen.Asc(agenttodo.FieldSessionID),
			gen.Asc(agenttodo.FieldSortOrder),
			gen.Asc(agenttodo.FieldItemID),
		).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: list agent todos by sessions: %w", err)
	}
	return rows, nil
}

// ReplaceAgentTodosForSession replaces agent todos for session.
func (s *AgentStore) ReplaceAgentTodosForSession(ctx context.Context, sessionID string, items []*gen.AgentTodo) error {
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("postgres: replace agent todos tx: %w", err)
	}
	if _, err := tx.AgentTodo.Delete().Where(agenttodo.SessionIDEQ(sessionID)).Exec(ctx); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("postgres: delete agent todos: %w", err)
	}
	builders := make([]*gen.AgentTodoCreate, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		builder := tx.AgentTodo.Create().
			SetFlag(item.Flag).
			SetSessionID(sessionID).
			SetItemID(item.ItemID).
			SetContent(item.Content).
			SetStatus(item.Status).
			SetSortOrder(item.SortOrder)
		if !item.CreatedAt.IsZero() {
			builder = builder.SetCreatedAt(item.CreatedAt)
		}
		if !item.UpdatedAt.IsZero() {
			builder = builder.SetUpdatedAt(item.UpdatedAt)
		}
		builders = append(builders, builder)
	}
	if len(builders) > 0 {
		if _, err := tx.AgentTodo.CreateBulk(builders...).Save(ctx); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("postgres: create agent todos: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("postgres: commit replace agent todos: %w", err)
	}
	return nil
}

// MergeAgentTodosForSession merges agent todos for session.
func (s *AgentStore) MergeAgentTodosForSession(ctx context.Context, sessionID string, items []*gen.AgentTodo) error {
	builders := make([]*gen.AgentTodoCreate, 0, len(items))
	now := time.Now()
	for _, item := range items {
		if item == nil {
			continue
		}
		builder := s.client.AgentTodo.Create().
			SetFlag(item.Flag).
			SetSessionID(sessionID).
			SetItemID(item.ItemID).
			SetContent(item.Content).
			SetStatus(item.Status).
			SetSortOrder(item.SortOrder).
			SetUpdatedAt(now)
		if !item.CreatedAt.IsZero() {
			builder = builder.SetCreatedAt(item.CreatedAt)
		}
		if !item.UpdatedAt.IsZero() {
			builder = builder.SetUpdatedAt(item.UpdatedAt)
		}
		builders = append(builders, builder)
	}
	if len(builders) == 0 {
		return nil
	}
	err := s.client.AgentTodo.CreateBulk(builders...).
		OnConflictColumns(agenttodo.FieldSessionID, agenttodo.FieldItemID).
		Update(func(u *gen.AgentTodoUpsert) {
			u.UpdateContent()
			u.UpdateStatus()
			u.UpdateSortOrder()
			u.UpdateUpdatedAt()
		}).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("postgres: merge agent todos: %w", err)
	}
	return nil
}

// ListAgentSkills returns agent skills.
func (s *AgentStore) ListAgentSkills(ctx context.Context, enabledOnly bool) ([]*gen.AgentSkill, error) {
	query := s.client.AgentSkill.Query()
	if enabledOnly {
		query = query.Where(agentskill.EnabledEQ(true))
	}
	rows, err := query.Order(gen.Asc(agentskill.FieldName)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: list agent skills: %w", err)
	}
	return rows, nil
}

// GetAgentSkillsMaxUpdatedAt returns the agent skills max updated at.
func (s *AgentStore) GetAgentSkillsMaxUpdatedAt(ctx context.Context) (time.Time, error) {
	var maxUpdated time.Time
	row, err := s.client.AgentSkill.Query().
		Where(agentskill.EnabledEQ(true)).
		Order(gen.Desc(agentskill.FieldUpdatedAt)).
		First(ctx)
	if err != nil {
		if !gen.IsNotFound(err) {
			return time.Time{}, fmt.Errorf("postgres: agent skills max updated_at: %w", err)
		}
	} else {
		maxUpdated = row.UpdatedAt
	}
	fileRow, err := s.client.AgentSkillFile.Query().
		Order(gen.Desc(agentskillfile.FieldUpdatedAt)).
		First(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return maxUpdated, nil
		}
		return time.Time{}, fmt.Errorf("postgres: agent skill files max updated_at: %w", err)
	}
	if fileRow.UpdatedAt.After(maxUpdated) {
		maxUpdated = fileRow.UpdatedAt
	}
	return maxUpdated, nil
}

// GetAgentSkillByName returns the agent skill by name.
func (s *AgentStore) GetAgentSkillByName(ctx context.Context, name string) (*gen.AgentSkill, error) {
	row, err := s.client.AgentSkill.Query().
		Where(agentskill.NameEQ(name), agentskill.EnabledEQ(true)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: get agent skill: %w", err)
	}
	return row, nil
}

// GetAgentSkillByFlag returns the agent skill by flag.
func (s *AgentStore) GetAgentSkillByFlag(ctx context.Context, flag string) (*gen.AgentSkill, error) {
	row, err := s.client.AgentSkill.Query().
		Where(agentskill.FlagEQ(flag)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: get agent skill by flag: %w", err)
	}
	return row, nil
}

// CreateAgentSkill persists a new agent skill.
func (s *AgentStore) CreateAgentSkill(ctx context.Context, skill *gen.AgentSkill) error {
	if skill == nil {
		return errors.New("postgres: nil agent skill")
	}
	builder := s.client.AgentSkill.Create().
		SetFlag(skill.Flag).
		SetName(skill.Name).
		SetDescription(skill.Description).
		SetContent(skill.Content).
		SetBaseDir(skill.BaseDir).
		SetSource(skill.Source).
		SetEnabled(skill.Enabled).
		SetDisableModelInvocation(skill.DisableModelInvocation)
	if !skill.CreatedAt.IsZero() {
		builder = builder.SetCreatedAt(skill.CreatedAt)
	}
	if !skill.UpdatedAt.IsZero() {
		builder = builder.SetUpdatedAt(skill.UpdatedAt)
	}
	_, err := builder.Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: create agent skill: %w", err)
	}
	return nil
}

// UpdateAgentSkill updates the agent skill.
func (s *AgentStore) UpdateAgentSkill(ctx context.Context, skill *gen.AgentSkill) error {
	if skill == nil {
		return errors.New("postgres: nil agent skill")
	}
	n, err := s.client.AgentSkill.Update().
		Where(agentskill.FlagEQ(skill.Flag)).
		SetName(skill.Name).
		SetDescription(skill.Description).
		SetContent(skill.Content).
		SetBaseDir(skill.BaseDir).
		SetSource(skill.Source).
		SetEnabled(skill.Enabled).
		SetDisableModelInvocation(skill.DisableModelInvocation).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: update agent skill: %w", err)
	}
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// DeleteAgentSkill deletes the agent skill.
func (s *AgentStore) DeleteAgentSkill(ctx context.Context, flag string) error {
	tx, err := s.client.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("postgres: begin delete agent skill tx: %w", err)
	}
	if _, err := tx.AgentSkillFile.Delete().
		Where(agentskillfile.SkillFlagEQ(flag)).
		Exec(ctx); err != nil {
		if rerr := tx.Rollback(); rerr != nil {
			return fmt.Errorf("postgres: delete agent skill files: %w (rollback: %v)", err, rerr)
		}
		return fmt.Errorf("postgres: delete agent skill files: %w", err)
	}
	n, err := tx.AgentSkill.Delete().
		Where(agentskill.FlagEQ(flag)).
		Exec(ctx)
	if err != nil {
		if rerr := tx.Rollback(); rerr != nil {
			return fmt.Errorf("postgres: delete agent skill: %w (rollback: %v)", err, rerr)
		}
		return fmt.Errorf("postgres: delete agent skill: %w", err)
	}
	if n == 0 {
		if rerr := tx.Rollback(); rerr != nil {
			return types.ErrNotFound
		}
		return types.ErrNotFound
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("postgres: commit delete agent skill: %w", err)
	}
	return nil
}

// ListAgentSkillFiles returns agent skill files.
func (s *AgentStore) ListAgentSkillFiles(ctx context.Context, skillFlag string) ([]*gen.AgentSkillFile, error) {
	rows, err := s.client.AgentSkillFile.Query().
		Where(agentskillfile.SkillFlagEQ(skillFlag)).
		Order(gen.Asc(agentskillfile.FieldPath)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: list agent skill files: %w", err)
	}
	return rows, nil
}

// GetAgentSkillFile returns the agent skill file.
func (s *AgentStore) GetAgentSkillFile(ctx context.Context, skillFlag, path string) (*gen.AgentSkillFile, error) {
	row, err := s.client.AgentSkillFile.Query().
		Where(
			agentskillfile.SkillFlagEQ(skillFlag),
			agentskillfile.PathEQ(path),
		).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: get agent skill file: %w", err)
	}
	return row, nil
}

// CreateAgentSkillFile persists a new agent skill file.
func (s *AgentStore) CreateAgentSkillFile(ctx context.Context, file *gen.AgentSkillFile) error {
	if file == nil {
		return errors.New("postgres: nil agent skill file")
	}
	builder := s.client.AgentSkillFile.Create().
		SetSkillFlag(file.SkillFlag).
		SetPath(file.Path).
		SetContent(file.Content)
	if !file.CreatedAt.IsZero() {
		builder = builder.SetCreatedAt(file.CreatedAt)
	}
	if !file.UpdatedAt.IsZero() {
		builder = builder.SetUpdatedAt(file.UpdatedAt)
	}
	_, err := builder.Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: create agent skill file: %w", err)
	}
	return nil
}

// UpdateAgentSkillFile updates the agent skill file.
func (s *AgentStore) UpdateAgentSkillFile(ctx context.Context, file *gen.AgentSkillFile) error {
	if file == nil {
		return errors.New("postgres: nil agent skill file")
	}
	n, err := s.client.AgentSkillFile.Update().
		Where(
			agentskillfile.SkillFlagEQ(file.SkillFlag),
			agentskillfile.PathEQ(file.Path),
		).
		SetContent(file.Content).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: update agent skill file: %w", err)
	}
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// DeleteAgentSkillFile deletes the agent skill file.
func (s *AgentStore) DeleteAgentSkillFile(ctx context.Context, skillFlag, path string) error {
	n, err := s.client.AgentSkillFile.Delete().
		Where(
			agentskillfile.SkillFlagEQ(skillFlag),
			agentskillfile.PathEQ(path),
		).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("postgres: delete agent skill file: %w", err)
	}
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// DeleteAgentSkillFilesByFlag deletes the agent skill files by flag.
func (s *AgentStore) DeleteAgentSkillFilesByFlag(ctx context.Context, skillFlag string) error {
	_, err := s.client.AgentSkillFile.Delete().
		Where(agentskillfile.SkillFlagEQ(skillFlag)).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("postgres: delete agent skill files by flag: %w", err)
	}
	return nil
}

// ListAgentKnowledge returns agent knowledge.
func (s *AgentStore) ListAgentKnowledge(ctx context.Context, filter AgentKnowledgeListFilter) ([]*gen.AgentKnowledge, error) {
	query := s.client.AgentKnowledge.Query()
	q := strings.TrimSpace(filter.Q)
	if q != "" {
		query = query.Where(agentknowledge.Or(
			agentknowledge.PathContainsFold(q),
			agentknowledge.TitleContainsFold(q),
		))
	}
	rows, err := query.Order(gen.Desc(agentknowledge.FieldUpdatedAt)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: list agent knowledge: %w", err)
	}
	return rows, nil
}

// SearchAgentKnowledge searches agent knowledge.
func (s *AgentStore) SearchAgentKnowledge(ctx context.Context, params AgentKnowledgeSearchParams) ([]*gen.AgentKnowledge, error) {
	queryText := strings.TrimSpace(params.Query)
	prefix := strings.TrimSpace(params.PathPrefix)
	tag := strings.TrimSpace(params.Tag)
	if queryText == "" && prefix == "" {
		return nil, fmt.Errorf("postgres: search agent knowledge: query or path_prefix is required")
	}
	limit := normalizeAgentKnowledgeSearchLimit(params.Limit)

	query := s.client.AgentKnowledge.Query()
	if prefix != "" {
		query = query.Where(agentknowledge.PathHasPrefix(prefix))
	}
	if queryText != "" {
		query = query.Where(agentknowledge.Or(
			agentknowledge.PathContainsFold(queryText),
			agentknowledge.TitleContainsFold(queryText),
			agentknowledge.SummaryContainsFold(queryText),
			agentknowledge.ContentContainsFold(queryText),
			agentKnowledgeTagsContainsFold(queryText),
		))
	}
	// Fetch a wider match window so relevance ranking can reorder before Limit.
	rows, err := query.Order(gen.Desc(agentknowledge.FieldUpdatedAt)).
		Limit(agentKnowledgeSearchFetchLimit(limit)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: search agent knowledge: %w", err)
	}
	filtered := filterAgentKnowledgeByTag(rows, tag)
	if queryText != "" {
		sortAgentKnowledgeByRelevance(filtered, queryText)
	}
	if len(filtered) > limit {
		filtered = filtered[:limit]
	}
	return filtered, nil
}

func normalizeAgentKnowledgeSearchLimit(limit int) int {
	if limit <= 0 {
		return 10
	}
	if limit > 50 {
		return 50
	}
	return limit
}

func agentKnowledgeSearchFetchLimit(limit int) int {
	fetchLimit := limit * 10
	if fetchLimit < 100 {
		return 100
	}
	if fetchLimit > 500 {
		return 500
	}
	return fetchLimit
}

func filterAgentKnowledgeByTag(rows []*gen.AgentKnowledge, tag string) []*gen.AgentKnowledge {
	if tag == "" {
		return rows
	}
	filtered := make([]*gen.AgentKnowledge, 0, len(rows))
	for _, row := range rows {
		if knowledgeHasTag(row.Tags, tag) {
			filtered = append(filtered, row)
		}
	}
	return filtered
}

// agentKnowledgeTagsContainsFold matches queryText against the JSON tags array
// as text (case-insensitive). Works for both Postgres jsonb and SQLite JSON.
func agentKnowledgeTagsContainsFold(queryText string) func(*entsql.Selector) {
	needle := "%" + strings.ToLower(strings.TrimSpace(queryText)) + "%"
	return func(s *entsql.Selector) {
		s.Where(entsql.P(func(b *entsql.Builder) {
			b.WriteString("LOWER(CAST(")
			b.Ident(s.C(agentknowledge.FieldTags))
			b.WriteString(" AS TEXT)) LIKE ")
			b.Arg(needle)
		}))
	}
}

func knowledgeHasTag(tags []string, want string) bool {
	for _, tag := range tags {
		if strings.EqualFold(strings.TrimSpace(tag), want) {
			return true
		}
	}
	return false
}

func sortAgentKnowledgeByRelevance(rows []*gen.AgentKnowledge, query string) {
	q := strings.ToLower(query)
	slices.SortStableFunc(rows, func(a, b *gen.AgentKnowledge) int {
		ra := knowledgeRelevanceRank(a, q)
		rb := knowledgeRelevanceRank(b, q)
		if ra != rb {
			return ra - rb
		}
		if a.UpdatedAt.After(b.UpdatedAt) {
			return -1
		}
		if a.UpdatedAt.Before(b.UpdatedAt) {
			return 1
		}
		return 0
	})
}

func knowledgeRelevanceRank(row *gen.AgentKnowledge, qLower string) int {
	title := strings.ToLower(row.Title)
	path := strings.ToLower(row.Path)
	switch {
	case strings.Contains(title, qLower):
		return 0
	case strings.Contains(path, qLower):
		return 1
	case knowledgeTagsContainFold(row.Tags, qLower):
		return 2
	default:
		return 3
	}
}

func knowledgeTagsContainFold(tags []string, qLower string) bool {
	for _, tag := range tags {
		if strings.Contains(strings.ToLower(strings.TrimSpace(tag)), qLower) {
			return true
		}
	}
	return false
}

// GetAgentKnowledgeByPath returns the agent knowledge by path.
func (s *AgentStore) GetAgentKnowledgeByPath(ctx context.Context, path string) (*gen.AgentKnowledge, error) {
	row, err := s.client.AgentKnowledge.Query().
		Where(agentknowledge.PathEQ(path)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: get agent knowledge by path: %w", err)
	}
	return row, nil
}

// GetAgentKnowledgeByID returns the agent knowledge with the given id.
func (s *AgentStore) GetAgentKnowledgeByID(ctx context.Context, id int64) (*gen.AgentKnowledge, error) {
	row, err := s.client.AgentKnowledge.Get(ctx, id)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: get agent knowledge by id: %w", err)
	}
	return row, nil
}

// CreateAgentKnowledge persists a new agent knowledge.
func (s *AgentStore) CreateAgentKnowledge(ctx context.Context, doc *gen.AgentKnowledge) error {
	if doc == nil {
		return errors.New("postgres: nil agent knowledge")
	}
	tags := doc.Tags
	if tags == nil {
		tags = []string{}
	}
	builder := s.client.AgentKnowledge.Create().
		SetPath(doc.Path).
		SetTitle(doc.Title).
		SetTags(tags).
		SetSummary(doc.Summary).
		SetContent(doc.Content)
	if !doc.CreatedAt.IsZero() {
		builder = builder.SetCreatedAt(doc.CreatedAt)
	}
	if !doc.UpdatedAt.IsZero() {
		builder = builder.SetUpdatedAt(doc.UpdatedAt)
	}
	row, err := builder.Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: create agent knowledge: %w", err)
	}
	doc.ID = row.ID
	doc.CreatedAt = row.CreatedAt
	doc.UpdatedAt = row.UpdatedAt
	return nil
}

// UpdateAgentKnowledge updates the agent knowledge.
func (s *AgentStore) UpdateAgentKnowledge(ctx context.Context, doc *gen.AgentKnowledge) error {
	if doc == nil {
		return errors.New("postgres: nil agent knowledge")
	}
	tags := doc.Tags
	if tags == nil {
		tags = []string{}
	}
	n, err := s.client.AgentKnowledge.Update().
		Where(agentknowledge.IDEQ(doc.ID)).
		SetPath(doc.Path).
		SetTitle(doc.Title).
		SetTags(tags).
		SetSummary(doc.Summary).
		SetContent(doc.Content).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: update agent knowledge: %w", err)
	}
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// DeleteAgentKnowledge deletes the agent knowledge.
func (s *AgentStore) DeleteAgentKnowledge(ctx context.Context, id int64) error {
	n, err := s.client.AgentKnowledge.Delete().
		Where(agentknowledge.IDEQ(id)).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("postgres: delete agent knowledge: %w", err)
	}
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// UpsertAgentMemoryFact inserts or updates agent memory fact.
func (s *AgentStore) UpsertAgentMemoryFact(ctx context.Context, fact AgentMemoryFactUpsert) (*gen.AgentMemoryFact, error) {
	scope := strings.TrimSpace(fact.Scope)
	key := strings.TrimSpace(fact.Key)
	value := strings.TrimSpace(fact.Value)
	if scope == "" || key == "" {
		return nil, fmt.Errorf("postgres: upsert agent memory fact: scope and key are required")
	}
	if value == "" {
		return nil, fmt.Errorf("postgres: upsert agent memory fact: value is required")
	}
	existing, err := s.client.AgentMemoryFact.Query().
		Where(
			agentmemoryfact.ScopeEQ(scope),
			agentmemoryfact.KeyEQ(key),
		).
		Only(ctx)
	if err != nil && !gen.IsNotFound(err) {
		return nil, fmt.Errorf("postgres: upsert agent memory fact lookup: %w", err)
	}
	now := time.Now()
	if gen.IsNotFound(err) {
		row, createErr := s.client.AgentMemoryFact.Create().
			SetScope(scope).
			SetKey(key).
			SetValue(value).
			SetPinned(fact.Pinned).
			SetCreatedAt(now).
			SetUpdatedAt(now).
			Save(ctx)
		if createErr != nil {
			return nil, fmt.Errorf("postgres: create agent memory fact: %w", createErr)
		}
		return row, nil
	}
	row, updateErr := existing.Update().
		SetValue(value).
		SetPinned(fact.Pinned).
		SetUpdatedAt(now).
		Save(ctx)
	if updateErr != nil {
		return nil, fmt.Errorf("postgres: update agent memory fact: %w", updateErr)
	}
	return row, nil
}

// GetAgentMemoryFact returns the agent memory fact.
func (s *AgentStore) GetAgentMemoryFact(ctx context.Context, scope, key string) (*gen.AgentMemoryFact, error) {
	row, err := s.client.AgentMemoryFact.Query().
		Where(
			agentmemoryfact.ScopeEQ(strings.TrimSpace(scope)),
			agentmemoryfact.KeyEQ(strings.TrimSpace(key)),
		).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: get agent memory fact: %w", err)
	}
	return row, nil
}

// ListAgentMemoryFacts returns agent memory facts.
func (s *AgentStore) ListAgentMemoryFacts(ctx context.Context, scope string) ([]*gen.AgentMemoryFact, error) {
	rows, err := s.client.AgentMemoryFact.Query().
		Where(agentmemoryfact.ScopeEQ(strings.TrimSpace(scope))).
		Order(gen.Desc(agentmemoryfact.FieldPinned), gen.Desc(agentmemoryfact.FieldUpdatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: list agent memory facts: %w", err)
	}
	return rows, nil
}

// DeleteAgentMemoryFact deletes the agent memory fact.
func (s *AgentStore) DeleteAgentMemoryFact(ctx context.Context, scope, key string) error {
	n, err := s.client.AgentMemoryFact.Delete().
		Where(
			agentmemoryfact.ScopeEQ(strings.TrimSpace(scope)),
			agentmemoryfact.KeyEQ(strings.TrimSpace(key)),
		).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("postgres: delete agent memory fact: %w", err)
	}
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// ListInjectableAgentMemoryFacts returns injectable agent memory facts.
func (s *AgentStore) ListInjectableAgentMemoryFacts(ctx context.Context, params AgentMemoryInjectableParams) ([]*gen.AgentMemoryFact, error) {
	maxCount := params.MaxCount
	if maxCount <= 0 {
		maxCount = 30
	}
	maxChars := params.MaxChars
	if maxChars <= 0 {
		maxChars = 4000
	}
	rows, err := s.ListAgentMemoryFacts(ctx, params.Scope)
	if err != nil {
		return nil, err
	}
	out := make([]*gen.AgentMemoryFact, 0, maxCount)
	used := 0
	for _, row := range rows {
		if len(out) >= maxCount {
			break
		}
		lineLen := len(row.Key) + len(row.Value) + 3
		if used+lineLen > maxChars {
			break
		}
		out = append(out, row)
		used += lineLen
	}
	return out, nil
}

// GetAgentMemoryFactsFingerprint returns the agent memory facts fingerprint.
func (s *AgentStore) GetAgentMemoryFactsFingerprint(ctx context.Context, scope string) (AgentMemoryFactsFingerprint, error) {
	rows, err := s.ListAgentMemoryFacts(ctx, scope)
	if err != nil {
		return AgentMemoryFactsFingerprint{}, err
	}
	h := sha256.New()
	var maxUpdated time.Time
	for _, row := range rows {
		if _, err := fmt.Fprintf(h, "%s\x00%s\x00%t\x00%s\x00%d\n", row.Key, row.Value, row.Pinned, row.UpdatedAt.UTC().Format(time.RFC3339Nano), row.ID); err != nil {
			return AgentMemoryFactsFingerprint{}, fmt.Errorf("postgres: agent memory facts fingerprint: %w", err)
		}
		if row.UpdatedAt.After(maxUpdated) {
			maxUpdated = row.UpdatedAt
		}
	}
	return AgentMemoryFactsFingerprint{
		Count:        len(rows),
		MaxUpdatedAt: maxUpdated,
		ContentHash:  hex.EncodeToString(h.Sum(nil)),
	}, nil
}

// UpsertAgentSessionSummaryPending inserts or updates agent session summary pending.
func (s *AgentStore) UpsertAgentSessionSummaryPending(ctx context.Context, sessionFlag, scope, title string) (*gen.AgentSessionSummary, error) {
	flag := strings.TrimSpace(sessionFlag)
	scope = strings.TrimSpace(scope)
	if flag == "" || scope == "" {
		return nil, fmt.Errorf("postgres: upsert session summary pending: session_flag and scope are required")
	}
	existing, err := s.client.AgentSessionSummary.Query().
		Where(agentsessionsummary.SessionFlagEQ(flag)).
		Only(ctx)
	if err != nil && !gen.IsNotFound(err) {
		return nil, fmt.Errorf("postgres: upsert session summary pending lookup: %w", err)
	}
	now := time.Now()
	if gen.IsNotFound(err) {
		row, createErr := s.client.AgentSessionSummary.Create().
			SetSessionFlag(flag).
			SetScope(scope).
			SetTitle(strings.TrimSpace(title)).
			SetSummary("").
			SetStatus(schema.AgentSessionSummaryPending).
			SetError("").
			SetClaimToken("").
			SetCreatedAt(now).
			SetUpdatedAt(now).
			Save(ctx)
		if createErr != nil {
			return nil, fmt.Errorf("postgres: create session summary pending: %w", createErr)
		}
		return row, nil
	}
	upd := existing.Update().
		SetScope(scope).
		SetStatus(schema.AgentSessionSummaryPending).
		SetError("").
		SetClaimToken("").
		ClearClaimedAt().
		SetUpdatedAt(now)
	if t := strings.TrimSpace(title); t != "" {
		upd = upd.SetTitle(t)
	}
	row, updateErr := upd.Save(ctx)
	if updateErr != nil {
		return nil, fmt.Errorf("postgres: update session summary pending: %w", updateErr)
	}
	return row, nil
}

// ClaimAgentSessionSummaryPending claims agent session summary pending.
func (s *AgentStore) ClaimAgentSessionSummaryPending(ctx context.Context, claimToken string) (*gen.AgentSessionSummary, error) {
	token := strings.TrimSpace(claimToken)
	if token == "" {
		return nil, fmt.Errorf("postgres: claim session summary: claim_token is required")
	}
	row, err := s.client.AgentSessionSummary.Query().
		Where(
			agentsessionsummary.StatusEQ(schema.AgentSessionSummaryPending),
			agentsessionsummary.ClaimTokenEQ(""),
		).
		Order(gen.Asc(agentsessionsummary.FieldUpdatedAt)).
		First(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: claim session summary lookup: %w", err)
	}
	now := time.Now()
	n, err := s.client.AgentSessionSummary.Update().
		Where(
			agentsessionsummary.IDEQ(row.ID),
			agentsessionsummary.StatusEQ(schema.AgentSessionSummaryPending),
			agentsessionsummary.ClaimTokenEQ(""),
		).
		SetClaimToken(token).
		SetClaimedAt(now).
		SetUpdatedAt(now).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: claim session summary: %w", err)
	}
	if n == 0 {
		return nil, types.ErrNotFound
	}
	return s.GetAgentSessionSummaryBySession(ctx, row.SessionFlag)
}

// MarkAgentSessionSummaryReady marks agent session summary ready.
func (s *AgentStore) MarkAgentSessionSummaryReady(ctx context.Context, sessionFlag, claimToken, title, summary string) error {
	flag := strings.TrimSpace(sessionFlag)
	token := strings.TrimSpace(claimToken)
	if flag == "" || token == "" {
		return fmt.Errorf("postgres: mark session summary ready: session_flag and claim_token are required")
	}
	n, err := s.client.AgentSessionSummary.Update().
		Where(
			agentsessionsummary.SessionFlagEQ(flag),
			agentsessionsummary.ClaimTokenEQ(token),
			agentsessionsummary.StatusEQ(schema.AgentSessionSummaryPending),
		).
		SetTitle(strings.TrimSpace(title)).
		SetSummary(summary).
		SetStatus(schema.AgentSessionSummaryReady).
		SetError("").
		SetClaimToken("").
		ClearClaimedAt().
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: mark session summary ready: %w", err)
	}
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// MarkAgentSessionSummaryFailed marks agent session summary failed.
func (s *AgentStore) MarkAgentSessionSummaryFailed(ctx context.Context, sessionFlag, claimToken, errMsg string) error {
	flag := strings.TrimSpace(sessionFlag)
	token := strings.TrimSpace(claimToken)
	if flag == "" || token == "" {
		return fmt.Errorf("postgres: mark session summary failed: session_flag and claim_token are required")
	}
	n, err := s.client.AgentSessionSummary.Update().
		Where(
			agentsessionsummary.SessionFlagEQ(flag),
			agentsessionsummary.ClaimTokenEQ(token),
			agentsessionsummary.StatusEQ(schema.AgentSessionSummaryPending),
		).
		SetStatus(schema.AgentSessionSummaryFailed).
		SetError(errMsg).
		SetClaimToken("").
		ClearClaimedAt().
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: mark session summary failed: %w", err)
	}
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// GetAgentSessionSummaryBySession returns the agent session summary by session.
func (s *AgentStore) GetAgentSessionSummaryBySession(ctx context.Context, sessionFlag string) (*gen.AgentSessionSummary, error) {
	row, err := s.client.AgentSessionSummary.Query().
		Where(agentsessionsummary.SessionFlagEQ(strings.TrimSpace(sessionFlag))).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: get session summary: %w", err)
	}
	return row, nil
}

// SearchAgentSessionSummaries searches agent session summaries.
func (s *AgentStore) SearchAgentSessionSummaries(ctx context.Context, params AgentSessionSummarySearchParams) ([]*gen.AgentSessionSummary, error) {
	queryText := strings.TrimSpace(params.Query)
	if queryText == "" {
		return nil, fmt.Errorf("postgres: search session summaries: query is required")
	}
	limit := normalizeAgentKnowledgeSearchLimit(params.Limit)
	query := s.client.AgentSessionSummary.Query().
		Where(
			agentsessionsummary.StatusEQ(schema.AgentSessionSummaryReady),
			agentsessionsummary.Or(
				agentsessionsummary.TitleContainsFold(queryText),
				agentsessionsummary.SummaryContainsFold(queryText),
			),
		)
	if scope := strings.TrimSpace(params.Scope); scope != "" {
		query = query.Where(agentsessionsummary.ScopeEQ(scope))
	}
	rows, err := query.Order(gen.Desc(agentsessionsummary.FieldUpdatedAt)).
		Limit(agentKnowledgeSearchFetchLimit(limit)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: search session summaries: %w", err)
	}
	sortAgentSessionSummariesByRelevance(rows, queryText)
	if len(rows) > limit {
		rows = rows[:limit]
	}
	return rows, nil
}

func sortAgentSessionSummariesByRelevance(rows []*gen.AgentSessionSummary, query string) {
	q := strings.ToLower(query)
	slices.SortStableFunc(rows, func(a, b *gen.AgentSessionSummary) int {
		ra := sessionSummaryRelevanceRank(a, q)
		rb := sessionSummaryRelevanceRank(b, q)
		if ra != rb {
			return ra - rb
		}
		if a.UpdatedAt.After(b.UpdatedAt) {
			return -1
		}
		if a.UpdatedAt.Before(b.UpdatedAt) {
			return 1
		}
		return 0
	})
}

func sessionSummaryRelevanceRank(row *gen.AgentSessionSummary, qLower string) int {
	if strings.Contains(strings.ToLower(row.Title), qLower) {
		return 0
	}
	return 1
}

// ListAgentSessionSummaries returns agent session summaries.
func (s *AgentStore) ListAgentSessionSummaries(ctx context.Context, filter AgentSessionSummaryListFilter) ([]*gen.AgentSessionSummary, error) {
	query := s.client.AgentSessionSummary.Query()
	if scope := strings.TrimSpace(filter.Scope); scope != "" {
		query = query.Where(agentsessionsummary.ScopeEQ(scope))
	}
	if status := strings.TrimSpace(filter.Status); status != "" {
		query = query.Where(agentsessionsummary.StatusEQ(status))
	}
	if q := strings.TrimSpace(filter.Q); q != "" {
		query = query.Where(agentsessionsummary.Or(
			agentsessionsummary.TitleContainsFold(q),
			agentsessionsummary.SummaryContainsFold(q),
		))
	}
	rows, err := query.Order(gen.Desc(agentsessionsummary.FieldUpdatedAt)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: list session summaries: %w", err)
	}
	return rows, nil
}

// RequeueStaleAgentSessionSummaryPending requeues stale agent session summary pending.
func (s *AgentStore) RequeueStaleAgentSessionSummaryPending(ctx context.Context, olderThan time.Duration) (int, error) {
	if olderThan <= 0 {
		olderThan = 10 * time.Minute
	}
	cutoff := time.Now().Add(-olderThan)
	n, err := s.client.AgentSessionSummary.Update().
		Where(
			agentsessionsummary.StatusEQ(schema.AgentSessionSummaryPending),
			agentsessionsummary.ClaimTokenNEQ(""),
			agentsessionsummary.ClaimedAtLT(cutoff),
		).
		SetClaimToken("").
		ClearClaimedAt().
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return 0, fmt.Errorf("postgres: requeue stale session summaries: %w", err)
	}
	return n, nil
}

// ListAgentSubagents returns agent subagents.
func (s *AgentStore) ListAgentSubagents(ctx context.Context, enabledOnly bool) ([]*gen.AgentSubagent, error) {
	query := s.client.AgentSubagent.Query()
	if enabledOnly {
		query = query.Where(agentsubagent.EnabledEQ(true))
	}
	rows, err := query.Order(gen.Asc(agentsubagent.FieldName)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: list agent subagents: %w", err)
	}
	return rows, nil
}

// GetAgentSubagentsMaxUpdatedAt returns the agent subagents max updated at.
func (s *AgentStore) GetAgentSubagentsMaxUpdatedAt(ctx context.Context) (time.Time, error) {
	row, err := s.client.AgentSubagent.Query().
		Where(agentsubagent.EnabledEQ(true)).
		Order(gen.Desc(agentsubagent.FieldUpdatedAt)).
		First(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return time.Time{}, nil
		}
		return time.Time{}, fmt.Errorf("postgres: agent subagents max updated_at: %w", err)
	}
	return row.UpdatedAt, nil
}

// GetAgentSubagentByName returns the agent subagent by name.
func (s *AgentStore) GetAgentSubagentByName(ctx context.Context, name string) (*gen.AgentSubagent, error) {
	row, err := s.client.AgentSubagent.Query().
		Where(agentsubagent.NameEQ(name), agentsubagent.EnabledEQ(true)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: get agent subagent: %w", err)
	}
	return row, nil
}

// GetAgentSubagentByFlag returns the agent subagent by flag.
func (s *AgentStore) GetAgentSubagentByFlag(ctx context.Context, flag string) (*gen.AgentSubagent, error) {
	row, err := s.client.AgentSubagent.Query().
		Where(agentsubagent.FlagEQ(flag)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: get agent subagent by flag: %w", err)
	}
	return row, nil
}

// CreateAgentSubagent persists a new agent subagent.
func (s *AgentStore) CreateAgentSubagent(ctx context.Context, subagent *gen.AgentSubagent) error {
	if subagent == nil {
		return errors.New("postgres: nil agent subagent")
	}
	builder := s.client.AgentSubagent.Create().
		SetFlag(subagent.Flag).
		SetName(subagent.Name).
		SetDescription(subagent.Description).
		SetSystemPrompt(subagent.SystemPrompt).
		SetTools(subagent.Tools).
		SetSkills(subagent.Skills).
		SetModel(subagent.Model).
		SetSource(subagent.Source).
		SetEnabled(subagent.Enabled)
	if !subagent.CreatedAt.IsZero() {
		builder = builder.SetCreatedAt(subagent.CreatedAt)
	}
	if !subagent.UpdatedAt.IsZero() {
		builder = builder.SetUpdatedAt(subagent.UpdatedAt)
	}
	row, err := builder.Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: create agent subagent: %w", err)
	}
	subagent.ID = row.ID
	return nil
}

// UpdateAgentSubagent updates the agent subagent.
func (s *AgentStore) UpdateAgentSubagent(ctx context.Context, subagent *gen.AgentSubagent) error {
	if subagent == nil {
		return errors.New("postgres: nil agent subagent")
	}
	n, err := s.client.AgentSubagent.Update().
		Where(agentsubagent.FlagEQ(subagent.Flag)).
		SetName(subagent.Name).
		SetDescription(subagent.Description).
		SetSystemPrompt(subagent.SystemPrompt).
		SetTools(subagent.Tools).
		SetSkills(subagent.Skills).
		SetModel(subagent.Model).
		SetSource(subagent.Source).
		SetEnabled(subagent.Enabled).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: update agent subagent: %w", err)
	}
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// DeleteAgentSubagent deletes the agent subagent.
func (s *AgentStore) DeleteAgentSubagent(ctx context.Context, flag string) error {
	n, err := s.client.AgentSubagent.Delete().
		Where(agentsubagent.FlagEQ(flag)).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("postgres: delete agent subagent: %w", err)
	}
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// CreateAgentSubagentTask persists a new agent subagent task.
func (s *AgentStore) CreateAgentSubagentTask(ctx context.Context, task *gen.AgentSubagentTask) error {
	if task == nil {
		return errors.New("postgres: nil agent subagent task")
	}
	builder := s.client.AgentSubagentTask.Create().
		SetSessionID(task.SessionID).
		SetSubagentName(task.SubagentName).
		SetDescription(task.Description).
		SetPrompt(task.Prompt).
		SetStatus(task.Status).
		SetResult(task.Result).
		SetErrorText(task.ErrorText).
		SetDepth(task.Depth)
	if !task.StartedAt.IsZero() {
		builder = builder.SetStartedAt(task.StartedAt)
	}
	if !task.CreatedAt.IsZero() {
		builder = builder.SetCreatedAt(task.CreatedAt)
	}
	if !task.UpdatedAt.IsZero() {
		builder = builder.SetUpdatedAt(task.UpdatedAt)
	}
	if task.FinishedAt != nil {
		builder = builder.SetFinishedAt(*task.FinishedAt)
	}
	row, err := builder.Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: create agent subagent task: %w", err)
	}
	task.ID = row.ID
	return nil
}

// UpdateAgentSubagentTask updates the agent subagent task.
func (s *AgentStore) UpdateAgentSubagentTask(ctx context.Context, task *gen.AgentSubagentTask) error {
	if task == nil {
		return errors.New("postgres: nil agent subagent task")
	}
	if task.ID == 0 {
		return errors.New("postgres: agent subagent task id is required")
	}
	builder := s.client.AgentSubagentTask.UpdateOneID(task.ID).
		SetStatus(task.Status).
		SetResult(task.Result).
		SetErrorText(task.ErrorText).
		SetUpdatedAt(time.Now())
	if task.FinishedAt != nil {
		builder = builder.SetFinishedAt(*task.FinishedAt)
	}
	if _, err := builder.Save(ctx); err != nil {
		if gen.IsNotFound(err) {
			return types.ErrNotFound
		}
		return fmt.Errorf("postgres: update agent subagent task: %w", err)
	}
	return nil
}

// ListAgentSubagentTasks returns agent subagent tasks.
func (s *AgentStore) ListAgentSubagentTasks(ctx context.Context, sessionID string, limit int) ([]*gen.AgentSubagentTask, error) {
	query := s.client.AgentSubagentTask.Query().
		Order(gen.Desc(agentsubagenttask.FieldCreatedAt))
	if sessionID != "" {
		query = query.Where(agentsubagenttask.SessionIDEQ(sessionID))
	}
	if limit > 0 {
		query = query.Limit(limit)
	}
	rows, err := query.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: list agent subagent tasks: %w", err)
	}
	return rows, nil
}

// GetAgentSubagentTask returns the agent subagent task.
func (s *AgentStore) GetAgentSubagentTask(ctx context.Context, id int64) (*gen.AgentSubagentTask, error) {
	row, err := s.client.AgentSubagentTask.Get(ctx, id)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: get agent subagent task: %w", err)
	}
	return row, nil
}

