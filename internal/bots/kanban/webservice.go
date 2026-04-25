package kanban

import (
	"fmt"
	"strconv"

	"github.com/flowline-io/flowbot/pkg/providers/kanboard"
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
	webservice.Get("/:id/metadata", getTaskMetadata),
	webservice.Get("/:id/metadata/:name", getTaskMetadataByName),
	webservice.Post("/:id/metadata", saveTaskMetadata),
	webservice.Delete("/:id/metadata/:name", removeTaskMetadata),
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

// list tasks
//
//	@Summary	List kanban tasks
//	@Tags		kanban
//	@Accept		json
//	@Produce	json
//	@Param		project_id	query		int	false	"project ID"
//	@Param		status_id	query		int	false	"status ID (1=active, 0=inactive)"
//	@Success	200			{object}	protocol.Response{data=[]kanboard.Task}
//	@Security	ApiKeyAuth
//	@Router		/service/kanban [get]
func listTasks(ctx fiber.Ctx) error {
	client, err := kanboard.GetClient()
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}

	projectId := kanboard.DefaultProjectId
	if v := ctx.Query("project_id"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			projectId = n
		}
	}

	status := kanboard.Active
	if v := ctx.Query("status_id"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			status = kanboard.StatusId(n)
		}
	}

	tasks, err := client.GetAllTasks(ctx.RequestCtx(), projectId, status)
	if err != nil {
		return fmt.Errorf("failed to get tasks: %w", err)
	}

	return ctx.JSON(protocol.NewSuccessResponse(tasks))
}

// get single task
//
//	@Summary	Get task by ID
//	@Tags		kanban
//	@Accept		json
//	@Produce	json
//	@Param		id	path		string	true	"task ID"
//	@Success	200	{object}	protocol.Response{data=kanboard.Task}
//	@Security	ApiKeyAuth
//	@Router		/service/kanban/{id} [get]
func getTask(ctx fiber.Ctx) error {
	idStr := ctx.Params("id")
	if idStr == "" {
		return protocol.ErrBadParam.New("id is required")
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return protocol.ErrBadParam.New("invalid task ID")
	}

	client, err := kanboard.GetClient()
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}

	task, err := client.GetTask(ctx.RequestCtx(), id)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	return ctx.JSON(protocol.NewSuccessResponse(task))
}

// create task
//
//	@Summary	Create a new task
//	@Tags		kanban
//	@Accept		json
//	@Produce	json
//	@Param		body	body		object{title=string,description=string,project_id=int,column_id=int}	true	"task data"
//	@Success	200		{object}	protocol.Response{data=map[string]int64}
//	@Security	ApiKeyAuth
//	@Router		/service/kanban [post]
func createTask(ctx fiber.Ctx) error {
	var body createTaskRequest
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

	projectId := body.ProjectID
	if projectId == 0 {
		projectId = kanboard.DefaultProjectId
	}

	task := &kanboard.Task{
		Title:       body.Title,
		Description: body.Description,
		ProjectID:   projectId,
		Priority:    kanboard.DefaultPriority,
	}
	if body.ColumnID > 0 {
		task.ColumnID = body.ColumnID
	}

	taskId, err := client.CreateTask(ctx.RequestCtx(), task)
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	return ctx.JSON(protocol.NewSuccessResponse(map[string]int64{"id": taskId}))
}

// update task
//
//	@Summary	Update a task
//	@Tags		kanban
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string									true	"task ID"
//	@Param		body	body		object{title=string,description=string}	true	"task data"
//	@Success	200		{object}	protocol.Response{data=map[string]bool}
//	@Security	ApiKeyAuth
//	@Router		/service/kanban/{id} [patch]
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

	client, err := kanboard.GetClient()
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}

	task := &kanboard.Task{
		Title:       body.Title,
		Description: body.Description,
	}

	result, err := client.UpdateTask(ctx.RequestCtx(), id, task)
	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	return ctx.JSON(protocol.NewSuccessResponse(map[string]bool{"success": result}))
}

