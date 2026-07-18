// Package hub implements webservice routes for hub management, bookmark,
// kanban, note, and reader capabilities.
package hub

import (
	"context"
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/capability"
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

// --- Karakeep routes (registered under /service/karakeep) ---

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
	return invokeBookmark(ctx, capability.OpBookmarkList, params)
}

func checkURLExists(ctx fiber.Ctx) error {
	url := ctx.Query("url")
	if url == "" {
		return types.Errorf(types.ErrInvalidArgument, "url is required")
	}
	return invokeBookmark(ctx, capability.OpBookmarkCheckURL, map[string]any{"url": url})
}

func searchBookmarks(ctx fiber.Ctx) error {
	params := pageParams(ctx)
	if v := ctx.Query("q"); v != "" {
		params["q"] = v
	}
	if v := ctx.Query("sort_order"); v != "" {
		params["sort_order"] = v
	}
	return invokeBookmark(ctx, capability.OpBookmarkSearch, params)
}

func getBookmark(ctx fiber.Ctx) error {
	id := ctx.Params("id")
	if id == "" {
		return types.Errorf(types.ErrInvalidArgument, "id is required")
	}
	return invokeBookmark(ctx, capability.OpBookmarkGet, map[string]any{"id": id})
}

func createBookmark(ctx fiber.Ctx) error {
	var body createBookmarkRequest
	if err := ctx.Bind().Body(&body); err != nil {
		return types.WrapError(types.ErrInvalidArgument, "decode create bookmark request", err)
	}
	if err := validate.Validate.Struct(body); err != nil {
		return types.WrapError(types.ErrInvalidArgument, "validate create bookmark request", err)
	}
	return invokeBookmark(ctx, capability.OpBookmarkCreate, map[string]any{"url": body.URL})
}

func archiveBookmark(ctx fiber.Ctx) error {
	id := ctx.Params("id")
	if id == "" {
		return types.Errorf(types.ErrInvalidArgument, "id is required")
	}
	return invokeBookmark(ctx, capability.OpBookmarkArchive, map[string]any{"id": id})
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
	return invokeBookmark(ctx, capability.OpBookmarkAttachTags, map[string]any{"id": id, "tags": body.Tags})
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
	return invokeBookmark(ctx, capability.OpBookmarkDetachTags, map[string]any{"id": id, "tags": body.Tags})
}

