package life_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/llms"

	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
	lifecap "github.com/flowline-io/flowbot/pkg/capability/life"
	"github.com/flowline-io/flowbot/pkg/types"
)

func testLLM(fake *agentllm.FakeModel) *lifecap.LLMService {
	return &lifecap.LLMService{
		ChatModel: func() string { return "test-model" },
		ResolveModel: func(_ context.Context, name string) (llms.Model, string, error) {
			return fake, name, nil
		},
	}
}

func evalReq(prompt string) lifecap.EvaluateQuestRequest {
	return lifecap.EvaluateQuestRequest{Prompt: prompt}
}

func TestLLMEvaluateQuestSuccess(t *testing.T) {
	t.Parallel()
	fake := agentllm.NewFakeModel(agentllm.ResponseScript{
		Content: `{"title":"Ship Auth Gateway","skill_name":"Systems Craft","stat_code":"int","difficulty":"A","quest_type":"One-Time"}`,
	})
	ev, err := testLLM(fake).EvaluateQuest(context.Background(), evalReq("refactor the payment API gateway carefully"))
	require.NoError(t, err)
	assert.Equal(t, "Ship Auth Gateway", ev.Title)
	assert.Equal(t, "INT", ev.StatCode)
	assert.Equal(t, "A", ev.Difficulty)
	assert.Equal(t, "Epic", ev.DropTier)
	assert.Equal(t, 65, ev.BaseExp)
	assert.Equal(t, 20, ev.BaseGold)
	assert.Equal(t, 3, ev.Fear)
	assert.Equal(t, 1, fake.Calls())
}

func TestLLMEvaluateQuestServerOwnsRewards(t *testing.T) {
	t.Parallel()
	fake := agentllm.NewFakeModel(agentllm.ResponseScript{
		Content: `{"title":"实现 AIContext","skill_name":"系统设计","stat_code":"INT","difficulty":"A","quest_type":"One-Time","fear":1,"base_exp":40,"base_gold":15,"drop_tier":"Common"}`,
	})
	ev, err := testLLM(fake).EvaluateQuest(context.Background(), evalReq("实现AIContext功能"))
	require.NoError(t, err)
	assert.Equal(t, "A", ev.Difficulty)
	assert.Equal(t, "Epic", ev.DropTier)
	assert.Equal(t, 65, ev.BaseExp)
	assert.Equal(t, 20, ev.BaseGold)
	assert.Equal(t, 3, ev.Fear)
}

func TestLLMEvaluateQuestExtractsEmbeddedJSON(t *testing.T) {
	t.Parallel()
	fake := agentllm.NewFakeModel(agentllm.ResponseScript{
		Content: `这里是评估：{"title":"研究水的滑与涩","skill_name":"科学探究","stat_code":"INT","difficulty":"B","quest_type":"One-Time"}`,
	})
	ev, err := testLLM(fake).EvaluateQuest(context.Background(), evalReq("研究为什么水可以很滑，也可以很涩?"))
	require.NoError(t, err)
	assert.Equal(t, "B", ev.Difficulty)
	assert.Equal(t, "Rare", ev.DropTier)
	assert.Equal(t, 40, ev.BaseExp)
	assert.Equal(t, 3, ev.Fear)
}

func TestLLMEvaluateQuestErrorsOnModelFailure(t *testing.T) {
	t.Parallel()
	svc := &lifecap.LLMService{
		ChatModel: func() string { return "test-model" },
		ResolveModel: func(context.Context, string) (llms.Model, string, error) {
			return nil, "", errors.New("boom")
		},
	}
	_, err := svc.EvaluateQuest(context.Background(), evalReq("go for a long run at the gym"))
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrProvider)
	assert.Contains(t, err.Error(), "life llm model")
}

