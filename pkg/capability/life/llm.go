package life

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/bytedance/sonic"
	"github.com/tmc/langchaingo/llms"

	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	pkglife "github.com/flowline-io/flowbot/pkg/life"
	"github.com/flowline-io/flowbot/pkg/types"
)

const (
	evaluateMaxTokens   = 512
	adjudicateMaxTokens = 900
	loreMaxTokens       = 256
	breakdownMaxTokens  = 1600
	evaluateTimeout     = 45 * time.Second
	adjudicateTimeout   = 45 * time.Second
	loreTimeout         = 45 * time.Second
	breakdownTimeout    = 60 * time.Second
	evaluatePromptLimit = 4000
)

var allowedDifficulties = map[string]struct{}{
	"E": {}, "D": {}, "C": {}, "B": {}, "A": {}, "S": {}, "SS": {}, "SSS": {},
}

var allowedQuestTypes = map[string]struct{}{
	"One-Time": {}, "Daily": {}, "Boss": {},
}

var allowedAdjudicationVerdicts = map[string]struct{}{
	"completed": {}, "partial": {}, "failed": {}, "needs_more_evidence": {},
}

var allowedStatCodes = func() map[string]struct{} {
	m := make(map[string]struct{}, len(pkglife.DefaultCharacteristics))
	for _, c := range pkglife.DefaultCharacteristics {
		m[c.Code] = struct{}{}
	}
	return m
}()

const evaluateSystemPrompt = `You are the Life dungeon master for a solo gamified productivity RPG.
Judge how hard the player's real-life task is. Respond with ONE JSON object only (no markdown fences, no commentary):
{"title":"...","skill_name":"...","stat_code":"INT|PHY|WIL|CHA|CRE|FIN|WRI|FOC","difficulty":"E|D|C|B|A|S|SS|SSS","quest_type":"One-Time|Daily|Boss"}

Do NOT invent fear, exp, gold, or drop_tier — the server derives those from difficulty.

Difficulty rubric (be strict; do not default everything to C):
- E: tiny chore, <10 minutes, no thinking (reply to one message, tidy desk)
- D: short simple task, <30 minutes
- C: routine single-session work, ~1 hour, low risk
- B: focused skilled work, half day, clear deliverable
- A: hard deep work or non-trivial feature/research, full day+, real failure risk
- S: multi-day hard project or high-stakes delivery
- SS: week-scale campaign / major milestone
- SSS: boss raid — launch, thesis, 30-day challenge; quest_type MUST be Boss

stat_code guide:
- INT: coding, systems, technical learning
- PHY: exercise, sleep, body
- WIL: habits, discipline, meditation
- CHA: meetings, talks, social
- CRE: design, ideation, art
- FIN: money, budget, invoices
- WRI: writing, docs, essays
- FOC: pure deep-focus / time-boxing with no better fit

quest_type:
- Daily: recurring habit (daily, every day, habit)
- Boss: SS/SSS milestones, launches, boss raids
- One-Time: everything else

title: concise (<=80 chars), prefer the player's language.
skill_name: concrete 2-6 word skill label in the player's language.

Examples:
prompt "reply to one email" → {"title":"Reply to email","skill_name":"Quick Outreach","stat_code":"CHA","difficulty":"E","quest_type":"One-Time"}
prompt "implement the AIContext feature" → {"title":"Implement AIContext","skill_name":"Systems Design","stat_code":"INT","difficulty":"A","quest_type":"One-Time"}
prompt "research why water can feel slippery or astringent" → {"title":"Water slip vs astringency","skill_name":"Scientific Inquiry","stat_code":"INT","difficulty":"B","quest_type":"One-Time"}
prompt "run 5km every day" → {"title":"Daily 5km run","skill_name":"Cardio Training","stat_code":"PHY","difficulty":"C","quest_type":"Daily"}
prompt "ship the product launch to production" → {"title":"Ship product launch","skill_name":"Launch Ops","stat_code":"INT","difficulty":"SSS","quest_type":"Boss"}`

