package sdk

import (
	"encoding/json"
	"fmt"
)

// Module is the interface for module plugins.
type Module interface {
	Init(config json.RawMessage) error
	Bootstrap() error
	Command(ctx *Context, content any) (*MsgPayload, error)
	Form(ctx *Context, values map[string]string) (*MsgPayload, error)
	Rules() (*Rules, error)
	Help() (map[string][]string, error)
	IsReady() bool
}

// ModuleBase provides no-op default implementations for Module.
type ModuleBase struct{}

func (ModuleBase) Init(_ json.RawMessage) error { return nil }
func (ModuleBase) Bootstrap() error             { return nil }
func (ModuleBase) Command(_ *Context, _ any) (*MsgPayload, error) {
	return nil, fmt.Errorf("command not implemented")
}
func (ModuleBase) Form(_ *Context, _ map[string]string) (*MsgPayload, error) {
	return nil, fmt.Errorf("form not implemented")
}
func (ModuleBase) Rules() (*Rules, error)             { return &Rules{}, nil }
func (ModuleBase) Help() (map[string][]string, error) { return nil, nil }
func (ModuleBase) IsReady() bool                      { return true }
