package functions

import (
	"io"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/pkg/types"
)

// ParseStdoutJSON trims stdout and decodes exactly one JSON value (≤ MaxJSONBytes).
func ParseStdoutJSON(stdout string) (any, error) {
	trimmed := strings.TrimSpace(stdout)
	if trimmed == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "stdout is empty")
	}
	if len(trimmed) > MaxJSONBytes {
		return nil, types.Errorf(types.ErrInvalidArgument, "stdout exceeds %d bytes", MaxJSONBytes)
	}
	dec := sonic.ConfigDefault.NewDecoder(strings.NewReader(trimmed))
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, types.WrapError(types.ErrInvalidArgument, "stdout is not valid JSON", err)
	}
	var extra any
	err := dec.Decode(&extra)
	if err == nil {
		return nil, types.Errorf(types.ErrInvalidArgument, "stdout must contain exactly one JSON value")
	}
	if err != io.EOF {
		return nil, types.Errorf(types.ErrInvalidArgument, "stdout must contain exactly one JSON value")
	}
	return v, nil
}
