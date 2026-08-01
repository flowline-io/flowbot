package pages

import (
	"fmt"
	"math"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/bytedance/sonic"

	pkglife "github.com/flowline-io/flowbot/pkg/life"
)

// LifeDefaultListPerPage is the default page size for Life list sections.
const LifeDefaultListPerPage = 10

// History section anchor and tab ids on the Life quests page.
const (
	LifeHistoryAnchor        = "life-history"
	LifeHistoryTabCompleted  = "completed"
	LifeHistoryTabActionLogs = "logs"
)

// Archive section anchor and tab ids on the Life rewards page.
const (
	LifeRewardsArchiveAnchor  = "life-rewards-archive"
	LifeRewardsTabRedemptions = "redemptions"
	LifeRewardsTabDeactivated = "deactivated"
)

// LifeBackpackAnchor is the backpack section fragment on the inventory page.
const LifeBackpackAnchor = "life-backpack"

// LifeDashboardData is the Life home dashboard model.
type LifeDashboardData = LifeCharacterData

// LifeStatRow is one characteristic row with progress for segmented bars.
type LifeStatRow struct {
	Code       string
	Name       string
	Level      int
	Exp        int64
	ExpToNext  int64
	FilledSegs int
	TotalSegs  int
	RadarValue float64
}

// LifeSkillRow is one skill row.
type LifeSkillRow struct {
	Name  string
	Level int
	Exp   int64
}

// LifeSkillTreeData is the skills tree page model.
type LifeSkillTreeData struct {
	PendingCount     int
	Roots            []LifeSkillTreeNodeRow
	SelectedNode     *LifeSkillTreeNodeDetail
	ActiveLeafCount  int
	TrackedLeafCount int
}

// LifeCharacterData is the Life character / dashboard page model.
type LifeCharacterData struct {
	Nickname         string
	ClassType        string
	Level            int
	Exp              int64
	ExpToNext        int64
	LevelFilledSegs  int
	LevelTotalSegs   int
	Gold             int
	HPCurrent        int
	HPMax            int
	HeartsFilled     int
	HeartsTotal      int
	MasterObjective  string
	MinorObjective   string
	Strength         string
	Weakness         string
	Characteristics  []LifeStatRow
	Skills           []LifeSkillRow
	DropRateBonus    float64
	GoldMult         float64
	PendingCount     int
	RadarLabelsJSON  string
	RadarValuesJSON  string
	ShowClassForm    bool
	Goals            []LifeGoalRow
	PlanTree         []LifePlanNodeRow
	PlanParents      []LifePlanParentOption
	BreakdownPreview *LifeBreakdownPreviewData
}

// LifeGoalsData is the Goals page model.
type LifeGoalsData struct {
	Goals         []LifeGoalRow
	Groups        []LifeGoalGroup
	ActiveAreas   []LifeGoalRow
	DropRateBonus float64
	GoldMult      float64
	PendingCount  int
}

// LifeGoalGroup is one Area section (or the ungrouped section when Area is nil).
type LifeGoalGroup struct {
	Area     *LifeGoalRow
	Children []LifeGoalRow
}

// LifePlanData is the Plan Tree page model.
type LifePlanData struct {
	PlanTree         []LifePlanNodeRow
	PlanParents      []LifePlanParentOption
	BreakdownPreview *LifeBreakdownPreviewData
	PendingCount     int
}

// LifeSkillTreeNodeRow is one rendered tree node.
type LifeSkillTreeNodeRow struct {
	Key               string
	Title             string
	Subtitle          string
	Status            string
	PracticeCount     int
	SkillCount        int
	LastActivityLabel string
	IsSelected        bool
	Children          []LifeSkillTreeNodeRow
}

// LifeSkillTreeNodeDetail is the evidence panel for one selected node.
type LifeSkillTreeNodeDetail struct {
	Title             string
	Subtitle          string
	Status            string
	PracticeCount     int
	SkillCount        int
	LastActivityLabel string
	Evidence          []LifeSkillEvidenceRow
}

// LifeSkillEvidenceRow is one recent training record linked to a tree node.
type LifeSkillEvidenceRow struct {
	Title      string
	SourceType string
	Detail     string
	When       string
}

// LifeGoalRow is one PARA goal for Goals / Quests UI.
type LifeGoalRow struct {
	Flag      string
	Title     string
	Category  string
	Status    string
	AreaFlag  string
	AreaTitle string
}

