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
