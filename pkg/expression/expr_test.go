package expression

import (
	"fmt"
	"testing"

	"github.com/expr-lang/expr/vm"
)

var lib = map[string]any{
	"toInt": func(v any) int { return v.(int) + 1000 },
}

func TestRun(t *testing.T) {
	LoadEnv("dev", lib)

	//code := `Debug(input.val1 + input.val2 + max(input.val1, input.val2) + lib.toInt(input.val1))`
	code := `dev.toInt(input.val2)`

	LoadEnv("input", map[string]any{
		"val1": 1,
		"val2": 2,
	})

	p, err := Compile(code)
	if err != nil {
		t.Fatal(err)
	}

	type args struct {
		program *vm.Program
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test",
			args: args{
				program: p,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Run(tt.args.program)
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			fmt.Println(got)
		})
	}
}
