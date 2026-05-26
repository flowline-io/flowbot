// Package hub implements webservice routes for hub management, bookmark,
// kanban, note, and reader capabilities.
package hub

import (
	"context"
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/providers/kanboard"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/validate"
)

func queryOrParam(ctx fiber.Ctx, name string) string {
	v := ctx.Query(name)
	if v == "_" {
		return ""
	}
	if v != "" {
		return v
	}
	return ctx.Params(name)
}

// --- Hub routes (registered under /service/hub) ---

var hubWebserviceRules = []webservice.Rule{
	webservice.Get("/resource-chain", queryByTag),
	webservice.Get("/:app/:entity_id/relations", getRelations),
}

func queryByTag(ctx fiber.Ctx) error {
	key := ctx.Query("key")
	value := ctx.Query("value")
	if key == "" || value == "" {
		return types.Errorf(types.ErrInvalidArgument, "key and value query params are required")
	}

	limit := 20
	if v := ctx.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	cursor := ctx.Query("cursor")

	events, nextCursor, err := rcStore.FindResourcesByTag(context.Background(), key, value, limit, cursor)
	if err != nil {
		return err
	}

	eventIDs := make([]string, len(events))
	for i, e := range events {
		eventIDs[i] = e.EventID
	}

	links, err := rcStore.FindResourceLinks(context.Background(), eventIDs)
	if err != nil {
		flog.Error(fmt.Errorf("hub resource-chain: find links: %w", err))
	}

	type resEntry struct {
		EntityID   string `json:"entity_id"`
		App        string `json:"app"`
		Capability string `json:"capability"`
		EventID    string `json:"event_id"`
		CreatedAt  string `json:"created_at"`
	}
	type linkEntry struct {
		Source       resEntry `json:"source"`
		Target       resEntry `json:"target"`
		PipelineName string   `json:"pipeline_name"`
		CreatedAt    string   `json:"created_at"`
	}

	resources := make([]resEntry, len(events))
	for i, e := range events {
		resources[i] = resEntry{
			EntityID: e.EntityID, App: e.App, Capability: e.Capability,
			EventID: e.EventID, CreatedAt: e.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	linkEntries := make([]linkEntry, 0, len(links))
	for _, l := range links {
		linkEntries = append(linkEntries, linkEntry{
			Source:       resEntry{EntityID: l.SourceEntityID, App: l.SourceApp},
			Target:       resEntry{EntityID: l.TargetEntityID, App: l.TargetApp},
			PipelineName: l.PipelineName,
			CreatedAt:    l.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	result := types.KV{
		"tag":       types.KV{"key": key, "value": value},
		"resources": resources,
		"links":     linkEntries,
	}
	if nextCursor != "" {
		result["cursor"] = nextCursor
	}

	return ctx.JSON(protocol.NewSuccessResponse(result))
}

func getRelations(ctx fiber.Ctx) error {
	app := queryOrParam(ctx, "app")
	entityID := queryOrParam(ctx, "entity_id")
	if app == "" || entityID == "" {
		return types.Errorf(types.ErrInvalidArgument, "app and entity_id path params are required")
	}

	relations, err := rcStore.FindRelations(context.Background(), app, entityID)
	if err != nil {
		return err
	}
	if relations == nil {
		relations = &schema.ResourceRelations{
			App: app, EntityID: entityID,
			Upstream: []schema.ResourceRef{}, Downstream: []schema.ResourceRef{},
		}
	}

	return ctx.JSON(protocol.NewSuccessResponse(relations))
}

// --- Bookmark routes (registered under /service/bookmark) ---

type createBookmarkRequest struct {
	URL string `json:"url" validate:"required,url,max=2048"`
}

type tagsRequest struct {
	Tags []string `json:"tags" validate:"required,dive,min=1,max=50"`
}

var bookmarkWebserviceRules = []webservice.Rule{
	webservice.Get("/", listBookmarks),
	webservice.Get("/check-url", checkURLExists),
	webservice.Get("/search", searchBookmarks),
	webservice.Get("/:id", getBookmark),
	webservice.Post("/", createBookmark),
	webservice.Patch("/:id", archiveBookmark),
	webservice.Post("/:id/tags", attachTags),
	webservice.Delete("/:id/tags", detachTags),
}

func listBookmarks(ctx fiber.Ctx) error {
	params := pageParams(ctx)
	if v := ctx.Query("archived"); v != "" {
		params["archived"] = v
	}
	if v := ctx.Query("favourited"); v != "" {
		params["favourited"] = v
	}
	return invokeBookmark(ctx, ability.OpBookmarkList, params)
}

func checkURLExists(ctx fiber.Ctx) error {
	url := ctx.Query("url")
	if url == "" {
		return types.Errorf(types.ErrInvalidArgument, "url is required")
	}
	return invokeBookmark(ctx, ability.OpBookmarkCheckURL, map[string]any{"url": url})
}

func searchBookmarks(ctx fiber.Ctx) error {
	params := pageParams(ctx)
	if v := ctx.Query("q"); v != "" {
		params["q"] = v
	}
	if v := ctx.Query("sort_order"); v != "" {
		params["sort_order"] = v
	}
	return invokeBookmark(ctx, ability.OpBookmarkSearch, params)
}

func getBookmark(ctx fiber.Ctx) error {
	id := ctx.Params("id")
	if id == "" {
		return types.Errorf(types.ErrInvalidArgument, "id is required")
	}
	return invokeBookmark(ctx, ability.OpBookmarkGet, map[string]any{"id": id})
}

func createBookmark(ctx fiber.Ctx) error {
	var body createBookmarkRequest
	if err := ctx.Bind().Body(&body); err != nil {
		return types.WrapError(types.ErrInvalidArgument, "decode create bookmark request", err)
	}
	if err := validate.Validate.Struct(body); err != nil {
		return types.WrapError(types.ErrInvalidArgument, "validate create bookmark request", err)
	}
	return invokeBookmark(ctx, ability.OpBookmarkCreate, map[string]any{"url": body.URL})
}

func archiveBookmark(ctx fiber.Ctx) error {
	id := ctx.Params("id")
	if id == "" {
		return types.Errorf(types.ErrInvalidArgument, "id is required")
	}
	return invokeBookmark(ctx, ability.OpBookmarkArchive, map[string]any{"id": id})
}

func attachTags(ctx fiber.Ctx) error {
	id := ctx.Params("id")
	if id == "" {
		return types.Errorf(types.ErrInvalidArgument, "id is required")
	}
	body, err := bindTags(ctx)
	if err != nil {
		return err
	}
	return invokeBookmark(ctx, ability.OpBookmarkAttachTags, map[string]any{"id": id, "tags": body.Tags})
}

func detachTags(ctx fiber.Ctx) error {
	id := ctx.Params("id")
	if id == "" {
		return types.Errorf(types.ErrInvalidArgument, "id is required")
	}
	body, err := bindTags(ctx)
	if err != nil {
		return err
	}
	return invokeBookmark(ctx, ability.OpBookmarkDetachTags, map[string]any{"id": id, "tags": body.Tags})
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
		if err := validate.ValidateVar(v, "gte=1,lte=100"); err == nil {
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

// --- Kanban routes (registered under /service/kanban) ---

type createTaskRequest struct {
	Title       string `json:"title" validate:"required,min=1,max=200"`
	Description string `json:"description" validate:"max=2000"`
	ProjectID   int    `json:"project_id" validate:"gte=0"`
	ColumnID    int    `json:"column_id" validate:"gte=0"`
}

type updateTaskRequest struct {
	Title       string `json:"title" validate:"omitempty,min=1,max=200"`
	Description string `json:"description" validate:"max=2000"`
}

type moveTaskRequest struct {
	ColumnID   int `json:"column_id" validate:"required,gte=1"`
	Position   int `json:"position" validate:"gte=0"`
	SwimlaneID int `json:"swimlane_id" validate:"gte=0"`
	ProjectID  int `json:"project_id" validate:"gte=0"`
}

type saveMetadataRequest struct {
	Values kanboard.TaskMetadata `json:"values" validate:"required"`
}

type createTagRequest struct {
	ProjectID int    `json:"project_id" validate:"required,gte=1"`
	Name      string `json:"name" validate:"required,min=1,max=100"`
	ColorID   string `json:"color_id" validate:"max=50"`
}

type updateTagRequest struct {
	Name    string `json:"name" validate:"required,min=1,max=100"`
	ColorID string `json:"color_id" validate:"max=50"`
}

type setTaskTagsRequest struct {
	ProjectID int      `json:"project_id" validate:"required,gte=1"`
	Tags      []string `json:"tags" validate:"required"`
}

type createSubtaskRequest struct {
	Title         string `json:"title" validate:"required,min=1,max=200"`
	UserID        int    `json:"user_id" validate:"gte=0"`
	TimeEstimated int    `json:"time_estimated" validate:"gte=0"`
	TimeSpent     int    `json:"time_spent" validate:"gte=0"`
	Status        int    `json:"status" validate:"gte=0"`
}

type updateSubtaskRequest struct {
	Title         string `json:"title" validate:"omitempty,min=1,max=200"`
	UserID        int    `json:"user_id" validate:"gte=-1"`
	TimeEstimated int    `json:"time_estimated" validate:"gte=-1"`
	TimeSpent     int    `json:"time_spent" validate:"gte=-1"`
	Status        int    `json:"status" validate:"gte=-1"`
}

var kanbanWebserviceRules = []webservice.Rule{
	webservice.Get("/", listTasks),
	webservice.Get("/search", searchTasks),
	webservice.Get("/:id", getTask),
	webservice.Post("/", createTask),
	webservice.Patch("/:id", updateTask),
	webservice.Delete("/:id", deleteTask),
	webservice.Post("/:id/move", moveTask),
	webservice.Get("/columns", listColumns),
	webservice.Get("/:id/metadata", getTaskMetadata),
	webservice.Get("/:id/metadata/:name", getTaskMetadataByName),
	webservice.Post("/:id/metadata", saveTaskMetadata),
	webservice.Delete("/:id/metadata/:name", removeTaskMetadata),
	webservice.Get("/tags", getAllTags),
	webservice.Get("/tags/project", getTagsByProject),
	webservice.Post("/tags", createTag),
	webservice.Patch("/tags/:id", updateTag),
	webservice.Delete("/tags/:id", removeTag),
	webservice.Get("/:id/tags", getTaskTags),
	webservice.Post("/:id/tags", setTaskTags),
	webservice.Get("/:id/subtasks", listSubtasks),
	webservice.Get("/:id/subtasks/:subtaskId", getSubtask),
	webservice.Post("/:id/subtasks", createSubtask),
	webservice.Patch("/:id/subtasks/:subtaskId", updateSubtask),
	webservice.Delete("/:id/subtasks/:subtaskId", deleteSubtask),
	webservice.Get("/:id/subtasks/:subtaskId/timer", hasSubtaskTimer),
	webservice.Post("/:id/subtasks/:subtaskId/timer/start", setSubtaskStartTime),
	webservice.Post("/:id/subtasks/:subtaskId/timer/stop", setSubtaskEndTime),
	webservice.Get("/:id/subtasks/:subtaskId/timer/spent", getSubtaskTimeSpent),
}

func listTasks(ctx fiber.Ctx) error {
	params := map[string]any{}
	if v := ctx.Query("project_id"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			params["project_id"] = n
		}
	}
	if v := ctx.Query("status_id"); v == "0" {
		params["status"] = "inactive"
	}

	res, err := ability.Invoke(ctx.Context(), hub.CapKanban, ability.OpKanbanListTasks, params)
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(res.Data))
}

func getTask(ctx fiber.Ctx) error {
	idStr := ctx.Params("id")
	if idStr == "" {
		return protocol.ErrBadParam.New("id is required")
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return protocol.ErrBadParam.New("invalid task ID")
	}

	res, err := ability.Invoke(ctx.Context(), hub.CapKanban, ability.OpKanbanGetTask, map[string]any{"id": id})
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(res.Data))
}

func createTask(ctx fiber.Ctx) error {
	var body createTaskRequest
	if err := ctx.Bind().Body(&body); err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}
	if err := validate.Validate.Struct(body); err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}

	res, err := ability.Invoke(ctx.Context(), hub.CapKanban, ability.OpKanbanCreateTask, map[string]any{
		"title":       body.Title,
		"description": body.Description,
		"project_id":  body.ProjectID,
		"column_id":   body.ColumnID,
	})
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(res.Data))
}

func updateTask(ctx fiber.Ctx) error {
	idStr := ctx.Params("id")
	if idStr == "" {
		return protocol.ErrBadParam.New("id is required")
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return protocol.ErrBadParam.New("invalid task ID")
	}

	var body updateTaskRequest
	if err := ctx.Bind().Body(&body); err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}
	if err := validate.Validate.Struct(body); err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}

	res, err := ability.Invoke(ctx.Context(), hub.CapKanban, ability.OpKanbanUpdateTask, map[string]any{
		"id":          id,
		"title":       body.Title,
		"description": body.Description,
	})
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(res.Data))
}

