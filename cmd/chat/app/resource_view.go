package app

import (
	"fmt"
	"strings"

	"github.com/flowline-io/flowbot/pkg/client"
)

// ResourceOverlay holds the full-screen resource preview state.
type ResourceOverlay struct {
	URI     string
	Title   string
	Content string
}

func formatResourcesHint(resources []client.ChatResourceRef) string {
	if len(resources) == 0 {
		return ""
	}
	parts := make([]string, 0, len(resources))
	for _, ref := range resources {
		label := ref.Title
		if label == "" {
			label = ref.URI
		}
		parts = append(parts, fmt.Sprintf("%s (%s) — /open %s", ref.URI, label, ref.URI))
	}
	return "Resources: " + strings.Join(parts, " · ")
}

func (m *Model) openResourceOverlay(resource client.ChatResource) {
	m.resourceOverlay = &ResourceOverlay{
		URI:     resource.URI,
		Title:   resource.Title,
		Content: RenderMarkdown(resource.Content, m.width),
	}
}

func (m *Model) closeResourceOverlay() {
	m.resourceOverlay = nil
}

func (m *Model) renderResourceOverlay() string {
	if m.resourceOverlay == nil || m.width <= 0 || m.height <= 0 {
		return ""
	}
	var b strings.Builder
	writeBuilder(&b, m.styles.UserMsg.Render(" Resource: "+m.resourceOverlay.Title)+" ")
	writeBuilder(&b, m.styles.Hint.Render(m.resourceOverlay.URI))
	writeBuilder(&b, "\n")
	writeBuilder(&b, m.styles.Hint.Render("Esc close"))
	writeBuilder(&b, "\n\n")
	body := m.resourceOverlay.Content
	if strings.TrimSpace(body) == "" {
		body = "(empty)"
	}
	maxLines := max(m.height-6, 3)
	lines := strings.Split(body, "\n")
	if len(lines) > maxLines {
		lines = lines[:maxLines]
		writeBuilder(&b, strings.Join(lines, "\n"))
		writeBuilder(&b, m.styles.Hint.Render("\n...(truncated)"))
	} else {
		writeBuilder(&b, body)
	}
	return b.String()
}

func mergeResourceRefs(existing, incoming []client.ChatResourceRef) []client.ChatResourceRef {
	if len(incoming) == 0 {
		return existing
	}
	seen := make(map[string]struct{}, len(existing)+len(incoming))
	out := make([]client.ChatResourceRef, 0, len(existing)+len(incoming))
	for _, ref := range existing {
		if ref.URI == "" {
			continue
		}
		if _, ok := seen[ref.URI]; ok {
			continue
		}
		seen[ref.URI] = struct{}{}
		out = append(out, ref)
	}
	for _, ref := range incoming {
		if ref.URI == "" {
			continue
		}
		if _, ok := seen[ref.URI]; ok {
			continue
		}
		seen[ref.URI] = struct{}{}
		out = append(out, ref)
	}
	return out
}

func planSummariesToRefs(plans []client.ChatPlanSummary) []client.ChatResourceRef {
	if len(plans) == 0 {
		return nil
	}
	refs := make([]client.ChatResourceRef, 0, len(plans))
	for _, plan := range plans {
		refs = append(refs, client.ChatResourceRef{
			URI:   plan.URI,
			Kind:  "plan",
			Title: plan.Title,
		})
	}
	return refs
}
