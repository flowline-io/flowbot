package docker

import (
	"context"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/stretchr/testify/assert"
	"os"
	"path"
	"sync"
	"testing"
)

func TestMountBindNotAllowed(t *testing.T) {
	m := &BindMounter{cfg: BindConfig{
		Allowed: false,
	}}

	err := m.Mount(context.Background(), &types.Mount{
		Type:   types.MountTypeBind,
		Source: "/tmp",
		Target: "/somevol",
	})
	assert.Error(t, err)
}

func TestMountCreate(t *testing.T) {
	m := NewBindMounter(BindConfig{
		Allowed: true,
	})
	dir := path.Join(os.TempDir(), utils.NewUUID())
	wg := sync.WaitGroup{}
	c := 10
	wg.Add(c)
	for i := 0; i < c; i++ {
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
}