func deleteTask(ctx fiber.Ctx) error {
	idStr := ctx.Params("id")
	if idStr == "" {
		return protocol.ErrBadParam.New("id is required")
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return protocol.ErrBadParam.New("invalid task ID")
	}

	_, err = ability.Invoke(ctx.Context(), hub.CapKanban, ability.OpKanbanDeleteTask, map[string]any{"id": id})
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(map[string]bool{"success": true}))
}

func moveTask(ctx fiber.Ctx) error {
	idStr := ctx.Params("id")
	if idStr == "" {
		return protocol.ErrBadParam.New("id is required")
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return protocol.ErrBadParam.New("invalid task ID")
	}

	var body moveTaskRequest
	if err := ctx.Bind().Body(&body); err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}
	if err := validate.Validate.Struct(body); err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}

	res, err := ability.Invoke(ctx.Context(), hub.CapKanban, ability.OpKanbanMoveTask, map[string]any{
		"id":          id,
		"column_id":   body.ColumnID,
		"position":    body.Position,
		"swimlane_id": body.SwimlaneID,
		"project_id":  body.ProjectID,
	})
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(res.Data))
}

func listColumns(ctx fiber.Ctx) error {
	projectID := 1
	if v := ctx.Query("project_id"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			projectID = n
		}
	}

	res, err := ability.Invoke(ctx.Context(), hub.CapKanban, ability.OpKanbanGetColumns, map[string]any{
		"project_id": projectID,
	})
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(res.Data))
}

