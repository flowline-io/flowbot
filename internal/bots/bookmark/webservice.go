package bookmark

import (
	"fmt"

	"github.com/flowline-io/flowbot/pkg/providers/karakeep"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/validate"
	"github.com/gofiber/fiber/v3"
)

var webserviceRules = []webservice.Rule{
	webservice.Get("/", listBookmarks),
	webservice.Get("/:id", getBookmark),
	webservice.Post("/", createBookmark),
	webservice.Patch("/:id", updateBookmark),
	webservice.Post("/:id/tags", attachTags),
	webservice.Delete("/:id/tags", detachTags),
}

type createBookmarkRequest struct {
	URL string `json:"url" validate:"required,url,max=2048"`
}

type tagsRequest struct {
	Tags []string `json:"tags" validate:"required,dive,min=1,max=50"`
}

// list bookmarks
//
//	@Summary	List bookmarks
//	@Tags		bookmark
//	@Accept		json
//	@Produce	json
//	@Param		limit		query		int		false	"page size"
//	@Param		cursor		query		string	false	"pagination cursor"
//	@Param		archived	query		bool	false	"include archived"
//	@Param		favourited	query		bool	false	"favourited only"
//	@Success	200			{object}	protocol.Response{data=karakeep.BookmarksResponse}
//	@Security	ApiKeyAuth
//	@Router		/service/bookmark [get]
func listBookmarks(ctx fiber.Ctx) error {
	client := karakeep.GetClient()

	query := &karakeep.BookmarksQuery{Limit: karakeep.MaxPageSize}
	if v := ctx.Query("limit"); v != "" {
		if n, err := validate.ValidateVar(v, "gte=1,lte=100"); err == nil && n != nil {
			query.Limit = int(n.(int64))
		}
	}
	if v := ctx.Query("cursor"); v != "" {
		query.Cursor = v
	}
	if v := ctx.Query("archived"); v == "true" {
		query.Archived = true
	}
	if v := ctx.Query("favourited"); v == "true" {
		query.Favourited = true
	}

	resp, err := client.GetAllBookmarks(query)
	if err != nil {
		return fmt.Errorf("failed to get bookmarks: %w", err)
	}

	return ctx.JSON(protocol.NewSuccessResponse(resp))
}

// get single bookmark
//
//	@Summary	Get bookmark by ID
//	@Tags		bookmark
//	@Accept		json
//	@Produce	json
//	@Param		id	path		string	true	"bookmark ID"
//	@Success	200	{object}	protocol.Response{data=karakeep.Bookmark}
//	@Security	ApiKeyAuth
//	@Router		/service/bookmark/{id} [get]
func getBookmark(ctx fiber.Ctx) error {
	id := ctx.Params("id")
	if id == "" {
		return protocol.ErrBadParam.New("id is required")
	}

	client := karakeep.GetClient()
	bookmark, err := client.GetBookmark(id)
	if err != nil {
		return fmt.Errorf("failed to get bookmark: %w", err)
	}

	return ctx.JSON(protocol.NewSuccessResponse(bookmark))
}

// create bookmark
//
//	@Summary	Create a new bookmark
//	@Tags		bookmark
//	@Accept		json
//	@Produce	json
//	@Param		body	body		object{url=string}	true	"bookmark URL"
//	@Success	200		{object}	protocol.Response{data=karakeep.Bookmark}
//	@Security	ApiKeyAuth
//	@Router		/service/bookmark [post]
func createBookmark(ctx fiber.Ctx) error {
	var body createBookmarkRequest
	if err := ctx.Bind().Body(&body); err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}

	if err := validate.Validate.Struct(body); err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}

	client := karakeep.GetClient()
	bookmark, err := client.CreateBookmark(body.URL)
	if err != nil {
		return fmt.Errorf("failed to create bookmark: %w", err)
	}

	return ctx.JSON(protocol.NewSuccessResponse(bookmark))
}

// update bookmark (archive/unarchive)
//
//	@Summary	Update bookmark (archive/unarchive)
//	@Tags		bookmark
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string					true	"bookmark ID"
//	@Param		body	body		object{archived=bool}	true	"archive status"
//	@Success	200		{object}	protocol.Response{data=karakeep.ArchiveResponse}
//	@Security	ApiKeyAuth
//	@Router		/service/bookmark/{id} [patch]
func updateBookmark(ctx fiber.Ctx) error {
	id := ctx.Params("id")
	if id == "" {
		return protocol.ErrBadParam.New("id is required")
	}

	client := karakeep.GetClient()
	archived, err := client.ArchiveBookmark(id)
	if err != nil {
		return fmt.Errorf("failed to archive bookmark: %w", err)
	}

	return ctx.JSON(protocol.NewSuccessResponse(karakeep.ArchiveResponse{Archived: archived}))
}

// attach tags to bookmark
//
//	@Summary	Attach tags to a bookmark
//	@Tags		bookmark
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string		true	"bookmark ID"
//	@Param		body	body		[]string	true	"tag names"
//	@Success	200		{object}	protocol.Response{data=karakeep.AttachTagsResponse}
//	@Security	ApiKeyAuth
//	@Router		/service/bookmark/{id}/tags [post]
func attachTags(ctx fiber.Ctx) error {
	id := ctx.Params("id")
	if id == "" {
		return protocol.ErrBadParam.New("id is required")
	}

	var body tagsRequest
	if err := ctx.Bind().Body(&body); err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}

	if err := validate.Validate.Struct(body); err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}

	if len(body.Tags) == 0 {
		return protocol.ErrBadParam.New("tags are required")
	}

	if len(body.Tags) > validate.MaxTagsCount {
		return protocol.ErrBadParam.New("too many tags")
	}

	client := karakeep.GetClient()
	attached, err := client.AttachTagsToBookmark(id, body.Tags)
	if err != nil {
		return fmt.Errorf("failed to attach tags: %w", err)
	}

	return ctx.JSON(protocol.NewSuccessResponse(karakeep.AttachTagsResponse{Attached: attached}))
}

// detach tags from bookmark
//
//	@Summary	Detach tags from a bookmark
//	@Tags		bookmark
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string		true	"bookmark ID"
//	@Param		body	body		[]string	true	"tag names"
//	@Success	200		{object}	protocol.Response{data=karakeep.DetachTagsResponse}
//	@Security	ApiKeyAuth
//	@Router		/service/bookmark/{id}/tags [delete]
func detachTags(ctx fiber.Ctx) error {
	id := ctx.Params("id")
	if id == "" {
		return protocol.ErrBadParam.New("id is required")
	}

	var body tagsRequest
	if err := ctx.Bind().Body(&body); err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}

	if err := validate.Validate.Struct(body); err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}

	if len(body.Tags) == 0 {
		return protocol.ErrBadParam.New("tags are required")
	}

	if len(body.Tags) > validate.MaxTagsCount {
		return protocol.ErrBadParam.New("too many tags")
	}

	client := karakeep.GetClient()
	detached, err := client.DetachTagsToBookmark(id, body.Tags)
	if err != nil {
		return fmt.Errorf("failed to detach tags: %w", err)
	}

	return ctx.JSON(protocol.NewSuccessResponse(karakeep.DetachTagsResponse{Detached: detached}))
}
