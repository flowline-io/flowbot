package reader

import (
	"fmt"
	"strconv"

	"github.com/flowline-io/flowbot/pkg/providers/miniflux"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/validate"
	"github.com/gofiber/fiber/v3"
	rssClient "miniflux.app/v2/client"
)

var webserviceRules = []webservice.Rule{
	webservice.Get("/", listFeeds),
	webservice.Get("/:id", getFeed),
	webservice.Post("/", createFeed),
	webservice.Patch("/:id", updateFeed),
	webservice.Post("/:id/refresh", refreshFeed),
	webservice.Get("/entries", listEntries),
	webservice.Patch("/entries", updateEntriesStatus),
	webservice.Get("/:id/entries", getFeedEntries),
}

type createFeedRequest struct {
	FeedURL    string `json:"feed_url" validate:"required,url,max=2048"`
	CategoryID int64  `json:"category_id"`
}

type updateFeedRequest struct {
	Title                       string `json:"title" validate:"omitempty,max=255"`
	FeedURL                     string `json:"feed_url" validate:"omitempty,url,max=2048"`
	SiteURL                     string `json:"site_url" validate:"omitempty,url,max=2048"`
	ScraperRules                string `json:"scraper_rules" validate:"max=1000"`
	RewriteRules                string `json:"rewrite_rules" validate:"max=1000"`
	UrlRewriteRules             string `json:"urlrewrite_rules" validate:"max=1000"`
	BlocklistRules              string `json:"blocklist_rules" validate:"max=1000"`
	KeeplistRules               string `json:"keeplist_rules" validate:"max=1000"`
	BlockFilterEntryRules       string `json:"block_filter_entry_rules" validate:"max=1000"`
	KeepFilterEntryRules        string `json:"keep_filter_entry_rules" validate:"max=1000"`
	UserAgent                   string `json:"user_agent" validate:"max=500"`
	Cookie                      string `json:"cookie" validate:"max=1000"`
	Username                    string `json:"username" validate:"max=255"`
	Password                    string `json:"password" validate:"max=255"`
	Crawler                     *bool  `json:"crawler,omitempty"`
	IgnoreHTTPCache             *bool  `json:"ignore_http_cache,omitempty"`
	AllowSelfSignedCertificates *bool  `json:"allow_self_signed_certificates,omitempty"`
	FetchViaProxy               *bool  `json:"fetch_via_proxy,omitempty"`
	IgnoreEntryUpdates          *bool  `json:"ignore_entry_updates,omitempty"`
	DisableHTTP2                *bool  `json:"disable_http2,omitempty"`
	HideGlobally                *bool  `json:"hide_globally,omitempty"`
	Disabled                    *bool  `json:"disabled,omitempty"`
}

type updateEntriesRequest struct {
	EntryIDs []int64 `json:"entry_ids" validate:"required,min=1,max=1000"`
	Status   string  `json:"status" validate:"required,oneof=read unread removed"`
}

// list feeds
//
//	@Summary	List all feeds
//	@Tags		reader
//	@Accept		json
//	@Produce	json
//	@Success	200	{object}	protocol.Response{data=rssClient.Feeds}
//	@Security	ApiKeyAuth
//	@Router		/service/reader [get]
func listFeeds(ctx fiber.Ctx) error {
	client := miniflux.GetClient()

	feeds, err := client.GetFeeds()
	if err != nil {
		return fmt.Errorf("failed to get feeds: %w", err)
	}

	return ctx.JSON(protocol.NewSuccessResponse(feeds))
}

// get single feed
//
//	@Summary	Get feed by ID
//	@Tags		reader
//	@Accept		json
//	@Produce	json
//	@Param		id	path		string	true	"feed ID"
//	@Success	200	{object}	protocol.Response{data=rssClient.Feed}
//	@Security	ApiKeyAuth
//	@Router		/service/reader/{id} [get]
func getFeed(ctx fiber.Ctx) error {
	idStr := ctx.Params("id")
	if idStr == "" {
		return protocol.ErrBadParam.New("id is required")
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return protocol.ErrBadParam.New("invalid feed ID")
	}

	client := miniflux.GetClient()

	feed, err := client.GetFeed(id)
	if err != nil {
		return fmt.Errorf("failed to get feed: %w", err)
	}

	return ctx.JSON(protocol.NewSuccessResponse(feed))
}

// create feed
//
//	@Summary	Create a new feed
//	@Tags		reader
//	@Accept		json
//	@Produce	json
//	@Param		body	body		object{feed_url=string,category_id=int}	true	"feed data"
//	@Success	200		{object}	protocol.Response{data=map[string]int64}
//	@Security	ApiKeyAuth
//	@Router		/service/reader [post]
func createFeed(ctx fiber.Ctx) error {
	var body createFeedRequest
	if err := ctx.Bind().Body(&body); err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}

	if err := validate.Validate.Struct(body); err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}

	client := miniflux.GetClient()

	req := &rssClient.FeedCreationRequest{
		FeedURL:    body.FeedURL,
		CategoryID: body.CategoryID,
	}

	feedID, err := client.CreateFeed(req)
	if err != nil {
		return fmt.Errorf("failed to create feed: %w", err)
	}

	return ctx.JSON(protocol.NewSuccessResponse(map[string]int64{"id": feedID}))
}

