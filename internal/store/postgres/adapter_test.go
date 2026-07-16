package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getTestClient(t *testing.T) *gen.Client {
	t.Helper()
	return newSQLiteTestClient(t)
}

func testAdapter(t *testing.T) *adapter {
	t.Helper()
	return &adapter{client: getTestClient(t)}
}

func TestListTokens(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		seeds   func(*testing.T, *adapter)
		wantLen int
	}{
		{
			name:    "empty database returns empty slice",
			seeds:   func(_ *testing.T, _ *adapter) {},
			wantLen: 0,
		},
		{
			name: "with valid tokens returns them",
			seeds: func(t *testing.T, a *adapter) {
				token, err := a.CreateToken(context.Background(), types.Uid("user:alice"), time.Now().Add(24*time.Hour), []string{"admin:*"})
				require.NoError(t, err)
				require.NotEmpty(t, token)
				_, err = a.CreateToken(context.Background(), types.Uid("user:bob"), time.Now().Add(7*24*time.Hour), []string{"hub:apps:read"})
				require.NoError(t, err)
			},
			wantLen: 2,
		},
		{
			name: "filters expired unused tokens older than 30 days",
			seeds: func(t *testing.T, a *adapter) {
				_, err := a.CreateToken(context.Background(), types.Uid("user:old"), time.Now().Add(-40*24*time.Hour), []string{"hub:apps:read"})
				require.NoError(t, err)
				_, err = a.CreateToken(context.Background(), types.Uid("user:recent"), time.Now().Add(24*time.Hour), []string{"pipeline:read"})
				require.NoError(t, err)
			},
			wantLen: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := testAdapter(t)
			tt.seeds(t, a)
			items, err := a.ListTokens(context.Background())
			require.NoError(t, err)
			assert.Len(t, items, tt.wantLen)
			if tt.wantLen > 0 {
				for _, item := range items {
					assert.NotEmpty(t, item.Token)
					assert.Len(t, item.Token, 64)
					assert.NotEmpty(t, item.UID)
				}
			}
		})
	}
}

func TestCreateToken(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		uid       types.Uid
		expiresAt time.Time
		scopes    []string
		wantErr   bool
	}{
		{
			name:      "creates token successfully",
			uid:       types.Uid("user:test"),
			expiresAt: time.Now().Add(24 * time.Hour),
			scopes:    []string{"admin:*"},
			wantErr:   false,
		},
		{
			name:      "creates token with multiple scopes",
			uid:       types.Uid("user:multi"),
			expiresAt: time.Now().Add(7 * 24 * time.Hour),
			scopes:    []string{"hub:apps:read", "pipeline:read"},
			wantErr:   false,
		},
		{
			name:      "creates token with past expiry still succeeds",
			uid:       types.Uid("user:expired"),
			expiresAt: time.Now().Add(-1 * time.Hour),
			scopes:    []string{"hub:apps:read"},
			wantErr:   false,
		},
		{
			name:      "rejects empty scopes",
			uid:       types.Uid("user:noscope"),
			expiresAt: time.Now().Add(24 * time.Hour),
			scopes:    nil,
			wantErr:   true,
		},
		{
			name:      "rejects empty scope slice",
			uid:       types.Uid("user:empty"),
			expiresAt: time.Now().Add(24 * time.Hour),
			scopes:    []string{},
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := testAdapter(t)
			token, err := a.CreateToken(context.Background(), tt.uid, tt.expiresAt, tt.scopes)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Greater(t, len(token), 10)
			assert.Contains(t, token, "fb_")
			items, err := a.ListTokens(context.Background())
			require.NoError(t, err)
			assert.Len(t, items, 1)
			assert.Equal(t, auth.HashToken(token), items[0].Token)
			assert.Equal(t, tt.uid, items[0].UID)
			assert.Equal(t, tt.scopes, items[0].Scopes)
		})
	}
}