func invokeBookmark(ctx fiber.Ctx, operation string, params map[string]any) error {
	res, err := capability.Invoke(context.Background(), hub.CapKarakeep, operation, params)
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

// --- Kanboard routes (registered under /service/kanboard) ---

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

	res, err := capability.Invoke(ctx.Context(), hub.CapKanboard, capability.OpKanbanListTasks, params)
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

	res, err := capability.Invoke(ctx.Context(), hub.CapKanboard, capability.OpKanbanGetTask, map[string]any{"id": id})
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

	res, err := capability.Invoke(ctx.Context(), hub.CapKanboard, capability.OpKanbanCreateTask, map[string]any{
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

	res, err := capability.Invoke(ctx.Context(), hub.CapKanboard, capability.OpKanbanUpdateTask, map[string]any{
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

	_, err = capability.Invoke(ctx.Context(), hub.CapKanboard, capability.OpKanbanDeleteTask, map[string]any{"id": id})
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

	res, err := capability.Invoke(ctx.Context(), hub.CapKanboard, capability.OpKanbanMoveTask, map[string]any{
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

	res, err := capability.Invoke(ctx.Context(), hub.CapKanboard, capability.OpKanbanGetColumns, map[string]any{
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

	res, err := capability.Invoke(ctx.Context(), hub.CapKanboard, capability.OpKanbanSearchTasks, params)
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

// --- Trilium routes (registered under /service/trilium) ---

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
	return invokeNote(ctx, capability.OpNoteList, params)
}

func getNote(ctx fiber.Ctx) error {
	return invokeNote(ctx, capability.OpNoteGet, map[string]any{
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
	return invokeNote(ctx, capability.OpNoteCreate, map[string]any{
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
	return invokeNote(ctx, capability.OpNoteUpdate, map[string]any{
		"id":      ctx.Params("id"),
		"title":   body.Title,
		"content": body.Content,
	})
}

func deleteNote(ctx fiber.Ctx) error {
	return invokeNote(ctx, capability.OpNoteDelete, map[string]any{
		"id": ctx.Params("id"),
	})
}

func getNoteContent(ctx fiber.Ctx) error {
	return invokeNote(ctx, capability.OpNoteGetContent, map[string]any{
		"id": ctx.Params("id"),
	})
}

func setNoteContent(ctx fiber.Ctx) error {
	content := string(ctx.Body())
	return invokeNote(ctx, capability.OpNoteSetContent, map[string]any{
		"id":      ctx.Params("id"),
		"content": content,
	})
}

func searchNotes(ctx fiber.Ctx) error {
	query := ctx.Query("q")
	if query == "" {
		return types.Errorf(types.ErrInvalidArgument, "query parameter 'q' is required")
	}
	return invokeNote(ctx, capability.OpNoteSearch, map[string]any{
		"query": query,
	})
}

func noteHealth(ctx fiber.Ctx) error {
	return invokeNote(ctx, capability.OpNoteGetAppInfo, map[string]any{})
}

func invokeNote(ctx fiber.Ctx, operation string, params map[string]any) error {
	res, err := capability.Invoke(context.Background(), hub.CapTrilium, operation, params)
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(res))
}

// --- Memos routes (registered under /service/memos) ---

var memoWebserviceRules = []webservice.Rule{
	webservice.Get("/", listMemos),
	webservice.Get("/health", memoHealth),
	webservice.Post("/", createMemo),
	webservice.Patch("/", updateMemo),
	webservice.Delete("/", deleteMemo),
}

func listMemos(ctx fiber.Ctx) error {
	if name := ctx.Query("name"); name != "" {
		return invokeMemo(ctx, capability.OpMemoGet, map[string]any{"name": name})
	}
	params := pageParams(ctx)
	return invokeMemo(ctx, capability.OpMemoList, params)
}

func createMemo(ctx fiber.Ctx) error {
	var body struct {
		Content    string `json:"content"`
		Visibility string `json:"visibility"`
	}
	if err := ctx.Bind().Body(&body); err != nil {
		return types.WrapError(types.ErrInvalidArgument, "decode memo request", err)
	}
	if body.Content == "" {
		return types.Errorf(types.ErrInvalidArgument, "content is required")
	}
	return invokeMemo(ctx, capability.OpMemoCreate, map[string]any{
		"content":    body.Content,
		"visibility": body.Visibility,
	})
}

func updateMemo(ctx fiber.Ctx) error {
	name := ctx.Query("name")
	if name == "" {
		return types.Errorf(types.ErrInvalidArgument, "name query param is required")
	}
	var body struct {
		Content    string `json:"content"`
		Visibility string `json:"visibility"`
		Pinned     *bool  `json:"pinned"`
	}
	if err := ctx.Bind().Body(&body); err != nil {
		return types.WrapError(types.ErrInvalidArgument, "decode memo update request", err)
	}
	params := map[string]any{"name": name}
	if body.Content != "" {
		params["content"] = body.Content
	}
	if body.Visibility != "" {
		params["visibility"] = body.Visibility
	}
	if body.Pinned != nil {
		params["pinned"] = *body.Pinned
	}
	return invokeMemo(ctx, capability.OpMemoUpdate, params)
}

func deleteMemo(ctx fiber.Ctx) error {
	name := ctx.Query("name")
	if name == "" {
		return types.Errorf(types.ErrInvalidArgument, "name query param is required")
	}
	return invokeMemo(ctx, capability.OpMemoDelete, map[string]any{"name": name})
}

func memoHealth(ctx fiber.Ctx) error {
	return invokeMemo(ctx, capability.OpMemoHealth, map[string]any{})
}

func invokeMemo(ctx fiber.Ctx, operation string, params map[string]any) error {
	res, err := capability.Invoke(context.Background(), hub.CapMemos, operation, params)
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(res))
}

// --- Miniflux routes (registered under /service/miniflux) ---

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

// --- Gitea routes (registered under /service/gitea) ---

var forgeWebserviceRules = []webservice.Rule{
	webservice.Get("/user", forgeGetUser),
	webservice.Get("/repo", forgeGetRepo),
	webservice.Get("/issues", forgeListIssues),
	webservice.Get("/issue", forgeGetIssue),
	webservice.Get("/commit-diff", forgeGetCommitDiff),
	webservice.Get("/file-content", forgeGetFileContent),
}

func forgeGetUser(ctx fiber.Ctx) error {
	res, err := capability.Invoke(ctx.Context(), hub.CapGitea, capability.OpForgeGetUser, nil)
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(res.Data))
}

func forgeGetRepo(ctx fiber.Ctx) error {
	owner := ctx.Query("owner")
	repo := ctx.Query("repo")
	if owner == "" || repo == "" {
		return types.Errorf(types.ErrInvalidArgument, "owner and repo query params are required")
	}

	res, err := capability.Invoke(ctx.Context(), hub.CapGitea, capability.OpForgeGetRepo, map[string]any{
		"owner": owner,
		"repo":  repo,
	})
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(res.Data))
}

func forgeListIssues(ctx fiber.Ctx) error {
	owner := ctx.Query("owner")
	if owner == "" {
		return types.Errorf(types.ErrInvalidArgument, "owner query param is required")
	}

	params := map[string]any{"owner": owner}
	if v := ctx.Query("state"); v != "" {
		params["state"] = v
	}
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

	res, err := capability.Invoke(ctx.Context(), hub.CapGitea, capability.OpForgeListIssues, params)
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(res.Data))
}

func forgeGetIssue(ctx fiber.Ctx) error {
	owner := ctx.Query("owner")
	repo := ctx.Query("repo")
	indexStr := ctx.Query("index")
	if owner == "" || repo == "" || indexStr == "" {
		return types.Errorf(types.ErrInvalidArgument, "owner, repo and index query params are required")
	}
	index, err := strconv.ParseInt(indexStr, 10, 64)
	if err != nil {
		return types.Errorf(types.ErrInvalidArgument, "invalid index: %v", err)
	}

	res, err := capability.Invoke(ctx.Context(), hub.CapGitea, capability.OpForgeGetIssue, map[string]any{
		"owner": owner,
		"repo":  repo,
		"index": index,
	})
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(res.Data))
}

func forgeGetCommitDiff(ctx fiber.Ctx) error {
	owner := ctx.Query("owner")
	repo := ctx.Query("repo")
	commitID := ctx.Query("commit_id")
	if owner == "" || repo == "" || commitID == "" {
		return types.Errorf(types.ErrInvalidArgument, "owner, repo and commit_id query params are required")
	}

	res, err := capability.Invoke(ctx.Context(), hub.CapGitea, capability.OpForgeGetCommitDiff, map[string]any{
		"owner":     owner,
		"repo":      repo,
		"commit_id": commitID,
	})
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(res.Data))
}

func forgeGetFileContent(ctx fiber.Ctx) error {
	owner := ctx.Query("owner")
	repo := ctx.Query("repo")
	commitID := ctx.Query("commit_id")
	filePath := ctx.Query("file_path")
	if owner == "" || repo == "" || commitID == "" || filePath == "" {
		return types.Errorf(types.ErrInvalidArgument, "owner, repo, commit_id and file_path query params are required")
	}

	params := map[string]any{
		"owner":     owner,
		"repo":      repo,
		"commit_id": commitID,
		"file_path": filePath,
	}
	if v := ctx.Query("line_start"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			params["line_start"] = n
		}
	}
	if v := ctx.Query("line_count"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			params["line_count"] = n
		}
	}

	res, err := capability.Invoke(ctx.Context(), hub.CapGitea, capability.OpForgeGetFileContent, params)
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(res.Data))
}

// --- GitHub routes (registered under /service/github) ---

var githubWebserviceRules = []webservice.Rule{
	webservice.Get("/user", githubGetUser),
	webservice.Get("/user/:login", githubGetUserByLogin),
	webservice.Get("/repo", githubGetRepo),
	webservice.Get("/issues", githubListIssues),
	webservice.Get("/issue", githubGetIssue),
	webservice.Get("/commit-diff", githubGetCommitDiff),
	webservice.Get("/file-content", githubGetFileContent),
	webservice.Get("/notifications", githubListNotifications),
	webservice.Get("/releases", githubListReleases),
}

func githubGetUser(ctx fiber.Ctx) error {
	res, err := capability.Invoke(ctx.Context(), hub.CapGithub, capability.OpGithubGetUser, nil)
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(res.Data))
}

func githubGetUserByLogin(ctx fiber.Ctx) error {
	login := ctx.Params("login")
	if login == "" {
		return types.Errorf(types.ErrInvalidArgument, "login path param is required")
	}

	res, err := capability.Invoke(ctx.Context(), hub.CapGithub, capability.OpGithubGetUserByLogin, map[string]any{
		"login": login,
	})
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(res.Data))
}

