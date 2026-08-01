package life

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/sqlitetest"
	"github.com/flowline-io/flowbot/pkg/types"
)

func TestGoalContextTitle(t *testing.T) {
	t.Parallel()
	area := &gen.LifeGoal{ID: 10, Title: "Health", Category: "Area"}
	byID := map[int64]*gen.LifeGoal{10: area}
	tests := []struct {
		name string
		g    *gen.LifeGoal
		want string
	}{
		{name: "nil goal", g: nil, want: ""},
		{name: "no area", g: &gen.LifeGoal{Title: "Solo", Category: "Project"}, want: "Solo"},
		{
			name: "with area",
			g:    &gen.LifeGoal{Title: "Run 5k", Category: "Project", AreaID: new(int64(10))},
			want: "Run 5k · Health",
		},
		{
			name: "missing area falls back",
			g:    &gen.LifeGoal{Title: "Orphan", Category: "Project", AreaID: new(int64(99))},
			want: "Orphan",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, goalContextTitle(tt.g, byID))
		})
	}
}

func TestMapGoalViewsResolvesArea(t *testing.T) {
	t.Parallel()
	areaID := int64(10)
	views := MapGoalViews([]*gen.LifeGoal{
		{ID: 10, Flag: "area-1", Title: "Health", Category: "Area", Status: "Active"},
		{ID: 11, Flag: "proj-1", Title: "Run 5k", Category: "Project", Status: "Active", AreaID: &areaID},
		{ID: 12, Flag: "proj-2", Title: "Orphan", Category: "Project", Status: "Active", AreaID: new(int64(99))},
	})
	require.Len(t, views, 3)
	assert.Empty(t, views[0].AreaFlag)
	assert.Equal(t, "area-1", views[1].AreaFlag)
	assert.Equal(t, "Health", views[1].AreaTitle)
	assert.Empty(t, views[2].AreaFlag)
}

//go:fix inline
func int64Ptr(v int64) *int64 { return new(v) }

func TestCreateGoalWithOptionalArea(t *testing.T) {
	t.Parallel()
	client := sqlitetest.OpenClient(t, t.Name())
	svc := NewService(store.NewLifeStore(client))
	ctx := context.Background()
	uid := "goal-area-create-user"

	area, err := svc.CreateGoal(ctx, uid, "Health", "Area", "")
	require.NoError(t, err)
	project, err := svc.CreateGoal(ctx, uid, "Run 5k", "Project", area.Flag)
	require.NoError(t, err)
	require.NotNil(t, project.AreaID)
	assert.Equal(t, area.ID, *project.AreaID)

	resource, err := svc.CreateGoal(ctx, uid, "Notes", "Resource", area.Flag)
	require.NoError(t, err)
	require.NotNil(t, resource.AreaID)

	solo, err := svc.CreateGoal(ctx, uid, "Solo", "Project", "")
	require.NoError(t, err)
	assert.Nil(t, solo.AreaID)

	_, err = svc.CreateGoal(ctx, uid, "Bad", "Project", "missing-area")
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrInvalidArgument)

	paused, err := svc.CreateGoal(ctx, uid, "Paused Area", "Area", "")
	require.NoError(t, err)
	require.NoError(t, svc.SetGoalStatus(ctx, uid, paused.Flag, "Paused"))
	_, err = svc.CreateGoal(ctx, uid, "Onto paused", "Project", paused.Flag)
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrInvalidArgument)
}

func TestDeleteAreaClearsChildLinks(t *testing.T) {
	t.Parallel()
	client := sqlitetest.OpenClient(t, t.Name())
	svc := NewService(store.NewLifeStore(client))
	ctx := context.Background()
	uid := "goal-area-delete-user"

	area, err := svc.CreateGoal(ctx, uid, "Career", "Area", "")
	require.NoError(t, err)
	project, err := svc.CreateGoal(ctx, uid, "Ship v1", "Project", area.Flag)
	require.NoError(t, err)
	require.NoError(t, svc.DeleteGoal(ctx, uid, area.Flag))

	got, err := svc.store.GetGoalByFlag(ctx, project.LifeProfileID, project.Flag)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Nil(t, got.AreaID)
}

func TestUpdateGoalCategoryClearsAreaEdges(t *testing.T) {
	t.Parallel()
	client := sqlitetest.OpenClient(t, t.Name())
	svc := NewService(store.NewLifeStore(client))
	ctx := context.Background()
	uid := "goal-area-update-user"

	area, err := svc.CreateGoal(ctx, uid, "Learning", "Area", "")
	require.NoError(t, err)
	project, err := svc.CreateGoal(ctx, uid, "Course", "Project", area.Flag)
	require.NoError(t, err)

	require.NoError(t, svc.UpdateGoal(ctx, uid, area.Flag, "Learning", "Project", ""))
	got, err := svc.store.GetGoalByFlag(ctx, project.LifeProfileID, project.Flag)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Nil(t, got.AreaID)

	area2, err := svc.CreateGoal(ctx, uid, "Fitness", "Area", "")
	require.NoError(t, err)
	require.NoError(t, svc.UpdateGoal(ctx, uid, project.Flag, "Course", "Project", area2.Flag))
	got, err = svc.store.GetGoalByFlag(ctx, project.LifeProfileID, project.Flag)
	require.NoError(t, err)
	require.NotNil(t, got.AreaID)
	assert.Equal(t, area2.ID, *got.AreaID)

	require.NoError(t, svc.UpdateGoal(ctx, uid, project.Flag, "Course", "Area", area2.Flag))
	got, err = svc.store.GetGoalByFlag(ctx, project.LifeProfileID, project.Flag)
	require.NoError(t, err)
	assert.Equal(t, "Area", got.Category)
	assert.Nil(t, got.AreaID)
}
