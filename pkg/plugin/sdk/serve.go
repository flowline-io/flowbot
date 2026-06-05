package sdk

import (
	"context"

	goPlugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"

	pb "github.com/flowline-io/flowbot/pkg/plugin/grpc/proto"
)

// ModulePlugin implements the go-plugin interface for module plugins.
type ModulePlugin struct {
	goPlugin.NetRPCUnsupportedPlugin
	impl Module
}

// GRPCServer registers the module as a PluginServiceServer.
func (*ModulePlugin) GRPCServer(_ *goPlugin.GRPCBroker, _ *grpc.Server) error {
	return nil
}

// GRPCClient creates a PluginServiceClient.
func (*ModulePlugin) GRPCClient(_ context.Context, _ *goPlugin.GRPCBroker, c *grpc.ClientConn) (any, error) {
	return pb.NewPluginServiceClient(c), nil
}

// ServeModule starts a go-plugin server for a module plugin.
// Called from the plugin binary's main().
func ServeModule(m Module) {
	goPlugin.Serve(&goPlugin.ServeConfig{
		HandshakeConfig: goPlugin.HandshakeConfig{
			ProtocolVersion:  1,
			MagicCookieKey:   "FLOWBOT_PLUGIN",
			MagicCookieValue: "flowbot-plugin-v1",
		},
		Plugins: map[string]goPlugin.Plugin{
			"module": &ModulePlugin{impl: m},
		},
		GRPCServer: goPlugin.DefaultGRPCServer,
	})
}