const loreSystemPrompt = `You are the Life dungeon master writing memorial lore for a dropped equipment instance.
Respond with a single JSON object only, no markdown fences:
{"name":"instance display name","lore":"1-3 sentences of flavor text"}
Rules:
- name should weave the equipment base name with the quest context when present.
- lore should feel earned and specific to the quest, under 280 characters.
- plain text only; no markdown.
- Prefer English unless the quest title clearly uses another language.`

const adjudicationSystemPrompt = `You are the Life dungeon master adjudicating whether a real-world quest should count as cleared.
Respond with ONE JSON object only, no markdown fences:
{"verdict":"completed|partial|failed|needs_more_evidence","reason":"...","suggested_exp":0,"suggested_gold":0,"suggested_next_steps":["..."]}

Rules:
- Base your ruling on the supplied evidence only.
- Use "needs_more_evidence" when proof is too weak or too vague.
- Use "partial" when real progress exists but the quest is not fully cleared.
- suggested_exp and suggested_gold must be integers between 0 and the supplied base rewards.
- suggested_next_steps should contain 0-3 short concrete steps.
- Keep reason under 240 characters.
- Plain text only.`

const breakdownSystemPrompt = `You are the Life planning dungeon master.
Break one concrete result goal into an execution tree. Respond with ONE JSON object only, no markdown fences.

Schema:
{
  "node_type":"goal",
  "title":"...",
  "description":"...",
  "children":[
    {
      "node_type":"milestone|project|action",
      "title":"...",
      "description":"...",
      "action":{
        "is_repeatable":true,
        "tracking_mode":"completion|consistency",
        "repeat_trigger":"time|condition|none",
        "suggested_cadence":"daily|weekly|",
        "is_identity_building":false,
        "difficulty":"E|D|C|B|A|S|SS|SSS",
        "reason":"..."
      },
      "children":[]
    }
  ]
}

Rules:
- Root node_type MUST be "goal".
- Valid hierarchy only: goal -> milestone/project; milestone -> project; project -> action.
- Keep trees practical and small. Usually 2-4 children per non-leaf, 1-3 action children per project.
- Leaves must be action nodes only.
- Use task semantics:
  - todo: action with is_repeatable=false
  - recurring: action with is_repeatable=true and tracking_mode="completion"
  - habit candidate: action with is_repeatable=true and tracking_mode="consistency"
- Only use suggested_cadence when repeat_trigger="time"; prefer daily or weekly.
- Titles should be concrete and concise.
- Every action must include difficulty (E through SSS). Do not invent dates, estimates, or reward numbers.
- Plain text only.`

// modelResolver resolves a configured chat model name to a langchaingo client.
type modelResolver func(context.Context, string) (llms.Model, string, error)

// LLMService evaluates quests and lore via the chat agent model.
type LLMService struct {
	ResolveModel modelResolver
	ChatModel    func() string
}

// NewLLM returns an LLM-backed life capability service.
func NewLLM() *LLMService {
	return &LLMService{
		ResolveModel: agentllm.GetOrCreateModel,
		ChatModel:    config.ChatAgentChatModel,
	}
}

// EvaluateQuest asks the LLM to score the prompt into a QuestEvaluation.
func (s *LLMService) EvaluateQuest(ctx context.Context, req EvaluateQuestRequest) (*QuestEvaluation, error) {
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "life capability: empty prompt")
	}
	ev, err := s.evaluateWithLLM(ctx, req, prompt)
	if err != nil {
		return nil, err
	}
	return normalizeEvaluation(ev, prompt), nil
}

