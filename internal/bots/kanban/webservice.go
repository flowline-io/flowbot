package kanban

import (
	"fmt"
	"strconv"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/providers/kanboard"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/validate"
	"github.com/gofiber/fiber/v3"
)

var webserviceRules = []webservice.Rule{
	webservice.Get("/", listTasks),
	webservice.Get("/search", searchTasks),
	webservice.Get("/:id", getTask),
	webservice.Post("/", createTask),
	webservice.Patch("/:id", updateTask),
	webservice.Delete("/:id", deleteTask),
	webservice.Post("/:id/move", moveTask),
	webservice.Get("/columns", listColumns),
	// metadata, tags, subtasks routes use direct provider calls (specialized operations)
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

// list tasks
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

	res, err := ability.Invoke(ctx.Context(), hub.CapKanban, "list_tasks", params)
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(res.Data))
}

// get single task
func getTask(ctx fiber.Ctx) error {
	idStr := ctx.Params("id")
	if idStr == "" {
		return protocol.ErrBadParam.New("id is required")
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return protocol.ErrBadParam.New("invalid task ID")
	}

	res, err := ability.Invoke(ctx.Context(), hub.CapKanban, "get_task", map[string]any{"id": id})
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(res.Data))
}

// create task
func createTask(ctx fiber.Ctx) error {
	var body createTaskRequest
	if err := ctx.Bind().Body(&body); err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}
	if err := validate.Validate.Struct(body); err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}

	res, err := ability.Invoke(ctx.Context(), hub.CapKanban, "create_task", map[string]any{
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

// update task
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

	res, err := ability.Invoke(ctx.Context(), hub.CapKanban, "update_task", map[string]any{
		"id":          id,
		"title":       body.Title,
		"description": body.Description,
	})
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(res.Data))
}

// delete task
func deleteTask(ctx fiber.Ctx) error {
	idStr := ctx.Params("id")
	if idStr == "" {
		return protocol.ErrBadParam.New("id is required")
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return protocol.ErrBadParam.New("invalid task ID")
	}

	_, err = ability.Invoke(ctx.Context(), hub.CapKanban, "delete_task", map[string]any{"id": id})
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(map[string]bool{"success": true}))
}

// move task
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

	res, err := ability.Invoke(ctx.Context(), hub.CapKanban, "move_task", map[string]any{
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

// list columns
func listColumns(ctx fiber.Ctx) error {
	projectID := 1
	if v := ctx.Query("project_id"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			projectID = n
		}
	}

	res, err := ability.Invoke(ctx.Context(), hub.CapKanban, "get_columns", map[string]any{
		"project_id": projectID,
	})
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(res.Data))
}

// search tasks
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

	res, err := ability.Invoke(ctx.Context(), hub.CapKanban, "search_tasks", params)
	if err != nil {
		return err
	}
	return ctx.JSON(protocol.NewSuccessResponse(res.Data))
}

// The following specialized handlers (metadata, tags, subtasks) use direct provider calls.
// These are deeply tied to kanboard's specific data model.
// TODO: Migrate to ability.Invoke when the ability interface is expanded.

func getTaskMetadata(ctx fiber.Ctx) error {
	return types.Errorf(types.ErrNotImplemented, "metadata operations not yet migrated to ability layer")
}

func getTaskMetadataByName(ctx fiber.Ctx) error {
	return types.Errorf(types.ErrNotImplemented, "metadata operations not yet migrated to ability layer")
}

func saveTaskMetadata(ctx fiber.Ctx) error {
	return types.Errorf(types.ErrNotImplemented, "metadata operations not yet migrated to ability layer")
}

func removeTaskMetadata(ctx fiber.Ctx) error {
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
