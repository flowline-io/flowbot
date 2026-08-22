package web

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"

	lifemod "github.com/flowline-io/flowbot/internal/modules/life"
	"github.com/flowline-io/flowbot/pkg/views/pages"
)

func lifeSkillsPage(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid, err := lifeUID(ctx)
	if err != nil {
		return err
	}
	tree, err := lifeService().BuildSkillTree(context.Background(), uid)
	if err != nil {
		return toastError(ctx, lifeUserError(ctx, err))
	}
	pending, err := lifeService().ListQuests(context.Background(), uid, "Pending")
	if err != nil {
		return toastError(ctx, lifeUserError(ctx, err))
	}
	selected := ctx.Query("node")
	if selected == "" {
		selected = tree.DefaultSelected
	}
	data := buildLifeSkillTreeData(ctx.Context(), tree, selected, len(pending))
	ctx.Type("html")
	return pages.LifeSkillsPage(data).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func buildLifeSkillTreeData(ctx context.Context, tree *lifemod.SkillTreeView, selected string, pendingCount int) pages.LifeSkillTreeData {
	if tree == nil {
		return pages.LifeSkillTreeData{}
	}
	roots := make([]pages.LifeSkillTreeNodeRow, 0, len(tree.Roots))
	detail := (*pages.LifeSkillTreeNodeDetail)(nil)
	for _, root := range tree.Roots {
		row, selectedDetail := mapLifeSkillTreeNode(ctx, root, selected)
		roots = append(roots, row)
		if selectedDetail != nil {
			detail = selectedDetail
		}
	}
	if detail == nil && tree.DefaultSelected != "" {
		for _, root := range tree.Roots {
			_, selectedDetail := mapLifeSkillTreeNode(ctx, root, tree.DefaultSelected)
			if selectedDetail != nil {
				detail = selectedDetail
				break
			}
		}
	}
	return pages.LifeSkillTreeData{
		PendingCount:     pendingCount,
		Roots:            roots,
		SelectedNode:     detail,
		ActiveLeafCount:  tree.ActiveNodeCount,
		TrackedLeafCount: tree.TrackedNodeCount,
	}
}

func mapLifeSkillTreeNode(ctx context.Context, node *lifemod.SkillTreeNodeView, selected string) (pages.LifeSkillTreeNodeRow, *pages.LifeSkillTreeNodeDetail) {
	row := pages.LifeSkillTreeNodeRow{
		Key:               node.Key,
		Title:             pages.LifeSkillTreeTitle(ctx, node.Key, node.Title),
		Subtitle:          pages.LifeSkillTreeSubtitle(ctx, node.Key, node.Subtitle),
		Status:            node.Status,
		PracticeCount:     node.PracticeCount,
		SkillCount:        node.SkillCount,
		LastActivityLabel: lifeSkillTreeTimeLabel(node.LastActivityAt),
		Children:          make([]pages.LifeSkillTreeNodeRow, 0, len(node.Children)),
		IsSelected:        node.Key == selected,
	}
	var detail *pages.LifeSkillTreeNodeDetail
	if row.IsSelected {
		evidence := make([]pages.LifeSkillEvidenceRow, 0, len(node.Evidence))
		for _, ev := range node.Evidence {
			evidence = append(evidence, pages.LifeSkillEvidenceRow{
				Title:      ev.Title,
				SourceType: ev.SourceType,
				Detail:     ev.Detail,
				When:       ev.OccurredAt.UTC().Format("2006-01-02 15:04"),
			})
		}
		detail = &pages.LifeSkillTreeNodeDetail{
			Title:             row.Title,
			Subtitle:          row.Subtitle,
			Status:            node.Status,
			PracticeCount:     node.PracticeCount,
			SkillCount:        node.SkillCount,
			LastActivityLabel: row.LastActivityLabel,
			Evidence:          evidence,
		}
	}
	for _, child := range node.Children {
		childRow, childDetail := mapLifeSkillTreeNode(ctx, child, selected)
		row.Children = append(row.Children, childRow)
		if detail == nil && childDetail != nil {
			detail = childDetail
		}
	}
	return row, detail
}

func lifeSkillTreeTimeLabel(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.UTC().Format("2006-01-02")
}