func TestRevokeToken(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		seed    func(*testing.T, *adapter) string
		wantErr bool
		errIs   error
	}{
		{
			name: "revokes existing token",
			seed: func(t *testing.T, a *adapter) string {
				token, err := a.CreateToken(context.Background(), types.Uid("user:revoke"), time.Now().Add(24*time.Hour), []string{"admin:*"})
				require.NoError(t, err)
				return auth.HashToken(token)
			},
			wantErr: false,
		},
		{
			name: "returns ErrNotFound for nonexistent token",
			seed: func(_ *testing.T, _ *adapter) string {
				return "fb_nonexistent_token_12345678"
			},
			wantErr: true,
			errIs:   types.ErrNotFound,
		},
		{
			name: "revoking already revoked token returns ErrNotFound",
			seed: func(t *testing.T, a *adapter) string {
				token, err := a.CreateToken(context.Background(), types.Uid("user:twice"), time.Now().Add(24*time.Hour), []string{"hub:apps:read"})
				require.NoError(t, err)
				flag := auth.HashToken(token)
				err = a.RevokeToken(context.Background(), flag)
				require.NoError(t, err)
				return flag
			},
			wantErr: true,
			errIs:   types.ErrNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := testAdapter(t)
			flag := tt.seed(t, a)
			err := a.RevokeToken(context.Background(), flag)
			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.errIs)
				return
			}
			require.NoError(t, err)
			items, err := a.ListTokens(context.Background())
			require.NoError(t, err)
			assert.Empty(t, items)
		})
	}
}

func TestCreatePlatformUser(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		item       *gen.PlatformUser
		wantEmail  string
		wantAvatar string
	}{
		{
			name: "preserves provided profile fields",
			item: &gen.PlatformUser{
				PlatformID: 1,
				UserID:     2,
				Flag:       "U123",
				Name:       "alice",
				Email:      "alice@example.com",
				AvatarURL:  "https://example.com/a.png",
				IsBot:      false,
			},
			wantEmail:  "alice@example.com",
			wantAvatar: "https://example.com/a.png",
		},
		{
			name: "fills missing email and avatar placeholders",
			item: &gen.PlatformUser{
				PlatformID: 1,
				UserID:     2,
				Flag:       "U01DMQDTV5W",
				Name:       "user",
				IsBot:      false,
			},
			wantEmail:  "U01DMQDTV5W@unknown.local",
			wantAvatar: "-",
		},
		{
			name: "fills only missing avatar when email is present",
			item: &gen.PlatformUser{
				PlatformID: 1,
				UserID:     2,
				Flag:       "U999",
				Name:       "user",
				Email:      "user@slack.local",
				IsBot:      false,
			},
			wantEmail:  "user@slack.local",
			wantAvatar: "-",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := testAdapter(t)
			id, err := a.CreatePlatformUser(context.Background(), tt.item)
			require.NoError(t, err)
			assert.Positive(t, id)

			created, err := a.client.PlatformUser.Get(context.Background(), id)
			require.NoError(t, err)
			assert.Equal(t, tt.wantEmail, created.Email)
			assert.Equal(t, tt.wantAvatar, created.AvatarURL)
		})
	}
}

