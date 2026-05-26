package platforms

import (
	"context"
	"errors"
	"fmt"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/types"
)

var callers = make(map[string]*Caller)

func PlatformRegister(name string, caller *Caller) error {
	_, err := store.Database.GetPlatformByName(context.Background(), name)
	if err != nil && !errors.Is(err, types.ErrNotFound) {
		return err
	}
	if errors.Is(err, types.ErrNotFound) {
		_, err = store.Database.CreatePlatform(context.Background(), &gen.Platform{
			Name: name,
		})
		if err != nil {
			return fmt.Errorf("failed to create platform %s, %w", name, err)
		}
	}
	callers[name] = caller
	return nil
}

func GetCaller(name string) (*Caller, error) {
	if c, ok := callers[name]; ok {
		return c, nil
	}
	return nil, errors.New("caller not found")
}