func searchTasks(ctx fiber.Ctx) error {
	params := map[string]any{}
	if v := ctx.Query("project_id"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			params["project_id"] = n
		}
	}
	if v := ctx.Query("q"); v != "" {
		params["q"] = v
	}

	res, err := ability.Invoke(ctx.Context(), hub.CapKanban, ability.OpKanbanSearchTasks, params)
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(res.Data))
}

func getTaskMetadata(_ fiber.Ctx) error {
	return types.Errorf(types.ErrNotImplemented, "metadata operations not yet migrated to ability layer")
}

func getTaskMetadataByName(_ fiber.Ctx) error {
	return types.Errorf(types.ErrNotImplemented, "metadata operations not yet migrated to ability layer")
}

func saveTaskMetadata(_ fiber.Ctx) error {
	return types.Errorf(types.ErrNotImplemented, "metadata operations not yet migrated to ability layer")
}

func removeTaskMetadata(_ fiber.Ctx) error {
	return types.Errorf(types.ErrNotImplemented, "metadata operations not yet migrated to ability layer")
}

func getAllTags(ctx fiber.Ctx) error {
	client, err := kanboard.GetClient()
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}
	tags, err := client.GetAllTags(ctx.Context())
	if err != nil {
		return fmt.Errorf("failed to get tags: %w", err)
	}
	return ctx.JSON(protocol.NewSuccessResponse(tags))
}