// AdjudicateQuest asks the LLM to turn quest evidence into a structured ruling.
func (s *LLMService) AdjudicateQuest(ctx context.Context, req AdjudicateQuestRequest) (*QuestAdjudication, error) {
	title := strings.TrimSpace(req.QuestTitle)
	if title == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "life capability: empty quest title")
	}
	model, resolvedName, err := s.resolveChatModel(ctx)
	if err != nil {
		return nil, err
	}
	genCtx, cancel := context.WithTimeout(ctx, adjudicateTimeout)
	defer cancel()
	raw, err := agentllm.Complete(genCtx, model, adjudicationSystemPrompt, []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, buildAdjudicationUserMessage(req, title)),
	}, resolvedName, adjudicateMaxTokens)
	if err != nil {
		return nil, wrapLLMComplete(err)
	}
	flog.InfoFields("life: adjudicate quest llm raw", map[string]any{
		"model": resolvedName,
		"quest": truncate(title, 120),
		"raw":   truncate(raw, 800),
	})
	adjudication, err := parseAdjudicationJSON(raw)
	if err != nil {
		return nil, err
	}
	return normalizeAdjudication(adjudication, req), nil
}

// GenerateInstanceLore asks the LLM for memorial item name and lore.
func (s *LLMService) GenerateInstanceLore(ctx context.Context, questTitle, equipmentName, rarity string) (*InstanceLore, error) {
	equipmentName = strings.TrimSpace(equipmentName)
	if equipmentName == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "life capability: empty equipment name")
	}
	if strings.TrimSpace(rarity) == "" {
		rarity = "Common"
	}
	model, resolvedName, err := s.resolveChatModel(ctx)
	if err != nil {
		return nil, err
	}
	genCtx, cancel := context.WithTimeout(ctx, loreTimeout)
	defer cancel()
	user := fmt.Sprintf("quest_title: %s\nequipment_name: %s\nrarity: %s",
		truncate(questTitle, 120), equipmentName, rarity)
	raw, err := agentllm.Complete(genCtx, model, loreSystemPrompt, []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, user),
	}, resolvedName, loreMaxTokens)
	if err != nil {
		return nil, wrapLLMComplete(err)
	}
	lore, err := parseLoreJSON(raw)
	if err != nil {
		return nil, err
	}
	return normalizeLore(lore, questTitle, equipmentName), nil
}

// BreakdownGoalTree asks the LLM for a structured suggested plan tree.
func (s *LLMService) BreakdownGoalTree(ctx context.Context, req GoalBreakdownRequest) (*GoalBreakdownSuggestion, error) {
	title := strings.TrimSpace(req.RootTitle)
	if title == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "life capability: empty root title")
	}
	model, resolvedName, err := s.resolveChatModel(ctx)
	if err != nil {
		return nil, err
	}
	genCtx, cancel := context.WithTimeout(ctx, breakdownTimeout)
	defer cancel()
	raw, err := agentllm.Complete(genCtx, model, breakdownSystemPrompt, []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, buildBreakdownUserMessage(req, title)),
	}, resolvedName, breakdownMaxTokens)
	if err != nil {
		return nil, wrapLLMComplete(err)
	}
	flog.InfoFields("life: breakdown goal llm raw", map[string]any{
		"model":      resolvedName,
		"root_title": truncate(title, 120),
		"raw":        truncate(raw, 800),
	})
	tree, err := parseBreakdownJSON(raw)
	if err != nil {
		return nil, err
	}
	return normalizeBreakdownSuggestion(tree, title), nil
}

func (s *LLMService) evaluateWithLLM(ctx context.Context, req EvaluateQuestRequest, prompt string) (*QuestEvaluation, error) {
	model, resolvedName, err := s.resolveChatModel(ctx)
	if err != nil {
		return nil, err
	}
	genCtx, cancel := context.WithTimeout(ctx, evaluateTimeout)
	defer cancel()
	userPrompt := buildEvaluateUserMessage(req, prompt)
	raw, err := agentllm.Complete(genCtx, model, evaluateSystemPrompt, []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, userPrompt),
	}, resolvedName, evaluateMaxTokens)
	if err != nil {
		return nil, wrapLLMComplete(err)
	}
	flog.InfoFields("life: evaluate quest llm raw", map[string]any{
		"model":  resolvedName,
		"prompt": truncate(prompt, 120),
		"raw":    truncate(raw, 500),
	})
	return parseEvaluationJSON(raw)
}

