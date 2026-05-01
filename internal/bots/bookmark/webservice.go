package bookmark

import (
	"context"
	"strconv"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/validate"
	"github.com/gofiber/fiber/v3"
)

var webserviceRules = []webservice.Rule{
	webservice.Get("/", listBookmarks),
	webservice.Get("/check-url", checkURLExists),
	webservice.Get("/search", searchBookmarks),
	webservice.Get("/:id", getBookmark),
	webservice.Post("/", createBookmark),
	webservice.Patch("/:id", archiveBookmark),
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
//	@Success	200			{object}	protocol.Response{data=ability.InvokeResult}
//	@Security	ApiKeyAuth
//	@Router		/service/bookmark [get]
func listBookmarks(ctx fiber.Ctx) error {
	params := pageParams(ctx)
	if v := ctx.Query("archived"); v != "" {
		params["archived"] = v
	}
	if v := ctx.Query("favourited"); v != "" {
		params["favourited"] = v
	}
	return invokeBookmark(ctx, "list", params)
}

// check if URL exists in bookmarks
//
//	@Summary	Check if URL exists in bookmarks
//	@Tags		bookmark
//	@Accept		json
//	@Produce	json
//	@Param		url	query		string	true	"URL to check"
//	@Success	200	{object}	protocol.Response{data=ability.InvokeResult}
//	@Security	ApiKeyAuth
//	@Router		/service/bookmark/check-url [get]
func checkURLExists(ctx fiber.Ctx) error {
	url := ctx.Query("url")
	if url == "" {
		return types.Errorf(types.ErrInvalidArgument, "url is required")
	}
	return invokeBookmark(ctx, "check_url", map[string]any{"url": url})
}

// search bookmarks
//
//	@Summary	Search bookmarks
//	@Tags		bookmark
//	@Accept		json
//	@Produce	json
//	@Param		q			query		string	true	"search query"
//	@Param		sort_order	query		string	false	"sort order"
//	@Param		limit		query		int		false	"page size"
//	@Param		cursor		query		string	false	"pagination cursor"
//	@Success	200			{object}	protocol.Response{data=ability.InvokeResult}
//	@Security	ApiKeyAuth
//	@Router		/service/bookmark/search [get]
func searchBookmarks(ctx fiber.Ctx) error {
	params := pageParams(ctx)
	if v := ctx.Query("q"); v != "" {
		params["q"] = v
	}
	if v := ctx.Query("sort_order"); v != "" {
		params["sort_order"] = v
	}
	return invokeBookmark(ctx, "search", params)
}

// get single bookmark
//
//	@Summary	Get bookmark by ID
//	@Tags		bookmark
//	@Accept		json
//	@Produce	json
//	@Param		id	path		string	true	"bookmark ID"
//	@Success	200	{object}	protocol.Response{data=ability.InvokeResult}
//	@Security	ApiKeyAuth
//	@Router		/service/bookmark/{id} [get]
func getBookmark(ctx fiber.Ctx) error {
	id := ctx.Params("id")
	if id == "" {
		return types.Errorf(types.ErrInvalidArgument, "id is required")
	}
	return invokeBookmark(ctx, "get", map[string]any{"id": id})
}

// create bookmark
//
//	@Summary	Create a new bookmark
//	@Tags		bookmark
//	@Accept		json
//	@Produce	json
//	@Param		body	body		object{url=string}	true	"bookmark URL"
//	@Success	200		{object}	protocol.Response{data=ability.InvokeResult}
//	@Security	ApiKeyAuth
//	@Router		/service/bookmark [post]
func createBookmark(ctx fiber.Ctx) error {
	var body createBookmarkRequest
	if err := ctx.Bind().Body(&body); err != nil {
		return types.WrapError(types.ErrInvalidArgument, "decode create bookmark request", err)
	}
	if err := validate.Validate.Struct(body); err != nil {
		return types.WrapError(types.ErrInvalidArgument, "validate create bookmark request", err)
	}
	return invokeBookmark(ctx, "create", map[string]any{"url": body.URL})
}

// archive bookmark
//
//	@Summary	Archive bookmark
//	@Tags		bookmark
//	@Accept		json
//	@Produce	json
//	@Param		id	path		string	true	"bookmark ID"
//	@Success	200	{object}	protocol.Response{data=ability.InvokeResult}
//	@Security	ApiKeyAuth
//	@Router		/service/bookmark/{id} [patch]
func archiveBookmark(ctx fiber.Ctx) error {
	id := ctx.Params("id")
	if id == "" {
		return types.Errorf(types.ErrInvalidArgument, "id is required")
	}
	return invokeBookmark(ctx, "archive", map[string]any{"id": id})
}

// attach tags to bookmark
//
//	@Summary	Attach tags to a bookmark
//	@Tags		bookmark
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string		true	"bookmark ID"
//	@Param		body	body		[]string	true	"tag names"
//	@Success	200		{object}	protocol.Response{data=ability.InvokeResult}
//	@Security	ApiKeyAuth
//	@Router		/service/bookmark/{id}/tags [post]
func attachTags(ctx fiber.Ctx) error {
	id := ctx.Params("id")
	if id == "" {
		return types.Errorf(types.ErrInvalidArgument, "id is required")
	}
	body, err := bindTags(ctx)
	if err != nil {
		return err
	}
	return invokeBookmark(ctx, "attach_tags", map[string]any{"id": id, "tags": body.Tags})
}

// detach tags from a bookmark
//
//	@Summary	Detach tags from a bookmark
//	@Tags		bookmark
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string		true	"bookmark ID"
//	@Param		body	body		[]string	true	"tag names"
//	@Success	200		{object}	protocol.Response{data=ability.InvokeResult}
//	@Security	ApiKeyAuth
//	@Router		/service/bookmark/{id}/tags [delete]
func detachTags(ctx fiber.Ctx) error {
	id := ctx.Params("id")
	if id == "" {
		return types.Errorf(types.ErrInvalidArgument, "id is required")
	}
	body, err := bindTags(ctx)
	if err != nil {
		return err
	}
	return invokeBookmark(ctx, "detach_tags", map[string]any{"id": id, "tags": body.Tags})
}

func invokeBookmark(ctx fiber.Ctx, operation string, params map[string]any) error {
	res, err := ability.Invoke(context.Background(), hub.CapBookmark, operation, params)
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(res))
}

func pageParams(ctx fiber.Ctx) map[string]any {
	params := map[string]any{}
	if v := ctx.Query("limit"); v != "" {
		if _, err := validate.ValidateVar(v, "gte=1,lte=100"); err == nil {
			if n, err := strconv.Atoi(v); err == nil {
				params["limit"] = n
			}
		}
	}
	if v := ctx.Query("cursor"); v != "" {
		params["cursor"] = v
	}
	return params
}

func bindTags(ctx fiber.Ctx) (tagsRequest, error) {
	var body tagsRequest
	if err := ctx.Bind().Body(&body); err != nil {
		return body, types.WrapError(types.ErrInvalidArgument, "decode tags request", err)
	}
	if err := validate.Validate.Struct(body); err != nil {
		return body, types.WrapError(types.ErrInvalidArgument, "validate tags request", err)
	}
	if len(body.Tags) == 0 {
		return body, types.Errorf(types.ErrInvalidArgument, "tags are required")
	}
	if len(body.Tags) > validate.MaxTagsCount {
		return body, types.Errorf(types.ErrInvalidArgument, "too many tags")
	}
	return body, nil
}