// update feed
//
//	@Summary	Update a feed
//	@Tags		reader
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string	true	"feed ID"
//	@Param		body	body		updateFeedRequest	true	"feed data"
//	@Success	200		{object}	protocol.Response{data=rssClient.Feed}
//	@Security	ApiKeyAuth
//	@Router		/service/reader/{id} [patch]
func updateFeed(ctx fiber.Ctx) error {
	idStr := ctx.Params("id")
	if idStr == "" {
		return protocol.ErrBadParam.New("id is required")
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return protocol.ErrBadParam.New("invalid feed ID")
	}

	var body updateFeedRequest
	if err := ctx.Bind().Body(&body); err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}

	if err := validate.Validate.Struct(body); err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}

	client := miniflux.GetClient()

	req := &rssClient.FeedModificationRequest{}
	if body.Title != "" {
		req.Title = rssClient.SetOptionalField(body.Title)
	}
	if body.FeedURL != "" {
		req.FeedURL = rssClient.SetOptionalField(body.FeedURL)
	}
	if body.SiteURL != "" {
		req.SiteURL = rssClient.SetOptionalField(body.SiteURL)
	}
	if body.ScraperRules != "" {
		req.ScraperRules = rssClient.SetOptionalField(body.ScraperRules)
	}
	if body.RewriteRules != "" {
		req.RewriteRules = rssClient.SetOptionalField(body.RewriteRules)
	}
	if body.UrlRewriteRules != "" {
		req.UrlRewriteRules = rssClient.SetOptionalField(body.UrlRewriteRules)
	}
	if body.BlocklistRules != "" {
		req.BlocklistRules = rssClient.SetOptionalField(body.BlocklistRules)
	}
	if body.KeeplistRules != "" {
		req.KeeplistRules = rssClient.SetOptionalField(body.KeeplistRules)
	}
	if body.BlockFilterEntryRules != "" {
		req.BlockFilterEntryRules = rssClient.SetOptionalField(body.BlockFilterEntryRules)
	}
	if body.KeepFilterEntryRules != "" {
		req.KeepFilterEntryRules = rssClient.SetOptionalField(body.KeepFilterEntryRules)
	}
	if body.UserAgent != "" {
		req.UserAgent = rssClient.SetOptionalField(body.UserAgent)
	}
	if body.Cookie != "" {
		req.Cookie = rssClient.SetOptionalField(body.Cookie)
	}
	if body.Username != "" {
		req.Username = rssClient.SetOptionalField(body.Username)
	}
	if body.Password != "" {
		req.Password = rssClient.SetOptionalField(body.Password)
	}
	if body.Crawler != nil {
		req.Crawler = rssClient.SetOptionalField(*body.Crawler)
	}
	if body.IgnoreHTTPCache != nil {
		req.IgnoreHTTPCache = rssClient.SetOptionalField(*body.IgnoreHTTPCache)
	}
	if body.AllowSelfSignedCertificates != nil {
		req.AllowSelfSignedCertificates = rssClient.SetOptionalField(*body.AllowSelfSignedCertificates)
	}
	if body.FetchViaProxy != nil {
		req.FetchViaProxy = rssClient.SetOptionalField(*body.FetchViaProxy)
	}
	if body.IgnoreEntryUpdates != nil {
		req.IgnoreEntryUpdates = rssClient.SetOptionalField(*body.IgnoreEntryUpdates)
	}
	if body.DisableHTTP2 != nil {
		req.DisableHTTP2 = rssClient.SetOptionalField(*body.DisableHTTP2)
	}
	if body.HideGlobally != nil {
		req.HideGlobally = rssClient.SetOptionalField(*body.HideGlobally)
	}
	if body.Disabled != nil {
		req.Disabled = rssClient.SetOptionalField(*body.Disabled)
	}

	feed, err := client.UpdateFeed(id, req)
	if err != nil {
		return fmt.Errorf("failed to update feed: %w", err)
	}

	return ctx.JSON(protocol.NewSuccessResponse(feed))
}

