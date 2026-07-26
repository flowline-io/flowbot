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
)

const (
	evaluateMaxTokens   = 512
	loreMaxTokens       = 256
	evaluateTimeout     = 45 * time.Second
	loreTimeout         = 45 * time.Second
	evaluatePromptLimit = 4000
)

var allowedDifficulties = map[string]struct{}{
	"E": {}, "D": {}, "C": {}, "B": {}, "A": {}, "S": {}, "SS": {}, "SSS": {},
}

var allowedQuestTypes = map[string]struct{}{
	"One-Time": {}, "Daily": {}, "Boss": {},
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
		return nil, fmt.Errorf("life capability: empty prompt")
	}
	ev, err := s.evaluateWithLLM(ctx, req, prompt)
	if err != nil {
		return nil, err
	}
	return normalizeEvaluation(ev, prompt), nil
}

// GenerateInstanceLore asks the LLM for memorial item name and lore.
func (s *LLMService) GenerateInstanceLore(ctx context.Context, questTitle, equipmentName, rarity string) (*InstanceLore, error) {
	equipmentName = strings.TrimSpace(equipmentName)
	if equipmentName == "" {
		return nil, fmt.Errorf("life capability: empty equipment name")
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
		return nil, err
	}
	lore, err := parseLoreJSON(raw)
	if err != nil {
		return nil, err
	}
	return normalizeLore(lore, questTitle, equipmentName), nil
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
		return nil, err
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

func (s *LLMService) resolveChatModel(ctx context.Context) (llms.Model, string, error) {
	if s == nil {
		return nil, "", fmt.Errorf("life capability: llm service is nil")
	}
	chatModel := ""
	if s.ChatModel != nil {
		chatModel = s.ChatModel()
	}
	if chatModel == "" {
		return nil, "", fmt.Errorf("chat agent model is not configured")
	}
	resolve := s.ResolveModel
	if resolve == nil {
		resolve = agentllm.GetOrCreateModel
	}
	model, resolvedName, err := resolve(ctx, chatModel)
	if err != nil {
		return nil, "", fmt.Errorf("life llm model: %w", err)
	}
	return model, resolvedName, nil
}

func parseEvaluationJSON(raw string) (*QuestEvaluation, error) {
	raw = extractJSONObject(raw)
	var ev QuestEvaluation
	if err := sonic.Unmarshal([]byte(raw), &ev); err != nil {
		return nil, fmt.Errorf("parse evaluate json: %w", err)
	}
	if strings.TrimSpace(ev.Title) == "" || strings.TrimSpace(ev.StatCode) == "" {
		return nil, fmt.Errorf("parse evaluate json: missing title or stat_code")
	}
	return &ev, nil
}

func parseLoreJSON(raw string) (*InstanceLore, error) {
	raw = extractJSONObject(raw)
	var lore InstanceLore
	if err := sonic.Unmarshal([]byte(raw), &lore); err != nil {
		return nil, fmt.Errorf("parse lore json: %w", err)
	}
	if strings.TrimSpace(lore.Name) == "" || strings.TrimSpace(lore.Lore) == "" {
		return nil, fmt.Errorf("parse lore json: missing name or lore")
	}
	return &lore, nil
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
	ev.Fear, ev.BaseExp, ev.BaseGold, ev.DropTier = defaultRewards(ev.Difficulty)
	return ev
}

func normalizeDifficulty(raw string) string {
	d := strings.ToUpper(strings.TrimSpace(raw))
	if _, ok := allowedDifficulties[d]; ok {
		return d
	}
	return "C"
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

func defaultRewards(diff string) (fear, exp, gold int, tier string) {
	switch diff {
	case "SSS":
		return 5, 300, 80, "Mythic"
	case "SS":
		return 5, 220, 60, "Legendary"
	case "S":
		return 4, 180, 50, "Epic"
	case "A":
		return 4, 150, 40, "Epic"
	case "B":
		return 3, 80, 25, "Rare"
	case "D":
		return 2, 30, 10, "Common"
	case "E":
		return 1, 20, 8, "Common"
	default: // C
		return 2, 40, 15, "Common"
	}
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
