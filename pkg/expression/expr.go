package expression

import (
	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

func Compile(code string) (*vm.Program, error) {
	program, err := expr.Compile(code, expr.Env(globalEnv), expr.WithContext("ctx"))
	if err != nil {
		return nil, err
	}
	return program, nil
}

func Run(program *vm.Program) (interface{}, error) {
	result, err := expr.Run(program, globalEnv)
	if err != nil {
		return nil, err
	}
	return result, nil
}