func TestAgentSkillByFlagAndDelete(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		setup   func(context.Context, *adapter) string
		action  func(context.Context, *adapter, string) error
		wantErr error
	}{
		{
			name: "get by flag returns stored skill",
			setup: func(ctx context.Context, a *adapter) string {
				require.NoError(t, a.CreateAgentSkill(ctx, &gen.AgentSkill{
					Flag:        "homelab-bookmark",
					Name:        "homelab-bookmark",
					Description: "Bookmark skill",
					Content:     "# Bookmark",
					Source:      "global",
					Enabled:     true,
				}))
				return "homelab-bookmark"
			},
			action: func(ctx context.Context, a *adapter, flag string) error {
				row, err := a.GetAgentSkillByFlag(ctx, flag)
				if err != nil {
					return err
				}
				if row.Name != "homelab-bookmark" {
					return types.Errorf(types.ErrInternal, "unexpected name %q", row.Name)
				}
				return nil
			},
		},
		{
			name: "get by flag returns not found",
			setup: func(_ context.Context, _ *adapter) string {
				return "missing"
			},
			action: func(ctx context.Context, a *adapter, flag string) error {
				_, err := a.GetAgentSkillByFlag(ctx, flag)
				return err
			},
			wantErr: types.ErrNotFound,
		},
		{
			name: "delete removes skill",
			setup: func(ctx context.Context, a *adapter) string {
				require.NoError(t, a.CreateAgentSkill(ctx, &gen.AgentSkill{
					Flag:        "to-delete",
					Name:        "to-delete",
					Description: "Delete me",
					Content:     "body",
					Enabled:     true,
				}))
				return "to-delete"
			},
			action: func(ctx context.Context, a *adapter, flag string) error {
				if err := a.DeleteAgentSkill(ctx, flag); err != nil {
					return err
				}
				_, err := a.GetAgentSkillByFlag(ctx, flag)
				return err
			},
			wantErr: types.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := testAdapter(t)
			ctx := context.Background()
			flag := tt.setup(ctx, a)
			err := tt.action(ctx, a, flag)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestAgentSkillFileCRUD(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		setup   func(context.Context, *adapter) string
		action  func(context.Context, *adapter, string) error
		wantErr error
	}{
		{
			name: "create list and get file",
			setup: func(ctx context.Context, a *adapter) string {
				require.NoError(t, a.CreateAgentSkill(ctx, &gen.AgentSkill{
					Flag:        "demo-skill",
					Name:        "demo-skill",
					Description: "Demo",
					Content:     "body",
					Enabled:     true,
				}))
				return "demo-skill"
			},
			action: func(ctx context.Context, a *adapter, flag string) error {
				require.NoError(t, a.CreateAgentSkillFile(ctx, &gen.AgentSkillFile{
					SkillFlag: flag,
					Path:      "reference.md",
					Content:   "reference body",
				}))
				rows, err := a.ListAgentSkillFiles(ctx, flag)
				if err != nil {
					return err
				}
				if len(rows) != 1 {
					return types.Errorf(types.ErrInternal, "expected 1 file, got %d", len(rows))
				}
				row, err := a.GetAgentSkillFile(ctx, flag, "reference.md")
				if err != nil {
					return err
				}
				if row.Content != "reference body" {
					return types.Errorf(types.ErrInternal, "unexpected content %q", row.Content)
				}
				return nil
			},
		},
		{
			name: "duplicate path rejected",
			setup: func(ctx context.Context, a *adapter) string {
				require.NoError(t, a.CreateAgentSkill(ctx, &gen.AgentSkill{
					Flag: "dup-skill", Name: "dup-skill", Description: "d", Content: "c", Enabled: true,
				}))
				require.NoError(t, a.CreateAgentSkillFile(ctx, &gen.AgentSkillFile{
					SkillFlag: "dup-skill", Path: "a.md", Content: "a",
				}))
				return "dup-skill"
			},
			action: func(ctx context.Context, a *adapter, flag string) error {
				return a.CreateAgentSkillFile(ctx, &gen.AgentSkillFile{
					SkillFlag: flag, Path: "a.md", Content: "duplicate",
				})
			},
		},
		{
			name: "delete skill cascades files",
			setup: func(ctx context.Context, a *adapter) string {
				require.NoError(t, a.CreateAgentSkill(ctx, &gen.AgentSkill{
					Flag: "cascade-skill", Name: "cascade-skill", Description: "d", Content: "c", Enabled: true,
				}))
				require.NoError(t, a.CreateAgentSkillFile(ctx, &gen.AgentSkillFile{
					SkillFlag: "cascade-skill", Path: "notes.md", Content: "notes",
				}))
				return "cascade-skill"
			},
			action: func(ctx context.Context, a *adapter, flag string) error {
				if err := a.DeleteAgentSkill(ctx, flag); err != nil {
					return err
				}
				rows, err := a.ListAgentSkillFiles(ctx, flag)
				if err != nil {
					return err
				}
				if len(rows) != 0 {
					return types.Errorf(types.ErrInternal, "expected 0 files after cascade delete")
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := testAdapter(t)
			ctx := context.Background()
			flag := tt.setup(ctx, a)
			err := tt.action(ctx, a, flag)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			if tt.name == "duplicate path rejected" {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestUpdateAgentSkillNotFound(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "missing flag returns not found"},
		{name: "update on empty database fails"},
		{name: "update without prior create fails"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := testAdapter(t)
			err := a.UpdateAgentSkill(context.Background(), &gen.AgentSkill{
				Flag:        "missing",
				Name:        "missing",
				Description: "Missing",
				Content:     "body",
			})
			require.ErrorIs(t, err, types.ErrNotFound)
		})
	}
}

func TestCreateAgentSubagentSetsID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		row  *gen.AgentSubagent
	}{
		{
			name: "enabled subagent gets generated id",
			row: &gen.AgentSubagent{
				Flag: "subagent-a", Name: "subagent-a",
				Description: "desc", SystemPrompt: "prompt",
				Tools: []string{"read_file"}, Source: "test", Enabled: true,
			},
		},
		{
			name: "disabled subagent gets generated id",
			row: &gen.AgentSubagent{
				Flag: "subagent-b", Name: "subagent-b",
				Description: "desc", SystemPrompt: "prompt",
				Source: "test", Enabled: false,
			},
		},
		{
			name: "subagent with model gets generated id",
			row: &gen.AgentSubagent{
				Flag: "subagent-c", Name: "subagent-c",
				Description: "desc", SystemPrompt: "prompt",
				Model: "gpt-4o", Source: "global", Enabled: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := testAdapter(t)
			ctx := context.Background()
			require.NoError(t, a.CreateAgentSubagent(ctx, tt.row))
			assert.Positive(t, tt.row.ID)

			got, err := a.GetAgentSubagentByFlag(ctx, tt.row.Flag)
			require.NoError(t, err)
			assert.Equal(t, tt.row.ID, got.ID)
		})
	}
}

func TestListChatSessions(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC().Truncate(time.Second)

	tests := []struct {
		name       string
		seeds      func(*testing.T, *adapter)
		opts       store.ListChatSessionsOptions
		wantLen    int
		wantCursor bool
	}{
		{
			name:    "empty database returns empty slice",
			seeds:   func(_ *testing.T, _ *adapter) {},
			opts:    store.ListChatSessionsOptions{Limit: 10},
			wantLen: 0,
		},
		{
			name: "returns seeded sessions newest first",
			seeds: func(t *testing.T, a *adapter) {
				require.NoError(t, a.CreateChatSession(context.Background(), &gen.ChatSession{
					Flag: "sess-old", UID: "user:a", State: int(schema.ChatSessionActive),
					CreatedAt: now.Add(-2 * time.Hour), UpdatedAt: now.Add(-2 * time.Hour),
				}))
				require.NoError(t, a.CreateChatSession(context.Background(), &gen.ChatSession{
					Flag: "sess-new", UID: "user:b", State: int(schema.ChatSessionClosed),
					CreatedAt: now, UpdatedAt: now,
				}))
			},
			opts:    store.ListChatSessionsOptions{Limit: 10},
			wantLen: 2,
		},
		{
			name: "cursor paginates remaining sessions",
			seeds: func(t *testing.T, a *adapter) {
				for i := range 3 {
					flag := "sess-" + string(rune('a'+i))
					require.NoError(t, a.CreateChatSession(context.Background(), &gen.ChatSession{
						Flag: flag, UID: "user:p", State: int(schema.ChatSessionActive),
						CreatedAt: now.Add(time.Duration(i) * time.Minute),
						UpdatedAt: now.Add(time.Duration(i) * time.Minute),
					}))
				}
			},
			opts:       store.ListChatSessionsOptions{Limit: 2},
			wantLen:    2,
			wantCursor: true,
		},
		{
			name: "uid filter returns only matching owner",
			seeds: func(t *testing.T, a *adapter) {
				require.NoError(t, a.CreateChatSession(context.Background(), &gen.ChatSession{
					Flag: "sess-a", UID: "user:alice", State: int(schema.ChatSessionActive),
					CreatedAt: now, UpdatedAt: now,
				}))
				require.NoError(t, a.CreateChatSession(context.Background(), &gen.ChatSession{
					Flag: "sess-b", UID: "user:bob", State: int(schema.ChatSessionActive),
					CreatedAt: now, UpdatedAt: now,
				}))
			},
			opts:    store.ListChatSessionsOptions{Limit: 10, UID: "user:alice"},
			wantLen: 1,
		},
		{
			name: "state filter returns only active sessions",
			seeds: func(t *testing.T, a *adapter) {
				require.NoError(t, a.CreateChatSession(context.Background(), &gen.ChatSession{
					Flag: "sess-active", UID: "user:s", State: int(schema.ChatSessionActive),
					CreatedAt: now, UpdatedAt: now,
				}))
				require.NoError(t, a.CreateChatSession(context.Background(), &gen.ChatSession{
					Flag: "sess-closed", UID: "user:s", State: int(schema.ChatSessionClosed),
					CreatedAt: now, UpdatedAt: now,
				}))
			},
			opts: func() store.ListChatSessionsOptions {
				active := int(schema.ChatSessionActive)
				return store.ListChatSessionsOptions{Limit: 10, State: &active}
			}(),
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := testAdapter(t)
			tt.seeds(t, a)

			got, cursor, err := a.ListChatSessions(context.Background(), tt.opts)
			require.NoError(t, err)
			assert.Len(t, got, tt.wantLen)
			if tt.wantCursor {
				assert.NotEmpty(t, cursor)
				page2, cursor2, err := a.ListChatSessions(context.Background(), store.ListChatSessionsOptions{
					Limit:  tt.opts.Limit,
					Cursor: cursor,
				})
				require.NoError(t, err)
				assert.NotEmpty(t, page2)
				assert.Empty(t, cursor2)
				return
			}
			assert.Empty(t, cursor)
		})
	}
}

func TestUpdateChatSessionTitle(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	a := testAdapter(t)

	require.NoError(t, a.CreateChatSession(ctx, &gen.ChatSession{
		Flag: "sess-title", UID: "user:t", State: int(schema.ChatSessionActive),
	}))

	tests := []struct {
		name    string
		title   string
		wantErr error
	}{
		{name: "sets title", title: "Deploy flowbot"},
		{name: "updates title", title: "Redis configuration"},
		{name: "missing session", title: "ghost", wantErr: types.ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := "sess-title"
			if tt.wantErr != nil {
				flag = "missing-session"
			}
			err := a.UpdateChatSessionTitle(ctx, flag, tt.title)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			got, err := a.GetChatSession(ctx, "sess-title")
			require.NoError(t, err)
			assert.Equal(t, tt.title, got.Title)
		})
	}
}

