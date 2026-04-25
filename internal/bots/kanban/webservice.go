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

// get all tags
//
//	@Summary	Get all tags
//	@Tags		kanban
//	@Accept		json
//	@Produce	json
//	@Success	200	{object}	protocol.Response{data=[]kanboard.Tag}
//	@Security	ApiKeyAuth
//	@Router		/service/kanban/tags [get]
func getAllTags(ctx fiber.Ctx) error {
	client, err := kanboard.GetClient()
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}

	tags, err := client.GetAllTags(ctx.RequestCtx())
	if err != nil {
		return fmt.Errorf("failed to get all tags: %w", err)
	}

	return ctx.JSON(protocol.NewSuccessResponse(tags))
}

// get tags by project
//
//	@Summary	Get tags by project
//	@Tags		kanban
//	@Accept		json
//	@Produce	json
//	@Param		project_id	query		int	true	"project ID"
//	@Success	200			{object}	protocol.Response{data=[]kanboard.Tag}
//	@Security	ApiKeyAuth
//	@Router		/service/kanban/tags/project [get]
func getTagsByProject(ctx fiber.Ctx) error {
	projectIdStr := ctx.Query("project_id")
	if projectIdStr == "" {
		return protocol.ErrBadParam.New("project_id is required")
	}

	projectId, err := strconv.Atoi(projectIdStr)
	if err != nil {
		return protocol.ErrBadParam.New("invalid project_id")
	}

	client, err := kanboard.GetClient()
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}

	tags, err := client.GetTagsByProject(ctx.RequestCtx(), projectId)
	if err != nil {
		return fmt.Errorf("failed to get tags by project: %w", err)
	}

	return ctx.JSON(protocol.NewSuccessResponse(tags))
}

// create tag
//
//	@Summary	Create a new tag
//	@Tags		kanban
//	@Accept		json
//	@Produce	json
//	@Param		body	body		object{project_id=int,name=string,color_id=string}	true	"tag data"
//	@Success	200		{object}	protocol.Response{data=map[string]int64}
//	@Security	ApiKeyAuth
//	@Router		/service/kanban/tags [post]
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

	tagId, err := client.CreateTag(ctx.RequestCtx(), body.ProjectID, body.Name, body.ColorID)
	if err != nil {
		return fmt.Errorf("failed to create tag: %w", err)
	}

	return ctx.JSON(protocol.NewSuccessResponse(map[string]int64{"id": tagId}))
}

// update tag
//
//	@Summary	Update a tag
//	@Tags		kanban
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string								true	"tag ID"
//	@Param		body	body		object{name=string,color_id=string}	true	"tag data"
//	@Success	200		{object}	protocol.Response{data=map[string]bool}
//	@Security	ApiKeyAuth
//	@Router		/service/kanban/tags/{id} [patch]
func updateTag(ctx fiber.Ctx) error {
	idStr := ctx.Params("id")
	if idStr == "" {
		return protocol.ErrBadParam.New("id is required")
	}

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

	result, err := client.UpdateTag(ctx.RequestCtx(), id, body.Name, body.ColorID)
	if err != nil {
		return fmt.Errorf("failed to update tag: %w", err)
	}

	return ctx.JSON(protocol.NewSuccessResponse(map[string]bool{"success": result}))
}

// remove tag
//
//	@Summary	Remove a tag
//	@Tags		kanban
//	@Accept		json
//	@Produce	json
//	@Param		id	path		string	true	"tag ID"
//	@Success	200	{object}	protocol.Response{data=map[string]bool}
//	@Security	ApiKeyAuth
//	@Router		/service/kanban/tags/{id} [delete]
func removeTag(ctx fiber.Ctx) error {
	idStr := ctx.Params("id")
	if idStr == "" {
		return protocol.ErrBadParam.New("id is required")
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return protocol.ErrBadParam.New("invalid tag ID")
	}

	client, err := kanboard.GetClient()
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}

	result, err := client.RemoveTag(ctx.RequestCtx(), id)
	if err != nil {
		return fmt.Errorf("failed to remove tag: %w", err)
	}

	return ctx.JSON(protocol.NewSuccessResponse(map[string]bool{"success": result}))
}