// LifeGoalGroupTestID returns a stable test id for a goals group section.
func LifeGoalGroupTestID(group LifeGoalGroup) string {
	if group.Area == nil {
		return "life-goal-group-ungrouped"
	}
	return "life-goal-group-" + group.Area.Flag
}

// LifeGoalStatusClass maps goal status to a life-meta-chip modifier.
func LifeGoalStatusClass(status string) string {
	switch strings.TrimSpace(status) {
	case pkglife.GoalStatusActive:
		return "life-meta-chip-ok"
	case pkglife.GoalStatusPaused:
		return "life-meta-chip-warn"
	case pkglife.GoalStatusCompleted:
		return "life-meta-chip-diff"
	default:
		return ""
	}
}

// LifeGoalAreaOptions returns Active Areas for a picker, keeping the current Area if missing.
func LifeGoalAreaOptions(active []LifeGoalRow, currentFlag, currentTitle string) []LifeGoalRow {
	if currentFlag == "" {
		return active
	}
	for _, a := range active {
		if a.Flag == currentFlag {
			return active
		}
	}
	out := make([]LifeGoalRow, 0, len(active)+1)
	out = append(out, active...)
	out = append(out, LifeGoalRow{Flag: currentFlag, Title: currentTitle, Category: pkglife.GoalCategoryArea})
	return out
}

// LifeGroupGoals groups Project/Resource rows under their Area; ungrouped items follow.
func LifeGroupGoals(goals []LifeGoalRow) []LifeGoalGroup {
	areas := make([]LifeGoalRow, 0)
	for _, g := range goals {
		if g.Category == pkglife.GoalCategoryArea {
			areas = append(areas, g)
		}
	}
	used := make(map[string]bool)
	groups := make([]LifeGoalGroup, 0, len(areas)+1)
	for i := range areas {
		area := areas[i]
		children := make([]LifeGoalRow, 0)
		for _, g := range goals {
			if g.Category == pkglife.GoalCategoryArea {
				continue
			}
			if g.AreaFlag == area.Flag {
				children = append(children, g)
				used[g.Flag] = true
			}
		}
		groups = append(groups, LifeGoalGroup{Area: &area, Children: children})
	}
	ungrouped := make([]LifeGoalRow, 0)
	for _, g := range goals {
		if g.Category == pkglife.GoalCategoryArea {
			continue
		}
		if !used[g.Flag] {
			ungrouped = append(ungrouped, g)
		}
	}
	if len(ungrouped) > 0 {
		groups = append(groups, LifeGoalGroup{Children: ungrouped})
	}
	if len(groups) == 0 {
		return nil
	}
	return groups
}

// LifePlanParentOption is one valid parent choice in the create form.
type LifePlanParentOption struct {
	Flag      string
	Label     string
	NodeType  string
	Depth     int
	AllowText string
}

// LifePlanNodeRow is one rendered plan node.
type LifePlanNodeRow struct {
	Flag              string
	ParentFlag        string
	NodeType          string
	Title             string
	Description       string
	Status            string
	TaskType          string
	TrackingMode      string
	SuggestedCadence  string
	Difficulty        string
	Exp               int
	Gold              int
	NeedsConfirmation bool
	Children          []LifePlanNodeRow
}

// LifeQuestRow is one quest list row.
type LifeQuestRow struct {
	Flag         string
	Title        string
	Prompt       string
	Type         string
	Difficulty   string
	Fear         int
	Exp          int
	Gold         int
	DropTier     string
	DropChance   float64
	Status       string
	Evidence     []LifeQuestEvidenceRow
	Adjudication *LifeQuestAdjudicationRow
}

// LifePageInfo holds pagination state for one Life list section.
type LifePageInfo struct {
	Page       int
	PerPage    int
	Total      int
	TotalPages int
	HasPrev    bool
	HasNext    bool
	PrevURL    string
	NextURL    string
}

// LifeQuestsData is the quests page model.
type LifeQuestsData struct {
	Pending        []LifeQuestRow
	Completed      []LifeQuestRow
	Goals          []LifeGoalRow
	TodayActions   []LifeTodayActionRow
	TodayHabits    []LifeTodayHabitRow
	ActionLogs     []LifeActionLogRow
	CompletedPage  LifePageInfo
	ActionLogsPage LifePageInfo
	HistoryTab     string
	PendingCount   int
}