func githubGetRepo(ctx fiber.Ctx) error {
	owner := ctx.Query("owner")
	repo := ctx.Query("repo")
	if owner == "" || repo == "" {
		return types.Errorf(types.ErrInvalidArgument, "owner and repo query params are required")
	}

	res, err := capability.Invoke(ctx.Context(), hub.CapGithub, capability.OpGithubGetRepo, map[string]any{
		"owner": owner,
		"repo":  repo,
	})
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(res.Data))
}

func githubListIssues(ctx fiber.Ctx) error {
	owner := ctx.Query("owner")
	if owner == "" {
		return types.Errorf(types.ErrInvalidArgument, "owner query param is required")
	}

	params := map[string]any{"owner": owner}
	if v := ctx.Query("state"); v != "" {
		params["state"] = v
	}
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

	res, err := capability.Invoke(ctx.Context(), hub.CapGithub, capability.OpGithubListIssues, params)
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(res.Data))
}

func githubGetIssue(ctx fiber.Ctx) error {
	owner := ctx.Query("owner")
	repo := ctx.Query("repo")
	numberStr := ctx.Query("number")
	if owner == "" || repo == "" || numberStr == "" {
		return types.Errorf(types.ErrInvalidArgument, "owner, repo and number query params are required")
	}
	number, err := strconv.ParseInt(numberStr, 10, 64)
	if err != nil {
		return types.Errorf(types.ErrInvalidArgument, "invalid number: %v", err)
	}

	res, err := capability.Invoke(ctx.Context(), hub.CapGithub, capability.OpGithubGetIssue, map[string]any{
		"owner":  owner,
		"repo":   repo,
		"number": number,
	})
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(res.Data))
}

