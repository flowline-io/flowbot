package confluence

import (
	"github.com/flowline-io/flowbot/pkg/capability"
	provider "github.com/flowline-io/flowbot/pkg/providers/confluence"
)

func toSpace(s *provider.Space) *capability.ConfluenceSpace {
	if s == nil {
		return nil
	}
	return &capability.ConfluenceSpace{
		ID:   s.ID,
		Key:  s.Key,
		Name: s.Name,
		Type: s.Type,
	}
}

func toSpaces(items []provider.Space) []*capability.ConfluenceSpace {
	out := make([]*capability.ConfluenceSpace, len(items))
	for i := range items {
		out[i] = toSpace(&items[i])
	}
	return out
}

func toPage(p *provider.Page, includeBody bool) *capability.ConfluencePage {
	if p == nil {
		return nil
	}
	page := &capability.ConfluencePage{
		ID:     p.ID,
		Type:   p.Type,
		Status: p.Status,
		Title:  p.Title,
	}
	if p.Space != nil {
		page.SpaceKey = p.Space.Key
	}
	if p.Version != nil {
		page.Version = p.Version.Number
	}
	if p.Links != nil {
		page.WebUI = p.Links.WebUI
	}
	if includeBody && p.Body != nil && p.Body.Storage != nil {
		page.Content = p.Body.Storage.Value
	}
	return page
}

func toPages(items []provider.Page) []*capability.ConfluencePage {
	out := make([]*capability.ConfluencePage, len(items))
	for i := range items {
		out[i] = toPage(&items[i], false)
	}
	return out
}