// delete task (close)
//
//	@Summary	Close a task
//	@Tags		kanban
//	@Accept		json
//	@Produce	json
//	@Param		id	path		string	true	"task ID"
//	@Success	200	{object}	protocol.Response{data=map[string]bool}
//	@Security	ApiKeyAuth
//	@Router		/service/kanban/{id} [delete]
func deleteTask(ctx fiber.Ctx) error {
	idStr := ctx.Params("id")
	if idStr == "" {
		return protocol.ErrBadParam.New("id is required")
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return protocol.ErrBadParam.New("invalid task ID")
	}

	client, err := kanboard.GetClient()
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}

	result, err := client.CloseTask(ctx.RequestCtx(), id)
	if err != nil {
		return fmt.Errorf("failed to close task: %w", err)
	}

	return ctx.JSON(protocol.NewSuccessResponse(map[string]bool{"success": result}))
}

// move task to another column/position
//
//	@Summary	Move task to another column/position
//	@Tags		kanban
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string																true	"task ID"
//	@Param		body	body		object{column_id=int,position=int,swimlane_id=int,project_id=int}	true	"move parameters"
//	@Success	200		{object}	protocol.Response{data=map[string]bool}
//	@Security	ApiKeyAuth
//	@Router		/service/kanban/{id}/move [post]
func moveTask(ctx fiber.Ctx) error {
	idStr := ctx.Params("id")
	if idStr == "" {
		return protocol.ErrBadParam.New("id is required")
	}

	taskId, err := strconv.Atoi(idStr)
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

	client, err := kanboard.GetClient()
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}

	projectId := body.ProjectID
	if projectId == 0 {
		projectId = kanboard.DefaultProjectId
	}

	swimlaneId := body.SwimlaneID
	if swimlaneId == 0 {
		swimlaneId = 1
	}

	result, err := client.MoveTaskPosition(ctx.RequestCtx(), projectId, taskId, body.ColumnID, body.Position, swimlaneId)
	if err != nil {
		return fmt.Errorf("failed to move task: %w", err)
	}

	return ctx.JSON(protocol.NewSuccessResponse(map[string]bool{"success": result}))
}

// list columns
//
//	@Summary	List kanban columns
//	@Tags		kanban
//	@Accept		json
//	@Produce	json
//	@Param		project_id	query		int	false	"project ID"
//	@Success	200			{object}	protocol.Response{data=[]map[string]any}
//	@Security	ApiKeyAuth
//	@Router		/service/kanban/columns [get]
func listColumns(ctx fiber.Ctx) error {
	client, err := kanboard.GetClient()
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}

	projectId := kanboard.DefaultProjectId
	if v := ctx.Query("project_id"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			projectId = n
		}
	}

	columns, err := client.GetColumns(ctx.RequestCtx(), projectId)
	if err != nil {
		return fmt.Errorf("failed to get columns: %w", err)
	}

	return ctx.JSON(protocol.NewSuccessResponse(columns))
}

// search tasks
//
//	@Summary	Search kanban tasks
//	@Tags		kanban
//	@Accept		json
//	@Produce	json
//	@Param		project_id	query		int		true	"project ID"
//	@Param		query		query		string	true	"search query"
//	@Success	200			{object}	protocol.Response{data=[]kanboard.Task}
//	@Security	ApiKeyAuth
//	@Router		/service/kanban/search [get]
func searchTasks(ctx fiber.Ctx) error {
	projectIdStr := ctx.Query("project_id")
	if projectIdStr == "" {
		return protocol.ErrBadParam.New("project_id is required")
	}

	projectId, err := strconv.Atoi(projectIdStr)
	if err != nil {
		return protocol.ErrBadParam.New("invalid project_id")
	}

	query := ctx.Query("query")
	if query == "" {
		return protocol.ErrBadParam.New("query is required")
	}

	client, err := kanboard.GetClient()
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}

	tasks, err := client.SearchTasks(ctx.RequestCtx(), projectId, query)
	if err != nil {
		return fmt.Errorf("failed to search tasks: %w", err)
	}

	return ctx.JSON(protocol.NewSuccessResponse(tasks))
}

