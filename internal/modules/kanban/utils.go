package kanban

import (
	"github.com/bytedance/sonic"
)

func unmarshal(data any, v any) error {
	r, err := sonic.Marshal(data)
	if err != nil {
		return err
	}
	return sonic.Unmarshal(r, &v)
}