func githubGetCommitDiff(ctx fiber.Ctx) error {
	owner := ctx.Query("owner")
	repo := ctx.Query("repo")
	commitID := ctx.Query("commit_id")
	if owner == "" || repo == "" || commitID == "" {
		return types.Errorf(types.ErrInvalidArgument, "owner, repo and commit_id query params are required")
	}

	res, err := capability.Invoke(ctx.Context(), hub.CapGithub, capability.OpGithubGetCommitDiff, map[string]any{
		"owner":     owner,
		"repo":      repo,
		"commit_id": commitID,
	})
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(res.Data))
}

func githubGetFileContent(ctx fiber.Ctx) error {
	owner := ctx.Query("owner")
	repo := ctx.Query("repo")
	commitID := ctx.Query("commit_id")
	filePath := ctx.Query("file_path")
	if owner == "" || repo == "" || commitID == "" || filePath == "" {
		return types.Errorf(types.ErrInvalidArgument, "owner, repo, commit_id and file_path query params are required")
	}

	params := map[string]any{
		"owner":     owner,
		"repo":      repo,
		"commit_id": commitID,
		"file_path": filePath,
	}
	if v := ctx.Query("line_start"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			params["line_start"] = n
		}
	}
	if v := ctx.Query("line_count"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			params["line_count"] = n
		}
	}

	res, err := capability.Invoke(ctx.Context(), hub.CapGithub, capability.OpGithubGetFileContent, params)
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(res.Data))
}

func githubListNotifications(ctx fiber.Ctx) error {
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

	res, err := capability.Invoke(ctx.Context(), hub.CapGithub, capability.OpGithubListNotifications, params)
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(res.Data))
}

func githubListReleases(ctx fiber.Ctx) error {
	owner := ctx.Query("owner")
	repo := ctx.Query("repo")
	if owner == "" || repo == "" {
		return types.Errorf(types.ErrInvalidArgument, "owner and repo query params are required")
	}

	params := map[string]any{
		"owner": owner,
		"repo":  repo,
	}
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

	res, err := capability.Invoke(ctx.Context(), hub.CapGithub, capability.OpGithubListReleases, params)
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(res.Data))
}

// webserviceRules merges all sub-module webservice rule sets into a single
// slice for registration via Rules().
var webserviceRules = append(append(append(append(append(append(append(
	hubWebserviceRules,
	bookmarkWebserviceRules...),
	kanbanWebserviceRules...),
	noteWebserviceRules...),
	readerWebserviceRules...),
	forgeWebserviceRules...),
	githubWebserviceRules...),
	memoWebserviceRules...)

func listFeeds(ctx fiber.Ctx) error {
	res, err := capability.Invoke(ctx.Context(), hub.CapMiniflux, capability.OpReaderListFeeds, nil)
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

	res, err := capability.Invoke(ctx.Context(), hub.CapMiniflux, capability.OpReaderCreateFeed, map[string]any{
		"feed_url": body.FeedURL,
	})
	if err != nil {
		return err
	}

	return ctx.JSON(protocol.NewSuccessResponse(res.Data))
}

func listEntries(ctx fiber.Ctx) error {
	params := pageParams(ctx)
	if v := ctx.Query("status"); v != "" {
		params["status"] = v
	}
	if v := ctx.Query("feed_id"); v != "" {
		params["feed_id"] = parseQueryInt(v)
	}

	res, err := capability.Invoke(ctx.Context(), hub.CapMiniflux, capability.OpReaderListEntries, params)
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
			operation = capability.OpReaderMarkEntryRead
		default:
			operation = capability.OpReaderMarkEntryUnread
		}
		_, err := capability.Invoke(ctx.Context(), hub.CapMiniflux, operation, map[string]any{
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
