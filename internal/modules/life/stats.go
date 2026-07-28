package life

import (
	"context"
	"time"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	pkglife "github.com/flowline-io/flowbot/pkg/life"
)

// GetStatsPage builds the Life stats dashboard for the given IANA timezone.
// Invalid or empty timezoneName falls back to UTC.
func (s *Service) GetStatsPage(ctx context.Context, userID, timezoneName string) (*pkglife.StatsPage, error) {
	p, err := s.EnsureProfile(ctx, userID, "", config.DefaultClass)
	if err != nil {
		return nil, err
	}
	loc, tzName := resolveStatsLocation(timezoneName)
	now := time.Now()
	since, until := pkglife.StatsWindowBounds(now, loc, pkglife.StatsWindowDays)

	actions, err := s.store.ListActionLogsInRange(ctx, p.ID, since, until)
	if err != nil {
		return nil, err
	}
	redemptions, err := s.store.ListRewardRedemptionsInRange(ctx, p.ID, since, until)
	if err != nil {
		return nil, err
	}
	unlocks, err := s.store.ListAchievementUnlocksInRange(ctx, p.ID, since, until)
	if err != nil {
		return nil, err
	}
	quests, err := s.store.ListQuestsCompletedInRange(ctx, p.ID, since, until)
	if err != nil {
		return nil, err
	}
	questCharCode, err := s.questCharacteristicCodes(ctx, p.ID, actions)
	if err != nil {
		return nil, err
	}

	page := pkglife.BuildStatsPage(pkglife.StatsInput{
		Actions:      mapStatsActions(actions, questCharCode),
		Quests:       mapStatsQuests(quests),
		Redemptions:  mapStatsRedemptions(redemptions),
		Unlocks:      mapStatsUnlocks(unlocks),
		Location:     loc,
		Now:          now,
		TimezoneName: tzName,
	})
	return &page, nil
}

func resolveStatsLocation(timezoneName string) (*time.Location, string) {
	if timezoneName == "" {
		return time.UTC, "UTC"
	}
	loc, err := time.LoadLocation(timezoneName)
	if err != nil {
		return time.UTC, "UTC"
	}
	return loc, timezoneName
}

func (s *Service) questCharacteristicCodes(ctx context.Context, profileID int64, actions []*gen.LifeActionLog) (map[int64]string, error) {
	questIDs := uniqueQuestIDs(actions)
	questRows, err := s.store.ListQuestsByIDs(ctx, profileID, questIDs)
	if err != nil {
		return nil, err
	}
	skills, err := s.store.ListSkills(ctx, profileID)
	if err != nil {
		return nil, err
	}
	chars, err := s.store.ListCharacteristics(ctx, profileID)
	if err != nil {
		return nil, err
	}
	skillChar := make(map[int64]int64, len(skills))
	for _, sk := range skills {
		skillChar[sk.ID] = sk.CharacteristicID
	}
	charCode := make(map[int64]string, len(chars))
	for _, c := range chars {
		charCode[c.ID] = c.Code
	}
	out := make(map[int64]string, len(questRows))
	for _, q := range questRows {
		charID, ok := skillChar[q.SkillID]
		if !ok {
			continue
		}
		code, ok := charCode[charID]
		if !ok {
			continue
		}
		out[q.ID] = code
	}
	return out, nil
}

func uniqueQuestIDs(actions []*gen.LifeActionLog) []int64 {
	seen := make(map[int64]struct{})
	ids := make([]int64, 0)
	for _, row := range actions {
		if row.QuestID == nil {
			continue
		}
		id := *row.QuestID
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids
}

func mapStatsActions(actions []*gen.LifeActionLog, questCharCode map[int64]string) []pkglife.StatsActionEvent {
	events := make([]pkglife.StatsActionEvent, 0, len(actions))
	for _, row := range actions {
		ev := pkglife.StatsActionEvent{
			At:         row.CreatedAt,
			GainedExp:  row.GainedExp,
			GainedGold: row.GainedGold,
			Dropped:    row.DroppedInventoryID != nil,
		}
		if row.QuestID != nil {
			ev.CharacteristicCode = questCharCode[*row.QuestID]
		}
		events = append(events, ev)
	}
	return events
}

func mapStatsQuests(quests []*gen.LifeQuest) []pkglife.StatsQuestCompletion {
	out := make([]pkglife.StatsQuestCompletion, 0, len(quests))
	for _, q := range quests {
		if q.CompletedAt == nil {
			continue
		}
		out = append(out, pkglife.StatsQuestCompletion{
			CompletedAt: *q.CompletedAt,
			Type:        q.Type,
		})
	}
	return out
}

func mapStatsRedemptions(rows []*gen.LifeRewardRedemption) []pkglife.StatsRedemptionEvent {
	out := make([]pkglife.StatsRedemptionEvent, 0, len(rows))
	for _, r := range rows {
		out = append(out, pkglife.StatsRedemptionEvent{
			At:        r.RedeemedAt,
			PricePaid: r.PricePaid,
		})
	}
	return out
}

func mapStatsUnlocks(rows []*gen.LifeAchievementUnlock) []pkglife.StatsUnlockEvent {
	out := make([]pkglife.StatsUnlockEvent, 0, len(rows))
	for _, u := range rows {
		out = append(out, pkglife.StatsUnlockEvent{At: u.UnlockedAt})
	}
	return out
}