// LifeQuestEvidenceRow is one submitted evidence item.
type LifeQuestEvidenceRow struct {
	Flag       string
	SourceType string
	Content    string
	SourceURL  string
	Summary    string
	When       string
}

// LifeQuestAdjudicationRow is one AI ruling shown on the quests page.
type LifeQuestAdjudicationRow struct {
	Flag               string
	Status             string
	Verdict            string
	Reason             string
	SuggestedExp       int
	SuggestedGold      int
	SuggestedNextSteps []string
	When               string
}

// LifeActionLogRow is one completion audit row.
type LifeActionLogRow struct {
	Flag       string
	SourceType string
	QuestTitle string
	GainedExp  int
	GainedGold int
	Dice       float64
	HasDice    bool
	Dropped    bool
	When       string
}

// LifeTodayActionRow is one pending todo/recurring occurrence.
type LifeTodayActionRow struct {
	Flag             string
	Title            string
	NodeFlag         string
	ContextPath      string
	Description      string
	Kind             string
	State            string
	DueLabel         string
	TaskType         string
	TrackingMode     string
	SuggestedCadence string
	Difficulty       string
	Exp              int
	Gold             int
}

// LifeTodayHabitRow is one habit plus today check-in state.
type LifeTodayHabitRow struct {
	NodeFlag         string
	Title            string
	CheckedIn        bool
	TaskType         string
	SuggestedCadence string
}

// LifeBreakdownPreviewData is the UI state for an AI suggestion preview.
type LifeBreakdownPreviewData struct {
	RootTitle   string
	Description string
	PayloadJSON string
	Tree        *LifeBreakdownSuggestionRow
}

// LifeBreakdownSuggestionRow is one suggested AI tree node.
type LifeBreakdownSuggestionRow struct {
	NodeType         string
	Title            string
	Description      string
	TaskType         string
	SuggestedCadence string
	Difficulty       string
	Exp              int
	Gold             int
	Children         []LifeBreakdownSuggestionRow
}

// LifeInventoryRow is one backpack row.
type LifeInventoryRow struct {
	Flag       string
	Name       string
	Rarity     string
	Slot       string
	SlotField  string
	BuffText   string
	PerkText   string
	Lore       string
	LoreStatus string
	Equipped   bool
	Tarnished  bool
}

// LifeEquipSlot is one wear-position cell on the inventory equipment board.
type LifeEquipSlot struct {
	Label     string
	SlotField string
	Item      *LifeInventoryRow
}

// LifeInventoryData is the inventory page model.
type LifeInventoryData struct {
	Slots        []LifeEquipSlot
	Items        []LifeInventoryRow
	BackpackPage LifePageInfo
	PendingCount int
}

// LifeAchievementRow is one memorial achievement on the achievements page.
type LifeAchievementRow struct {
	Flag         string
	Name         string
	Description  string
	Unlocked     bool
	UnlockedAt   string
	ShowProgress bool
	Current      int
	Target       int
	Retired      bool
}

// LifeAchievementsData is the achievements page model.
type LifeAchievementsData struct {
	Items         []LifeAchievementRow
	UnlockedCount int
	PendingCount  int
}

// LifeRewardRow is one player-defined reward on the Rewards page.
type LifeRewardRow struct {
	Flag           string
	Name           string
	Notes          string
	Price          int
	CooldownHours  int
	OnCooldown     bool
	CooldownEndsAt string
	CanAfford      bool
	CanRedeem      bool
}

// LifeRedemptionRow is one redemption history entry.
type LifeRedemptionRow struct {
	Flag       string
	RewardName string
	PricePaid  int
	RedeemedAt string
}

// LifeRewardsData is the Rewards market page model.
type LifeRewardsData struct {
	Gold            int
	Active          []LifeRewardRow
	Inactive        []LifeRewardRow
	Redemptions     []LifeRedemptionRow
	RedemptionsPage LifePageInfo
	InactivePage    LifePageInfo
	ArchiveTab      string
	PendingCount    int
}

// LifeStatsShellData is the Stats page shell before the HTMX panel loads.
type LifeStatsShellData struct {
	PendingCount int
}