func TestLLMEvaluateQuestErrorsOnBadJSON(t *testing.T) {
	t.Parallel()
	fake := agentllm.NewFakeModel(agentllm.ResponseScript{Content: "not-json"})
	_, err := testLLM(fake).EvaluateQuest(context.Background(), evalReq("draft a blog essay"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse evaluate json")
}

func TestLLMEvaluateQuestNormalizesUnknownStat(t *testing.T) {
	t.Parallel()
	fake := agentllm.NewFakeModel(agentllm.ResponseScript{
		Content: `{"title":"Mystery Task","skill_name":"Odd Job","stat_code":"XYZ","difficulty":"Z","quest_type":"Raid"}`,
	})
	ev, err := testLLM(fake).EvaluateQuest(context.Background(), evalReq("do something vague"))
	require.NoError(t, err)
	assert.Equal(t, "FOC", ev.StatCode)
	assert.Equal(t, "C", ev.Difficulty)
	assert.Equal(t, "Common", ev.DropTier)
	assert.Equal(t, "One-Time", ev.QuestType)
	assert.Equal(t, 2, ev.Fear)
	assert.Equal(t, 25, ev.BaseExp)
}

func TestLLMEvaluateQuestEmptyPrompt(t *testing.T) {
	t.Parallel()
	_, err := lifecap.NewLLM().EvaluateQuest(context.Background(), evalReq("  "))
	require.Error(t, err)
}

func TestLLMGenerateInstanceLoreSuccess(t *testing.T) {
	t.Parallel()
	fake := agentllm.NewFakeModel(agentllm.ResponseScript{
		Content: `{"name":"Truth Codex of Finish API","lore":"Forged when the gateway finally held."}`,
	})
	lore, err := testLLM(fake).GenerateInstanceLore(context.Background(), "Finish API", "Truth Codex", "Epic")
	require.NoError(t, err)
	assert.Equal(t, "Truth Codex of Finish API", lore.Name)
	assert.Contains(t, lore.Lore, "gateway")
	assert.Equal(t, 1, fake.Calls())
}

func TestLLMGenerateInstanceLoreErrorsOnBadJSON(t *testing.T) {
	t.Parallel()
	fake := agentllm.NewFakeModel(agentllm.ResponseScript{Content: "{}"})
	_, err := testLLM(fake).GenerateInstanceLore(context.Background(), "Q", "Sword", "Rare")
	require.Error(t, err)
}

func TestLLMBreakdownGoalTreeSuccess(t *testing.T) {
	t.Parallel()
	fake := agentllm.NewFakeModel(agentllm.ResponseScript{
		Content: `{"node_type":"goal","title":"Ship Flowbot V2","description":"Deliver the execution layer","children":[{"node_type":"project","title":"Implement execution layer","children":[{"node_type":"action","title":"Add occurrence store","action":{"is_repeatable":false,"tracking_mode":"completion","repeat_trigger":"none","difficulty":"B","base_exp":999,"base_gold":999,"reason":"Unblock today board"}}]}]}`,
	})
	tree, err := testLLM(fake).BreakdownGoalTree(context.Background(), lifecap.GoalBreakdownRequest{
		RootTitle:   "Ship Flowbot V2",
		Description: "Deliver the execution layer",
	})
	require.NoError(t, err)
	require.NotNil(t, tree)
	assert.Equal(t, "goal", tree.NodeType)
	assert.Equal(t, "Ship Flowbot V2", tree.Title)
	require.Len(t, tree.Children, 1)
	require.Len(t, tree.Children[0].Children, 1)
	assert.Equal(t, "action", tree.Children[0].Children[0].NodeType)
	require.NotNil(t, tree.Children[0].Children[0].Action)
	assert.Equal(t, "B", tree.Children[0].Children[0].Action.Difficulty)
	assert.Equal(t, 40, tree.Children[0].Children[0].Action.BaseExp)
	assert.Equal(t, 12, tree.Children[0].Children[0].Action.BaseGold)
}

func TestLLMBreakdownGoalTreeEmptyTitle(t *testing.T) {
	t.Parallel()
	_, err := lifecap.NewLLM().BreakdownGoalTree(context.Background(), lifecap.GoalBreakdownRequest{RootTitle: " "})
	require.Error(t, err)
}

func TestLLMAdjudicateQuestSuccess(t *testing.T) {
	t.Parallel()
	fake := agentllm.NewFakeModel(agentllm.ResponseScript{
		Content: `{"verdict":"completed","reason":"The evidence shows the feature shipped.","suggested_exp":150,"suggested_gold":40,"suggested_next_steps":["Write a short retro"]}`,
	})
	ruling, err := testLLM(fake).AdjudicateQuest(context.Background(), lifecap.AdjudicateQuestRequest{
		QuestTitle: "Ship AI DM MVP",
		QuestPrompt: "Ship the first quest adjudication flow",
		BaseExp:    150,
		BaseGold:   40,
		Evidence: []lifecap.QuestEvidence{
			{SourceType: "note", Content: "Implemented and tested the flow."},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, ruling)
	assert.Equal(t, "completed", ruling.Verdict)
	assert.Equal(t, 150, ruling.SuggestedExp)
	assert.Equal(t, 40, ruling.SuggestedGold)
	assert.Equal(t, []string{"Write a short retro"}, ruling.SuggestedNextSteps)
}

func TestLLMAdjudicateQuestClampsUnsupportedOutput(t *testing.T) {
	t.Parallel()
	fake := agentllm.NewFakeModel(agentllm.ResponseScript{
		Content: `{"verdict":"legendary","reason":"","suggested_exp":999,"suggested_gold":999,"suggested_next_steps":["one","two","three","four"]}`,
	})
	ruling, err := testLLM(fake).AdjudicateQuest(context.Background(), lifecap.AdjudicateQuestRequest{
		QuestTitle: "Collect evidence",
		BaseExp:    40,
		BaseGold:   15,
		Evidence: []lifecap.QuestEvidence{
			{SourceType: "note", Content: "Made progress, but not done."},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "partial", ruling.Verdict)
	assert.Equal(t, 40, ruling.SuggestedExp)
	assert.Equal(t, 15, ruling.SuggestedGold)
	assert.Len(t, ruling.SuggestedNextSteps, 3)
	assert.NotEmpty(t, ruling.Reason)
}
