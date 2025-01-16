package kanban

import json "github.com/json-iterator/go"

func unmarshal(data any, v any) error {
	r, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(r, &v)
}