func buildEvaluateUserMessage(req EvaluateQuestRequest, prompt string) string {
	parts := []string{
		"Player quest prompt:",
		truncateRunes(prompt, evaluatePromptLimit),
		"",
		"Context:",
	}
	if req.AIPersonality != "" {
		parts = append(parts, "- DM personality: "+req.AIPersonality)
	}
	if req.CompletionRate > 0 {
		parts = append(parts, fmt.Sprintf("- historical completion rate: %.2f", req.CompletionRate))
	}
	if len(req.ActiveGoals) > 0 {
		parts = append(parts, "- active goals: "+strings.Join(req.ActiveGoals, " | "))
	}
	if depth, ok := req.Privileges["ai_breakdown_depth"].(string); ok && depth != "" {
		req.BreakdownDepth = depth
	}
	if req.BreakdownDepth != "" {
		parts = append(parts, "- equipped AI privilege breakdown_depth="+req.BreakdownDepth+" (be more precise on skill naming and difficulty)")
	}
	if len(req.Mood) > 0 {
		if raw, err := sonic.Marshal(req.Mood); err == nil {
			parts = append(parts, "- recent mood: "+string(raw))
		}
	}
	return strings.Join(parts, "\n")
}

func buildBreakdownUserMessage(req GoalBreakdownRequest, title string) string {
	parts := []string{
		"Root goal:",
		title,
	}
	if desc := strings.TrimSpace(req.Description); desc != "" {
		parts = append(parts, "", "Goal description:", truncateRunes(desc, evaluatePromptLimit))
	}
	parts = append(parts, "", "Context:")
	if req.AIPersonality != "" {
		parts = append(parts, "- DM personality: "+req.AIPersonality)
	}
	if len(req.ActiveGoals) > 0 {
		parts = append(parts, "- active goals: "+strings.Join(req.ActiveGoals, " | "))
	}
	if depth, ok := req.Privileges["ai_breakdown_depth"].(string); ok && depth != "" {
		req.BreakdownDepth = depth
	}
	if req.BreakdownDepth != "" {
		parts = append(parts, "- breakdown depth hint: "+req.BreakdownDepth)
	}
	return strings.Join(parts, "\n")
}

func buildAdjudicationUserMessage(req AdjudicateQuestRequest, title string) string {
	parts := []string{
		"Quest:",
		title,
		"",
		"Quest context:",
		fmt.Sprintf("- type: %s", defaultString(req.QuestType, "One-Time")),
		fmt.Sprintf("- difficulty: %s", defaultString(req.Difficulty, "E")),
		fmt.Sprintf("- base rewards: %d exp, %d gold", max(req.BaseExp, 0), max(req.BaseGold, 0)),
	}
	if prompt := strings.TrimSpace(req.QuestPrompt); prompt != "" {
		parts = append(parts, "- prompt: "+truncateRunes(prompt, evaluatePromptLimit))
	}
	if req.AIPersonality != "" {
		parts = append(parts, "- DM personality: "+req.AIPersonality)
	}
	if req.CompletionRate > 0 {
		parts = append(parts, fmt.Sprintf("- historical completion rate: %.2f", req.CompletionRate))
	}
	if len(req.ActiveGoals) > 0 {
		parts = append(parts, "- active goals: "+strings.Join(req.ActiveGoals, " | "))
	}
	if len(req.Mood) > 0 {
		if raw, err := sonic.Marshal(req.Mood); err == nil {
			parts = append(parts, "- recent mood: "+string(raw))
		}
	}
	parts = append(parts, "", "Evidence:")
	if len(req.Evidence) == 0 {
		parts = append(parts, "- none submitted")
	} else {
		for _, ev := range req.Evidence {
			line := fmt.Sprintf("- [%s] %s", defaultString(ev.SourceType, "note"), truncateRunes(strings.TrimSpace(ev.Content), 500))
			if summary := strings.TrimSpace(ev.Summary); summary != "" {
				line += " | summary: " + truncateRunes(summary, 120)
			}
			if sourceURL := strings.TrimSpace(ev.SourceURL); sourceURL != "" {
				line += " | url: " + truncateRunes(sourceURL, 160)
			}
			parts = append(parts, line)
		}
	}
	if len(req.RecentActionLog) > 0 {
		parts = append(parts, "", "Recent action log context:")
		for _, row := range req.RecentActionLog {
			if raw, err := sonic.Marshal(row); err == nil {
				parts = append(parts, "- "+truncateRunes(string(raw), 220))
			}
		}
	}
	return strings.Join(parts, "\n")
}