// refresh feed
//
//	@Summary	Refresh a feed
//	@Tags		reader
//	@Accept		json
//	@Produce	json
//	@Param		id	path		string	true	"feed ID"
//	@Success	200	{object}	protocol.Response{data=map[string]bool}
//	@Security	ApiKeyAuth
//	@Router		/service/reader/{id}/refresh [post]
func refreshFeed(ctx fiber.Ctx) error {
	idStr := ctx.Params("id")
	if idStr == "" {
		return protocol.ErrBadParam.New("id is required")
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return protocol.ErrBadParam.New("invalid feed ID")
	}

	client := miniflux.GetClient()

	err = client.RefreshFeed(id)
	if err != nil {
		return fmt.Errorf("failed to refresh feed: %w", err)
	}

	return ctx.JSON(protocol.NewSuccessResponse(map[string]bool{"success": true}))
}

// list entries
//
//	@Summary	List entries
//	@Tags		reader
//	@Accept		json
//	@Produce	json
//	@Param		status		query		string	false	"status filter (read, unread, removed)"
//	@Param		limit		query		int		false	"page size"
//	@Param		offset		query		int		false	"pagination offset"
//	@Param		order		query		string	false	"order field (id, published_at, status)"
//	@Param		direction	query		string	false	"sort direction (asc, desc)"
//	@Param		starred		query		bool	false	"starred only"
//	@Param		feed_id		query		int		false	"filter by feed ID"
//	@Param		category_id	query		int		false	"filter by category ID"
//	@Success	200			{object}	protocol.Response{data=rssClient.EntryResultSet}
//	@Security	ApiKeyAuth
//	@Router		/service/reader/entries [get]
func listEntries(ctx fiber.Ctx) error {
	filter := &rssClient.Filter{}

	if v := ctx.Query("status"); v != "" {
		filter.Status = v
	}
	if v := ctx.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			filter.Limit = n
		}
	}
	if v := ctx.Query("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			filter.Offset = n
		}
	}
	if v := ctx.Query("order"); v != "" {
		filter.Order = v
	}
	if v := ctx.Query("direction"); v != "" {
		filter.Direction = v
	}
	if v := ctx.Query("starred"); v == "true" {
		filter.Starred = "true"
	}
	if v := ctx.Query("feed_id"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			filter.FeedID = n
		}
	}
	if v := ctx.Query("category_id"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			filter.CategoryID = n
		}
	}

	client := miniflux.GetClient()

	entries, err := client.GetEntries(filter)
	if err != nil {
		return fmt.Errorf("failed to get entries: %w", err)
	}

	return ctx.JSON(protocol.NewSuccessResponse(entries))
}

// update entries status
//
//	@Summary	Update entries status
//	@Tags		reader
//	@Accept		json
//	@Produce	json
//	@Param		body	body		updateEntriesRequest	true	"entry IDs and status"
//	@Success	200		{object}	protocol.Response{data=map[string]bool}
//	@Security	ApiKeyAuth
//	@Router		/service/reader/entries [patch]
func updateEntriesStatus(ctx fiber.Ctx) error {
	var body updateEntriesRequest
	if err := ctx.Bind().Body(&body); err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}

	if err := validate.Validate.Struct(body); err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}

	client := miniflux.GetClient()

	err := client.UpdateEntries(body.EntryIDs, body.Status)
	if err != nil {
		return fmt.Errorf("failed to update entries status: %w", err)
	}

	return ctx.JSON(protocol.NewSuccessResponse(map[string]bool{"success": true}))
}

// get feed entries
//
//	@Summary	Get entries for a specific feed
//	@Tags		reader
//	@Accept		json
//	@Produce	json
//	@Param		id			path		string	true	"feed ID"
//	@Param		status		query		string	false	"status filter (read, unread, removed)"
//	@Param		limit		query		int		false	"page size"
//	@Param		offset		query		int		false	"pagination offset"
//	@Param		order		query		string	false	"order field"
//	@Param		direction	query		string	false	"sort direction (asc, desc)"
//	@Param		starred		query		bool	false	"starred only"
//	@Success	200			{object}	protocol.Response{data=rssClient.EntryResultSet}
//	@Security	ApiKeyAuth
//	@Router		/service/reader/{id}/entries [get]
func getFeedEntries(ctx fiber.Ctx) error {
	idStr := ctx.Params("id")
	if idStr == "" {
		return protocol.ErrBadParam.New("id is required")
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return protocol.ErrBadParam.New("invalid feed ID")
	}

	filter := &rssClient.Filter{}

	if v := ctx.Query("status"); v != "" {
		filter.Status = v
	}
	if v := ctx.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			filter.Limit = n
		}
	}
	if v := ctx.Query("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			filter.Offset = n
		}
	}
	if v := ctx.Query("order"); v != "" {
		filter.Order = v
	}
	if v := ctx.Query("direction"); v != "" {
		filter.Direction = v
	}
	if v := ctx.Query("starred"); v == "true" {
		filter.Starred = "true"
	}

	client := miniflux.GetClient()

	entries, err := client.GetFeedEntries(id, filter)
	if err != nil {
		return fmt.Errorf("failed to get feed entries: %w", err)
	}

	return ctx.JSON(protocol.NewSuccessResponse(entries))
}
