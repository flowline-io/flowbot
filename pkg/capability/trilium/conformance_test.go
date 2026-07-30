// Package trilium implements the Trilium adapter for the note capability.
package trilium

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/capability/conformance"
	provider "github.com/flowline-io/flowbot/pkg/providers/trilium"
)

// newConformanceService wraps an Adapter to satisfy the conformance NoteServiceFactory contract.
// It constructs a fakeClient from the conformance NoteConfig and returns the adapter.
func newConformanceService(t *testing.T, cfg conformance.NoteConfig) Service {
	t.Helper()
	return NewWithClient(fakeClientFromNoteConfig(cfg))
}

func fakeClientFromNoteConfig(cfg conformance.NoteConfig) *fakeClient {
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
	applyConformanceSearch(c, cfg)
	applyConformanceCRUD(c, cfg)
	applyConformanceContent(c, cfg)
	applyConformanceRaw(c, cfg)
	return c
}

func applyConformanceSearch(c *fakeClient, cfg conformance.NoteConfig) {
	// Provide a default empty search response for List when no error is configured.
	if cfg.ListErr == nil && cfg.SearchErr == nil {
		c.searchResp = &provider.SearchResponse{}
	}
	// List also maps error to searchErr.
	if cfg.ListErr != nil {
		c.searchErr = cfg.ListErr
	}
	if len(cfg.ListItems) > 0 {
		c.searchResp = &provider.SearchResponse{Results: providerNotesFromCapability(cfg.ListItems)}
	}
	if len(cfg.SearchItems) > 0 {
		c.searchResp = &provider.SearchResponse{Results: providerNotesFromCapability(cfg.SearchItems)}
	}
}

func applyConformanceCRUD(c *fakeClient, cfg conformance.NoteConfig) {
	if cfg.GetItem != nil {
		c.getResp = providerNoteFromCapability(cfg.GetItem)
	}
	if cfg.CreateItem != nil {
		c.createResp = &provider.NoteWithBranch{Note: *providerNoteFromCapability(cfg.CreateItem)}
	}
	if cfg.UpdateItem != nil {
		c.patchResp = providerNoteFromCapability(cfg.UpdateItem)
		// Update also calls GetNote after patch to return fresh state.
		c.getResp = c.patchResp
	}
}

func applyConformanceContent(c *fakeClient, cfg conformance.NoteConfig) {
	if cfg.Content != "" {
		c.getContentResp = cfg.Content
	}
	if cfg.AppInfo != nil {
		c.appInfoResp = &provider.AppInfo{
			AppVersion:   "0.63.7",
			InstanceName: cfg.AppInfo.ID,
		}
	}
}

func applyConformanceRaw(c *fakeClient, cfg conformance.NoteConfig) {
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
}

func providerNoteFromCapability(item *capability.Note) *provider.Note {
	return &provider.Note{
		NoteID: item.ID,
		Title:  item.Title,
		Type:   item.Type,
	}
}

func providerNotesFromCapability(items []*capability.Note) []provider.Note {
	results := make([]provider.Note, len(items))
	for i, item := range items {
		results[i] = *providerNoteFromCapability(item)
	}
	return results
}

func TestTriliumNoteConformance(t *testing.T) {
	conformance.RunNoteConformance(t, func(t *testing.T, cfg conformance.NoteConfig) conformance.NoteService {
		t.Helper()
		return newConformanceService(t, cfg)
	})
}

// Compile-time check: fakeClient satisfies the client interface.
var _ client = (*fakeClient)(nil)

// Compile-time check: Adapter satisfies note.Service.
var _ Service = (*Adapter)(nil)