// lifeEquipSlotOrder is the paper-doll display order for wear positions.
var lifeEquipSlotOrder = []struct {
	label string
	field string
}{
	{"Head", "head_slot"},
	{"Weapon", "weapon_slot"},
	{"Armor", "armor_slot"},
	{"Shoes", "shoes_slot"},
	{"Accessory", "accessory_slot"},
	{"Artifact", "artifact_slot"},
}

// LifeBuildEquipSlots builds the six wear-position cells from backpack rows.
// Empty positions keep a labeled empty cell; equipped items fill their slot.
func LifeBuildEquipSlots(items []LifeInventoryRow) []LifeEquipSlot {
	byField := make(map[string]LifeInventoryRow, len(items))
	for _, it := range items {
		if !it.Equipped || it.SlotField == "" {
			continue
		}
		byField[it.SlotField] = it
	}
	out := make([]LifeEquipSlot, 0, len(lifeEquipSlotOrder))
	for _, def := range lifeEquipSlotOrder {
		cell := LifeEquipSlot{Label: def.label, SlotField: def.field}
		if it, ok := byField[def.field]; ok {
			cp := it
			cell.Item = &cp
		}
		out = append(out, cell)
	}
	return out
}

func lifeEquippedCount(slots []LifeEquipSlot) string {
	n := 0
	for _, s := range slots {
		if s.Item != nil {
			n++
		}
	}
	return fmt.Sprintf("%d / %d", n, len(slots))
}

func lifeAchievementProgressPct(current, target int) int {
	if target <= 0 {
		return 0
	}
	if current <= 0 {
		return 0
	}
	if current >= target {
		return 100
	}
	return int(math.Round(float64(current) / float64(target) * 100))
}

// LifeRarityClass maps equipment rarity to a CSS modifier class.
func LifeRarityClass(rarity string) string {
	switch strings.ToLower(strings.TrimSpace(rarity)) {
	case "common":
		return "rarity-common"
	case "uncommon":
		return "rarity-uncommon"
	case "rare":
		return "rarity-rare"
	case "epic":
		return "rarity-epic"
	case "legendary":
		return "rarity-legendary"
	case "mythic":
		return "rarity-mythic"
	default:
		return "rarity-common"
	}
}

// LifeVerdictChipClass maps an adjudication verdict to a meta-chip modifier.
func LifeVerdictChipClass(verdict string) string {
	switch strings.ToLower(strings.TrimSpace(verdict)) {
	case "completed":
		return "life-meta-chip-ok"
	case "partial":
		return "life-meta-chip-diff"
	case "failed":
		return "life-meta-chip-danger"
	case "needs_more_evidence":
		return "life-meta-chip-warn"
	default:
		return ""
	}
}

// LifeSlotRarityClass returns the rarity CSS class for a filled equip slot.
func LifeSlotRarityClass(slot LifeEquipSlot) string {
	if slot.Item == nil {
		return ""
	}
	return LifeRarityClass(slot.Item.Rarity)
}

// LifePlanNodeTypeLabel returns a human label for one node type.
func LifePlanNodeTypeLabel(nodeType string) string {
	switch strings.ToLower(strings.TrimSpace(nodeType)) {
	case "goal":
		return "Goal"
	case "milestone":
		return "Milestone"
	case "project":
		return "Project"
	case "action":
		return "Action"
	default:
		return "Node"
	}
}

// LifeTaskTypeLabel returns a concise label for one action type.
func LifeTaskTypeLabel(taskType string) string {
	return pkglife.TaskTypeLabel(taskType)
}

// LifeActionLogSourceLabel returns a concise label for one audit source.
func LifeActionLogSourceLabel(sourceType string) string {
	return pkglife.SourceTypeLabel(sourceType)
}

// LifeBuildPageInfo builds pagination state for a list section.
func LifeBuildPageInfo(page, perPage, total int) LifePageInfo {
	if perPage < 1 {
		perPage = LifeDefaultListPerPage
	}
	if page < 1 {
		page = 1
	}
	if total < 0 {
		total = 0
	}
	totalPages := 0
	if total > 0 {
		totalPages = (total + perPage - 1) / perPage
	}
	if totalPages > 0 && page > totalPages {
		page = totalPages
	}
	return LifePageInfo{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages,
		HasPrev:    totalPages > 0 && page > 1,
		HasNext:    totalPages > 0 && page < totalPages,
	}
}