func getTagsByProject(ctx fiber.Ctx) error {
	client, err := kanboard.GetClient()
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}
	projectID := kanboard.DefaultProjectId
	if v := ctx.Query("project_id"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			projectID = n
		}
	}
	tags, err := client.GetTagsByProject(ctx.Context(), projectID)
	if err != nil {
		return fmt.Errorf("failed to get tags: %w", err)
	}
	return ctx.JSON(protocol.NewSuccessResponse(tags))
}

func createTag(ctx fiber.Ctx) error {
	var body createTagRequest
	if err := ctx.Bind().Body(&body); err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}
	if err := validate.Validate.Struct(body); err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}
	client, err := kanboard.GetClient()
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}
	tagID, err := client.CreateTag(ctx.Context(), body.ProjectID, body.Name, body.ColorID)
	if err != nil {
		return fmt.Errorf("failed to create tag: %w", err)
	}
	return ctx.JSON(protocol.NewSuccessResponse(map[string]int64{"id": tagID}))
}

func updateTag(ctx fiber.Ctx) error {
	idStr := ctx.Params("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return protocol.ErrBadParam.New("invalid tag ID")
	}
	var body updateTagRequest
	if err := ctx.Bind().Body(&body); err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}
	if err := validate.Validate.Struct(body); err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}
	client, err := kanboard.GetClient()
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}
	_, err = client.UpdateTag(ctx.Context(), id, body.Name, body.ColorID)
	if err != nil {
		return fmt.Errorf("failed to update tag: %w", err)
	}
	return ctx.JSON(protocol.NewSuccessResponse(map[string]bool{"success": true}))
}