func (s *LLMService) resolveChatModel(ctx context.Context) (llms.Model, string, error) {
	if s == nil {
		return nil, "", types.Errorf(types.ErrProvider, "life capability: llm service is nil")
	}
	chatModel := ""
	if s.ChatModel != nil {
		chatModel = s.ChatModel()
	}
	if chatModel == "" {
		return nil, "", types.Errorf(types.ErrProvider, "chat agent model is not configured")
	}
	resolve := s.ResolveModel
	if resolve == nil {
		resolve = agentllm.GetOrCreateModel
	}
	model, resolvedName, err := resolve(ctx, chatModel)
	if err != nil {
		return nil, "", types.WrapError(types.ErrProvider, "life llm model", err)
	}
	return model, resolvedName, nil
}

func wrapLLMComplete(err error) error {
	return types.WrapError(types.ErrProvider, "life llm complete", err)
}

func parseEvaluationJSON(raw string) (*QuestEvaluation, error) {
	raw = extractJSONObject(raw)
	var ev QuestEvaluation
	if err := sonic.Unmarshal([]byte(raw), &ev); err != nil {
		return nil, types.WrapError(types.ErrProvider, "parse evaluate json", err)
	}
	if strings.TrimSpace(ev.Title) == "" || strings.TrimSpace(ev.StatCode) == "" {
		return nil, types.Errorf(types.ErrProvider, "parse evaluate json: missing title or stat_code")
	}
	return &ev, nil
}

func parseLoreJSON(raw string) (*InstanceLore, error) {
	raw = extractJSONObject(raw)
	var lore InstanceLore
	if err := sonic.Unmarshal([]byte(raw), &lore); err != nil {
		return nil, types.WrapError(types.ErrProvider, "parse lore json", err)
	}
	if strings.TrimSpace(lore.Name) == "" || strings.TrimSpace(lore.Lore) == "" {
		return nil, types.Errorf(types.ErrProvider, "parse lore json: missing name or lore")
	}
	return &lore, nil
}

func parseAdjudicationJSON(raw string) (*QuestAdjudication, error) {
	raw = extractJSONObject(raw)
	var ruling QuestAdjudication
	if err := sonic.Unmarshal([]byte(raw), &ruling); err != nil {
		return nil, types.WrapError(types.ErrProvider, "parse adjudication json", err)
	}
	if strings.TrimSpace(ruling.Verdict) == "" {
		return nil, types.Errorf(types.ErrProvider, "parse adjudication json: missing verdict")
	}
	return &ruling, nil
}

func parseBreakdownJSON(raw string) (*GoalBreakdownSuggestion, error) {
	raw = extractJSONObject(raw)
	var tree GoalBreakdownSuggestion
	if err := sonic.Unmarshal([]byte(raw), &tree); err != nil {
		return nil, types.WrapError(types.ErrProvider, "parse breakdown json", err)
	}
	if strings.TrimSpace(tree.Title) == "" {
		return nil, types.Errorf(types.ErrProvider, "parse breakdown json: missing title")
	}
	return &tree, nil
}

func extractJSONObject(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start >= 0 && end > start {
		return raw[start : end+1]
	}
	return raw
}

