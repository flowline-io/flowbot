package machine

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/flowline-io/flowbot/pkg/utils/syncx"
	"golang.org/x/crypto/ssh"
)

type Config struct {
	Host     string
	Port     int
	Username string
	Password string
}

type Runtime struct {
	client *ssh.Client
	tasks  *syncx.Map[string, *ssh.Session]
	config Config
}

type Option = func(rt *Runtime)

func WithConfig(config Config) Option {
	return func(rt *Runtime) {
		rt.config = config
	}
}

func NewRuntime(opts ...Option) (*Runtime, error) {
	rt := &Runtime{
		tasks: new(syncx.Map[string, *ssh.Session]),
	}
	for _, o := range opts {
		o(rt)
	}

	cfg := &ssh.ClientConfig{
		User: rt.config.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(rt.config.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         time.Minute,
	}
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", rt.config.Host, rt.config.Port), cfg)
	if err != nil {
		return nil, err
	}
	rt.client = client

	return rt, nil
}

func (r *Runtime) Run(ctx context.Context, t *types.Task) error {
	// execute pre-tasks
	for _, pre := range t.Pre {
		pre.ID = utils.NewUUID()
		if err := r.doRun(ctx, pre); err != nil {
			return err
		}
	}
	// run the actual task
	if err := r.doRun(ctx, t); err != nil {
		return err
	}
	// execute post-tasks
	for _, post := range t.Post {
		post.ID = utils.NewUUID()
		if err := r.doRun(ctx, post); err != nil {
			return err
		}
	}
	return nil
}

func (r *Runtime) Stop(_ context.Context, t *types.Task) error {
	sess, ok := r.tasks.Get(t.ID)
	if !ok {
		return nil
	}
	r.tasks.Delete(t.ID)
	flog.Debug("Attempting to stop and remove session, task %v", t.ID)
	return sess.Close()
}

func (r *Runtime) HealthCheck(_ context.Context) error {
	sess, err := r.client.NewSession()
	if err != nil {
		return err
	}
	defer func() { _ = sess.Close() }()
	return sess.Run("/usr/bin/hostname")
}

func (r *Runtime) doRun(ctx context.Context, t *types.Task) error {
	if t.ID == "" {
		return errors.New("task id is required")
	}

	sess, err := r.client.NewSession()
	if err != nil {
		return err
	}

	r.tasks.Set(t.ID, sess)

	flog.Debug("created session for task %v", t.ID)

	defer func() {
		stopContext, cancel := context.WithTimeout(context.Background(), time.Second*60)
		defer cancel()
		if err := r.Stop(stopContext, t); err != nil {
			flog.Error(fmt.Errorf("error stopping session for task %v, %w", t.ID, err))
		}
	}()

	var b bytes.Buffer
	sess.Stdout = &b

	if err := sess.Run(t.Run); err != nil {
		return r.Stop(ctx, t)
	}
	t.Result = b.String()

	return nil
}
