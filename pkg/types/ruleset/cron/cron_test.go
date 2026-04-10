package cron

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestRule_ID(t *testing.T) {
	r := Rule{Name: "test_cron"}
	assert.Equal(t, "test_cron", r.ID())
}

func TestRule_TYPE(t *testing.T) {
	r := Rule{Name: "test_cron"}
	assert.Equal(t, types.CronRule, r.TYPE())
}

func TestCronScope_Constants(t *testing.T) {
	assert.Equal(t, CronScope("system"), CronScopeSystem)
	assert.Equal(t, CronScope("user"), CronScopeUser)
}

func TestNewCronRuleset(t *testing.T) {
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

	rs := NewCronRuleset("test", rules)
	assert.NotNil(t, rs)
	assert.Equal(t, "test", rs.Type)
	assert.Len(t, rs.cronRules, 1)
	assert.NotNil(t, rs.stop)
	assert.NotNil(t, rs.outCh)
}

func TestNewCronRuleset_EmptyRules(t *testing.T) {
	rs := NewCronRuleset("empty", []Rule{})
	assert.NotNil(t, rs)
	assert.Equal(t, "empty", rs.Type)
	assert.Len(t, rs.cronRules, 0)
}

func TestNewCronRuleset_MultipleRules(t *testing.T) {
	rules := []Rule{
		{Name: "rule1", Scope: CronScopeSystem, When: "0 * * * *"},
		{Name: "rule2", Scope: CronScopeUser, When: "*/5 * * * *"},
		{Name: "rule3", Scope: CronScopeSystem, When: "0 0 * * *"},
	}

	rs := NewCronRuleset("multi", rules)
	assert.Len(t, rs.cronRules, 3)
}

func TestRule_AllFields(t *testing.T) {
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
}

func TestUn(t *testing.T) {
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
			got := un(tt.payload)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRuleset_Shutdown(t *testing.T) {
	rs := NewCronRuleset("test", []Rule{})
	go func() {
		rs.Shutdown()
	}()
	<-rs.stop
}

func TestRule_ScopeValues(t *testing.T) {
	assert.Equal(t, CronScope("system"), CronScopeSystem)
	assert.Equal(t, CronScope("user"), CronScopeUser)
	assert.NotEqual(t, CronScopeSystem, CronScopeUser)
}

func TestNewCronRuleset_ChannelCapacity(t *testing.T) {
	rs := NewCronRuleset("test", []Rule{})
	assert.Equal(t, 100, cap(rs.outCh))
}

func TestRule_ActionReturnsEmpty(t *testing.T) {
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
}

func TestRule_ActionReturnsMultiple(t *testing.T) {
	r := Rule{
		Name:  "multi_action",
		Scope: CronScopeSystem,
		When:  "* * * * *",
		Action: func(ctx types.Context) []types.MsgPayload {
			return []types.MsgPayload{
				types.TextMsg{Text: "msg1"},
				types.TextMsg{Text: "msg2"},
			}
		},
	}
	result := r.Action(types.Context{})
	assert.Len(t, result, 2)
}
