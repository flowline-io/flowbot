package core

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/capability"
)

func coreHealthInvoker(_ context.Context, _ map[string]any) (*capability.InvokeResult, error) {
	return &capability.InvokeResult{Data: true}, nil
}
