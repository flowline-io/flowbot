package homelab

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	require.NotNil(t, r)
	assert.Empty(t, r.List())
}

func TestRegistry_Replace(t *testing.T) {
	r := NewRegistry()

	apps := []App{
		{Name: "archivebox", Path: "/apps/archivebox", Status: AppStatusRunning},
		{Name: "karakeep", Path: "/apps/karakeep", Status: AppStatusStopped},
	}
	r.Replace(apps)

	list := r.List()
	require.Len(t, list, 2)
	assert.Equal(t, "archivebox", list[0].Name)
	assert.Equal(t, "karakeep", list[1].Name)
}

func TestRegistry_ReplaceOverwrites(t *testing.T) {
	r := NewRegistry()

	r.Replace([]App{{Name: "first", Path: "/apps/first"}})
	r.Replace([]App{{Name: "second", Path: "/apps/second"}})

	list := r.List()
	require.Len(t, list, 1)
	assert.Equal(t, "second", list[0].Name)
}

func TestRegistry_ReplaceEmpty(t *testing.T) {
	r := NewRegistry()
	r.Replace([]App{{Name: "test", Path: "/apps/test"}})
	r.Replace([]App{})

	assert.Empty(t, r.List())
}

func TestRegistry_ListSorted(t *testing.T) {
	r := NewRegistry()

	apps := []App{
		{Name: "zulu"},
		{Name: "alpha"},
		{Name: "mike"},
	}
	r.Replace(apps)

	list := r.List()
	require.Len(t, list, 3)
	assert.Equal(t, "alpha", list[0].Name)
	assert.Equal(t, "mike", list[1].Name)
	assert.Equal(t, "zulu", list[2].Name)
}

func TestRegistry_Get(t *testing.T) {
	r := NewRegistry()

	r.Replace([]App{
		{Name: "archivebox", Path: "/apps/archivebox"},
		{Name: "karakeep", Path: "/apps/karakeep"},
	})

	app, ok := r.Get("archivebox")
	assert.True(t, ok)
	assert.Equal(t, "archivebox", app.Name)
	assert.Equal(t, "/apps/archivebox", app.Path)
}

func TestRegistry_GetMissing(t *testing.T) {
	r := NewRegistry()

	_, ok := r.Get("nonexistent")
	assert.False(t, ok)
}

func TestRegistry_GetEmptyRegistry(t *testing.T) {
	r := NewRegistry()

	_, ok := r.Get("anything")
	assert.False(t, ok)
}

func TestRegistry_Permissions(t *testing.T) {
	r := NewRegistry()

	p := Permissions{
		Status:  true,
		Logs:    true,
		Start:   false,
		Stop:    false,
		Restart: false,
		Pull:    false,
		Update:  false,
		Exec:    false,
	}
	r.SetPermissions(p)

	got := r.Permissions()
	assert.Equal(t, p, got)
}

func TestRegistry_PermissionsDefaults(t *testing.T) {
	r := NewRegistry()

	got := r.Permissions()
	assert.False(t, got.Status)
	assert.False(t, got.Logs)
	assert.False(t, got.Start)
	assert.False(t, got.Stop)
	assert.False(t, got.Restart)
	assert.False(t, got.Pull)
	assert.False(t, got.Update)
	assert.False(t, got.Exec)
}

func TestDefaultRegistry(t *testing.T) {
	assert.NotNil(t, DefaultRegistry)
}
