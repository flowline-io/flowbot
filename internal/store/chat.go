package store

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/chatscheduledtask"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/chatscheduledtaskrun"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/chatsession"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/chatsessionentry"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/types"
)

// ChatStore persists chat sessions, entries, and scheduled tasks.
type ChatStore struct {
	client *gen.Client
}

// NewChatStore creates a ChatStore with the given ent client.
func NewChatStore(client *gen.Client) *ChatStore {
	return &ChatStore{client: client}
}

// ChatStoreFromDB returns a ChatStore using the global database client.
func ChatStoreFromDB() *ChatStore {
	return NewChatStore(ClientFromDB())
}

// Client returns the underlying ent client.
func (s *ChatStore) Client() *gen.Client {
	if s == nil {
		return nil
	}
	return s.client
}

// CreateChatSession persists a new chat session.
func (s *ChatStore) CreateChatSession(ctx context.Context, session *gen.ChatSession) error {
	if session == nil {
		return errors.New("postgres: nil chat session")
	}
	builder := s.client.ChatSession.Create().
		SetFlag(session.Flag).
		SetUID(session.UID).
		SetLeafID(session.LeafID).
		SetState(session.State)
	if !session.CreatedAt.IsZero() {
		builder = builder.SetCreatedAt(session.CreatedAt)
	}
	if !session.UpdatedAt.IsZero() {
		builder = builder.SetUpdatedAt(session.UpdatedAt)
	}
	_, err := builder.Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: create chat session: %w", err)
	}
	return nil
}

// GetChatSession returns the chat session.
func (s *ChatStore) GetChatSession(ctx context.Context, flag string) (*gen.ChatSession, error) {
	row, err := s.client.ChatSession.Query().
		Where(chatsession.FlagEQ(flag)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: get chat session: %w", err)
	}
	return row, nil
}

// ListChatSessions returns chat sessions.
func (s *ChatStore) ListChatSessions(ctx context.Context, opts ListChatSessionsOptions) ([]*gen.ChatSession, string, error) {
	if opts.Limit <= 0 || opts.Limit > 100 {
		opts.Limit = 20
	}

	order := []chatsession.OrderOption{
		gen.Desc(chatsession.FieldUpdatedAt),
		gen.Desc(chatsession.FieldID),
	}
	if opts.PinnedFirst {
		order = []chatsession.OrderOption{
			gen.Desc(chatsession.FieldPinned),
			gen.Desc(chatsession.FieldUpdatedAt),
			gen.Desc(chatsession.FieldID),
		}
	}

	q := s.chatSessionFilterQuery(opts).
		Order(order...).
		Limit(opts.Limit + 1)

	if opts.Cursor != "" {
		id, err := strconv.ParseInt(opts.Cursor, 10, 64)
		if err == nil {
			q = q.Where(chatsession.IDLT(id))
		}
	}

	rows, err := q.All(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("postgres: list chat sessions: %w", err)
	}

	var nextCursor string
	if len(rows) > opts.Limit {
		nextCursor = strconv.FormatInt(rows[opts.Limit-1].ID, 10)
		rows = rows[:opts.Limit]
	}

	return rows, nextCursor, nil
}

// CountChatSessions returns the number of chat sessions.
func (s *ChatStore) CountChatSessions(ctx context.Context, opts ListChatSessionsOptions) (int, error) {
	n, err := s.chatSessionFilterQuery(opts).Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("postgres: count chat sessions: %w", err)
	}
	return n, nil
}

func (s *ChatStore) chatSessionFilterQuery(opts ListChatSessionsOptions) *gen.ChatSessionQuery {
	q := s.client.ChatSession.Query()
	if opts.UID != "" {
		q = q.Where(chatsession.UIDEQ(opts.UID))
	}
	if opts.State != nil {
		q = q.Where(chatsession.StateEQ(*opts.State))
	}
	if opts.Archived != nil {
		q = q.Where(chatsession.ArchivedEQ(*opts.Archived))
	}
	if len(opts.Flags) > 0 {
		q = q.Where(chatsession.FlagIn(opts.Flags...))
	}
	return q
}