// LifeNormalizeHistoryTab returns a valid history tab id.
func LifeNormalizeHistoryTab(tab string) string {
	return lifeNormalizeAltTab(tab, LifeHistoryTabActionLogs, LifeHistoryTabCompleted)
}

// LifeHistoryTabClass returns the CSS class for one history tab control.
func LifeHistoryTabClass(activeTab, tab string) string {
	return lifeTabClass(LifeNormalizeHistoryTab(activeTab), tab)
}

// LifeQuestsListURL builds the quests page URL with optional page/tab query params and fragment.
func LifeQuestsListURL(completedPage, logsPage int, tab, anchor string) string {
	u := "/service/web/life/quests"
	q := url.Values{}
	if completedPage > 1 {
		q.Set("completed_page", strconv.Itoa(completedPage))
	}
	if logsPage > 1 {
		q.Set("logs_page", strconv.Itoa(logsPage))
	}
	if LifeNormalizeHistoryTab(tab) == LifeHistoryTabActionLogs {
		q.Set("history_tab", LifeHistoryTabActionLogs)
	}
	if enc := q.Encode(); enc != "" {
		u += "?" + enc
	}
	if anchor != "" {
		u += "#" + anchor
	}
	return u
}

// LifeWithCompletedPager attaches prev/next URLs for the completed quests tab.
func LifeWithCompletedPager(info LifePageInfo, logsPage int) LifePageInfo {
	return lifeAttachPager(
		info,
		LifeQuestsListURL(info.Page-1, logsPage, LifeHistoryTabCompleted, LifeHistoryAnchor),
		LifeQuestsListURL(info.Page+1, logsPage, LifeHistoryTabCompleted, LifeHistoryAnchor),
	)
}

// LifeWithActionLogsPager attaches prev/next URLs for the action log tab.
func LifeWithActionLogsPager(info LifePageInfo, completedPage int) LifePageInfo {
	return lifeAttachPager(
		info,
		LifeQuestsListURL(completedPage, info.Page-1, LifeHistoryTabActionLogs, LifeHistoryAnchor),
		LifeQuestsListURL(completedPage, info.Page+1, LifeHistoryTabActionLogs, LifeHistoryAnchor),
	)
}

// LifeNormalizeRewardsArchiveTab returns a valid rewards archive tab id.
func LifeNormalizeRewardsArchiveTab(tab string) string {
	return lifeNormalizeAltTab(tab, LifeRewardsTabDeactivated, LifeRewardsTabRedemptions)
}

// LifeRewardsArchiveTabClass returns the CSS class for one rewards archive tab control.
func LifeRewardsArchiveTabClass(activeTab, tab string) string {
	return lifeTabClass(LifeNormalizeRewardsArchiveTab(activeTab), tab)
}

// LifeRewardsListURL builds the rewards page URL with optional page/tab query params and fragment.
func LifeRewardsListURL(redemptionsPage, inactivePage int, tab, anchor string) string {
	u := "/service/web/life/rewards"
	q := url.Values{}
	if redemptionsPage > 1 {
		q.Set("redemptions_page", strconv.Itoa(redemptionsPage))
	}
	if inactivePage > 1 {
		q.Set("inactive_page", strconv.Itoa(inactivePage))
	}
	if LifeNormalizeRewardsArchiveTab(tab) == LifeRewardsTabDeactivated {
		q.Set("archive_tab", LifeRewardsTabDeactivated)
	}
	if enc := q.Encode(); enc != "" {
		u += "?" + enc
	}
	if anchor != "" {
		u += "#" + anchor
	}
	return u
}

// LifeWithRedemptionsPager attaches prev/next URLs for the redemptions tab.
func LifeWithRedemptionsPager(info LifePageInfo, inactivePage int) LifePageInfo {
	return lifeAttachPager(
		info,
		LifeRewardsListURL(info.Page-1, inactivePage, LifeRewardsTabRedemptions, LifeRewardsArchiveAnchor),
		LifeRewardsListURL(info.Page+1, inactivePage, LifeRewardsTabRedemptions, LifeRewardsArchiveAnchor),
	)
}

