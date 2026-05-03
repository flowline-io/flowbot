package archivebox

import (
	"testing"

	arc "github.com/flowline-io/flowbot/pkg/ability/archive"
	"github.com/flowline-io/flowbot/pkg/ability/conformance"
	provider "github.com/flowline-io/flowbot/pkg/providers/archivebox"
)

func TestArchiveboxConformance(t *testing.T) {
	conformance.RunArchiveConformance(t, func(t *testing.T, cfg conformance.ArchiveConfig) arc.Service {
		c := &fakeClient{
			resp: cfgToAddResp(cfg),
			err:  cfg.AddErr,
		}
		return NewWithClient(c)
	})
}

func cfgToAddResp(cfg conformance.ArchiveConfig) *provider.Response {
	if cfg.AddErr != nil {
		return nil
	}
	if cfg.AddItem == nil {
		return &provider.Response{Success: true, Result: []string{"snap-1"}}
	}
	return &provider.Response{Success: true, Result: []string{cfg.AddItem.ID}}
}
