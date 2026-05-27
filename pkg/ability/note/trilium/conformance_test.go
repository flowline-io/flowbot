// Package trilium implements the Trilium adapter for the note capability.
package trilium

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/ability/conformance"
	notesvc "github.com/flowline-io/flowbot/pkg/ability/note"
	provider "github.com/flowline-io/flowbot/pkg/providers/trilium"
)

// newConformanceService wraps an Adapter to satisfy the conformance NoteServiceFactory contract.
// It constructs a fakeClient from the conformance NoteConfig and returns the adapter.
//
//revive:disable:cyclomatic — conformance wiring maps many config fields to fake client fields.
func newConformanceService(t *testing.T, cfg conformance.NoteConfig) notesvc.Service {
	t.Helper()

	c := &fakeClient{
		getErr:           cfg.GetErr,
		createErr:        cfg.CreateErr,
		patchErr:         cfg.UpdateErr,
		deleteErr:        cfg.DeleteErr,
		getContentErr:    cfg.ContentErr,
		updateContentErr: cfg.SetContentErr,
		searchErr:        cfg.SearchErr,
		appInfoErr:       cfg.AppInfoErr,
	}
	// Provide a default empty search response for List when no error is configured.
	if cfg.ListErr == nil && cfg.SearchErr == nil {
		c.searchResp = &provider.SearchResponse{}
	}
	// List also maps error to searchErr.
	if cfg.ListErr != nil {
		c.searchErr = cfg.ListErr
	}

	if cfg.GetItem != nil {
		c.getResp = &provider.Note{
			NoteID: cfg.GetItem.ID,
			Title:  cfg.GetItem.Title,
			Type:   cfg.GetItem.Type,
		}
	}
	if cfg.CreateItem != nil {
		c.createResp = &provider.NoteWithBranch{
			Note: provider.Note{
				NoteID: cfg.CreateItem.ID,
				Title:  cfg.CreateItem.Title,
				Type:   cfg.CreateItem.Type,
			},
		}
	}
	if cfg.UpdateItem != nil {
		c.patchResp = &provider.Note{
			NoteID: cfg.UpdateItem.ID,
			Title:  cfg.UpdateItem.Title,
			Type:   cfg.UpdateItem.Type,
		}
		// Update also calls GetNote after patch to return fresh state.
		c.getResp = c.patchResp
	}
	if len(cfg.ListItems) > 0 {
		results := make([]provider.Note, len(cfg.ListItems))
		for i, item := range cfg.ListItems {
			results[i] = provider.Note{
				NoteID: item.ID,
				Title:  item.Title,
				Type:   item.Type,
			}
		}
		c.searchResp = &provider.SearchResponse{Results: results}
	}
	if len(cfg.SearchItems) > 0 {
		results := make([]provider.Note, len(cfg.SearchItems))
		for i, item := range cfg.SearchItems {
			results[i] = provider.Note{
				NoteID: item.ID,
				Title:  item.Title,
				Type:   item.Type,
			}
		}
		c.searchResp = &provider.SearchResponse{Results: results}
	}
	if cfg.Content != "" {
		c.getContentResp = cfg.Content
	}
	if cfg.AppInfo != nil {
		c.appInfoResp = &provider.AppInfo{
			AppVersion:   "0.63.7",
			InstanceName: cfg.AppInfo.ID,
		}
	}
	if cfg.RawItems != nil {
		c.listRawEventsResp = make([]map[string]any, len(cfg.RawItems))
		for i, item := range cfg.RawItems {
			if m, ok := item.(map[string]any); ok {
				c.listRawEventsResp[i] = m
			}
		}
		c.listRawEventsNext = cfg.RawCursor
	}
	if cfg.RawErr != nil {
		c.listRawEventsErr = cfg.RawErr
	}

	return NewWithClient(c)
}

func TestTriliumNoteConformance(t *testing.T) {
	conformance.RunNoteConformance(t, func(t *testing.T, cfg conformance.NoteConfig) notesvc.Service {
		t.Helper()
		return newConformanceService(t, cfg)
	})
}

// Compile-time check: fakeClient satisfies the client interface.
var _ client = (*fakeClient)(nil)

// Compile-time check: Adapter satisfies note.Service.
var _ notesvc.Service = (*Adapter)(nil)
