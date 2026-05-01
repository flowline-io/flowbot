package reader

import (
	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/validate"
	"github.com/gofiber/fiber/v3"
)

var webserviceRules = []webservice.Rule{
	webservice.Get("/", listFeeds),
	webservice.Post("/", createFeed),
	webservice.Get("/entries", listEntries),
	webservice.Patch("/entries", updateEntriesStatus),
}

type createFeedRequest struct {
	FeedURL    string `json:"feed_url" validate:"required,url,max=2048"`
	CategoryID int64  `json:"category_id"`
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
//	@Success	200	{object}	protocol.Response
//	@Security	ApiKeyAuth
//	@Router		/service/reader [get]
func listFeeds(ctx fiber.Ctx) error {
	res, err := ability.Invoke(ctx.Context(), hub.CapReader, ability.OpReaderListFeeds, nil)
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(res.Data))
}

// create feed
//
//	@Summary	Create a new feed
//	@Tags		reader
//	@Accept		json
//	@Produce	json
//	@Param		body	body		createFeedRequest	true	"feed data"
//	@Success	200		{object}	protocol.Response
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

	res, err := ability.Invoke(ctx.Context(), hub.CapReader, ability.OpReaderCreateFeed, map[string]any{
		"feed_url": body.FeedURL,
	})
	if err != nil {
		return err
	}

	return ctx.JSON(protocol.NewSuccessResponse(res.Data))
}

// list entries
//
//	@Summary	List entries
//	@Tags		reader
//	@Accept		json
//	@Produce	json
//	@Param		status	query		string	false	"status filter (read, unread, removed)"
//	@Param		feed_id	query		int		false	"filter by feed ID"
//	@Success	200		{object}	protocol.Response
//	@Security	ApiKeyAuth
//	@Router		/service/reader/entries [get]
func listEntries(ctx fiber.Ctx) error {
	params := map[string]any{}
	if v := ctx.Query("status"); v != "" {
		params["status"] = v
	}
	if v := ctx.Query("feed_id"); v != "" {
		params["feed_id"] = parseQueryInt(v)
	}

	res, err := ability.Invoke(ctx.Context(), hub.CapReader, ability.OpReaderListEntries, params)
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(res.Data))
}

// update entries status
//
//	@Summary	Update entries status
//	@Tags		reader
//	@Accept		json
//	@Produce	json
//	@Param		body	body		updateEntriesRequest	true	"entry IDs and status"
//	@Success	200		{object}	protocol.Response
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

	for _, entryID := range body.EntryIDs {
		var operation string
		switch body.Status {
		case "read":
			operation = ability.OpReaderMarkEntryRead
		default:
			operation = ability.OpReaderMarkEntryUnread
		}
		_, err := ability.Invoke(ctx.Context(), hub.CapReader, operation, map[string]any{
			"id": entryID,
		})
		if err != nil {
			return err
		}
	}

	return ctx.JSON(protocol.NewSuccessResponse(map[string]bool{"success": true}))
}

func parseQueryInt(s string) int64 {
	var v int64
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		v = v*10 + int64(c-'0')
	}
	return v
}

var _ types.MsgPayload
