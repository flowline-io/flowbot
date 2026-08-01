package life

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/flog"
	pkglife "github.com/flowline-io/flowbot/pkg/life"
)

// RewardView is one catalog reward for the Rewards UI.
type RewardView struct {
	Flag            string
	Name            string
	Notes           string
	Price           int
	CooldownHours   int
	Active          bool
	OnCooldown      bool
	CooldownEndsAt  string
	CanAfford       bool
	LastRedeemedAt  string
}

// RedemptionView is one redemption history row for the Rewards UI.
type RedemptionView struct {
	Flag        string
	RewardName  string
	PricePaid   int
	RedeemedAt  string
}

// RewardsPage is the full Rewards page model.
type RewardsPage struct {
	Gold             int
	Active           []RewardView
	Inactive         []RewardView
	Redemptions      []RedemptionView
	InactiveTotal    int
	RedemptionsTotal int
}

// CreateRewardInput is the write-set for creating or updating a reward.
type CreateRewardInput struct {
	Name          string
	Notes         string
	Price         int
	CooldownHours int
}

// ListRewardsPage returns gold, active rewards, and paged inactive/redemption lists.
func (s *Service) ListRewardsPage(ctx context.Context, userID string, redemptionsPage, inactivePage, perPage int) (*RewardsPage, error) {
	p, err := s.EnsureProfile(ctx, userID, "", config.DefaultClass)
	if err != nil {
		return nil, err
	}
	activeOnly := true
	activeRows, err := s.store.ListRewards(ctx, p.ID, &activeOnly)
	if err != nil {
		return nil, err
	}
	inactiveOnly := false
	inactivePage, perPage, inactiveOffset := normalizeLifeListPage(inactivePage, perPage)
	inactiveRows, inactiveTotal, err := s.store.ListRewardsPage(ctx, p.ID, &inactiveOnly, perPage, inactiveOffset)
	if err != nil {
		return nil, err
	}
	if inactiveTotal > 0 {
		maxPage := (inactiveTotal + perPage - 1) / perPage
		if inactivePage > maxPage {
			inactivePage = maxPage
			inactiveOffset = (inactivePage - 1) * perPage
			inactiveRows, inactiveTotal, err = s.store.ListRewardsPage(ctx, p.ID, &inactiveOnly, perPage, inactiveOffset)
			if err != nil {
				return nil, err
			}
		}
	}
	redemptionsPage, perPage, redemptionsOffset := normalizeLifeListPage(redemptionsPage, perPage)
	redemptionRows, redemptionsTotal, err := s.store.ListRewardRedemptionsPage(ctx, p.ID, perPage, redemptionsOffset)
	if err != nil {
		return nil, err
	}
	if redemptionsTotal > 0 {
		maxPage := (redemptionsTotal + perPage - 1) / perPage
		if redemptionsPage > maxPage {
			redemptionsPage = maxPage
			redemptionsOffset = (redemptionsPage - 1) * perPage
			redemptionRows, redemptionsTotal, err = s.store.ListRewardRedemptionsPage(ctx, p.ID, perPage, redemptionsOffset)
			if err != nil {
				return nil, err
			}
		}
	}
	now := time.Now()
	page := &RewardsPage{
		Gold:             p.Gold,
		Active:           make([]RewardView, 0, len(activeRows)),
		Inactive:         make([]RewardView, 0, len(inactiveRows)),
		Redemptions:      make([]RedemptionView, 0, len(redemptionRows)),
		InactiveTotal:    inactiveTotal,
		RedemptionsTotal: redemptionsTotal,
	}
	for _, row := range activeRows {
		page.Active = append(page.Active, rewardToView(row, p.Gold, now))
	}
	for _, row := range inactiveRows {
		page.Inactive = append(page.Inactive, rewardToView(row, p.Gold, now))
	}
	for _, row := range redemptionRows {
		page.Redemptions = append(page.Redemptions, RedemptionView{
			Flag:       row.Flag,
			RewardName: row.RewardName,
			PricePaid:  row.PricePaid,
			RedeemedAt: row.RedeemedAt.Format("2006-01-02 15:04"),
		})
	}
	return page, nil
}

// CreateReward creates a player-defined reward.
func (s *Service) CreateReward(ctx context.Context, userID string, in CreateRewardInput) (*gen.LifeReward, error) {
	p, err := s.EnsureProfile(ctx, userID, "", config.DefaultClass)
	if err != nil {
		return nil, err
	}
	normalized, err := normalizeRewardInput(in)
	if err != nil {
		return nil, err
	}
	row, err := s.store.CreateReward(ctx, p.ID, store.LifeRewardCreate{
		Name: normalized.Name, Notes: normalized.Notes,
		Price: normalized.Price, CooldownHours: normalized.CooldownHours,
	})
	if err != nil {
		return nil, err
	}
	flog.InfoFields("life: reward created", map[string]any{
		"uid": userID, "profile_id": p.ID, "reward_flag": row.Flag, "price": row.Price,
	})
	return row, nil
}