func TestChatScheduledTaskStore(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	runAt := now.Add(2 * time.Hour)

	tests := []struct {
		name string
		run  func(*testing.T, *adapter)
	}{
		{
			name: "create list and update task",
			run: func(t *testing.T, a *adapter) {
				require.NoError(t, a.CreateChatScheduledTask(ctx, &gen.ChatScheduledTask{
					Flag:         "task-1",
					UID:          "user:alice",
					Name:         "daily",
					ScheduleKind: string(schema.ChatScheduledTaskKindCron),
					Cron:         "0 9 * * *",
					Prompt:       "check logs",
					State:        string(schema.ChatScheduledTaskStateActive),
					CreatedAt:    now,
					UpdatedAt:    now,
				}))
				rows, err := a.ListChatScheduledTasks(ctx, store.ListChatScheduledTasksOptions{UID: "user:alice"})
				require.NoError(t, err)
				require.Len(t, rows, 1)

				prompt := "updated prompt"
				require.NoError(t, a.UpdateChatScheduledTask(ctx, "task-1", store.UpdateChatScheduledTaskParams{Prompt: &prompt}))
				got, err := a.GetChatScheduledTaskForUID(ctx, "task-1", "user:alice")
				require.NoError(t, err)
				assert.Equal(t, prompt, got.Prompt)
			},
		},
		{
			name: "create once task run record",
			run: func(t *testing.T, a *adapter) {
				require.NoError(t, a.CreateChatScheduledTask(ctx, &gen.ChatScheduledTask{
					Flag:         "task-once",
					UID:          "user:bob",
					Name:         "reminder",
					ScheduleKind: string(schema.ChatScheduledTaskKindOnce),
					Prompt:       "submit report",
					RunAt:        &runAt,
					State:        string(schema.ChatScheduledTaskStateActive),
				}))
				require.NoError(t, a.CreateChatScheduledTaskRun(ctx, &gen.ChatScheduledTaskRun{
					Flag:         "run-1",
					TaskID:       "task-once",
					RunSessionID: "sess-run",
					State:        string(schema.ChatScheduledTaskRunStateCompleted),
					Reply:        "done",
					StartedAt:    now,
				}))
				runs, err := a.ListChatScheduledTaskRuns(ctx, "task-once", 10)
				require.NoError(t, err)
				require.Len(t, runs, 1)
				assert.Equal(t, "done", runs[0].Reply)
			},
		},
		{
			name: "delete task",
			run: func(t *testing.T, a *adapter) {
				require.NoError(t, a.CreateChatScheduledTask(ctx, &gen.ChatScheduledTask{
					Flag:         "task-delete",
					UID:          "user:alice",
					Name:         "temp",
					ScheduleKind: string(schema.ChatScheduledTaskKindCron),
					Cron:         "0 7 * * *",
					Prompt:       "temp",
					State:        string(schema.ChatScheduledTaskStateActive),
				}))
				require.NoError(t, a.DeleteChatScheduledTask(ctx, "task-delete"))
				_, err := a.GetChatScheduledTask(ctx, "task-delete")
				require.ErrorIs(t, err, types.ErrNotFound)
			},
		},
		{
			name: "fail stale running task runs",
			run: func(t *testing.T, a *adapter) {
				require.NoError(t, a.CreateChatScheduledTask(ctx, &gen.ChatScheduledTask{
					Flag:         "task-stale",
					UID:          "user:alice",
					Name:         "stale",
					ScheduleKind: string(schema.ChatScheduledTaskKindCron),
					Cron:         "0 6 * * *",
					Prompt:       "stale",
					State:        string(schema.ChatScheduledTaskStateActive),
				}))
				require.NoError(t, a.CreateChatScheduledTaskRun(ctx, &gen.ChatScheduledTaskRun{
					Flag:         "run-stale",
					TaskID:       "task-stale",
					RunSessionID: "sess-stale",
					State:        string(schema.ChatScheduledTaskRunStateRunning),
					StartedAt:    now,
				}))
				require.NoError(t, a.FailStaleChatScheduledTaskRuns(ctx))
				runs, err := a.ListChatScheduledTaskRuns(ctx, "task-stale", 5)
				require.NoError(t, err)
				require.Len(t, runs, 1)
				assert.Equal(t, string(schema.ChatScheduledTaskRunStateFailed), runs[0].State)
				assert.NotEmpty(t, runs[0].Error)
			},
		},
		{
			name: "uid scoped get returns not found for other user",
			run: func(t *testing.T, a *adapter) {
				require.NoError(t, a.CreateChatScheduledTask(ctx, &gen.ChatScheduledTask{
					Flag:         "task-private",
					UID:          "user:owner",
					Name:         "private",
					ScheduleKind: string(schema.ChatScheduledTaskKindCron),
					Cron:         "0 8 * * *",
					Prompt:       "secret",
					State:        string(schema.ChatScheduledTaskStateActive),
				}))
				_, err := a.GetChatScheduledTaskForUID(ctx, "task-private", "user:other")
				require.ErrorIs(t, err, types.ErrNotFound)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.run(t, testAdapter(t))
		})
	}
}