func removeTag(ctx fiber.Ctx) error {
	idStr := ctx.Params("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return protocol.ErrBadParam.New("invalid tag ID")
	}
	client, err := kanboard.GetClient()
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}
	_, err = client.RemoveTag(ctx.Context(), id)
	if err != nil {
		return fmt.Errorf("failed to remove tag: %w", err)
	}
	return ctx.JSON(protocol.NewSuccessResponse(map[string]bool{"success": true}))
}

func getTaskTags(ctx fiber.Ctx) error {
	idStr := ctx.Params("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return protocol.ErrBadParam.New("invalid task ID")
	}
	client, err := kanboard.GetClient()
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}
	tags, err := client.GetTaskTags(ctx.Context(), id)
	if err != nil {
		return fmt.Errorf("failed to get task tags: %w", err)
	}
	return ctx.JSON(protocol.NewSuccessResponse(tags))
}

func setTaskTags(ctx fiber.Ctx) error {
	idStr := ctx.Params("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return protocol.ErrBadParam.New("invalid task ID")
	}
	var body setTaskTagsRequest
	if err := ctx.Bind().Body(&body); err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}
	if err := validate.Validate.Struct(body); err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}
	client, err := kanboard.GetClient()
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}
	resp, err := client.SetTaskTags(ctx.Context(), body.ProjectID, id, body.Tags)
	if err != nil {
		return fmt.Errorf("failed to set task tags: %w", err)
	}
	return ctx.JSON(protocol.NewSuccessResponse(resp))
}

func listSubtasks(ctx fiber.Ctx) error {
	idStr := ctx.Params("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return protocol.ErrBadParam.New("invalid task ID")
	}
	client, err := kanboard.GetClient()
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}
	subtasks, err := client.GetAllSubtasks(ctx.Context(), id)
	if err != nil {
		return fmt.Errorf("failed to get subtasks: %w", err)
	}
	return ctx.JSON(protocol.NewSuccessResponse(subtasks))
}

