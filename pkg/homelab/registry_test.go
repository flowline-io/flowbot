package homelab

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRegistry(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "new registry is non-nil and empty"},
		{name: "new registry returns not found for any name"},
		{name: "multiple registries are independent"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := NewRegistry()
			require.NotNil(t, r)
			assert.Empty(t, r.List())

			switch tt.name {
			case "new registry returns not found for any name":
				_, ok := r.Get("anything")
				assert.False(t, ok)
			case "multiple registries are independent":
				r2 := NewRegistry()
				require.NotNil(t, r2)
				assert.NotSame(t, r, r2)
				r2.Replace([]App{{Name: "only-in-r2"}})
				assert.Len(t, r2.List(), 1)
				assert.Empty(t, r.List())
			}
		})
	}
}

func TestRegistry_Replace(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		apps     []App
		wantLen  int
		wantName string
	}{
		{
			name: "replace stores and returns apps in order",
			apps: []App{
				{Name: "archivebox", Path: "/apps/archivebox", Status: AppStatusRunning},
				{Name: "karakeep", Path: "/apps/karakeep", Status: AppStatusStopped},
			},
			wantLen:  2,
			wantName: "karakeep",
		},
		{
			name: "replace with single app",
			apps: []App{
				{Name: "lone", Path: "/apps/lone"},
			},
			wantLen:  1,
			wantName: "lone",
		},
		{
			name:    "replace with nil slice clears",
			apps:    nil,
			wantLen: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := NewRegistry()
			r.Replace(tt.apps)

			list := r.List()
			require.Len(t, list, tt.wantLen)
			if tt.wantLen > 0 {
				assert.Equal(t, tt.wantName, list[tt.wantLen-1].Name)
			}
		})
	}
}

func TestRegistry_ReplaceOverwrites(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		first    []App
		second   []App
		wantLen  int
		wantName string
	}{
		{
			name:     "second replace overwrites first",
			first:    []App{{Name: "first", Path: "/apps/first"}},
			second:   []App{{Name: "second", Path: "/apps/second"}},
			wantLen:  1,
			wantName: "second",
		},
		{
			name:     "second replace with different app count",
			first:    []App{{Name: "a"}, {Name: "b"}},
			second:   []App{{Name: "x"}, {Name: "y"}, {Name: "z"}},
			wantLen:  3,
			wantName: "z",
		},
		{
			name:     "replace after nil replace",
			first:    []App{{Name: "original"}},
			second:   nil,
			wantLen:  0,
			wantName: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := NewRegistry()

			r.Replace(tt.first)
			r.Replace(tt.second)

			list := r.List()
			require.Len(t, list, tt.wantLen)
			if tt.wantLen > 0 {
				assert.Equal(t, tt.wantName, list[tt.wantLen-1].Name)
			}
		})
	}
}

func TestRegistry_ReplaceEmpty(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		initial []App
		clear   []App
		wantErr bool
	}{
		{
			name:    "replace with empty slice clears registry",
			initial: []App{{Name: "test", Path: "/apps/test"}},
			clear:   []App{},
		},
		{
			name:    "replace with nil slice also clears",
			initial: []App{{Name: "a"}, {Name: "b"}, {Name: "c"}},
			clear:   nil,
		},
		{
			name:    "replace empty after empty registry",
			initial: []App{},
			clear:   []App{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := NewRegistry()
			r.Replace(tt.initial)
			r.Replace(tt.clear)

			assert.Empty(t, r.List())
		})
	}
}

func TestRegistry_ListSorted(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		apps []App
		want []string
	}{
		{
			name: "list returns apps sorted by name",
			apps: []App{
				{Name: "zulu"},
				{Name: "alpha"},
				{Name: "mike"},
			},
			want: []string{"alpha", "mike", "zulu"},
		},
		{
			name: "list on empty registry returns empty",
			apps: nil,
			want: []string{},
		},
		{
			name: "list with single app returns that app",
			apps: []App{{Name: "single"}},
			want: []string{"single"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := NewRegistry()
			r.Replace(tt.apps)

			list := r.List()
			require.Len(t, list, len(tt.want))
			for i, name := range tt.want {
				assert.Equal(t, name, list[i].Name)
			}
		})
	}
}

