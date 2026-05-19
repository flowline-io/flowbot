package cron

import (
	"context"
	"crypto/sha1"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/types"
)

func newTestStore(t *testing.T) *cache.RedisStore {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return cache.NewRedisStore(client)
}

func TestRule_ID(t *testing.T) {
	t.Parallel()
	t.Run("rule id", func(t *testing.T) {
		t.Parallel()
		r := Rule{Name: "test_cron"}
		assert.Equal(t, "test_cron", r.ID())
	})
}

func TestRule_TYPE(t *testing.T) {
	t.Parallel()
	t.Run("rule type", func(t *testing.T) {
		t.Parallel()
		r := Rule{Name: "test_cron"}
		assert.Equal(t, types.CronRule, r.TYPE())
	})
}

func TestCronScope_Constants(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		constant CronScope
		want     CronScope
	}{
		{
			name:     "system scope",
			constant: CronScopeSystem,
			want:     CronScope("system"),
		},
		{
			name:     "user scope",
			constant: CronScopeUser,
			want:     CronScope("user"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.constant)
		})
	}
}

func TestNewCronRuleset(t *testing.T) {
	t.Parallel()
	t.Run("single rule", func(t *testing.T) {
		t.Parallel()
		rules := []Rule{
			{
				Name:  "rule1",
				Help:  "Test rule 1",
				Scope: CronScopeSystem,
				When:  "* * * * *",
				Action: func(ctx types.Context) []types.MsgPayload {
					return nil
				},
			},
		}

		store := newTestStore(t)
		rs := NewCronRuleset("test", rules, store)
		assert.NotNil(t, rs)
		assert.Equal(t, "test", rs.Type)
		assert.NotNil(t, rs.store)
		assert.Len(t, rs.cronRules, 1)
		assert.NotNil(t, rs.stop)
		assert.NotNil(t, rs.outCh)
	})
}

func TestNewCronRuleset_EmptyRules(t *testing.T) {
	t.Parallel()
	t.Run("empty rules", func(t *testing.T) {
		t.Parallel()
		store := newTestStore(t)
		rs := NewCronRuleset("empty", []Rule{}, store)
		assert.NotNil(t, rs)
		assert.Equal(t, "empty", rs.Type)
		assert.Empty(t, rs.cronRules)
	})
}

func TestNewCronRuleset_MultipleRules(t *testing.T) {
	t.Parallel()
	t.Run("multiple rules", func(t *testing.T) {
		t.Parallel()
		rules := []Rule{
			{Name: "rule1", Scope: CronScopeSystem, When: "0 * * * *"},
			{Name: "rule2", Scope: CronScopeUser, When: "*/5 * * * *"},
			{Name: "rule3", Scope: CronScopeSystem, When: "0 0 * * *"},
		}

		store := newTestStore(t)
		rs := NewCronRuleset("multi", rules, store)
		assert.Len(t, rs.cronRules, 3)
	})
}

func TestRule_AllFields(t *testing.T) {
	t.Parallel()
	t.Run("all fields", func(t *testing.T) {
		t.Parallel()
		called := false
		r := Rule{
			Name:  "daily_summary",
			Help:  "Generate daily summary",
			Scope: CronScopeUser,
			When:  "0 9 * * *",
			Action: func(ctx types.Context) []types.MsgPayload {
				called = true
				return []types.MsgPayload{types.TextMsg{Text: "summary"}}
			},
		}

		assert.Equal(t, "daily_summary", r.Name)
		assert.Equal(t, "Generate daily summary", r.Help)
		assert.Equal(t, CronScopeUser, r.Scope)
		assert.Equal(t, "0 9 * * *", r.When)

		result := r.Action(types.Context{})
		assert.True(t, called)
		assert.Len(t, result, 1)
	})
}

func TestUn(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		payload types.MsgPayload
		want    []byte
	}{
		{
			name:    "TextMsg",
			payload: types.TextMsg{Text: "hello"},
			want:    []byte("hello"),
		},
		{
			name:    "InfoMsg",
			payload: types.InfoMsg{Title: "info title"},
			want:    []byte("info title"),
		},
		{
			name:    "LinkMsg",
			payload: types.LinkMsg{Url: "https://example.com"},
			want:    []byte("https://example.com"),
		},
		{
			name:    "nil payload",
			payload: nil,
			want:    nil,
		},
		{
			name:    "unhandled type",
			payload: types.EmptyMsg{},
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := un(tt.payload)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRuleset_Shutdown(t *testing.T) {
	t.Parallel()
	t.Run("shutdown", func(t *testing.T) {
		t.Parallel()
		store := newTestStore(t)
		rs := NewCronRuleset("test", []Rule{}, store)
		go func() {
			rs.Shutdown()
		}()
		<-rs.stop
	})
}

func TestRule_ScopeValues(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		constant CronScope
		want     CronScope
	}{
		{
			name:     "system scope",
			constant: CronScopeSystem,
			want:     CronScope("system"),
		},
		{
			name:     "user scope",
			constant: CronScopeUser,
			want:     CronScope("user"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.constant)
		})
	}

	t.Run("system and user are not equal", func(t *testing.T) {
		t.Parallel()
		assert.NotEqual(t, CronScopeSystem, CronScopeUser)
	})
}

func TestNewCronRuleset_ChannelCapacity(t *testing.T) {
	t.Parallel()
	t.Run("channel capacity", func(t *testing.T) {
		t.Parallel()
		store := newTestStore(t)
		rs := NewCronRuleset("test", []Rule{}, store)
		assert.Equal(t, 100, cap(rs.outCh))
	})
}

