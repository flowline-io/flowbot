// Package trilium implements the Trilium adapter for the note capability.
package trilium

import (
	"testing"

	notesvc "github.com/flowline-io/flowbot/pkg/ability/note"
	provider "github.com/flowline-io/flowbot/pkg/providers/trilium"
)

// conformanceService wraps an Adapter to satisfy the conformance ServiceFactory contract.
// It constructs a fakeClient from the conformance Config and returns the adapter.
func newConformanceService(t *testing.T, cfg notesvc.Config) notesvc.Service {
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
	// List uses SearchNotes internally, so ListErr maps to searchErr.
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
	if cfg.ListResult != nil {
		results := make([]provider.Note, len(cfg.ListResult.Items))
		for i, item := range cfg.ListResult.Items {
			results[i] = provider.Note{
				NoteID: item.ID,
				Title:  item.Title,
				Type:   item.Type,
			}
		}
		c.searchResp = &provider.SearchResponse{Results: results}
	}
	if cfg.SearchResult != nil {
		results := make([]provider.Note, len(cfg.SearchResult.Items))
		for i, item := range cfg.SearchResult.Items {
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

	return NewWithClient(c)
}

func TestTriliumNoteConformance(t *testing.T) {
	notesvc.RunNoteConformance(t, func(t *testing.T, cfg notesvc.Config) notesvc.Service {
		t.Helper()
		return newConformanceService(t, cfg)
	})
}

// Compile-time check: fakeClient satisfies the client interface.
var _ client = (*fakeClient)(nil)

// Compile-time check: Adapter satisfies note.Service.
var _ notesvc.Service = (*Adapter)(nil)