// get task metadata
//
//	@Summary	Get all metadata for a task
//	@Tags		kanban
//	@Accept		json
//	@Produce	json
//	@Param		id	path		string	true	"task ID"
//	@Success	200	{object}	protocol.Response{data=[]kanboard.TaskMetadata}
//	@Security	ApiKeyAuth
//	@Router		/service/kanban/{id}/metadata [get]
func getTaskMetadata(ctx fiber.Ctx) error {
	idStr := ctx.Params("id")
	if idStr == "" {
		return protocol.ErrBadParam.New("id is required")
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return protocol.ErrBadParam.New("invalid task ID")
	}

	client, err := kanboard.GetClient()
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}

	metadata, err := client.GetTaskMetadata(ctx.RequestCtx(), id)
	if err != nil {
		return fmt.Errorf("failed to get task metadata: %w", err)
	}

	return ctx.JSON(protocol.NewSuccessResponse(metadata))
}

// get task metadata by name
//
//	@Summary	Get task metadata by name
//	@Tags		kanban
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string	true	"task ID"
//	@Param		name	path		string	true	"metadata name"
//	@Success	200		{object}	protocol.Response{data=string}
//	@Security	ApiKeyAuth
//	@Router		/service/kanban/{id}/metadata/{name} [get]
func getTaskMetadataByName(ctx fiber.Ctx) error {
	idStr := ctx.Params("id")
	if idStr == "" {
		return protocol.ErrBadParam.New("id is required")
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return protocol.ErrBadParam.New("invalid task ID")
	}

	name := ctx.Params("name")
	if name == "" {
		return protocol.ErrBadParam.New("name is required")
	}

	client, err := kanboard.GetClient()
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}

	value, err := client.GetTaskMetadataByName(ctx.RequestCtx(), id, name)
	if err != nil {
		return fmt.Errorf("failed to get task metadata by name: %w", err)
	}

	return ctx.JSON(protocol.NewSuccessResponse(value))
}

// save task metadata
//
//	@Summary	Save task metadata
//	@Tags		kanban
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string						true	"task ID"
//	@Param		body	body		object{values=map[string]string}	true	"metadata values"
//	@Success	200		{object}	protocol.Response{data=map[string]bool}
//	@Security	ApiKeyAuth
//	@Router		/service/kanban/{id}/metadata [post]
func saveTaskMetadata(ctx fiber.Ctx) error {
	idStr := ctx.Params("id")
	if idStr == "" {
		return protocol.ErrBadParam.New("id is required")
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return protocol.ErrBadParam.New("invalid task ID")
	}

	var body saveMetadataRequest
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

	result, err := client.SaveTaskMetadata(ctx.RequestCtx(), id, body.Values)
	if err != nil {
		return fmt.Errorf("failed to save task metadata: %w", err)
	}

	return ctx.JSON(protocol.NewSuccessResponse(map[string]bool{"success": result}))
}

// remove task metadata
//
//	@Summary	Remove task metadata
//	@Tags		kanban
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string	true	"task ID"
//	@Param		name	path		string	true	"metadata name"
//	@Success	200		{object}	protocol.Response{data=map[string]bool}
//	@Security	ApiKeyAuth
//	@Router		/service/kanban/{id}/metadata/{name} [delete]
func removeTaskMetadata(ctx fiber.Ctx) error {
	idStr := ctx.Params("id")
	if idStr == "" {
		return protocol.ErrBadParam.New("id is required")
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return protocol.ErrBadParam.New("invalid task ID")
	}

	name := ctx.Params("name")
	if name == "" {
		return protocol.ErrBadParam.New("name is required")
	}

	client, err := kanboard.GetClient()
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}

	result, err := client.RemoveTaskMetadata(ctx.RequestCtx(), id, name)
	if err != nil {
		return fmt.Errorf("failed to remove task metadata: %w", err)
	}

	return ctx.JSON(protocol.NewSuccessResponse(map[string]bool{"success": result}))
}