// LifeWithInactiveRewardsPager attaches prev/next URLs for the deactivated tab.
func LifeWithInactiveRewardsPager(info LifePageInfo, redemptionsPage int) LifePageInfo {
	return lifeAttachPager(
		info,
		LifeRewardsListURL(redemptionsPage, info.Page-1, LifeRewardsTabDeactivated, LifeRewardsArchiveAnchor),
		LifeRewardsListURL(redemptionsPage, info.Page+1, LifeRewardsTabDeactivated, LifeRewardsArchiveAnchor),
	)
}

// LifeInventoryListURL builds the inventory page URL with optional backpack page and fragment.
func LifeInventoryListURL(backpackPage int, anchor string) string {
	u := "/service/web/life/inventory"
	if backpackPage > 1 {
		u += "?backpack_page=" + strconv.Itoa(backpackPage)
	}
	if anchor != "" {
		u += "#" + anchor
	}
	return u
}

// LifeWithBackpackPager attaches prev/next URLs for the backpack section.
func LifeWithBackpackPager(info LifePageInfo) LifePageInfo {
	return lifeAttachPager(
		info,
		LifeInventoryListURL(info.Page-1, LifeBackpackAnchor),
		LifeInventoryListURL(info.Page+1, LifeBackpackAnchor),
	)
}

func lifeAttachPager(info LifePageInfo, prevURL, nextURL string) LifePageInfo {
	if info.HasPrev {
		info.PrevURL = prevURL
	}
	if info.HasNext {
		info.NextURL = nextURL
	}
	return info
}

func lifeNormalizeAltTab(tab, alt, defaultTab string) string {
	if strings.EqualFold(strings.TrimSpace(tab), alt) {
		return alt
	}
	return defaultTab
}

func lifeTabClass(activeTab, tab string) string {
	if activeTab == tab {
		return "life-tab is-active"
	}
	return "life-tab"
}

// LifeOccurrenceKindLabel returns a human label for one occurrence kind.
func LifeOccurrenceKindLabel(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "one_time":
		return "One-time"
	case "recurring":
		return "Recurring"
	default:
		return kind
	}
}

// LifeSkillStatusClass maps skill activity status to a chip modifier class.
func LifeSkillStatusClass(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "active":
		return "life-skill-status-active"
	case "cooling":
		return "life-skill-status-cooling"
	default:
		return "life-skill-status-quiet"
	}
}

// LifeSkillMonogram returns a short badge label for one skill node title.
func LifeSkillMonogram(title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return "?"
	}
	for _, r := range title {
		return strings.ToUpper(string(r))
	}
	return "?"
}

// LifeIndentStyle returns a simple indent style for tree rows.
func LifeIndentStyle(depth int) string {
	if depth <= 0 {
		return ""
	}
	return fmt.Sprintf("margin-left:%drem;", depth*1)
}

// LifeSkillTreeBranchOpen reports whether a skill-tree branch should start expanded
// so the path to a selected descendant stays visible.
func LifeSkillTreeBranchOpen(node LifeSkillTreeNodeRow) bool {
	for _, child := range node.Children {
		if child.IsSelected || LifeSkillTreeBranchOpen(child) {
			return true
		}
	}
	return false
}

// LifeFormatBuffText formats equipment stat buffs for UI chips.
func LifeFormatBuffText(statBuffs map[string]any) string {
	if len(statBuffs) == 0 {
		return ""
	}
	keys := make([]string, 0, len(statBuffs))
	for k := range statBuffs {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		value, ok := lifeAnyToFloat(statBuffs[key])
		if !ok {
			continue
		}
		switch key {
		case "DropRate":
			parts = append(parts, fmt.Sprintf("Drop +%.0f%%", value*100))
		case "GoldMult":
			parts = append(parts, fmt.Sprintf("Gold +%.0f%%", value*100))
		default:
			parts = append(parts, fmt.Sprintf("%s +%s", key, lifeFormatNumber(value)))
		}
	}
	return strings.Join(parts, " · ")
}

// LifeFormatPerkText formats AI unlocked privileges for UI chips.
func LifeFormatPerkText(privileges map[string]any) string {
	if len(privileges) == 0 {
		return ""
	}
	keys := make([]string, 0, len(privileges))
	for k := range privileges {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%v", key, privileges[key]))
	}
	return strings.Join(parts, " · ")
}

func lifeAnyToFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	default:
		return 0, false
	}
}

func lifeFormatNumber(v float64) string {
	if math.Mod(v, 1) == 0 {
		return fmt.Sprintf("%.0f", v)
	}
	return fmt.Sprintf("%.1f", v)
}

