// Package note implements the note module HTTP webservice routes.
package note

import (
	"context"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
)

var webserviceRules = []webservice.Rule{
	webservice.Get("/", listNotes),
	webservice.Get("/search", searchNotes),
	webservice.Get("/health", noteHealth),
	webservice.Get("/:id", getNote),
	webservice.Post("/", createNote),
	webservice.Patch("/:id", updateNote),
	webservice.Delete("/:id", deleteNote),
	webservice.Get("/:id/content", getNoteContent),
	webservice.Put("/:id/content", setNoteContent),
}

// listNotes handles GET /service/note
//
//	@Summary	List notes
//	@Tags		note
//	@Param		query	query		string	false	"search query"
//	@Param		limit	query		int		false	"page size"
//	@Param		cursor	query		string	false	"pagination cursor"
//	@Success	200		{object}	protocol.Response{data=ability.InvokeResult}
//	@Security	ApiKeyAuth
//	@Router		/service/note [get]
func listNotes(ctx fiber.Ctx) error {
	params := map[string]any{}
	if q := ctx.Query("query"); q != "" {
		params["query"] = q
	}
	if l := ctx.Query("limit"); l != "" {
		params["limit"] = l
	}
	if c := ctx.Query("cursor"); c != "" {
		params["cursor"] = c
	}
	return invokeNote(ctx, ability.OpNoteList, params)
}

// getNote handles GET /service/note/{id}
//
//	@Summary	Get a note by ID
//	@Tags		note
//	@Param		id	path		string	true	"note ID"
//	@Success	200	{object}	protocol.Response{data=ability.InvokeResult}
//	@Security	ApiKeyAuth
//	@Router		/service/note/{id} [get]
func getNote(ctx fiber.Ctx) error {
	return invokeNote(ctx, ability.OpNoteGet, map[string]any{
		"id": ctx.Params("id"),
	})
}

// createNote handles POST /service/note
//
//	@Summary	Create a new note
//	@Tags		note
//	@Accept		json
//	@Produce	json
//	@Param		body	body		object{title=string,content=string,type=string,parent_note_id=string}	true	"note fields"
//	@Success	200		{object}	protocol.Response{data=ability.InvokeResult}
//	@Security	ApiKeyAuth
//	@Router		/service/note [post]
func createNote(ctx fiber.Ctx) error {
	var body struct {
		Title        string `json:"title"`
		Content      string `json:"content"`
		Type         string `json:"type"`
		ParentNoteID string `json:"parent_note_id"`
	}
	if err := ctx.Bind().Body(&body); err != nil {
		return types.WrapError(types.ErrInvalidArgument, "decode note request", err)
	}
	if body.Title == "" {
		return types.Errorf(types.ErrInvalidArgument, "title is required")
	}
	return invokeNote(ctx, ability.OpNoteCreate, map[string]any{
		"title":          body.Title,
		"content":        body.Content,
		"type":           body.Type,
		"parent_note_id": body.ParentNoteID,
	})
}

// updateNote handles PATCH /service/note/{id}
//
//	@Summary	Update a note
//	@Tags		note
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string								true	"note ID"
//	@Param		body	body		object{title=string,content=string}	true	"fields to update"
//	@Success	200		{object}	protocol.Response{data=ability.InvokeResult}
//	@Security	ApiKeyAuth
//	@Router		/service/note/{id} [patch]
func updateNote(ctx fiber.Ctx) error {
	var body struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	if err := ctx.Bind().Body(&body); err != nil {
		return types.WrapError(types.ErrInvalidArgument, "decode note update request", err)
	}
	return invokeNote(ctx, ability.OpNoteUpdate, map[string]any{
		"id":      ctx.Params("id"),
		"title":   body.Title,
		"content": body.Content,
	})
}

// deleteNote handles DELETE /service/note/{id}
//
//	@Summary	Delete a note
//	@Tags		note
//	@Param		id	path		string	true	"note ID"
//	@Success	200	{object}	protocol.Response{}
//	@Security	ApiKeyAuth
//	@Router		/service/note/{id} [delete]
func deleteNote(ctx fiber.Ctx) error {
	return invokeNote(ctx, ability.OpNoteDelete, map[string]any{
		"id": ctx.Params("id"),
	})
}

// getNoteContent handles GET /service/note/{id}/content
//
//	@Summary	Get note content
//	@Tags		note
//	@Param		id	path		string	true	"note ID"
//	@Success	200	{object}	protocol.Response{data=string}
//	@Security	ApiKeyAuth
//	@Router		/service/note/{id}/content [get]
func getNoteContent(ctx fiber.Ctx) error {
	return invokeNote(ctx, ability.OpNoteGetContent, map[string]any{
		"id": ctx.Params("id"),
	})
}

// setNoteContent handles PUT /service/note/{id}/content
//
//	@Summary	Set note content
//	@Tags		note
//	@Accept		plain
//	@Produce	json
//	@Param		id		path		string	true	"note ID"
//	@Param		body	body		string	true	"content body"
//	@Success	200		{object}	protocol.Response{}
//	@Security	ApiKeyAuth
//	@Router		/service/note/{id}/content [put]
func setNoteContent(ctx fiber.Ctx) error {
	content := string(ctx.Body())
	return invokeNote(ctx, ability.OpNoteSetContent, map[string]any{
		"id":      ctx.Params("id"),
		"content": content,
	})
}

// searchNotes handles GET /service/note/search
//
//	@Summary	Search notes
//	@Tags		note
//	@Param		q	query		string	true	"search query"
//	@Success	200	{object}	protocol.Response{data=ability.InvokeResult}
//	@Security	ApiKeyAuth
//	@Router		/service/note/search [get]
func searchNotes(ctx fiber.Ctx) error {
	query := ctx.Query("q")
	if query == "" {
		return types.Errorf(types.ErrInvalidArgument, "query parameter 'q' is required")
	}
	return invokeNote(ctx, ability.OpNoteSearch, map[string]any{
		"query": query,
	})
}

// noteHealth handles GET /service/note/health
//
//	@Summary	Note capability health check
//	@Tags		note
//	@Success	200	{object}	protocol.Response{data=ability.InvokeResult}
//	@Security	ApiKeyAuth
//	@Router		/service/note/health [get]
func noteHealth(ctx fiber.Ctx) error {
	return invokeNote(ctx, ability.OpNoteGetAppInfo, map[string]any{})
}

// invokeNote invokes a note capability operation and writes the response.
func invokeNote(ctx fiber.Ctx, operation string, params map[string]any) error {
	res, err := ability.Invoke(context.Background(), hub.CapNote, operation, params)
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(res))
}