func TestRegistry_Get(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		apps    []App
		getName string
		wantOk  bool
		wantApp App
	}{
		{
			name: "get existing app returns it",
			apps: []App{
				{Name: "archivebox", Path: "/apps/archivebox"},
				{Name: "karakeep", Path: "/apps/karakeep"},
			},
			getName: "archivebox",
			wantOk:  true,
			wantApp: App{Name: "archivebox", Path: "/apps/archivebox"},
		},
		{
			name: "get multiple apps by different names",
			apps: []App{
				{Name: "a", Path: "/a"},
				{Name: "b", Path: "/b"},
				{Name: "c", Path: "/c"},
			},
			getName: "b",
			wantOk:  true,
			wantApp: App{Name: "b", Path: "/b"},
		},
		{
			name: "get app after replace with updated values",
			apps: []App{
				{Name: "update-me", Path: "/old/path"},
			},
			getName: "update-me",
			wantOk:  true,
			wantApp: App{Name: "update-me", Path: "/new/path"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := NewRegistry()
			if tt.name == "get app after replace with updated values" {
				r.Replace(tt.apps)
				r.Replace([]App{{Name: "update-me", Path: "/new/path"}})
			} else {
				r.Replace(tt.apps)
			}

			app, ok := r.Get(tt.getName)
			assert.Equal(t, tt.wantOk, ok)
			assert.Equal(t, tt.wantApp.Name, app.Name)
			assert.Equal(t, tt.wantApp.Path, app.Path)
		})
	}
}

func TestRegistry_GetMissing(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		apps    []App
		getName string
	}{
		{
			name:    "get missing app returns not found",
			getName: "nonexistent",
		},
		{
			name:    "get missing app with empty name",
			getName: "",
		},
		{
			name: "get missing app among existing apps",
			apps: []App{
				{Name: "real-app-1"},
				{Name: "real-app-2"},
			},
			getName: "not-here",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := NewRegistry()
			if len(tt.apps) > 0 {
				r.Replace(tt.apps)
			}

			_, ok := r.Get(tt.getName)
			assert.False(t, ok)
		})
	}
}

func TestRegistry_GetEmptyRegistry(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		getName string
	}{
		{name: "get on empty registry returns not found", getName: "anything"},
		{name: "get on empty registry with empty name", getName: ""},
		{name: "get on empty registry after previous replace cleared it", getName: "gone"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := NewRegistry()

			if tt.name == "get on empty registry after previous replace cleared it" {
				r.Replace([]App{{Name: "gone", Path: "/gone"}})
				r.Replace([]App{})
			}

			_, ok := r.Get(tt.getName)
			assert.False(t, ok)
		})
	}
}

func TestRegistry_Permissions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		perms1    Permissions
		perms2    Permissions
		want      Permissions
		setSecond bool
	}{
		{
			name: "set and get permissions",
			perms1: Permissions{
				Status:  true,
				Logs:    true,
				Start:   false,
				Stop:    false,
				Restart: false,
				Pull:    false,
				Update:  false,
				Exec:    false,
			},
			want: Permissions{
				Status:  true,
				Logs:    true,
				Start:   false,
				Stop:    false,
				Restart: false,
				Pull:    false,
				Update:  false,
				Exec:    false,
			},
		},
		{
			name:      "set permissions multiple times last wins",
			perms1:    Permissions{Status: true, Logs: true},
			perms2:    Permissions{Status: false, Exec: true},
			want:      Permissions{Status: false, Exec: true},
			setSecond: true,
		},
		{
			name:   "set all-true permissions",
			perms1: Permissions{Status: true, Logs: true, Start: true, Stop: true, Restart: true, Pull: true, Update: true, Exec: true},
			want:   Permissions{Status: true, Logs: true, Start: true, Stop: true, Restart: true, Pull: true, Update: true, Exec: true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := NewRegistry()

			r.SetPermissions(tt.perms1)
			if tt.setSecond {
				r.SetPermissions(tt.perms2)
			}

			got := r.Permissions()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRegistry_PermissionsDefaults(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "default permissions are all false"},
		{name: "permissions are independent between registries"},
		{name: "permissions after zero-value SetPermissions are all false"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := NewRegistry()

			switch tt.name {
			case "default permissions are all false":
				got := r.Permissions()
				assert.False(t, got.Status)
				assert.False(t, got.Logs)
				assert.False(t, got.Start)
				assert.False(t, got.Stop)
				assert.False(t, got.Restart)
				assert.False(t, got.Pull)
				assert.False(t, got.Update)
				assert.False(t, got.Exec)
			case "permissions are independent between registries":
				r2 := NewRegistry()
				r.SetPermissions(Permissions{Status: true})
				got2 := r2.Permissions()
				assert.False(t, got2.Status)
				got1 := r.Permissions()
				assert.True(t, got1.Status)
			case "permissions after zero-value SetPermissions are all false":
				r.SetPermissions(Permissions{Status: true, Logs: true})
				r.SetPermissions(Permissions{})
				got := r.Permissions()
				assert.False(t, got.Status)
				assert.False(t, got.Logs)
			}
		})
	}
}

func TestDefaultRegistry(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "default registry is non-nil"},
		{name: "default registry is empty initially"},
		{name: "default registry supports Get and List"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.NotNil(t, DefaultRegistry)

			switch tt.name {
			case "default registry is empty initially":
				assert.Empty(t, DefaultRegistry.List())
			case "default registry supports Get and List":
				_, ok := DefaultRegistry.Get("_nonexistent_")
				assert.False(t, ok)
				assert.NotNil(t, DefaultRegistry.List())
			}
		})
	}
}
