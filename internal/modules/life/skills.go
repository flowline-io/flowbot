package life

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	pkglife "github.com/flowline-io/flowbot/pkg/life"
)

// SkillTreeView is the aggregated skills page model.
type SkillTreeView struct {
	Roots            []*SkillTreeNodeView
	DefaultSelected  string
	ActiveNodeCount  int
	TrackedNodeCount int
}

// SkillTreeNodeView is one node in the controlled skills tree.
type SkillTreeNodeView struct {
	Key            string
	Title          string
	Subtitle       string
	Characteristic string
	Children       []*SkillTreeNodeView
	PracticeCount  int
	SkillCount     int
	LastActivityAt *time.Time
	Status         string
	Evidence       []SkillTreeEvidenceView

	directPractice []SkillTreeEvidenceView
}

// SkillTreeEvidenceView is one recent practice record linked to a tree node.
type SkillTreeEvidenceView struct {
	Title      string
	SourceType string
	Detail     string
	OccurredAt time.Time
	WindowDays int
}

type skillTreeBlueprint struct {
	key            string
	title          string
	subtitle       string
	characteristic string
	aliases        []string
	children       []skillTreeBlueprint
}

type skillTreeInputs struct {
	charCodeByID map[int64]string
	nodeByID     map[int64]*gen.LifePlanNode
	specByNodeID map[int64]*gen.LifeActionSpec
	skills       []*gen.LifeSkill
	logs         []*gen.LifeActionLog
}

var skillTreeBlueprints = []skillTreeBlueprint{
	{
		key: "int", title: "Intelligence", subtitle: "Research, analysis, and learning loops", characteristic: "INT",
		children: []skillTreeBlueprint{
			{key: "int-research", title: "Research", aliases: []string{"research", "read", "reading", "study", "learn"}},
			{key: "int-analysis", title: "Analysis", aliases: []string{"analysis", "analy", "debug", "problem", "reason"}},
			{key: "int-systems", title: "Systems", aliases: []string{"system", "architecture", "design doc", "model"}},
		},
	},
	{
		key: "wri", title: "Writing", subtitle: "Drafting, documentation, and publishing", characteristic: "WRI",
		children: []skillTreeBlueprint{
			{key: "wri-draft", title: "Drafting", aliases: []string{"write", "writing", "draft", "essay", "article", "blog"}},
			{key: "wri-docs", title: "Documentation", aliases: []string{"doc", "docs", "documentation", "spec", "prd"}},
			{key: "wri-story", title: "Storytelling", aliases: []string{"story", "narrative", "copy", "script"}},
		},
	},
	{
		key: "foc", title: "Focus", subtitle: "Planning, deep work, and follow-through", characteristic: "FOC",
		children: []skillTreeBlueprint{
			{key: "foc-plan", title: "Planning", aliases: []string{"plan", "planning", "roadmap", "schedule"}},
			{key: "foc-exec", title: "Execution", aliases: []string{"ship", "execute", "deliver", "finish", "complete"}},
			{key: "foc-deep", title: "Deep Work", aliases: []string{"focus", "deep work", "concentrate", "pomodoro"}},
		},
	},
	{
		key: "cre", title: "Creativity", subtitle: "Ideation, prototyping, and making", characteristic: "CRE",
		children: []skillTreeBlueprint{
			{key: "cre-ideas", title: "Ideation", aliases: []string{"idea", "brainstorm", "concept"}},
			{key: "cre-design", title: "Design", aliases: []string{"design", "ui", "ux", "visual"}},
			{key: "cre-proto", title: "Prototyping", aliases: []string{"prototype", "mock", "experiment", "build"}},
		},
	},
	{
		key: "cha", title: "Charisma", subtitle: "Communication, teaching, and leadership", characteristic: "CHA",
		children: []skillTreeBlueprint{
			{key: "cha-comm", title: "Communication", aliases: []string{"commun", "talk", "meeting", "present", "speak"}},
			{key: "cha-lead", title: "Leadership", aliases: []string{"lead", "mentor", "manage", "coach"}},
			{key: "cha-teach", title: "Teaching", aliases: []string{"teach", "course", "share", "explain"}},
		},
	},
	{
		key: "phy", title: "Physique", subtitle: "Training, mobility, and recovery", characteristic: "PHY",
		children: []skillTreeBlueprint{
			{key: "phy-strength", title: "Strength", aliases: []string{"gym", "lift", "strength", "workout", "train"}},
			{key: "phy-endurance", title: "Endurance", aliases: []string{"run", "cardio", "walk", "swim", "cycle"}},
			{key: "phy-recovery", title: "Recovery", aliases: []string{"sleep", "stretch", "recover", "mobility", "rest"}},
		},
	},
	{
		key: "wil", title: "Willpower", subtitle: "Discipline and consistency habits", characteristic: "WIL",
		children: []skillTreeBlueprint{
			{key: "wil-discipline", title: "Discipline", aliases: []string{"discipline", "habit", "routine"}},
			{key: "wil-consistency", title: "Consistency", aliases: []string{"consistency", "streak", "daily", "repeat"}},
			{key: "wil-recovery", title: "Resilience", aliases: []string{"resilience", "grit", "recover", "bounce back"}},
		},
	},
	{
		key: "fin", title: "Finance", subtitle: "Budgeting, sales, and negotiation", characteristic: "FIN",
		children: []skillTreeBlueprint{
			{key: "fin-budget", title: "Budgeting", aliases: []string{"budget", "saving", "expense", "cashflow"}},
			{key: "fin-sales", title: "Sales", aliases: []string{"sell", "sales", "offer", "proposal"}},
			{key: "fin-negotiation", title: "Negotiation", aliases: []string{"negot", "pricing", "deal", "contract"}},
		},
	},
	{
		key: "general", title: "General", subtitle: "Unmapped growth signals that still deserve visibility",
		children: []skillTreeBlueprint{
			{key: "general-explore", title: "Exploration", aliases: []string{"explore", "misc", "general"}},
		},
	},
}