// get task tags
//
//	@Summary	Get tags assigned to a task
//	@Tags		kanban
//	@Accept		json
//	@Produce	json
//	@Param		id	path		string	true	"task ID"
//	@Success	200	{object}	protocol.Response{data=map[string]string}
//	@Security	ApiKeyAuth
//	@Router		/service/kanban/{id}/tags [get]
func getTaskTags(ctx fiber.Ctx) error {
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

	tags, err := client.GetTaskTags(ctx.RequestCtx(), id)
	if err != nil {
		return fmt.Errorf("failed to get task tags: %w", err)
	}

	return ctx.JSON(protocol.NewSuccessResponse(tags))
}

// set task tags
//
//	@Summary	Set tags for a task
//	@Tags		kanban
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string									true	"task ID"
//	@Param		body	body		object{project_id=int,tags=[]string}	true	"tags data"
//	@Success	200		{object}	protocol.Response{data=map[string]bool}
//	@Security	ApiKeyAuth
//	@Router		/service/kanban/{id}/tags [post]
func setTaskTags(ctx fiber.Ctx) error {
	idStr := ctx.Params("id")
	if idStr == "" {
		return protocol.ErrBadParam.New("id is required")
	}

	taskId, err := strconv.Atoi(idStr)
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

	result, err := client.SetTaskTags(ctx.RequestCtx(), body.ProjectID, taskId, body.Tags)
	if err != nil {
		return fmt.Errorf("failed to set task tags: %w", err)
	}

	return ctx.JSON(protocol.NewSuccessResponse(map[string]bool{"success": result}))
}

// list subtasks
//
//	@Summary	List all subtasks for a task
//	@Tags		kanban
//	@Accept		json
//	@Produce	json
//	@Param		id	path		string	true	"task ID"
//	@Success	200	{object}	protocol.Response{data=[]kanboard.Subtask}
//	@Security	ApiKeyAuth
//	@Router		/service/kanban/{id}/subtasks [get]
func listSubtasks(ctx fiber.Ctx) error {
	idStr := ctx.Params("id")
	if idStr == "" {
		return protocol.ErrBadParam.New("id is required")
	}

	taskId, err := strconv.Atoi(idStr)
	if err != nil {
		return protocol.ErrBadParam.New("invalid task ID")
	}

	client, err := kanboard.GetClient()
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}

	subtasks, err := client.GetAllSubtasks(ctx.RequestCtx(), taskId)
	if err != nil {
		return fmt.Errorf("failed to get subtasks: %w", err)
	}

	return ctx.JSON(protocol.NewSuccessResponse(subtasks))
}

// get subtask
//
//	@Summary	Get a subtask by ID
//	@Tags		kanban
//	@Accept		json
//	@Produce	json
//	@Param		id			path		string	true	"task ID"
//	@Param		subtaskId	path		string	true	"subtask ID"
//	@Success	200			{object}	protocol.Response{data=kanboard.Subtask}
//	@Security	ApiKeyAuth
//	@Router		/service/kanban/{id}/subtasks/{subtaskId} [get]
func getSubtask(ctx fiber.Ctx) error {
	subtaskIdStr := ctx.Params("subtaskId")
	if subtaskIdStr == "" {
		return protocol.ErrBadParam.New("subtask_id is required")
	}

	subtaskId, err := strconv.Atoi(subtaskIdStr)
	if err != nil {
		return protocol.ErrBadParam.New("invalid subtask ID")
	}

	client, err := kanboard.GetClient()
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}

	subtask, err := client.GetSubtask(ctx.RequestCtx(), subtaskId)
	if err != nil {
		return fmt.Errorf("failed to get subtask: %w", err)
	}

	return ctx.JSON(protocol.NewSuccessResponse(subtask))
}

// create subtask
//
//	@Summary	Create a new subtask
//	@Tags		kanban
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string										true	"task ID"
//	@Param		body	body		object{title=string,user_id=int,time_estimated=int,time_spent=int,status=int}	true	"subtask data"
//	@Success	200		{object}	protocol.Response{data=map[string]int64}
//	@Security	ApiKeyAuth
//	@Router		/service/kanban/{id}/subtasks [post]
func createSubtask(ctx fiber.Ctx) error {
	idStr := ctx.Params("id")
	if idStr == "" {
		return protocol.ErrBadParam.New("id is required")
	}

	taskId, err := strconv.Atoi(idStr)
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

	subtaskId, err := client.CreateSubtask(ctx.RequestCtx(), taskId, body.Title, body.UserID, body.TimeEstimated, body.TimeSpent, body.Status)
	if err != nil {
		return fmt.Errorf("failed to create subtask: %w", err)
	}

	return ctx.JSON(protocol.NewSuccessResponse(map[string]int64{"id": subtaskId}))
}

