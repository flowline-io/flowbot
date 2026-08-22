package web

import (
	"context"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v3"

	lifemod "github.com/flowline-io/flowbot/internal/modules/life"
	"github.com/flowline-io/flowbot/pkg/views/pages"
)

func lifeRewardsPage(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid, err := lifeUID(ctx)
	if err != nil {
		return err
	}
	redemptionsPage := parsePositiveIntQuery(ctx, "redemptions_page", 1)
	inactivePage := parsePositiveIntQuery(ctx, "inactive_page", 1)
	archiveTab := pages.LifeNormalizeRewardsArchiveTab(ctx.Query("archive_tab"))
	page, err := lifeService().ListRewardsPage(
		context.Background(), uid, redemptionsPage, inactivePage, pages.LifeDefaultListPerPage,
	)
	if err != nil {
		return toastError(ctx, lifeUserError(err))
	}
	pending, err := lifeService().ListQuests(context.Background(), uid, "Pending")
	if err != nil {
		return toastError(ctx, lifeUserError(err))
	}
	ctx.Type("html")
	return pages.LifeRewardsPage(mapLifeRewardsData(page, len(pending), redemptionsPage, inactivePage, archiveTab)).
		Render(ctx.Context(), ctx.Response().BodyWriter())
}

func lifeCreateReward(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid, err := lifeUID(ctx)
	if err != nil {
		return err
	}
	in, err := parseLifeRewardForm(ctx)
	if err != nil {
		return toastError(ctx, lifeUserError(err))
	}
	if _, err := lifeService().CreateReward(context.Background(), uid, in); err != nil {
		return toastError(ctx, lifeUserError(err))
	}
	ctx.Set("HX-Redirect", "/service/web/life/rewards")
	return ctx.SendStatus(http.StatusOK)
}

func lifeUpdateReward(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid, err := lifeUID(ctx)
	if err != nil {
		return err
	}
	in, err := parseLifeRewardForm(ctx)
	if err != nil {
		return toastError(ctx, lifeUserError(err))
	}
	if err := lifeService().UpdateReward(context.Background(), uid, ctx.Params("flag"), in); err != nil {
		return toastError(ctx, lifeUserError(err))
	}
	ctx.Set("HX-Redirect", "/service/web/life/rewards")
	return ctx.SendStatus(http.StatusOK)
}

func lifeDeactivateReward(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid, err := lifeUID(ctx)
	if err != nil {
		return err
	}
	if err := lifeService().DeactivateReward(context.Background(), uid, ctx.Params("flag")); err != nil {
		return toastError(ctx, lifeUserError(err))
	}
	ctx.Set("HX-Redirect", "/service/web/life/rewards")
	return ctx.SendStatus(http.StatusOK)
}

func lifeRestoreReward(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid, err := lifeUID(ctx)
	if err != nil {
		return err
	}
	if err := lifeService().RestoreReward(context.Background(), uid, ctx.Params("flag")); err != nil {
		return toastError(ctx, lifeUserError(err))
	}
	ctx.Set("HX-Redirect", "/service/web/life/rewards")
	return ctx.SendStatus(http.StatusOK)
}

func lifeRedeemReward(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid, err := lifeUID(ctx)
	if err != nil {
		return err
	}
	if err := lifeService().RedeemReward(context.Background(), uid, ctx.Params("flag")); err != nil {
		return toastError(ctx, lifeUserError(err))
	}
	ctx.Set("HX-Redirect", "/service/web/life/rewards")
	return ctx.SendStatus(http.StatusOK)
}

func parseLifeRewardForm(ctx fiber.Ctx) (lifemod.CreateRewardInput, error) {
	price, err := lifemod.ParseRewardPrice(ctx.FormValue("price"))
	if err != nil {
		return lifemod.CreateRewardInput{}, err
	}
	cooldown, err := lifemod.ParseRewardCooldownHours(ctx.FormValue("cooldown_hours"))
	if err != nil {
		return lifemod.CreateRewardInput{}, err
	}
	return lifemod.CreateRewardInput{
		Name:          strings.TrimSpace(ctx.FormValue("name")),
		Notes:         strings.TrimSpace(ctx.FormValue("notes")),
		Price:         price,
		CooldownHours: cooldown,
	}, nil
}

func mapLifeRewardsData(
	page *lifemod.RewardsPage,
	pendingCount, redemptionsPage, inactivePage int,
	archiveTab string,
) pages.LifeRewardsData {
	if page == nil {
		return pages.LifeRewardsData{PendingCount: pendingCount, ArchiveTab: pages.LifeNormalizeRewardsArchiveTab(archiveTab)}
	}
	redemptionsInfo := pages.LifeWithRedemptionsPager(
		pages.LifeBuildPageInfo(redemptionsPage, pages.LifeDefaultListPerPage, page.RedemptionsTotal),
		inactivePage,
	)
	inactiveInfo := pages.LifeWithInactiveRewardsPager(
		pages.LifeBuildPageInfo(inactivePage, pages.LifeDefaultListPerPage, page.InactiveTotal),
		redemptionsPage,
	)
	return pages.LifeRewardsData{
		Gold:            page.Gold,
		Active:          mapLifeRewardRows(page.Active),
		Inactive:        mapLifeRewardRows(page.Inactive),
		Redemptions:     mapLifeRedemptionRows(page.Redemptions),
		RedemptionsPage: redemptionsInfo,
		InactivePage:    inactiveInfo,
		ArchiveTab:      pages.LifeNormalizeRewardsArchiveTab(archiveTab),
		PendingCount:    pendingCount,
	}
}

func mapLifeRewardRows(items []lifemod.RewardView) []pages.LifeRewardRow {
	out := make([]pages.LifeRewardRow, 0, len(items))
	for _, it := range items {
		out = append(out, pages.LifeRewardRow{
			Flag:           it.Flag,
			Name:           it.Name,
			Notes:          it.Notes,
			Price:          it.Price,
			CooldownHours:  it.CooldownHours,
			OnCooldown:     it.OnCooldown,
			CooldownEndsAt: it.CooldownEndsAt,
			CanAfford:      it.CanAfford,
			CanRedeem:      it.Active && it.CanAfford && !it.OnCooldown,
		})
	}
	return out
}

func mapLifeRedemptionRows(items []lifemod.RedemptionView) []pages.LifeRedemptionRow {
	out := make([]pages.LifeRedemptionRow, 0, len(items))
	for _, it := range items {
		out = append(out, pages.LifeRedemptionRow{
			Flag:       it.Flag,
			RewardName: it.RewardName,
			PricePaid:  it.PricePaid,
			RedeemedAt: it.RedeemedAt,
		})
	}
	return out
}