func TestRule_ActionReturnsEmpty(t *testing.T) {
	t.Parallel()
	t.Run("action returns empty", func(t *testing.T) {
		t.Parallel()
		r := Rule{
			Name:  "empty_action",
			Scope: CronScopeSystem,
			When:  "* * * * *",
			Action: func(ctx types.Context) []types.MsgPayload {
				return nil
			},
		}
		result := r.Action(types.Context{})
		assert.Nil(t, result)
	})
}

func TestRuleset_Filter(t *testing.T) {
	tests := []struct {
		name       string
		results    []result
		wantFilter []bool
	}{
		{
			name: "first call passes through",
			results: []result{
				{
					name:    "test_cron",
					ctx:     types.Context{AsUser: types.Uid("user1")},
					payload: types.TextMsg{Text: "hello"},
				},
			},
			wantFilter: []bool{false},
		},
		{
			name: "duplicate content is filtered",
			results: []result{
				{
					name:    "test_cron",
					ctx:     types.Context{AsUser: types.Uid("user1")},
					payload: types.TextMsg{Text: "hello"},
				},
				{
					name:    "test_cron",
					ctx:     types.Context{AsUser: types.Uid("user1")},
					payload: types.TextMsg{Text: "hello"},
				},
			},
			wantFilter: []bool{false, true},
		},
		{
			name: "different content passes through",
			results: []result{
				{
					name:    "test_cron",
					ctx:     types.Context{AsUser: types.Uid("user1")},
					payload: types.TextMsg{Text: "hello"},
				},
				{
					name:    "test_cron",
					ctx:     types.Context{AsUser: types.Uid("user1")},
					payload: types.TextMsg{Text: "world"},
				},
			},
			wantFilter: []bool{false, false},
		},
		{
			name: "different users pass through",
			results: []result{
				{
					name:    "test_cron",
					ctx:     types.Context{AsUser: types.Uid("user1")},
					payload: types.TextMsg{Text: "hello"},
				},
				{
					name:    "test_cron",
					ctx:     types.Context{AsUser: types.Uid("user2")},
					payload: types.TextMsg{Text: "hello"},
				},
			},
			wantFilter: []bool{false, false},
		},
		{
			name: "different cron names pass through",
			results: []result{
				{
					name:    "cron_a",
					ctx:     types.Context{AsUser: types.Uid("user1")},
					payload: types.TextMsg{Text: "hello"},
				},
				{
					name:    "cron_b",
					ctx:     types.Context{AsUser: types.Uid("user1")},
					payload: types.TextMsg{Text: "hello"},
				},
			},
			wantFilter: []bool{false, false},
		},
		{
			name: "triple duplicate only first passes",
			results: []result{
				{
					name:    "test_cron",
					ctx:     types.Context{AsUser: types.Uid("user1")},
					payload: types.TextMsg{Text: "repeat"},
				},
				{
					name:    "test_cron",
					ctx:     types.Context{AsUser: types.Uid("user1")},
					payload: types.TextMsg{Text: "repeat"},
				},
				{
					name:    "test_cron",
					ctx:     types.Context{AsUser: types.Uid("user1")},
					payload: types.TextMsg{Text: "repeat"},
				},
			},
			wantFilter: []bool{false, true, true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newTestStore(t)
			rs := NewCronRuleset("test", []Rule{}, store)

			for i, res := range tt.results {
				filtered := rs.filter(res)
				if tt.wantFilter[i] {
					assert.Equal(t, result{}, filtered, "step %d: expected filtered result", i)
				} else {
					assert.Equal(t, res, filtered, "step %d: expected result to pass through", i)
				}
			}
		})
	}
}

func TestRuleset_Filter_KeyIsolationByUser(t *testing.T) {
	t.Run("keys are isolated by AsUser", func(t *testing.T) {
		store := newTestStore(t)
		rs := NewCronRuleset("test", []Rule{}, store)

		res1 := result{
			name:    "test_cron",
			ctx:     types.Context{AsUser: types.Uid("user1")},
			payload: types.TextMsg{Text: "hello"},
		}
		res2 := result{
			name:    "test_cron",
			ctx:     types.Context{AsUser: types.Uid("user2")},
			payload: types.TextMsg{Text: "hello"},
		}

		filtered1 := rs.filter(res1)
		assert.Equal(t, res1, filtered1, "user1 first call should pass")

		filtered2 := rs.filter(res2)
		assert.Equal(t, res2, filtered2, "user2 first call should pass")

		filtered1dup := rs.filter(res1)
		assert.Equal(t, result{}, filtered1dup, "user1 duplicate should be filtered")
	})
}

func TestRuleset_Filter_HashCorrectness(t *testing.T) {
	t.Run("hash matches expected pattern", func(t *testing.T) {
		store := newTestStore(t)
		rs := NewCronRuleset("test_cron", []Rule{}, store)

		payload := types.TextMsg{Text: "hello"}
		expectedHash := sha1.Sum([]byte(payload.Text))

		res := result{
			name:    "test_cron",
			ctx:     types.Context{AsUser: types.Uid("user1")},
			payload: payload,
		}

		key := cache.NewKey("cron", "filter", "test_cron:user1")
		ctx := context.Background()
		ok, _ := store.IsMember(ctx, key, string(expectedHash[:]))
		assert.False(t, ok, "hash should not exist before filter call")

		rs.filter(res)
		ok, _ = store.IsMember(ctx, key, string(expectedHash[:]))
		assert.True(t, ok, "hash should exist after filter call")
	})
}