// update subtask
//
//	@Summary	Update a subtask
//	@Tags		kanban
//	@Accept		json
//	@Produce	json
//	@Param		id			path		string															true	"task ID"
//	@Param		subtaskId		path		string															true	"subtask ID"
//	@Param		body		body		object{title=string,user_id=int,time_estimated=int,time_spent=int,status=int}	true	"subtask data"
//	@Success	200			{object}	protocol.Response{data=map[string]bool}
//	@Security	ApiKeyAuth
//	@Router		/service/kanban/{id}/subtasks/{subtaskId} [patch]
func updateSubtask(ctx fiber.Ctx) error {
	idStr := ctx.Params("id")
	if idStr == "" {
		return protocol.ErrBadParam.New("id is required")
	}

	taskId, err := strconv.Atoi(idStr)
	if err != nil {
		return protocol.ErrBadParam.New("invalid task ID")
	}

	subtaskIdStr := ctx.Params("subtaskId")
	if subtaskIdStr == "" {
		return protocol.ErrBadParam.New("subtask_id is required")
	}

	subtaskId, err := strconv.Atoi(subtaskIdStr)
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

	userId := -1
	if body.UserID >= 0 {
		userId = body.UserID
	}
	timeEstimated := -1
	if body.TimeEstimated >= 0 {
		timeEstimated = body.TimeEstimated
	}
	timeSpent := -1
	if body.TimeSpent >= 0 {
		timeSpent = body.TimeSpent
	}
	status := -1
	if body.Status >= 0 {
		status = body.Status
	}

	result, err := client.UpdateSubtask(ctx.RequestCtx(), subtaskId, taskId, body.Title, userId, timeEstimated, timeSpent, status)
	if err != nil {
		return fmt.Errorf("failed to update subtask: %w", err)
	}

	return ctx.JSON(protocol.NewSuccessResponse(map[string]bool{"success": result}))
}

// delete subtask
//
//	@Summary	Delete a subtask
//	@Tags		kanban
//	@Accept		json
//	@Produce	json
//	@Param		id			path		string	true	"task ID"
//	@Param		subtaskId	path		string	true	"subtask ID"
//	@Success	200			{object}	protocol.Response{data=map[string]bool}
//	@Security	ApiKeyAuth
//	@Router		/service/kanban/{id}/subtasks/{subtaskId} [delete]
func deleteSubtask(ctx fiber.Ctx) error {
	subtaskIdStr := ctx.Params("subtaskId")
	if subtaskIdStr == "" {
		return protocol.ErrBadParam.New("subtask_id is required")
	}

	subtaskId, err := strconv.Atoi(subtaskIdStr)
	if err != nil {
		return protocol.ErrBadParam.New("invalid subtask ID")
	}

	client, err := kanboard.GetClient()
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}

	result, err := client.RemoveSubtask(ctx.RequestCtx(), subtaskId)
	if err != nil {
		return fmt.Errorf("failed to remove subtask: %w", err)
	}

	return ctx.JSON(protocol.NewSuccessResponse(map[string]bool{"success": result}))
}

// has subtask timer
//
//	@Summary	Check if a timer is started for the subtask
//	@Tags		kanban
//	@Accept		json
//	@Produce	json
//	@Param		id			path		string	true	"task ID"
//	@Param		subtaskId	path		string	true	"subtask ID"
//	@Param		user_id		query		int		false	"user ID"
//	@Success	200			{object}	protocol.Response{data=map[string]bool}
//	@Security	ApiKeyAuth
//	@Router		/service/kanban/{id}/subtasks/{subtaskId}/timer [get]
func hasSubtaskTimer(ctx fiber.Ctx) error {
	subtaskIdStr := ctx.Params("subtaskId")
	if subtaskIdStr == "" {
		return protocol.ErrBadParam.New("subtask_id is required")
	}

	subtaskId, err := strconv.Atoi(subtaskIdStr)
	if err != nil {
		return protocol.ErrBadParam.New("invalid subtask ID")
	}

	userId := 0
	if v := ctx.Query("user_id"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			userId = n
		}
	}

	client, err := kanboard.GetClient()
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}

	result, err := client.HasSubtaskTimer(ctx.RequestCtx(), subtaskId, userId)
	if err != nil {
		return fmt.Errorf("failed to check subtask timer: %w", err)
	}

	return ctx.JSON(protocol.NewSuccessResponse(map[string]bool{"result": result}))
}

