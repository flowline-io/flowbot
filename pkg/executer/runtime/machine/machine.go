package machine

import "github.com/flowline-io/flowbot/pkg/utils/syncx"

type Runtime struct {
	client any
	tasks   *syncx.Map[string, string]
	config  string
}

type Option = func(rt *Runtime)

func WithConfig(config string) Option {
	return func(rt *Runtime) {
		rt.config = config
	}
}

func NewRuntime(opts ...Option) (*Runtime, error) {
	rt := &Runtime{
		client: "ssh client", // todo
		tasks:  new(syncx.Map[string, string]),
	}
	for _, o := range opts {
		o(rt)
	}

	return rt, nil
}