const lifeSegmentCount = 24

// LifeBuildStatRow builds a characteristic row with segment fill.
// Bar fill and "exp / need" are within the current level (levels have no cap).
// RadarValue is absolute progress (level + in-level fraction) with no upper bound.
func LifeBuildStatRow(code, name string, level int, exp int64) LifeStatRow {
	if level < 1 {
		level = 1
	}
	need := pkglife.ExpToNextLevel(level)
	filled := lifeFilledSegments(exp, need, lifeSegmentCount)
	return LifeStatRow{
		Code: code, Name: name, Level: level, Exp: exp, ExpToNext: need,
		FilledSegs: filled, TotalSegs: lifeSegmentCount,
		RadarValue: lifeAbsoluteProgress(level, exp, need),
	}
}

func lifeAbsoluteProgress(level int, exp, need int64) float64 {
	radar := float64(level)
	if need > 0 && exp > 0 {
		frac := float64(exp) / float64(need)
		if frac > 1 {
			frac = 1
		}
		radar += frac
	}
	return radar
}

func lifeFilledFromRatio(ratio float64, segments int) int {
	if segments <= 0 || ratio <= 0 {
		return 0
	}
	if ratio >= 1 {
		return segments
	}
	n := int(math.Round(ratio * float64(segments)))
	if n < 1 {
		return 1
	}
	if n > segments {
		return segments
	}
	return n
}

func lifeFilledSegments(current, ceiling int64, segments int) int {
	if ceiling <= 0 {
		return 0
	}
	return lifeFilledFromRatio(float64(current)/float64(ceiling), segments)
}

// LifeDisplayName picks a short display name from nickname or uid.
func LifeDisplayName(nickname, uid string) string {
	n := strings.TrimSpace(nickname)
	if n != "" {
		return n
	}
	u := strings.TrimPrefix(uid, "user:")
	u = strings.TrimPrefix(u, "user-")
	if u == "" {
		return "Operator"
	}
	return u
}

// LifeHPFromStats derives a soft HP pool from Willpower (or average level).
func LifeHPFromStats(stats []LifeStatRow, profileLevel int) (current, maxHP, heartsFilled, heartsTotal int) {
	heartsTotal = pkglife.SoftHPHeartCount
	wilLevel := profileLevel
	wilExp := int64(0)
	wilNeed := pkglife.ExpToNextLevel(profileLevel)
	for _, s := range stats {
		if s.Code == "WIL" {
			wilLevel = s.Level
			wilExp = s.Exp
			wilNeed = s.ExpToNext
			break
		}
	}
	current, maxHP = pkglife.SoftHPFromWillpower(wilLevel, wilExp, wilNeed, profileLevel)
	heartsFilled = lifeFilledSegments(int64(current), int64(maxHP), heartsTotal)
	return current, maxHP, heartsFilled, heartsTotal
}

// LifeClassStrength maps class to a strength label.
func LifeClassStrength(classType string) string {
	switch classType {
	case "Mage":
		return "Analytical"
	case "Warrior":
		return "Resilient"
	case "Rogue":
		return "Adaptive"
	default:
		return "Creative"
	}
}

// LifeClassWeakness maps class to a mild weakness label.
func LifeClassWeakness(classType string) string {
	switch classType {
	case "Mage":
		return "Overthinking"
	case "Warrior":
		return "Stubborn"
	case "Rogue":
		return "Restless"
	default:
		return "Impatient"
	}
}

// LifeMarshalRadar encodes radar chart labels/values as JSON arrays.
func LifeMarshalRadar(stats []LifeStatRow) (labelsJSON, valuesJSON string) {
	labels := make([]string, 0, len(stats))
	values := make([]float64, 0, len(stats))
	for _, s := range stats {
		labels = append(labels, s.Name)
		values = append(values, s.RadarValue)
	}
	lb, err := sonic.Marshal(labels)
	if err != nil {
		return "[]", "[]"
	}
	vb, err := sonic.Marshal(values)
	if err != nil {
		return "[]", "[]"
	}
	return string(lb), string(vb)
}

// LifeSegmentIndexes returns 0..n-1 for templ loops.
func LifeSegmentIndexes(n int) []int {
	if n <= 0 {
		return nil
	}
	out := make([]int, n)
	for i := range out {
		out[i] = i
	}
	return out
}