func getSubtask(ctx fiber.Ctx) error {
	subtaskID, err := strconv.Atoi(ctx.Params("subtaskId"))
	if err != nil {
		return protocol.ErrBadParam.New("invalid subtask ID")
	}
	client, err := kanboard.GetClient()
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}
	subtask, err := client.GetSubtask(ctx.Context(), subtaskID)
	if err != nil {
		return fmt.Errorf("failed to get subtask: %w", err)
	}
	return ctx.JSON(protocol.NewSuccessResponse(subtask))
}

func createSubtask(ctx fiber.Ctx) error {
	idStr := ctx.Params("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return protocol.ErrBadParam.New("invalid task ID")
	}
	var body createSubtaskRequest
	if err := ctx.Bind().Body(&body); err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}
	if err := validate.Validate.Struct(body); err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}
	client, err := kanboard.GetClient()
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}
	subtaskID, err := client.CreateSubtask(ctx.Context(), id, body.Title, body.UserID, body.TimeEstimated, body.TimeSpent, body.Status)
	if err != nil {
		return fmt.Errorf("failed to create subtask: %w", err)
	}
	return ctx.JSON(protocol.NewSuccessResponse(map[string]int64{"id": subtaskID}))
}

func updateSubtask(ctx fiber.Ctx) error {
	idStr := ctx.Params("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return protocol.ErrBadParam.New("invalid task ID")
	}
	subtaskID, err := strconv.Atoi(ctx.Params("subtaskId"))
	if err != nil {
		return protocol.ErrBadParam.New("invalid subtask ID")
	}
	var body updateSubtaskRequest
	if err := ctx.Bind().Body(&body); err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}
	if err := validate.Validate.Struct(body); err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}
	client, err := kanboard.GetClient()
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}
	_, err = client.UpdateSubtask(ctx.Context(), subtaskID, id, body.Title, body.UserID, body.TimeEstimated, body.TimeSpent, body.Status)
	if err != nil {
		return fmt.Errorf("failed to update subtask: %w", err)
	}
	return ctx.JSON(protocol.NewSuccessResponse(map[string]bool{"success": true}))
}

func deleteSubtask(ctx fiber.Ctx) error {
	subtaskID, err := strconv.Atoi(ctx.Params("subtaskId"))
	if err != nil {
		return protocol.ErrBadParam.New("invalid subtask ID")
	}
	client, err := kanboard.GetClient()
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}
	_, err = client.RemoveSubtask(ctx.Context(), subtaskID)
	if err != nil {
		return fmt.Errorf("failed to delete subtask: %w", err)
	}
	return ctx.JSON(protocol.NewSuccessResponse(map[string]bool{"success": true}))
}

func hasSubtaskTimer(ctx fiber.Ctx) error {
	subtaskID, err := strconv.Atoi(ctx.Params("subtaskId"))
	if err != nil {
		return protocol.ErrBadParam.New("invalid subtask ID")
	}
	client, err := kanboard.GetClient()
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}
	has, err := client.HasSubtaskTimer(ctx.Context(), subtaskID, 0)
	if err != nil {
		return fmt.Errorf("failed to check subtask timer: %w", err)
	}
	return ctx.JSON(protocol.NewSuccessResponse(map[string]bool{"has_timer": has}))
}

func setSubtaskStartTime(ctx fiber.Ctx) error {
	subtaskID, err := strconv.Atoi(ctx.Params("subtaskId"))
	if err != nil {
		return protocol.ErrBadParam.New("invalid subtask ID")
	}
	client, err := kanboard.GetClient()
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}
	_, err = client.SetSubtaskStartTime(ctx.Context(), subtaskID, 0)
	if err != nil {
		return fmt.Errorf("failed to set subtask start time: %w", err)
	}
	return ctx.JSON(protocol.NewSuccessResponse(map[string]bool{"success": true}))
}