func normalizeBreakdownSuggestion(tree *GoalBreakdownSuggestion, fallbackTitle string) *GoalBreakdownSuggestion {
	if tree == nil {
		return &GoalBreakdownSuggestion{NodeType: "goal", Title: fallbackTitle}
	}
	normalized := *tree
	if normalizePlanNodeTypeValue(normalized.NodeType) == "" {
		normalized.NodeType = "goal"
	} else {
		normalized.NodeType = normalizePlanNodeTypeValue(normalized.NodeType)
	}
	normalized.Title = strings.TrimSpace(normalized.Title)
	if normalized.Title == "" {
		normalized.Title = fallbackTitle
	}
	normalized.Description = strings.TrimSpace(normalized.Description)
	if normalized.Action != nil {
		normalized.Action.TrackingMode = normalizeTrackingModeValue(normalized.Action.TrackingMode)
		normalized.Action.RepeatTrigger = normalizeRepeatTriggerValue(normalized.Action.RepeatTrigger)
		normalized.Action.SuggestedCadence = normalizeCadenceValue(normalized.Action.SuggestedCadence)
		normalized.Action.Reason = strings.TrimSpace(normalized.Action.Reason)
		normalized.Action.Difficulty = normalizeDifficulty(normalized.Action.Difficulty)
		_, normalized.Action.BaseExp, normalized.Action.BaseGold, _ = pkglife.DefaultRewards(normalized.Action.Difficulty)
	}
	children := make([]GoalBreakdownSuggestion, 0, len(normalized.Children))
	for _, child := range normalized.Children {
		childCopy := normalizeBreakdownSuggestion(&child, child.Title)
		children = append(children, *childCopy)
	}
	normalized.Children = children
	return &normalized
}

func normalizePlanNodeTypeValue(nodeType string) string {
	switch strings.ToLower(strings.TrimSpace(nodeType)) {
	case "goal":
		return "goal"
	case "milestone":
		return "milestone"
	case "project":
		return "project"
	case "action":
		return "action"
	default:
		return ""
	}
}

func normalizeTrackingModeValue(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "consistency":
		return "consistency"
	default:
		return "completion"
	}
}

func normalizeRepeatTriggerValue(trigger string) string {
	switch strings.ToLower(strings.TrimSpace(trigger)) {
	case "time":
		return "time"
	case "condition":
		return "condition"
	default:
		return "none"
	}
}

func normalizeCadenceValue(cadence string) string {
	switch strings.ToLower(strings.TrimSpace(cadence)) {
	case "daily":
		return "daily"
	case "weekly":
		return "weekly"
	default:
		return ""
	}
}

func normalizeEvaluation(ev *QuestEvaluation, prompt string) *QuestEvaluation {
	if ev == nil {
		ev = &QuestEvaluation{}
	}
	ev.Title = strings.TrimSpace(ev.Title)
	if ev.Title == "" {
		ev.Title = firstSentence(prompt)
	}
	if len(ev.Title) > 80 {
		ev.Title = truncate(ev.Title, 80)
	}
	ev.SkillName = strings.TrimSpace(ev.SkillName)
	if ev.SkillName == "" {
		ev.SkillName = "General Focus"
	}
	ev.StatCode = strings.ToUpper(strings.TrimSpace(ev.StatCode))
	if _, ok := allowedStatCodes[ev.StatCode]; !ok {
		ev.StatCode = "FOC"
	}
	ev.Difficulty = normalizeDifficulty(ev.Difficulty)
	ev.QuestType = strings.TrimSpace(ev.QuestType)
	if _, ok := allowedQuestTypes[ev.QuestType]; !ok {
		if ev.Difficulty == "SSS" || ev.Difficulty == "SS" {
			ev.QuestType = "Boss"
		} else {
			ev.QuestType = "One-Time"
		}
	}
	if (ev.Difficulty == "SSS" || ev.Difficulty == "SS") && ev.QuestType != "Boss" {
		ev.QuestType = "Boss"
	}
	// Rewards are owned by the server difficulty table, not the LLM.
	ev.Fear, ev.BaseExp, ev.BaseGold, ev.DropTier = pkglife.DefaultRewards(ev.Difficulty)
	return ev
}

func normalizeDifficulty(raw string) string {
	return pkglife.NormalizeDifficulty(raw)
}