// UpdateChatSessionLeaf updates the chat session leaf.
func (s *ChatStore) UpdateChatSessionLeaf(ctx context.Context, flag, leafID string) error {
	n, err := s.client.ChatSession.Update().
		Where(chatsession.FlagEQ(flag)).
		SetLeafID(leafID).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: update chat session leaf: %w", err)
	}
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// UpdateChatSessionMode updates the chat session mode.
func (s *ChatStore) UpdateChatSessionMode(ctx context.Context, flag, mode string) error {
	n, err := s.client.ChatSession.Update().
		Where(chatsession.FlagEQ(flag)).
		SetMode(mode).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: update chat session mode: %w", err)
	}
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// UpdateChatSessionSettings updates the chat session settings.
func (s *ChatStore) UpdateChatSessionSettings(ctx context.Context, flag, modelName, thinkingLevel string) error {
	n, err := s.client.ChatSession.Update().
		Where(chatsession.FlagEQ(flag)).
		SetModel(modelName).
		SetThinkingLevel(thinkingLevel).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: update chat session settings: %w", err)
	}
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// UpdateChatSessionTitle updates the chat session title.
func (s *ChatStore) UpdateChatSessionTitle(ctx context.Context, flag, title string) error {
	n, err := s.client.ChatSession.Update().
		Where(chatsession.FlagEQ(flag)).
		SetTitle(title).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: update chat session title: %w", err)
	}
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// UpdateChatSessionPreview updates the chat session preview.
func (s *ChatStore) UpdateChatSessionPreview(ctx context.Context, flag, preview string) error {
	n, err := s.client.ChatSession.Update().
		Where(chatsession.FlagEQ(flag)).
		SetPreview(preview).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: update chat session preview: %w", err)
	}
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// UpdateChatSessionPinned updates the chat session pinned.
func (s *ChatStore) UpdateChatSessionPinned(ctx context.Context, flag string, pinned bool) error {
	n, err := s.client.ChatSession.Update().
		Where(chatsession.FlagEQ(flag)).
		SetPinned(pinned).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: update chat session pinned: %w", err)
	}
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// UpdateChatSessionArchived updates the chat session archived.
func (s *ChatStore) UpdateChatSessionArchived(ctx context.Context, flag string, archived bool) error {
	n, err := s.client.ChatSession.Update().
		Where(chatsession.FlagEQ(flag)).
		SetArchived(archived).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: update chat session archived: %w", err)
	}
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// CloseChatSession closes the chat session.
func (s *ChatStore) CloseChatSession(ctx context.Context, flag string) error {
	_, err := s.client.ChatSession.Update().
		Where(chatsession.FlagEQ(flag)).
		SetState(int(schema.ChatSessionClosed)).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: close chat session: %w", err)
	}
	return nil
}

// CreateChatSessionEntry persists a new chat session entry.
func (s *ChatStore) CreateChatSessionEntry(ctx context.Context, entry *gen.ChatSessionEntry) error {
	if entry == nil {
		return errors.New("postgres: nil chat session entry")
	}
	builder := s.client.ChatSessionEntry.Create().
		SetFlag(entry.Flag).
		SetSessionID(entry.SessionID).
		SetParentID(entry.ParentID).
		SetEntryType(entry.EntryType)
	if entry.Payload != nil {
		builder = builder.SetPayload(entry.Payload)
	}
	if !entry.CreatedAt.IsZero() {
		builder = builder.SetCreatedAt(entry.CreatedAt)
	}
	_, err := builder.Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: create chat session entry: %w", err)
	}
	return nil
}

// AppendChatSessionEntry appends a chat session entry.
func (s *ChatStore) AppendChatSessionEntry(ctx context.Context, entry *gen.ChatSessionEntry) error {
	if entry == nil {
		return errors.New("postgres: nil chat session entry")
	}
	tx, err := s.client.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("postgres: begin chat session tx: %w", err)
	}
	builder := tx.ChatSessionEntry.Create().
		SetFlag(entry.Flag).
		SetSessionID(entry.SessionID).
		SetParentID(entry.ParentID).
		SetEntryType(entry.EntryType)
	if entry.Payload != nil {
		builder = builder.SetPayload(entry.Payload)
	}
	if !entry.CreatedAt.IsZero() {
		builder = builder.SetCreatedAt(entry.CreatedAt)
	}
	if _, err := builder.Save(ctx); err != nil {
		if rerr := tx.Rollback(); rerr != nil {
			return fmt.Errorf("postgres: create chat session entry: %w (rollback: %v)", err, rerr)
		}
		return fmt.Errorf("postgres: create chat session entry: %w", err)
	}
	n, err := tx.ChatSession.Update().
		Where(chatsession.FlagEQ(entry.SessionID)).
		SetLeafID(entry.Flag).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		if rerr := tx.Rollback(); rerr != nil {
			return fmt.Errorf("postgres: update chat session leaf: %w (rollback: %v)", err, rerr)
		}
		return fmt.Errorf("postgres: update chat session leaf: %w", err)
	}
	if n == 0 {
		if rerr := tx.Rollback(); rerr != nil {
			return types.ErrNotFound
		}
		return types.ErrNotFound
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("postgres: commit chat session entry: %w", err)
	}
	return nil
}

