// Package grpc provides the gRPC-based plugin runtime implementation.
package grpc

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/flowline-io/flowbot/pkg/plugin"
	pb "github.com/flowline-io/flowbot/pkg/plugin/grpc/proto"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/utils"
)

// HostServer implements pb.HostServiceServer and delegates to a plugin.HostAPI.
type HostServer struct {
	pb.UnimplementedHostServiceServer
	api plugin.HostAPI
}

// NewHostServer creates a HostServer with the given HostAPI.
func NewHostServer(api plugin.HostAPI) *HostServer {
	return &HostServer{api: api}
}

// Register registers this server on a gRPC server.
func (h *HostServer) Register(s *grpc.Server) {
	pb.RegisterHostServiceServer(s, h)
}

func (h *HostServer) GetConfig(ctx context.Context, req *pb.GetConfigRequest) (*pb.GetConfigResponse, error) {
	val, err := h.api.GetConfig(ctx, req.Key)
	if err != nil {
		return &pb.GetConfigResponse{Error: err.Error()}, nil
	}
	return &pb.GetConfigResponse{Value: val}, nil
}

func (h *HostServer) Log(ctx context.Context, req *pb.LogRequest) (*emptypb.Empty, error) {
	h.api.Log(ctx, req.Level, req.Message, req.Fields)
	return &emptypb.Empty{}, nil
}

func (h *HostServer) KVGet(ctx context.Context, req *pb.KVGetRequest) (*pb.KVGetResponse, error) {
	val, err := h.api.KVGet(ctx, req.Key)
	if err != nil {
		return &pb.KVGetResponse{Error: err.Error()}, nil
	}
	return &pb.KVGetResponse{Value: val}, nil
}

func (h *HostServer) KVSet(ctx context.Context, req *pb.KVSetRequest) (*emptypb.Empty, error) {
	_ = h.api.KVSet(ctx, req.Key, req.Value)
	return &emptypb.Empty{}, nil
}

func (h *HostServer) KVDelete(ctx context.Context, req *pb.KVDeleteRequest) (*emptypb.Empty, error) {
	_ = h.api.KVDelete(ctx, req.Key)
	return &emptypb.Empty{}, nil
}

func (h *HostServer) HTTPRequest(ctx context.Context, req *pb.HTTPCallRequest) (*pb.HTTPCallResponse, error) {
	resp, err := h.api.HTTPRequest(ctx, &plugin.HostHTTPRequest{
		Method:  req.Method,
		URL:     req.Url,
		Headers: req.Headers,
		Body:    req.Body,
	})
	if err != nil {
		return &pb.HTTPCallResponse{Error: err.Error()}, nil
	}
	status, ok := utils.IntToInt32(resp.Status)
	if !ok {
		return &pb.HTTPCallResponse{Error: "http status out of range"}, nil
	}
	return &pb.HTTPCallResponse{
		Status:  status,
		Headers: resp.Headers,
		Body:    resp.Body,
	}, nil
}

func (h *HostServer) EmitEvent(ctx context.Context, req *pb.EmitEventRequest) (*emptypb.Empty, error) {
	event := types.DataEvent{
		Source:    req.Source,
		EventType: req.EventType,
	}
	_ = h.api.EmitEvent(ctx, event)
	return &emptypb.Empty{}, nil
}