func normalizeLore(lore *InstanceLore, questTitle, equipmentName string) *InstanceLore {
	if lore == nil {
		lore = &InstanceLore{}
	}
	lore.Name = strings.TrimSpace(lore.Name)
	if lore.Name == "" {
		lore.Name = equipmentName
		if questTitle != "" {
			lore.Name = fmt.Sprintf("%s of %s", equipmentName, truncate(questTitle, 32))
		}
	}
	lore.Lore = strings.TrimSpace(lore.Lore)
	if len(lore.Lore) > 400 {
		lore.Lore = truncate(lore.Lore, 400)
	}
	return lore
}

func normalizeAdjudication(ruling *QuestAdjudication, req AdjudicateQuestRequest) *QuestAdjudication {
	if ruling == nil {
		ruling = &QuestAdjudication{}
	}
	ruling.Verdict = normalizeAdjudicationVerdict(ruling.Verdict, len(req.Evidence) > 0)
	ruling.Reason = normalizeAdjudicationReason(ruling.Reason, ruling.Verdict)
	if len(ruling.Reason) > 240 {
		ruling.Reason = truncate(ruling.Reason, 240)
	}
	ruling.SuggestedExp, ruling.SuggestedGold = normalizeAdjudicationRewards(
		ruling.Verdict,
		ruling.SuggestedExp,
		ruling.SuggestedGold,
		req.BaseExp,
		req.BaseGold,
	)
	ruling.SuggestedNextSteps = normalizeNextSteps(ruling.SuggestedNextSteps)
	return ruling
}

func normalizeAdjudicationVerdict(raw string, hasEvidence bool) string {
	verdict := strings.ToLower(strings.TrimSpace(raw))
	if _, ok := allowedAdjudicationVerdicts[verdict]; ok {
		return verdict
	}
	if hasEvidence {
		return "partial"
	}
	return "needs_more_evidence"
}

func normalizeAdjudicationReason(raw, verdict string) string {
	reason := strings.TrimSpace(raw)
	if reason != "" {
		return reason
	}
	switch verdict {
	case "completed":
		return "Evidence suggests the quest objective was completed."
	case "failed":
		return "Evidence does not show the quest objective was completed."
	case "partial":
		return "Evidence shows meaningful progress, but not full completion yet."
	default:
		return "More specific evidence is required before this quest can be cleared."
	}
}

func normalizeAdjudicationRewards(verdict string, suggestedExp, suggestedGold, baseExp, baseGold int) (int, int) {
	maxExp := max(baseExp, 0)
	maxGold := max(baseGold, 0)
	suggestedExp = clampInt(suggestedExp, 0, maxExp)
	suggestedGold = clampInt(suggestedGold, 0, maxGold)
	switch verdict {
	case "completed":
		if maxExp > 0 && suggestedExp == 0 {
			suggestedExp = maxExp
		}
		if maxGold > 0 && suggestedGold == 0 {
			suggestedGold = maxGold
		}
	case "failed", "needs_more_evidence":
		suggestedExp = 0
		suggestedGold = 0
	}
	return suggestedExp, suggestedGold
}

func normalizeNextSteps(items []string) []string {
	if len(items) > 3 {
		items = items[:3]
	}
	for i := range items {
		items[i] = strings.TrimSpace(items[i])
	}
	return compactStrings(items)
}

func firstSentence(s string) string {
	s = strings.TrimSpace(s)
	for i, r := range s {
		if r == '.' || r == '!' || r == '?' {
			return strings.TrimSpace(s[:i+1])
		}
		if i > 72 && unicode.IsSpace(r) {
			return strings.TrimSpace(s[:i]) + "…"
		}
	}
	return truncate(s, 80)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func truncateRunes(s string, limit int) string {
	if limit <= 0 || s == "" {
		return s
	}
	n := 0
	for i := range s {
		if n == limit {
			return s[:i]
		}
		n++
	}
	return s
}

func defaultString(v, fallback string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return fallback
	}
	return v
}

func clampInt(v, minValue, maxValue int) int {
	if v < minValue {
		return minValue
	}
	if v > maxValue {
		return maxValue
	}
	return v
}

func compactStrings(items []string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		if item == "" {
			continue
		}
		out = append(out, item)
	}
	return out
}