// BuildSkillTree builds the read-only controlled skill tree for the Life UI.
func (s *Service) BuildSkillTree(ctx context.Context, userID string) (*SkillTreeView, error) {
	inputs, err := s.loadSkillTreeInputs(ctx, userID)
	if err != nil {
		return nil, err
	}
	roots, byKey := cloneSkillTree()
	recordSkillTreeSkills(inputs.skills, inputs.charCodeByID, byKey)
	if err := s.recordSkillTreeEvidence(ctx, inputs, byKey); err != nil {
		return nil, err
	}
	activeCount, trackedCount, bestKey := summarizeSkillTree(roots)
	return &SkillTreeView{
		Roots:            roots,
		DefaultSelected:  bestKey,
		ActiveNodeCount:  activeCount,
		TrackedNodeCount: trackedCount,
	}, nil
}

func (s *Service) loadSkillTreeInputs(ctx context.Context, userID string) (*skillTreeInputs, error) {
	p, err := s.EnsureProfile(ctx, userID, "", config.DefaultClass)
	if err != nil {
		return nil, err
	}
	chars, err := s.store.ListCharacteristics(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	skills, err := s.store.ListSkills(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	nodes, err := s.store.ListPlanNodes(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	specs, err := s.store.ListActionSpecs(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	logs, err := s.store.ListActionLogs(ctx, p.ID, 60)
	if err != nil {
		return nil, err
	}
	inputs := &skillTreeInputs{
		charCodeByID: make(map[int64]string, len(chars)),
		nodeByID:     make(map[int64]*gen.LifePlanNode, len(nodes)),
		specByNodeID: make(map[int64]*gen.LifeActionSpec, len(specs)),
		skills:       skills,
		logs:         logs,
	}
	for _, ch := range chars {
		inputs.charCodeByID[ch.ID] = strings.ToUpper(strings.TrimSpace(ch.Code))
	}
	for _, node := range nodes {
		inputs.nodeByID[node.ID] = node
	}
	for _, spec := range specs {
		inputs.specByNodeID[spec.PlanNodeID] = spec
	}
	return inputs, nil
}

func recordSkillTreeSkills(skills []*gen.LifeSkill, charCodeByID map[int64]string, byKey map[string]*SkillTreeNodeView) {
	for _, skill := range skills {
		if skill == nil {
			continue
		}
		charCode := charCodeByID[skill.CharacteristicID]
		target := classifySkillTreeNode(skill.Name, charCode, byKey)
		target.SkillCount++
	}
}

func (s *Service) recordSkillTreeEvidence(ctx context.Context, inputs *skillTreeInputs, byKey map[string]*SkillTreeNodeView) error {
	for _, row := range inputs.logs {
		if row == nil {
			continue
		}
		target, ev, err := s.mapSkillTreeEvidence(ctx, row, inputs.charCodeByID, inputs.nodeByID, inputs.specByNodeID, byKey)
		if err != nil {
			return err
		}
		target.directPractice = append(target.directPractice, ev)
	}
	return nil
}

func summarizeSkillTree(roots []*SkillTreeNodeView) (activeCount, trackedCount int, bestKey string) {
	bestRank := -1
	var bestTime time.Time
	for _, root := range roots {
		aggregateSkillTreeNode(root)
		for _, node := range flattenSkillTree(root) {
			if len(node.Children) == 0 {
				trackedCount++
				if node.Status == "Active" {
					activeCount++
				}
			}
			rank, seen := skillTreeNodeRank(node)
			if !seen {
				continue
			}
			last := time.Time{}
			if node.LastActivityAt != nil {
				last = *node.LastActivityAt
			}
			if rank > bestRank || (rank == bestRank && last.After(bestTime)) {
				bestRank = rank
				bestTime = last
				bestKey = node.Key
			}
		}
	}
	if bestKey == "" && len(roots) > 0 {
		bestKey = roots[0].Key
	}
	return activeCount, trackedCount, bestKey
}

func (s *Service) mapSkillTreeEvidence(
	ctx context.Context,
	row *gen.LifeActionLog,
	charCodeByID map[int64]string,
	nodeByID map[int64]*gen.LifePlanNode,
	specByNodeID map[int64]*gen.LifeActionSpec,
	byKey map[string]*SkillTreeNodeView,
) (*SkillTreeNodeView, SkillTreeEvidenceView, error) {
	evidence := SkillTreeEvidenceView{
		Title:      strings.TrimSpace(row.Summary),
		SourceType: skillTreeSourceLabel(row.SourceType),
		Detail:     "",
		OccurredAt: row.CreatedAt.UTC(),
		WindowDays: 14,
	}
	if row.QuestID != nil {
		quest, err := s.store.GetQuest(ctx, *row.QuestID)
		if err != nil {
			return nil, SkillTreeEvidenceView{}, err
		}
		if quest != nil {
			evidence.Title = strings.TrimSpace(quest.Title)
			evidence.Detail = fmt.Sprintf("Quest · %s · %d EXP", quest.AiEvaluatedDifficulty, quest.BaseExpReward)
			skill, err := s.store.GetSkill(ctx, quest.SkillID)
			if err != nil {
				return nil, SkillTreeEvidenceView{}, err
			}
			if skill != nil {
				charCode := charCodeByID[skill.CharacteristicID]
				return classifySkillTreeNode(skill.Name, charCode, byKey), evidence, nil
			}
		}
	}
	if row.PlanNodeID != nil {
		node := nodeByID[*row.PlanNodeID]
		if node != nil {
			evidence.Title = strings.TrimSpace(node.Title)
			evidence.Detail = strings.TrimSpace(node.Description)
			spec := specByNodeID[node.ID]
			if spec != nil {
				evidence.WindowDays = skillTreeWindowDays(spec.SuggestedCadence)
				sourceLabel := skillTreeTaskLabel(spec.TaskType)
				if sourceLabel != "" {
					evidence.SourceType = sourceLabel
				}
			}
			return classifySkillTreeNode(node.Title+" "+node.Description, "", byKey), evidence, nil
		}
	}
	return classifySkillTreeNode(evidence.Title, "", byKey), evidence, nil
}

func cloneSkillTree() ([]*SkillTreeNodeView, map[string]*SkillTreeNodeView) {
	index := map[string]*SkillTreeNodeView{}
	roots := make([]*SkillTreeNodeView, 0, len(skillTreeBlueprints))
	for _, bp := range skillTreeBlueprints {
		roots = append(roots, cloneSkillTreeNode(bp, index))
	}
	return roots, index
}

func cloneSkillTreeNode(bp skillTreeBlueprint, index map[string]*SkillTreeNodeView) *SkillTreeNodeView {
	node := &SkillTreeNodeView{
		Key:            bp.key,
		Title:          bp.title,
		Subtitle:       bp.subtitle,
		Characteristic: bp.characteristic,
		Children:       make([]*SkillTreeNodeView, 0, len(bp.children)),
	}
	index[node.Key] = node
	for _, child := range bp.children {
		node.Children = append(node.Children, cloneSkillTreeNode(child, index))
	}
	return node
}

func flattenSkillTree(root *SkillTreeNodeView) []*SkillTreeNodeView {
	if root == nil {
		return nil
	}
	out := []*SkillTreeNodeView{root}
	for _, child := range root.Children {
		out = append(out, flattenSkillTree(child)...)
	}
	return out
}

func classifySkillTreeNode(raw, charCode string, byKey map[string]*SkillTreeNodeView) *SkillTreeNodeView {
	text := skillTreeNormalize(raw)
	charCode = strings.ToUpper(strings.TrimSpace(charCode))
	if node := findSkillTreeAliasNode(text, charCode, byKey); node != nil {
		return node
	}
	if node := findSkillTreeRootNode(charCode, byKey); node != nil {
		return node
	}
	if node := findSkillTreeAliasNode(text, "", byKey); node != nil {
		return node
	}
	if node := byKey["general-explore"]; node != nil {
		return node
	}
	return byKey["general"]
}

func findSkillTreeAliasNode(text, charCode string, byKey map[string]*SkillTreeNodeView) *SkillTreeNodeView {
	for _, bp := range skillTreeBlueprints {
		if charCode != "" && bp.characteristic != "" && bp.characteristic != charCode {
			continue
		}
		for _, child := range bp.children {
			if !skillTreeMatches(text, child.aliases) {
				continue
			}
			if node := byKey[child.key]; node != nil {
				return node
			}
		}
	}
	return nil
}

func findSkillTreeRootNode(charCode string, byKey map[string]*SkillTreeNodeView) *SkillTreeNodeView {
	if charCode == "" {
		return nil
	}
	for _, bp := range skillTreeBlueprints {
		if bp.characteristic != charCode {
			continue
		}
		if node := byKey[bp.key]; node != nil {
			return node
		}
	}
	return nil
}

func aggregateSkillTreeNode(node *SkillTreeNodeView) {
	if node == nil {
		return
	}
	allEvidence := make([]SkillTreeEvidenceView, 0, len(node.directPractice))
	allEvidence = append(allEvidence, node.directPractice...)
	node.PracticeCount = len(node.directPractice)
	node.LastActivityAt = skillTreeLastActivity(node.directPractice)
	for _, child := range node.Children {
		aggregateSkillTreeNode(child)
		node.PracticeCount += child.PracticeCount
		node.SkillCount += child.SkillCount
		if child.LastActivityAt != nil && (node.LastActivityAt == nil || child.LastActivityAt.After(*node.LastActivityAt)) {
			t := *child.LastActivityAt
			node.LastActivityAt = &t
		}
		allEvidence = append(allEvidence, child.Evidence...)
	}
	slices.SortFunc(allEvidence, func(a, b SkillTreeEvidenceView) int {
		return b.OccurredAt.Compare(a.OccurredAt)
	})
	node.Evidence = allEvidence
	if len(node.Evidence) > 8 {
		node.Evidence = node.Evidence[:8]
	}
	node.Status = skillTreeStatus(allEvidence)
}

func skillTreeNodeRank(node *SkillTreeNodeView) (int, bool) {
	if node == nil || node.PracticeCount == 0 {
		return 0, false
	}
	switch node.Status {
	case "Active":
		return 3, true
	case "Cooling":
		return 2, true
	default:
		return 1, true
	}
}

func skillTreeStatus(evidence []SkillTreeEvidenceView) string {
	if len(evidence) == 0 {
		return "Quiet"
	}
	now := time.Now().UTC()
	bestWindow := 14
	for _, ev := range evidence {
		window := ev.WindowDays
		if window <= 0 {
			window = 14
		}
		if now.Sub(ev.OccurredAt) <= time.Duration(window)*24*time.Hour {
			return "Active"
		}
		if window > bestWindow {
			bestWindow = window
		}
	}
	if now.Sub(evidence[0].OccurredAt) <= time.Duration(bestWindow*2)*24*time.Hour {
		return "Cooling"
	}
	return "Quiet"
}

func skillTreeLastActivity(evidence []SkillTreeEvidenceView) *time.Time {
	if len(evidence) == 0 {
		return nil
	}
	last := evidence[0].OccurredAt
	for _, ev := range evidence[1:] {
		if ev.OccurredAt.After(last) {
			last = ev.OccurredAt
		}
	}
	return &last
}

func skillTreeMatches(text string, aliases []string) bool {
	for _, alias := range aliases {
		if strings.Contains(text, skillTreeNormalize(alias)) {
			return true
		}
	}
	return false
}

func skillTreeNormalize(raw string) string {
	lower := strings.ToLower(strings.TrimSpace(raw))
	replacer := strings.NewReplacer("-", " ", "_", " ", "/", " ", ".", " ", ",", " ", ":", " ")
	return replacer.Replace(lower)
}

func skillTreeWindowDays(cadence string) int {
	return pkglife.SkillTreeWindowDays(cadence)
}

func skillTreeSourceLabel(sourceType string) string {
	return pkglife.SourceTypeLabel(sourceType)
}

func skillTreeTaskLabel(taskType string) string {
	return pkglife.TaskTypeLabel(taskType)
}
