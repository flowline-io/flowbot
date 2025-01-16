package kanboard

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/jhttp"
	"net/http"
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
