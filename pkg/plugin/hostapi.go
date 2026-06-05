// Package plugin provides the plugin runner framework for extending flowbot
// via external binaries communicating over gRPC.
package plugin

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/types"
)

// HostAPI defines the services available to plugins from the host.
type HostAPI interface {
	GetConfig(ctx context.Context, key string) (string, error)
	Log(ctx context.Context, level string, msg string, fields map[string]string)
	KVGet(ctx context.Context, key string) ([]byte, error)
	KVSet(ctx context.Context, key string, value []byte) error
	KVDelete(ctx context.Context, key string) error
	HTTPRequest(ctx context.Context, req *HostHTTPRequest) (*HostHTTPResponse, error)
	EmitEvent(ctx context.Context, event types.DataEvent) error
}

// HostHTTPRequest is an HTTP request from a plugin to the host.
type HostHTTPRequest struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    []byte            `json:"body"`
}

// HostHTTPResponse is an HTTP response from the host to a plugin.
type HostHTTPResponse struct {
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers"`
	Body    []byte            `json:"body"`
}
