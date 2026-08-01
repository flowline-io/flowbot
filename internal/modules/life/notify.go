package life

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/notify"
	"github.com/flowline-io/flowbot/pkg/types"
)

const (
	lifeInboxQuestsURL    = "/service/web/life/quests"
	lifeInboxInventoryURL = "/service/web/life/inventory"
	lifeNotifyTimeout     = 5 * time.Second
)

// questNotifySend is the GatewaySend seam (overridable in tests).
var questNotifySend = notify.GatewaySend

func notifyQuestCompleted(userID string, result *CompleteResult) {
	if result == nil || result.Quest == nil {
		return
	}
	payload := buildQuestCompletedPayload(result)
	enqueueLifeNotify(userID, notify.LifeQuestCompletedTemplateID, payload)
}

func notifyQuestFailed(userID, questTitle string) {
	payload := buildQuestFailedPayload(questTitle)
	enqueueLifeNotify(userID, notify.LifeQuestFailedTemplateID, payload)
}

func enqueueLifeNotify(userID, templateID string, payload map[string]any) {
	if strings.TrimSpace(userID) == "" || payload == nil {
		return
	}
	uid := types.Uid(userID)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), lifeNotifyTimeout)
		defer cancel()
		if err := questNotifySend(ctx, uid, templateID, []string{notify.ChannelInapp}, payload); err != nil {
			flog.WarnFields("life: quest notify send failed", map[string]any{
				"uid": userID, "template_id": templateID, "error": err.Error(),
			})
		}
	}()
}

func buildQuestCompletedPayload(result *CompleteResult) map[string]any {
	title := "Quest completed"
	if name := strings.TrimSpace(result.Quest.Title); name != "" {
		title = "Quest completed · " + name
	}
	url := lifeInboxQuestsURL
	if result.Dropped {
		url = lifeInboxInventoryURL
	}
	return map[string]any{
		notify.PayloadKeyTitle:   title,
		notify.PayloadKeySummary: formatQuestCompletedSummary(result),
		notify.PayloadKeyURL:     url,
	}
}

func buildQuestFailedPayload(questTitle string) map[string]any {
	title := "Quest failed"
	if name := strings.TrimSpace(questTitle); name != "" {
		title = "Quest failed · " + name
	}
	return map[string]any{
		notify.PayloadKeyTitle:   title,
		notify.PayloadKeySummary: "Equipment tarnished for 24h.",
		notify.PayloadKeyURL:     lifeInboxQuestsURL,
	}
}

func formatQuestCompletedSummary(result *CompleteResult) string {
	lines := []string{
		fmt.Sprintf("+%d EXP · +%d gold", result.GainedExp, result.GainedGold),
	}
	if result.Dropped {
		drop := strings.TrimSpace(result.ItemName)
		if drop == "" {
			drop = "Unknown item"
		}
		if rarity := strings.TrimSpace(result.ItemRarity); rarity != "" {
			lines = append(lines, fmt.Sprintf("Dropped: %s (%s)", drop, rarity))
		} else {
			lines = append(lines, "Dropped: "+drop)
		}
	}
	if line := formatLevelUpLine(result); line != "" {
		lines = append(lines, line)
	}
	if names := achievementNames(result.NewlyUnlocked); len(names) > 0 {
		lines = append(lines, "Achievements: "+strings.Join(names, ", "))
	}
	return strings.Join(lines, "\n")
}

func formatLevelUpLine(result *CompleteResult) string {
	if result.ProfileLevelAfter > result.ProfileLevelBefore {
		return fmt.Sprintf("Level up: Profile Lv %d → %d",
			result.ProfileLevelBefore, result.ProfileLevelAfter)
	}
	if result.SkillLevelAfter > result.SkillLevelBefore {
		skill := strings.TrimSpace(result.SkillName)
		if skill == "" {
			skill = "Skill"
		}
		return fmt.Sprintf("Level up: %s Lv %d → %d",
			skill, result.SkillLevelBefore, result.SkillLevelAfter)
	}
	return ""
}

func achievementNames(unlocked []UnlockedAchievement) []string {
	if len(unlocked) == 0 {
		return nil
	}
	names := make([]string, 0, len(unlocked))
	for _, a := range unlocked {
		name := strings.TrimSpace(a.Name)
		if name == "" {
			name = strings.TrimSpace(a.Flag)
		}
		if name == "" {
			continue
		}
		names = append(names, name)
	}
	return names
}
