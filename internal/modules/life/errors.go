package life

import (
	"fmt"

	"github.com/flowline-io/flowbot/pkg/types"
)

func lifeNotFound(format string, args ...any) error {
	return &types.Error{Kind: types.ErrNotFound, Message: "life: " + fmt.Sprintf(format, args...)}
}

func lifeInvalid(format string, args ...any) error {
	return &types.Error{Kind: types.ErrInvalidArgument, Message: "life: " + fmt.Sprintf(format, args...)}
}

func lifeConflict(format string, args ...any) error {
	return &types.Error{Kind: types.ErrConflict, Message: "life: " + fmt.Sprintf(format, args...)}
}

func lifeUnavailable(format string, args ...any) error {
	return &types.Error{Kind: types.ErrUnavailable, Message: "life: " + fmt.Sprintf(format, args...)}
}
