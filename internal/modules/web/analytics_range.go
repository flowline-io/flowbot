package web

import (
	"net/http"
	"net/url"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/pkg/types"
)

type analyticsRange struct {
	Since       time.Time
	Until       time.Time
	ActiveRange string
	RangeLabel  string
	Query       string // range=… or since=&until= for appending to URLs
}

func resolveAnalyticsRange(c fiber.Ctx) (analyticsRange, error) {
	since, until, activeRange, rangeLabel, err := types.ResolveTokenUsageRange(
		c.Query("range", "7d"),
		c.Query("since", ""),
		c.Query("until", ""),
		time.Now().UTC(),
	)
	if err != nil {
		return analyticsRange{}, err
	}
	q := url.Values{}
	if activeRange == "custom" {
		q.Set("since", since.UTC().Format("2006-01-02"))
		q.Set("until", until.UTC().Format("2006-01-02"))
	} else {
		q.Set("range", activeRange)
	}
	return analyticsRange{
		Since: since, Until: until, ActiveRange: activeRange, RangeLabel: rangeLabel, Query: q.Encode(),
	}, nil
}

func invalidAnalyticsRange(c fiber.Ctx) error {
	return c.Status(http.StatusBadRequest).SendString(webMsg(c, "error.analytics.invalid_request"))
}
