package grpc

import (
	"context"
	"io"

	"github.com/hashicorp/go-hclog"
	goPlugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"

	pb "github.com/flowline-io/flowbot/pkg/plugin/grpc/proto"
)

// GrpcPlugin is the hashicorp/go-plugin adapter.
type GrpcPlugin struct {
	goPlugin.NetRPCUnsupportedPlugin
}

func (p *GrpcPlugin) GRPCServer(broker *goPlugin.GRPCBroker, s *grpc.Server) error {
	return nil
}

func (p *GrpcPlugin) GRPCClient(ctx context.Context, broker *goPlugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	client := pb.NewPluginServiceClient(c)
	return &client, nil
}

// HCLogAdapter adapts go-plugin's hclog to flowbot's logging.
type HCLogAdapter struct {
	hclog.Logger
}

// NewHCLogAdapter creates a logger adapter that discards go-plugin internal logs.
func NewHCLogAdapter() hclog.Logger {
	return hclog.New(&hclog.LoggerOptions{
		Output: io.Discard,
		Level:  hclog.Info,
	})
}
