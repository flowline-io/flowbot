package archivebox

import (
	"context"
	"time"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/ability/archive"
	provider "github.com/flowline-io/flowbot/pkg/providers/archivebox"
	"github.com/flowline-io/flowbot/pkg/types"
)

type client interface {
	Add(data provider.Data) (*provider.Response, error)
}

type Adapter struct {
	client client
}

func New() archive.Service {
	return NewWithClient(provider.GetClient())
}

func NewWithClient(client client) archive.Service {
	return &Adapter{client: client}
}

func (a *Adapter) Add(ctx context.Context, req archive.AddRequest) (*ability.ArchiveItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "archive add canceled", err)
	}
	if req.URL == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "url is required")
	}
	resp, err := a.client.Add(provider.Data{
		Urls:      []string{req.URL},
		Tag:       req.Tag,
		Depth:     req.Depth,
		Update:    req.Update,
		IndexOnly: req.IndexOnly,
	})
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "archivebox add", err)
	}
	if resp == nil {
		return nil, types.Errorf(types.ErrProvider, "archivebox returned empty response")
	}
	if !resp.Success {
		return nil, types.Errorf(types.ErrProvider, "archivebox add failed")
	}
	return &ability.ArchiveItem{
		ID:        firstResult(resp.Result, req.URL),
		URL:       req.URL,
		Title:     req.URL,
		Status:    "created",
		CreatedAt: time.Now(),
	}, nil
}

func (a *Adapter) Search(ctx context.Context, q *archive.SearchQuery) (*ability.ListResult[ability.ArchiveItem], error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "archive search canceled", err)
	}
	return nil, types.Errorf(types.ErrNotImplemented, "archivebox search is not implemented")
}

func (a *Adapter) Get(ctx context.Context, id string) (*ability.ArchiveItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "archive get canceled", err)
	}
	if id == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "id is required")
	}
	return nil, types.Errorf(types.ErrNotImplemented, "archivebox get is not implemented")
}

func firstResult(values []string, fallback string) string {
	if len(values) == 0 || values[0] == "" {
		return fallback
	}
	return values[0]
}