func setSubtaskEndTime(ctx fiber.Ctx) error {
	subtaskID, err := strconv.Atoi(ctx.Params("subtaskId"))
	if err != nil {
		return protocol.ErrBadParam.New("invalid subtask ID")
	}
	client, err := kanboard.GetClient()
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}
	_, err = client.SetSubtaskEndTime(ctx.Context(), subtaskID, 0)
	if err != nil {
		return fmt.Errorf("failed to set subtask end time: %w", err)
	}
	return ctx.JSON(protocol.NewSuccessResponse(map[string]bool{"success": true}))
}

func getSubtaskTimeSpent(ctx fiber.Ctx) error {
	subtaskID, err := strconv.Atoi(ctx.Params("subtaskId"))
	if err != nil {
		return protocol.ErrBadParam.New("invalid subtask ID")
	}
	client, err := kanboard.GetClient()
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}
	spent, err := client.GetSubtaskTimeSpent(ctx.Context(), subtaskID, 0)
	if err != nil {
		return fmt.Errorf("failed to get subtask time spent: %w", err)
	}
	return ctx.JSON(protocol.NewSuccessResponse(map[string]float64{"time_spent": spent}))
}

// --- Note routes (registered under /service/note) ---

var noteWebserviceRules = []webservice.Rule{
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

func getNote(ctx fiber.Ctx) error {
	return invokeNote(ctx, ability.OpNoteGet, map[string]any{
		"id": ctx.Params("id"),
	})
}

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

func deleteNote(ctx fiber.Ctx) error {
	return invokeNote(ctx, ability.OpNoteDelete, map[string]any{
		"id": ctx.Params("id"),
	})
}

func getNoteContent(ctx fiber.Ctx) error {
	return invokeNote(ctx, ability.OpNoteGetContent, map[string]any{
		"id": ctx.Params("id"),
	})
}

func setNoteContent(ctx fiber.Ctx) error {
	content := string(ctx.Body())
	return invokeNote(ctx, ability.OpNoteSetContent, map[string]any{
		"id":      ctx.Params("id"),
		"content": content,
	})
}

func searchNotes(ctx fiber.Ctx) error {
	query := ctx.Query("q")
	if query == "" {
		return types.Errorf(types.ErrInvalidArgument, "query parameter 'q' is required")
	}
	return invokeNote(ctx, ability.OpNoteSearch, map[string]any{
		"query": query,
	})
}

func noteHealth(ctx fiber.Ctx) error {
	return invokeNote(ctx, ability.OpNoteGetAppInfo, map[string]any{})
}

func invokeNote(ctx fiber.Ctx, operation string, params map[string]any) error {
	res, err := ability.Invoke(context.Background(), hub.CapNote, operation, params)
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(res))
}

// --- Reader routes (registered under /service/reader) ---

type createFeedRequest struct {
	FeedURL    string `json:"feed_url" validate:"required,url,max=2048"`
	CategoryID int64  `json:"category_id"`
}

type updateEntriesRequest struct {
	EntryIDs []int64 `json:"entry_ids" validate:"required,min=1,max=1000"`
	Status   string  `json:"status" validate:"required,oneof=read unread removed"`
}

var readerWebserviceRules = []webservice.Rule{
	webservice.Get("/", listFeeds),
	webservice.Post("/", createFeed),
	webservice.Get("/entries", listEntries),
	webservice.Patch("/entries", updateEntriesStatus),
}

// webserviceRules merges all sub-module webservice rule sets into a single
// slice for registration via Rules().
var webserviceRules = append(append(append(append(
	hubWebserviceRules,
	bookmarkWebserviceRules...),
	kanbanWebserviceRules...),
	noteWebserviceRules...),
	readerWebserviceRules...)

func listFeeds(ctx fiber.Ctx) error {
	res, err := ability.Invoke(ctx.Context(), hub.CapReader, ability.OpReaderListFeeds, nil)
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(res.Data))
}

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
