package store

import (
	"context"
	"testing"
	"time"
	"strconv"
	"strings"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/agentsessionsummary"
	"github.com/flowline-io/flowbot/internal/store/sqlitetest"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentSkillByFlagAndDelete(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		setup   func(context.Context, *gen.Client) string
		action  func(context.Context, *gen.Client, string) error
		wantErr error
	}{
		{
			name: "get by flag returns stored skill",
			setup: func(ctx context.Context, client *gen.Client) string {
				require.NoError(t, NewAgentStore(client).CreateAgentSkill(ctx, &gen.AgentSkill{
					Flag:        "karakeep",
					Name:        "karakeep",
					Description: "Bookmark skill",
					Content:     "# Bookmark",
					Source:      "global",
					Enabled:     true,
				}))
				return "karakeep"
			},
			action: func(ctx context.Context, client *gen.Client, flag string) error {
				row, err := NewAgentStore(client).GetAgentSkillByFlag(ctx, flag)
				if err != nil {
					return err
				}
				if row.Name != "karakeep" {
					return types.Errorf(types.ErrInternal, "unexpected name %q", row.Name)
				}
				return nil
			},
		},
		{
			name: "get by flag returns not found",
			setup: func(_ context.Context, _ *gen.Client) string {
				return "missing"
			},
			action: func(ctx context.Context, client *gen.Client, flag string) error {
				_, err := NewAgentStore(client).GetAgentSkillByFlag(ctx, flag)
				return err
			},
			wantErr: types.ErrNotFound,
		},
		{
			name: "delete removes skill",
			setup: func(ctx context.Context, client *gen.Client) string {
				require.NoError(t, NewAgentStore(client).CreateAgentSkill(ctx, &gen.AgentSkill{
					Flag:        "to-delete",
					Name:        "to-delete",
					Description: "Delete me",
					Content:     "body",
					Enabled:     true,
				}))
				return "to-delete"
			},
			action: func(ctx context.Context, client *gen.Client, flag string) error {
				if err := NewAgentStore(client).DeleteAgentSkill(ctx, flag); err != nil {
					return err
				}
				_, err := NewAgentStore(client).GetAgentSkillByFlag(ctx, flag)
				return err
			},
			wantErr: types.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client := sqlitetest.OpenClient(t, t.Name())
			ctx := context.Background()
			flag := tt.setup(ctx, client)
			err := tt.action(ctx, client, flag)
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
		setup   func(context.Context, *gen.Client) string
		action  func(context.Context, *gen.Client, string) error
		wantErr error
	}{
		{
			name: "create list and get file",
			setup: func(ctx context.Context, client *gen.Client) string {
				require.NoError(t, NewAgentStore(client).CreateAgentSkill(ctx, &gen.AgentSkill{
					Flag:        "demo-skill",
					Name:        "demo-skill",
					Description: "Demo",
					Content:     "body",
					Enabled:     true,
				}))
				return "demo-skill"
			},
			action: func(ctx context.Context, client *gen.Client, flag string) error {
				require.NoError(t, NewAgentStore(client).CreateAgentSkillFile(ctx, &gen.AgentSkillFile{
					SkillFlag: flag,
					Path:      "reference.md",
					Content:   "reference body",
				}))
				rows, err := NewAgentStore(client).ListAgentSkillFiles(ctx, flag)
				if err != nil {
					return err
				}
				if len(rows) != 1 {
					return types.Errorf(types.ErrInternal, "expected 1 file, got %d", len(rows))
				}
				row, err := NewAgentStore(client).GetAgentSkillFile(ctx, flag, "reference.md")
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
			setup: func(ctx context.Context, client *gen.Client) string {
				require.NoError(t, NewAgentStore(client).CreateAgentSkill(ctx, &gen.AgentSkill{
					Flag: "dup-skill", Name: "dup-skill", Description: "d", Content: "c", Enabled: true,
				}))
				require.NoError(t, NewAgentStore(client).CreateAgentSkillFile(ctx, &gen.AgentSkillFile{
					SkillFlag: "dup-skill", Path: "a.md", Content: "a",
				}))
				return "dup-skill"
			},
			action: func(ctx context.Context, client *gen.Client, flag string) error {
				return NewAgentStore(client).CreateAgentSkillFile(ctx, &gen.AgentSkillFile{
					SkillFlag: flag, Path: "a.md", Content: "duplicate",
				})
			},
		},
		{
			name: "delete skill cascades files",
			setup: func(ctx context.Context, client *gen.Client) string {
				require.NoError(t, NewAgentStore(client).CreateAgentSkill(ctx, &gen.AgentSkill{
					Flag: "cascade-skill", Name: "cascade-skill", Description: "d", Content: "c", Enabled: true,
				}))
				require.NoError(t, NewAgentStore(client).CreateAgentSkillFile(ctx, &gen.AgentSkillFile{
					SkillFlag: "cascade-skill", Path: "notes.md", Content: "notes",
				}))
				return "cascade-skill"
			},
			action: func(ctx context.Context, client *gen.Client, flag string) error {
				if err := NewAgentStore(client).DeleteAgentSkill(ctx, flag); err != nil {
					return err
				}
				rows, err := NewAgentStore(client).ListAgentSkillFiles(ctx, flag)
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
			client := sqlitetest.OpenClient(t, t.Name())
			ctx := context.Background()
			flag := tt.setup(ctx, client)
			err := tt.action(ctx, client, flag)
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
			client := sqlitetest.OpenClient(t, t.Name())
			err := NewAgentStore(client).UpdateAgentSkill(context.Background(), &gen.AgentSkill{
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
			client := sqlitetest.OpenClient(t, t.Name())
			ctx := context.Background()
			require.NoError(t, NewAgentStore(client).CreateAgentSubagent(ctx, tt.row))
			assert.Positive(t, tt.row.ID)

			got, err := NewAgentStore(client).GetAgentSubagentByFlag(ctx, tt.row.Flag)
			require.NoError(t, err)
			assert.Equal(t, tt.row.ID, got.ID)
		})
	}
}

func TestAgentKnowledgeCRUDAndSearch(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tests := []struct {
		name string
		run  func(t *testing.T, client *gen.Client)
	}{
		{
			name: "create get list and delete",
			run: func(t *testing.T, client *gen.Client) {
				doc := &gen.AgentKnowledge{
					Path:    "/docs/ops/backup.md",
					Title:   "Backup",
					Tags:    []string{"ops"},
					Summary: "how to backup",
					Content: "postgres backup steps",
				}
				require.NoError(t, NewAgentStore(client).CreateAgentKnowledge(ctx, doc))
				assert.Positive(t, doc.ID)

				got, err := NewAgentStore(client).GetAgentKnowledgeByPath(ctx, "/docs/ops/backup.md")
				require.NoError(t, err)
				assert.Equal(t, "Backup", got.Title)

				listed, err := NewAgentStore(client).ListAgentKnowledge(ctx, AgentKnowledgeListFilter{Q: "backup"})
				require.NoError(t, err)
				require.NotEmpty(t, listed)

				require.NoError(t, NewAgentStore(client).DeleteAgentKnowledge(ctx, doc.ID))
				_, err = NewAgentStore(client).GetAgentKnowledgeByID(ctx, doc.ID)
				require.ErrorIs(t, err, types.ErrNotFound)
			},
		},
		{
			name: "search ranks title hits first",
			run: func(t *testing.T, client *gen.Client) {
				require.NoError(t, NewAgentStore(client).CreateAgentKnowledge(ctx, &gen.AgentKnowledge{
					Path:    "/docs/a.md",
					Title:   "Other",
					Tags:    []string{},
					Content: "mentions widget in body only",
				}))
				require.NoError(t, NewAgentStore(client).CreateAgentKnowledge(ctx, &gen.AgentKnowledge{
					Path:    "/docs/b.md",
					Title:   "Widget Guide",
					Tags:    []string{},
					Content: "unrelated",
				}))
				rows, err := NewAgentStore(client).SearchAgentKnowledge(ctx, AgentKnowledgeSearchParams{Query: "widget", Limit: 10})
				require.NoError(t, err)
				require.GreaterOrEqual(t, len(rows), 2)
				assert.Equal(t, "/docs/b.md", rows[0].Path)
			},
		},
		{
			name: "search finds content match outside recent window",
			run: func(t *testing.T, client *gen.Client) {
				require.NoError(t, NewAgentStore(client).CreateAgentKnowledge(ctx, &gen.AgentKnowledge{
					Path:      "/docs/old-match.md",
					Title:     "Old",
					Tags:      []string{},
					Content:   "unique-needle-token",
					UpdatedAt: time.Now().Add(-48 * time.Hour),
				}))
				for i := range 120 {
					require.NoError(t, NewAgentStore(client).CreateAgentKnowledge(ctx, &gen.AgentKnowledge{
						Path:    "/docs/recent-" + strconv.Itoa(i) + ".md",
						Title:   "Recent",
						Tags:    []string{},
						Content: "noise",
					}))
				}
				rows, err := NewAgentStore(client).SearchAgentKnowledge(ctx, AgentKnowledgeSearchParams{Query: "unique-needle-token", Limit: 10})
				require.NoError(t, err)
				require.Len(t, rows, 1)
				assert.Equal(t, "/docs/old-match.md", rows[0].Path)
			},
		},
		{
			name: "search finds tag-only match",
			run: func(t *testing.T, client *gen.Client) {
				require.NoError(t, NewAgentStore(client).CreateAgentKnowledge(ctx, &gen.AgentKnowledge{
					Path:    "/scripts/run.md",
					Title:   "Homelab Data Hub & Capability Orchestration Center",
					Tags:    []string{"flowbot", "homelab"},
					Summary: "",
					Content: "Homelab Data Hub overview without the product codename in body",
				}))
				rows, err := NewAgentStore(client).SearchAgentKnowledge(ctx, AgentKnowledgeSearchParams{Query: "flowbot", Limit: 10})
				require.NoError(t, err)
				require.Len(t, rows, 1)
				assert.Equal(t, "/scripts/run.md", rows[0].Path)
			},
		},
		{
			name: "search tag match is case-insensitive",
			run: func(t *testing.T, client *gen.Client) {
				require.NoError(t, NewAgentStore(client).CreateAgentKnowledge(ctx, &gen.AgentKnowledge{
					Path:    "/docs/tag-case.md",
					Title:   "Other Title",
					Tags:    []string{"FlowBot"},
					Content: "no needle in body",
				}))
				rows, err := NewAgentStore(client).SearchAgentKnowledge(ctx, AgentKnowledgeSearchParams{Query: "flowbot", Limit: 10})
				require.NoError(t, err)
				require.Len(t, rows, 1)
				assert.Equal(t, "/docs/tag-case.md", rows[0].Path)
			},
		},
		{
			name: "search requires query or path prefix",
			run: func(t *testing.T, client *gen.Client) {
				_, err := NewAgentStore(client).SearchAgentKnowledge(ctx, AgentKnowledgeSearchParams{})
				require.Error(t, err)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.run(t, sqlitetest.OpenClient(t, t.Name()))
		})
	}
}

func TestAgentMemoryFactsAndSessionSummaries(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tests := []struct {
		name string
		run  func(t *testing.T, client *gen.Client)
	}{
		{
			name: "upsert get list delete memory fact",
			run: func(t *testing.T, client *gen.Client) {
				row, err := NewAgentStore(client).UpsertAgentMemoryFact(ctx, AgentMemoryFactUpsert{
					Scope: "default", Key: "user.name", Value: "Robin", Pinned: true,
				})
				require.NoError(t, err)
				assert.Positive(t, row.ID)
				got, err := NewAgentStore(client).GetAgentMemoryFact(ctx, "default", "user.name")
				require.NoError(t, err)
				assert.Equal(t, "Robin", got.Value)
				assert.True(t, got.Pinned)
				listed, err := NewAgentStore(client).ListAgentMemoryFacts(ctx, "default")
				require.NoError(t, err)
				require.Len(t, listed, 1)
				require.NoError(t, NewAgentStore(client).DeleteAgentMemoryFact(ctx, "default", "user.name"))
				_, err = NewAgentStore(client).GetAgentMemoryFact(ctx, "default", "user.name")
				require.ErrorIs(t, err, types.ErrNotFound)
			},
		},
		{
			name: "injectable facts prefer pinned then truncate by count",
			run: func(t *testing.T, client *gen.Client) {
				_, err := NewAgentStore(client).UpsertAgentMemoryFact(ctx, AgentMemoryFactUpsert{
					Scope: "s", Key: "a", Value: "1", Pinned: false,
				})
				require.NoError(t, err)
				time.Sleep(2 * time.Millisecond)
				_, err = NewAgentStore(client).UpsertAgentMemoryFact(ctx, AgentMemoryFactUpsert{
					Scope: "s", Key: "b", Value: "2", Pinned: true,
				})
				require.NoError(t, err)
				time.Sleep(2 * time.Millisecond)
				_, err = NewAgentStore(client).UpsertAgentMemoryFact(ctx, AgentMemoryFactUpsert{
					Scope: "s", Key: "c", Value: "3", Pinned: false,
				})
				require.NoError(t, err)
				rows, err := NewAgentStore(client).ListInjectableAgentMemoryFacts(ctx, AgentMemoryInjectableParams{
					Scope: "s", MaxCount: 2, MaxChars: 4000,
				})
				require.NoError(t, err)
				require.Len(t, rows, 2)
				assert.Equal(t, "b", rows[0].Key)
				fp, err := NewAgentStore(client).GetAgentMemoryFactsFingerprint(ctx, "s")
				require.NoError(t, err)
				assert.Equal(t, 3, fp.Count)
				assert.NotEmpty(t, fp.ContentHash)
			},
		},
		{
			name: "injectable facts skip oversized first fact",
			run: func(t *testing.T, client *gen.Client) {
				big := strings.Repeat("x", 5000)
				_, err := NewAgentStore(client).UpsertAgentMemoryFact(ctx, AgentMemoryFactUpsert{
					Scope: "budget", Key: "huge", Value: big, Pinned: true,
				})
				require.NoError(t, err)
				rows, err := NewAgentStore(client).ListInjectableAgentMemoryFacts(ctx, AgentMemoryInjectableParams{
					Scope: "budget", MaxCount: 30, MaxChars: 4000,
				})
				require.NoError(t, err)
				assert.Empty(t, rows)
			},
		},
		{
			name: "session summary pending claim ready and search",
			run: func(t *testing.T, client *gen.Client) {
				_, err := NewAgentStore(client).UpsertAgentSessionSummaryPending(ctx, "sess-1", "default", "Topic A")
				require.NoError(t, err)
				claimed, err := NewAgentStore(client).ClaimAgentSessionSummaryPending(ctx, "tok-1")
				require.NoError(t, err)
				assert.Equal(t, "sess-1", claimed.SessionFlag)
				require.NoError(t, NewAgentStore(client).MarkAgentSessionSummaryReady(ctx, "sess-1", "tok-1", "Topic A", "discussed widgets and backups"))
				rows, err := NewAgentStore(client).SearchAgentSessionSummaries(ctx, AgentSessionSummarySearchParams{
					Query: "widgets", Scope: "default", Limit: 10,
				})
				require.NoError(t, err)
				require.Len(t, rows, 1)
				assert.Equal(t, "sess-1", rows[0].SessionFlag)
			},
		},
		{
			name: "stale claim cannot mark after requeue",
			run: func(t *testing.T, client *gen.Client) {
				_, err := NewAgentStore(client).UpsertAgentSessionSummaryPending(ctx, "sess-race", "default", "Race")
				require.NoError(t, err)
				_, err = NewAgentStore(client).ClaimAgentSessionSummaryPending(ctx, "old-tok")
				require.NoError(t, err)
				_, err = NewAgentStore(client).UpsertAgentSessionSummaryPending(ctx, "sess-race", "default", "Race")
				require.NoError(t, err)
				_, err = NewAgentStore(client).ClaimAgentSessionSummaryPending(ctx, "new-tok")
				require.NoError(t, err)
				err = NewAgentStore(client).MarkAgentSessionSummaryReady(ctx, "sess-race", "old-tok", "Race", "stale write")
				require.ErrorIs(t, err, types.ErrNotFound)
				require.NoError(t, NewAgentStore(client).MarkAgentSessionSummaryReady(ctx, "sess-race", "new-tok", "Race", "fresh write"))
				row, err := NewAgentStore(client).GetAgentSessionSummaryBySession(ctx, "sess-race")
				require.NoError(t, err)
				assert.Equal(t, "fresh write", row.Summary)
			},
		},
		{
			name: "requeue stale claimed pending",
			run: func(t *testing.T, client *gen.Client) {
				_, err := NewAgentStore(client).UpsertAgentSessionSummaryPending(ctx, "sess-stale", "default", "Stale")
				require.NoError(t, err)
				_, err = NewAgentStore(client).ClaimAgentSessionSummaryPending(ctx, "tok-stale")
				require.NoError(t, err)
				// Force claimed_at into the past via update.
				_, err = client.AgentSessionSummary.Update().
					Where(agentsessionsummary.SessionFlagEQ("sess-stale")).
					SetClaimedAt(time.Now().Add(-time.Hour)).
					Save(ctx)
				require.NoError(t, err)
				n, err := NewAgentStore(client).RequeueStaleAgentSessionSummaryPending(ctx, 5*time.Minute)
				require.NoError(t, err)
				assert.Equal(t, 1, n)
				claimed, err := NewAgentStore(client).ClaimAgentSessionSummaryPending(ctx, "tok-2")
				require.NoError(t, err)
				assert.Equal(t, "sess-stale", claimed.SessionFlag)
			},
		},
		{
			name: "search requires query",
			run: func(t *testing.T, client *gen.Client) {
				_, err := NewAgentStore(client).SearchAgentSessionSummaries(ctx, AgentSessionSummarySearchParams{})
				require.Error(t, err)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.run(t, sqlitetest.OpenClient(t, t.Name()))
		})
	}
}