// UpdateReward updates name/notes/price/cooldown for a reward by flag.
func (s *Service) UpdateReward(ctx context.Context, userID, rewardFlag string, in CreateRewardInput) error {
	p, err := s.EnsureProfile(ctx, userID, "", config.DefaultClass)
	if err != nil {
		return err
	}
	row, err := s.store.GetRewardByFlag(ctx, p.ID, rewardFlag)
	if err != nil || row == nil {
		return lifeNotFound("reward not found")
	}
	normalized, err := normalizeRewardInput(in)
	if err != nil {
		return err
	}
	if err := s.store.UpdateReward(ctx, row.ID, store.LifeRewardCreate{
		Name: normalized.Name, Notes: normalized.Notes,
		Price: normalized.Price, CooldownHours: normalized.CooldownHours,
	}); err != nil {
		return err
	}
	flog.InfoFields("life: reward updated", map[string]any{
		"uid": userID, "profile_id": p.ID, "reward_flag": rewardFlag,
	})
	return nil
}

// DeactivateReward soft-deletes a reward (active=false).
func (s *Service) DeactivateReward(ctx context.Context, userID, rewardFlag string) error {
	return s.setRewardActive(ctx, userID, rewardFlag, false)
}

// RestoreReward reactivates a soft-deleted reward.
func (s *Service) RestoreReward(ctx context.Context, userID, rewardFlag string) error {
	return s.setRewardActive(ctx, userID, rewardFlag, true)
}

func (s *Service) setRewardActive(ctx context.Context, userID, rewardFlag string, active bool) error {
	p, err := s.EnsureProfile(ctx, userID, "", config.DefaultClass)
	if err != nil {
		return err
	}
	row, err := s.store.GetRewardByFlag(ctx, p.ID, rewardFlag)
	if err != nil || row == nil {
		return lifeNotFound("reward not found")
	}
	if row.Active == active {
		return nil
	}
	if err := s.store.SetRewardActive(ctx, row.ID, active); err != nil {
		return err
	}
	action := "deactivated"
	if active {
		action = "restored"
	}
	flog.InfoFields("life: reward "+action, map[string]any{
		"uid": userID, "profile_id": p.ID, "reward_flag": rewardFlag,
	})
	return nil
}

// RedeemReward deducts gold and records a redemption in one transaction.
func (s *Service) RedeemReward(ctx context.Context, userID, rewardFlag string) error {
	p, err := s.EnsureProfile(ctx, userID, "", config.DefaultClass)
	if err != nil {
		return err
	}
	now := time.Now()
	err = s.store.WithTx(ctx, func(tx *store.LifeStore) error {
		row, err := tx.GetRewardByFlag(ctx, p.ID, rewardFlag)
		if err != nil {
			return err
		}
		if row == nil {
			return lifeNotFound("reward not found")
		}
		if !row.Active {
			return lifeConflict("reward is inactive")
		}
		profile, err := tx.GetProfileByID(ctx, p.ID)
		if err != nil || profile == nil {
			return lifeNotFound("profile not found")
		}
		if profile.Gold < row.Price {
			return lifeConflict("insufficient gold")
		}
		if pkglife.RewardOnCooldown(row.LastRedeemedAt, row.CooldownHours, now) {
			return lifeConflict("reward on cooldown")
		}
		if err := tx.SetProfileGold(ctx, profile.ID, profile.Gold-row.Price); err != nil {
			return err
		}
		if err := tx.MarkRewardRedeemed(ctx, row.ID, now); err != nil {
			return err
		}
		_, err = tx.CreateRewardRedemption(ctx, profile.ID, row.ID, row.Name, row.Price, now)
		return err
	})
	if err != nil {
		return err
	}
	flog.InfoFields("life: reward redeemed", map[string]any{
		"uid": userID, "profile_id": p.ID, "reward_flag": rewardFlag,
	})
	return nil
}

func normalizeRewardInput(in CreateRewardInput) (CreateRewardInput, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return CreateRewardInput{}, lifeInvalid("reward name required")
	}
	if in.Price < 1 {
		return CreateRewardInput{}, lifeInvalid("reward price must be at least 1")
	}
	cooldown := in.CooldownHours
	if cooldown < 0 {
		return CreateRewardInput{}, lifeInvalid("reward cooldown cannot be negative")
	}
	return CreateRewardInput{
		Name:          name,
		Notes:         strings.TrimSpace(in.Notes),
		Price:         in.Price,
		CooldownHours: cooldown,
	}, nil
}

func rewardToView(row *gen.LifeReward, gold int, now time.Time) RewardView {
	view := RewardView{
		Flag:          row.Flag,
		Name:          row.Name,
		Notes:         row.Notes,
		Price:         row.Price,
		CooldownHours: row.CooldownHours,
		Active:        row.Active,
		CanAfford:     gold >= row.Price,
		OnCooldown:    pkglife.RewardOnCooldown(row.LastRedeemedAt, row.CooldownHours, now),
	}
	if row.LastRedeemedAt != nil {
		view.LastRedeemedAt = row.LastRedeemedAt.Format("2006-01-02 15:04")
	}
	if ends := pkglife.RewardCooldownEndsAt(row.LastRedeemedAt, row.CooldownHours, now); ends != nil {
		view.CooldownEndsAt = ends.Format("2006-01-02 15:04")
	}
	return view
}

// ParseRewardPrice parses a form price string into a positive int.
func ParseRewardPrice(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, lifeInvalid("reward price required")
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, lifeInvalid("invalid reward price")
	}
	return n, nil
}

// ParseRewardCooldownHours parses optional cooldown hours (empty → 0).
func ParseRewardCooldownHours(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, lifeInvalid("invalid reward cooldown")
	}
	return n, nil
}
