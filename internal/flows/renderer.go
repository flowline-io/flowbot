package flows

import (
	"fmt"
	"strings"

	"github.com/flowline-io/flowbot/pkg/types"
)

type TemplateRenderer interface {
	Render(s string, vars types.KV) string
}

type SimpleTemplateRenderer struct{}

func NewSimpleTemplateRenderer() TemplateRenderer {
	return &SimpleTemplateRenderer{}
}

func (r *SimpleTemplateRenderer) Render(s string, vars types.KV) string {
	if s == "" || len(vars) == 0 {
		return s
	}
	out := s
	for k, v := range vars {
		out = strings.ReplaceAll(out, "{{"+k+"}}", fmt.Sprint(v))
	}
	return out
}