// ListChatSessionEntries returns chat session entries.
func (s *ChatStore) ListChatSessionEntries(ctx context.Context, sessionID string) ([]*gen.ChatSessionEntry, error) {
	rows, err := s.client.ChatSessionEntry.Query().
		Where(chatsessionentry.SessionIDEQ(sessionID)).
		Order(gen.Asc(chatsessionentry.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: list chat session entries: %w", err)
	}
	return rows, nil
}

// ListChatSessionEntriesBySessions returns chat session entries by sessions.
func (s *ChatStore) ListChatSessionEntriesBySessions(ctx context.Context, sessionIDs []string) ([]*gen.ChatSessionEntry, error) {
	if len(sessionIDs) == 0 {
		return nil, nil
	}
	rows, err := s.client.ChatSessionEntry.Query().
		Where(chatsessionentry.SessionIDIn(sessionIDs...)).
		Order(gen.Asc(chatsessionentry.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: list chat session entries by sessions: %w", err)
	}
	return rows, nil
}

// GetChatSessionEntry returns the chat session entry.
func (s *ChatStore) GetChatSessionEntry(ctx context.Context, flag string) (*gen.ChatSessionEntry, error) {
	row, err := s.client.ChatSessionEntry.Query().
		Where(chatsessionentry.FlagEQ(flag)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: get chat session entry: %w", err)
	}
	return row, nil
}

// GetChatSessionEntryInSession returns the chat session entry in session.
func (s *ChatStore) GetChatSessionEntryInSession(ctx context.Context, sessionID, flag string) (*gen.ChatSessionEntry, error) {
	row, err := s.client.ChatSessionEntry.Query().
		Where(
			chatsessionentry.SessionIDEQ(sessionID),
			chatsessionentry.FlagEQ(flag),
		).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: get chat session entry in session: %w", err)
	}
	return row, nil
}

// CreateChatScheduledTask persists a new chat scheduled task.
func (s *ChatStore) CreateChatScheduledTask(ctx context.Context, task *gen.ChatScheduledTask) error {
	if task == nil {
		return errors.New("postgres: nil chat scheduled task")
	}
	builder := s.client.ChatScheduledTask.Create().
		SetFlag(task.Flag).
		SetUID(task.UID).
		SetName(task.Name).
		SetScheduleKind(task.ScheduleKind).
		SetCron(task.Cron).
		SetPrompt(task.Prompt).
		SetSourceSessionID(task.SourceSessionID).
		SetState(task.State)
	if task.RunAt != nil {
		builder = builder.SetRunAt(*task.RunAt)
	}
	if task.Delivery != nil {
		builder = builder.SetDelivery(task.Delivery)
	}
	if task.LastRunAt != nil {
		builder = builder.SetLastRunAt(*task.LastRunAt)
	}
	if task.NextRunAt != nil {
		builder = builder.SetNextRunAt(*task.NextRunAt)
	}
	if !task.CreatedAt.IsZero() {
		builder = builder.SetCreatedAt(task.CreatedAt)
	}
	if !task.UpdatedAt.IsZero() {
		builder = builder.SetUpdatedAt(task.UpdatedAt)
	}
	_, err := builder.Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: create chat scheduled task: %w", err)
	}
	return nil
}

// DeleteChatScheduledTask deletes the chat scheduled task.
func (s *ChatStore) DeleteChatScheduledTask(ctx context.Context, flag string) error {
	n, err := s.client.ChatScheduledTask.Delete().
		Where(chatscheduledtask.FlagEQ(flag)).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("postgres: delete chat scheduled task: %w", err)
	}
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// GetChatScheduledTask returns the chat scheduled task.
func (s *ChatStore) GetChatScheduledTask(ctx context.Context, flag string) (*gen.ChatScheduledTask, error) {
	row, err := s.client.ChatScheduledTask.Query().
		Where(chatscheduledtask.FlagEQ(flag)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: get chat scheduled task: %w", err)
	}
	return row, nil
}

// GetChatScheduledTaskForUID returns the chat scheduled task for uid.
func (s *ChatStore) GetChatScheduledTaskForUID(ctx context.Context, flag, uid string) (*gen.ChatScheduledTask, error) {
	row, err := s.client.ChatScheduledTask.Query().
		Where(
			chatscheduledtask.FlagEQ(flag),
			chatscheduledtask.UIDEQ(uid),
		).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: get chat scheduled task for uid: %w", err)
	}
	return row, nil
}

