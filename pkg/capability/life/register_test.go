package life

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/hub"
)

type captureAdjudicateService struct {
	last AdjudicateQuestRequest
}

func (*captureAdjudicateService) EvaluateQuest(context.Context, EvaluateQuestRequest) (*QuestEvaluation, error) {
	return &QuestEvaluation{}, nil
}

func (s *captureAdjudicateService) AdjudicateQuest(_ context.Context, req AdjudicateQuestRequest) (*QuestAdjudication, error) {
	s.last = req
	return &QuestAdjudication{Verdict: "completed", Reason: "ok", SuggestedExp: req.BaseExp, SuggestedGold: req.BaseGold}, nil
}

func (*captureAdjudicateService) GenerateInstanceLore(context.Context, string, string, string) (*InstanceLore, error) {
	return &InstanceLore{}, nil
}

func (*captureAdjudicateService) BreakdownGoalTree(context.Context, GoalBreakdownRequest) (*GoalBreakdownSuggestion, error) {
	return &GoalBreakdownSuggestion{}, nil
}

func TestInvokeAdjudicateAcceptsModuleEvidenceMaps(t *testing.T) {
	hub.Default.Unregister(hub.CapLife)
	capability.DefaultRegistry.Unregister(hub.CapLife, OpAdjudicateQuest)
	t.Cleanup(func() {
		hub.Default.Unregister(hub.CapLife)
		capability.DefaultRegistry.Unregister(hub.CapLife, OpAdjudicateQuest)
	})

	svc := &captureAdjudicateService{}
	require.NoError(t, Register(svc))

	// Module builds evidence as []map[string]any (not []any).
	_, err := capability.Invoke(context.Background(), hub.CapLife, OpAdjudicateQuest, map[string]any{
		"quest_title": "打卡一次",
		"base_exp":    10,
		"base_gold":   3,
		"evidence": []map[string]any{
			{
				"source_type": "note",
				"content":     "打卡时间：2026-08-01 21:06:57",
				"source_url":  "",
				"summary":     "打卡时间：2026-08-01 21:06:57",
			},
		},
		"recent_action_log": []map[string]any{
			{"source_type": "quest", "summary": "earlier quest", "gained_exp": 10, "gained_gold": 3},
		},
	})
	require.NoError(t, err)
	require.Len(t, svc.last.Evidence, 1)
	assert.Equal(t, "note", svc.last.Evidence[0].SourceType)
	assert.Contains(t, svc.last.Evidence[0].Content, "21:06:57")
	assert.Equal(t, 10, svc.last.BaseExp)
	assert.Equal(t, 3, svc.last.BaseGold)
	require.Len(t, svc.last.RecentActionLog, 1)
	assert.Equal(t, "earlier quest", svc.last.RecentActionLog[0]["summary"])
}
