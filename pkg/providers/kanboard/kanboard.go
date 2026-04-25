package kanboard

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/jhttp"
	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/types"
)

const (
	ID              = "kanboard"
	EndpointKey     = "endpoint"
	UsernameKey     = "username"
	PasswordKey     = "password"
	WebhookTokenKey = "webhook_token"
)

type Kanboard struct {
	c       *jrpc2.Client
	channel *jhttp.Channel
}

type AuthTransport struct {
	Transport http.RoundTripper
	Username  string
	Password  string
}

func (t *AuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	setAuthHeader(req.Header, t.Username, t.Password)
	return t.Transport.RoundTrip(req)
}

func setAuthHeader(header http.Header, username string, password string) {
	auth := fmt.Sprintf("%s:%s", username, password)
	buf := bytes.Buffer{}
	encoder := base64.NewEncoder(base64.StdEncoding, &buf)
	_, _ = encoder.Write([]byte(auth))
	_ = encoder.Close()

	header.Set("Authorization", fmt.Sprintf("Basic %s", buf.String()))
}

func GetClient() (*Kanboard, error) {
	endpoint, _ := providers.GetConfig(ID, EndpointKey)
	username, _ := providers.GetConfig(ID, UsernameKey)
	password, _ := providers.GetConfig(ID, PasswordKey)
	if endpoint.String() == "" {
		return nil, fmt.Errorf("kanboard disabled")
	}

	return NewKanboard(endpoint.String(), username.String(), password.String())
}

func NewKanboard(endpoint string, username string, password string) (*Kanboard, error) {
	v := &Kanboard{}
	v.channel = jhttp.NewChannel(endpoint, &jhttp.ChannelOptions{
		Client: &http.Client{
			Transport: &AuthTransport{
				Transport: http.DefaultTransport,
				Username:  username,
				Password:  password,
			},
		},
	})
	v.c = jrpc2.NewClient(v.channel, nil)

	return v, nil
}

func (v *Kanboard) Close() error {
	err := v.channel.Close()
	if err != nil {
		return err
	}
	return v.c.Close()
}

func (v *Kanboard) CreateTask(ctx context.Context, task *Task) (taskId int64, err error) {
	err = v.c.CallResult(ctx, "createTask", task, &taskId)
	if err != nil {
		err = fmt.Errorf("failed to create task, %w", err)
		return
	}
	return
}

func (v *Kanboard) GetAllTasks(ctx context.Context, projectId int, status StatusId) (tasks []*Task, err error) {
	err = v.c.CallResult(ctx, "getAllTasks", types.KV{"project_id": projectId, "status_id": status}, &tasks)
	if err != nil {
		err = fmt.Errorf("failed to get all tasks, %w", err)
		return
	}
	return
}

func (v *Kanboard) GetTask(ctx context.Context, taskId int) (task *Task, err error) {
	err = v.c.CallResult(ctx, "getTask", types.KV{"task_id": taskId}, &task)
	if err != nil {
		err = fmt.Errorf("failed to get task, %w", err)
		return
	}
	return
}

func (v *Kanboard) UpdateTask(ctx context.Context, taskId int, task *Task) (result bool, err error) {
	params := types.KV{"id": taskId}
	if task.Title != "" {
		params["title"] = task.Title
	}
	if task.Description != "" {
		params["description"] = task.Description
	}
	err = v.c.CallResult(ctx, "updateTask", params, &result)
	if err != nil {
		err = fmt.Errorf("failed to update task, %w", err)
		return
	}
	return
}

func (v *Kanboard) CloseTask(ctx context.Context, taskId int) (result bool, err error) {
	err = v.c.CallResult(ctx, "closeTask", types.KV{"task_id": taskId}, &result)
	if err != nil {
		err = fmt.Errorf("failed to close task, %w", err)
		return
	}
	return
}

func (v *Kanboard) OpenTask(ctx context.Context, taskId int) (result bool, err error) {
	err = v.c.CallResult(ctx, "openTask", types.KV{"task_id": taskId}, &result)
	if err != nil {
		err = fmt.Errorf("failed to open task, %w", err)
		return
	}
	return
}

func (v *Kanboard) RemoveTask(ctx context.Context, taskId int) (result bool, err error) {
	err = v.c.CallResult(ctx, "removeTask", types.KV{"task_id": taskId}, &result)
	if err != nil {
		err = fmt.Errorf("failed to remove task, %w", err)
		return
	}
	return
}

func (v *Kanboard) MoveTaskPosition(ctx context.Context, projectId int, taskId int, columnId int, position int, swimlaneId int) (result bool, err error) {
	params := types.KV{
		"project_id":  projectId,
		"task_id":     taskId,
		"column_id":   columnId,
		"position":    position,
		"swimlane_id": swimlaneId,
	}
	err = v.c.CallResult(ctx, "moveTaskPosition", params, &result)
	if err != nil {
		err = fmt.Errorf("failed to move task position, %w", err)
		return
	}
	return
}

func (v *Kanboard) GetColumns(ctx context.Context, projectId int) (columns []types.KV, err error) {
	err = v.c.CallResult(ctx, "getColumns", types.KV{"project_id": projectId}, &columns)
	if err != nil {
		err = fmt.Errorf("failed to get columns, %w", err)
		return
	}
	return
}

func (v *Kanboard) SearchTasks(ctx context.Context, projectId int, query string) (tasks []*Task, err error) {
	err = v.c.CallResult(ctx, "searchTasks", types.KV{"project_id": projectId, "query": query}, &tasks)
	if err != nil {
		err = fmt.Errorf("failed to search tasks, %w", err)
		return
	}
	return
}

func (v *Kanboard) GetTaskMetadata(ctx context.Context, taskId int) (metadata []TaskMetadata, err error) {
	err = v.c.CallResult(ctx, "getTaskMetadata", types.KV{"task_id": taskId}, &metadata)
	if err != nil {
		err = fmt.Errorf("failed to get task metadata, %w", err)
		return
	}
	return
}

func (v *Kanboard) GetTaskMetadataByName(ctx context.Context, taskId int, name string) (value string, err error) {
	err = v.c.CallResult(ctx, "getTaskMetadataByName", types.KV{"task_id": taskId, "name": name}, &value)
	if err != nil {
		err = fmt.Errorf("failed to get task metadata by name, %w", err)
		return
	}
	return
}

func (v *Kanboard) SaveTaskMetadata(ctx context.Context, taskId int, values TaskMetadata) (result bool, err error) {
	err = v.c.CallResult(ctx, "saveTaskMetadata", types.KV{"task_id": taskId, "values": values}, &result)
	if err != nil {
		err = fmt.Errorf("failed to save task metadata, %w", err)
		return
	}
	return
}

func (v *Kanboard) RemoveTaskMetadata(ctx context.Context, taskId int, name string) (result bool, err error) {
	err = v.c.CallResult(ctx, "removeTaskMetadata", types.KV{"task_id": taskId, "name": name}, &result)
	if err != nil {
		err = fmt.Errorf("failed to remove task metadata, %w", err)
		return
	}
	return
}