// ListChatScheduledTasks returns chat scheduled tasks.
func (s *ChatStore) ListChatScheduledTasks(ctx context.Context, opts ListChatScheduledTasksOptions) ([]*gen.ChatScheduledTask, error) {
	q := s.client.ChatScheduledTask.Query().
		Order(
			gen.Desc(chatscheduledtask.FieldUpdatedAt),
			gen.Desc(chatscheduledtask.FieldID),
		)
	if opts.UID != "" {
		q = q.Where(chatscheduledtask.UIDEQ(opts.UID))
	}
	if len(opts.States) > 0 {
		q = q.Where(chatscheduledtask.StateIn(opts.States...))
	}
	rows, err := q.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: list chat scheduled tasks: %w", err)
	}
	return rows, nil
}

// UpdateChatScheduledTask updates the chat scheduled task.
func (s *ChatStore) UpdateChatScheduledTask(ctx context.Context, flag string, params UpdateChatScheduledTaskParams) error {
	builder := s.client.ChatScheduledTask.Update().
		Where(chatscheduledtask.FlagEQ(flag)).
		SetUpdatedAt(time.Now())
	if params.Name != nil {
		builder = builder.SetName(*params.Name)
	}
	if params.Cron != nil {
		builder = builder.SetCron(*params.Cron)
	}
	if params.RunAt != nil {
		builder = builder.SetRunAt(*params.RunAt)
	}
	if params.Prompt != nil {
		builder = builder.SetPrompt(*params.Prompt)
	}
	if params.State != nil {
		builder = builder.SetState(*params.State)
	}
	if params.LastRunAt != nil {
		builder = builder.SetLastRunAt(*params.LastRunAt)
	}
	if params.NextRunAt != nil {
		builder = builder.SetNextRunAt(*params.NextRunAt)
	}
	n, err := builder.Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: update chat scheduled task: %w", err)
	}
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// CreateChatScheduledTaskRun persists a new chat scheduled task run.
func (s *ChatStore) CreateChatScheduledTaskRun(ctx context.Context, run *gen.ChatScheduledTaskRun) error {
	if run == nil {
		return errors.New("postgres: nil chat scheduled task run")
	}
	builder := s.client.ChatScheduledTaskRun.Create().
		SetFlag(run.Flag).
		SetTaskID(run.TaskID).
		SetRunSessionID(run.RunSessionID).
		SetState(run.State).
		SetReply(run.Reply).
		SetError(run.Error)
	if !run.StartedAt.IsZero() {
		builder = builder.SetStartedAt(run.StartedAt)
	}
	if run.FinishedAt != nil {
		builder = builder.SetFinishedAt(*run.FinishedAt)
	}
	_, err := builder.Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: create chat scheduled task run: %w", err)
	}
	return nil
}

// UpdateChatScheduledTaskRun updates the chat scheduled task run.
func (s *ChatStore) UpdateChatScheduledTaskRun(ctx context.Context, flag string, params UpdateChatScheduledTaskRunParams) error {
	builder := s.client.ChatScheduledTaskRun.Update().
		Where(chatscheduledtaskrun.FlagEQ(flag))
	if params.State != nil {
		builder = builder.SetState(*params.State)
	}
	if params.Reply != nil {
		builder = builder.SetReply(*params.Reply)
	}
	if params.Error != nil {
		builder = builder.SetError(*params.Error)
	}
	if params.FinishedAt != nil {
		builder = builder.SetFinishedAt(*params.FinishedAt)
	}
	n, err := builder.Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: update chat scheduled task run: %w", err)
	}
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// FailStaleChatScheduledTaskRuns marks stale chat scheduled task runs.
func (s *ChatStore) FailStaleChatScheduledTaskRuns(ctx context.Context) error {
	now := time.Now().UTC()
	msg := "interrupted by server restart"
	_, err := s.client.ChatScheduledTaskRun.Update().
		Where(chatscheduledtaskrun.StateEQ(string(schema.ChatScheduledTaskRunStateRunning))).
		SetState(string(schema.ChatScheduledTaskRunStateFailed)).
		SetError(msg).
		SetFinishedAt(now).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: fail stale chat scheduled task runs: %w", err)
	}
	return nil
}

// ListChatScheduledTaskRuns returns chat scheduled task runs.
func (s *ChatStore) ListChatScheduledTaskRuns(ctx context.Context, taskID string, limit int) ([]*gen.ChatScheduledTaskRun, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := s.client.ChatScheduledTaskRun.Query().
		Where(chatscheduledtaskrun.TaskIDEQ(taskID)).
		Order(
			gen.Desc(chatscheduledtaskrun.FieldStartedAt),
			gen.Desc(chatscheduledtaskrun.FieldID),
		).
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: list chat scheduled task runs: %w", err)
	}
	return rows, nil
}