// set subtask start time
//
//	@Summary	Start subtask timer for a user
//	@Tags		kanban
//	@Accept		json
//	@Produce	json
//	@Param		id			path		string	true	"task ID"
//	@Param		subtaskId	path		string	true	"subtask ID"
//	@Param		user_id		query		int		false	"user ID"
//	@Success	200			{object}	protocol.Response{data=map[string]bool}
//	@Security	ApiKeyAuth
//	@Router		/service/kanban/{id}/subtasks/{subtaskId}/timer/start [post]
func setSubtaskStartTime(ctx fiber.Ctx) error {
	subtaskIdStr := ctx.Params("subtaskId")
	if subtaskIdStr == "" {
		return protocol.ErrBadParam.New("subtask_id is required")
	}

	subtaskId, err := strconv.Atoi(subtaskIdStr)
	if err != nil {
		return protocol.ErrBadParam.New("invalid subtask ID")
	}

	userId := 0
	if v := ctx.Query("user_id"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			userId = n
		}
	}

	client, err := kanboard.GetClient()
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}

	result, err := client.SetSubtaskStartTime(ctx.RequestCtx(), subtaskId, userId)
	if err != nil {
		return fmt.Errorf("failed to start subtask timer: %w", err)
	}

	return ctx.JSON(protocol.NewSuccessResponse(map[string]bool{"result": result}))
}

// set subtask end time
//
//	@Summary	Stop subtask timer for a user
//	@Tags		kanban
//	@Accept		json
//	@Produce	json
//	@Param		id			path		string	true	"task ID"
//	@Param		subtaskId	path		string	true	"subtask ID"
//	@Param		user_id		query		int		false	"user ID"
//	@Success	200			{object}	protocol.Response{data=map[string]bool}
//	@Security	ApiKeyAuth
//	@Router		/service/kanban/{id}/subtasks/{subtaskId}/timer/stop [post]
func setSubtaskEndTime(ctx fiber.Ctx) error {
	subtaskIdStr := ctx.Params("subtaskId")
	if subtaskIdStr == "" {
		return protocol.ErrBadParam.New("subtask_id is required")
	}

	subtaskId, err := strconv.Atoi(subtaskIdStr)
	if err != nil {
		return protocol.ErrBadParam.New("invalid subtask ID")
	}

	userId := 0
	if v := ctx.Query("user_id"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			userId = n
		}
	}

	client, err := kanboard.GetClient()
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}

	result, err := client.SetSubtaskEndTime(ctx.RequestCtx(), subtaskId, userId)
	if err != nil {
		return fmt.Errorf("failed to stop subtask timer: %w", err)
	}

	return ctx.JSON(protocol.NewSuccessResponse(map[string]bool{"result": result}))
}

// get subtask time spent
//
//	@Summary	Get time spent on a subtask for a user
//	@Tags		kanban
//	@Accept		json
//	@Produce	json
//	@Param		id			path		string	true	"task ID"
//	@Param		subtaskId	path		string	true	"subtask ID"
//	@Param		user_id		query		int		false	"user ID"
//	@Success	200			{object}	protocol.Response{data=map[string]float64}
//	@Security	ApiKeyAuth
//	@Router		/service/kanban/{id}/subtasks/{subtaskId}/timer/spent [get]
func getSubtaskTimeSpent(ctx fiber.Ctx) error {
	subtaskIdStr := ctx.Params("subtaskId")
	if subtaskIdStr == "" {
		return protocol.ErrBadParam.New("subtask_id is required")
	}

	subtaskId, err := strconv.Atoi(subtaskIdStr)
	if err != nil {
		return protocol.ErrBadParam.New("invalid subtask ID")
	}

	userId := 0
	if v := ctx.Query("user_id"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			userId = n
		}
	}

	client, err := kanboard.GetClient()
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}

	result, err := client.GetSubtaskTimeSpent(ctx.RequestCtx(), subtaskId, userId)
	if err != nil {
		return fmt.Errorf("failed to get subtask time spent: %w", err)
	}

	return ctx.JSON(protocol.NewSuccessResponse(map[string]float64{"result": result}))
}
