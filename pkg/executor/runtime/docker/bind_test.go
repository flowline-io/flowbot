package docker

import (
	"context"
	"os"
	"path"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/utils"
)

func TestMountBindNotAllowed(t *testing.T) {
	t.Parallel()

	t.Run("mount bind when not allowed returns error", func(t *testing.T) {
		t.Parallel()
		m := &BindMounter{cfg: BindConfig{
			Allowed: false,
		}}

		err := m.Mount(context.Background(), &types.Mount{
			Type:   types.MountTypeBind,
			Source: "/tmp",
			Target: "/somevol",
		})
		assert.Error(t, err)
	})
}

func TestMountCreate(t *testing.T) {
	t.Parallel()

	t.Run("concurrent bind mounts succeed", func(t *testing.T) {
		t.Parallel()
		m := NewBindMounter(BindConfig{
			Allowed: true,
		})
		dir := path.Join(os.TempDir(), utils.NewUUID())
		wg := sync.WaitGroup{}
		c := 10
		wg.Add(c)
		for range c {
			go func() {
				defer wg.Done()
				err := m.Mount(context.Background(), &types.Mount{
					Type:   types.MountTypeBind,
					Source: dir,
					Target: "/somevol",
				})
				assert.NoError(t, err)
			}()
		}
		wg.Wait()
	})
}
