package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func foo1() {}
func foo2(a, b string) string {
	return ""
}
func foo3(a, b string, c int) *testing.InternalExample {
	return nil
}

func TestGetFunctionName(t *testing.T) {
	t.Parallel()
	type args struct {
		i any
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "case1",
			args: args{
				i: foo1,
			},
			want: "github.com/flowline-io/flowbot/pkg/utils.foo1",
		},
		{
			name: "case2",
			args: args{
				i: foo2,
			},
			want: "github.com/flowline-io/flowbot/pkg/utils.foo2",
		},
		{
			name: "case3",
			args: args{
				i: foo3,
			},
			want: "github.com/flowline-io/flowbot/pkg/utils.foo3",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, GetFunctionName(tt.args.i))
		})
	}
}

func TestParseFunctionName(t *testing.T) {
	t.Parallel()
	type args struct {
		name string
	}
	tests := []struct {
		name  string
		args  args
		want  string
		want1 string
	}{
		{
			name: "case1",
			args: args{
				name: "github.com/flowline-io/flowbot/pkg/utils.foo1",
			},
			want:  "github.com/flowline-io/flowbot/pkg/utils",
			want1: "foo1",
		},
		{
			name: "case2",
			args: args{
				name: "dev.github.com/flowline-io/flowbot/pkg/utils.foo2",
			},
			want:  "dev.github.com/flowline-io/flowbot/pkg/utils",
			want1: "foo2",
		},
		{
			name: "no_dots_in_name",
			args: args{
				name: "SingleFunc",
			},
			want:  "",
			want1: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, got1 := ParseFunctionName(tt.args.name)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.want1, got1)
		})
	}
}

func FuzzParseFunctionName(f *testing.F) {
	f.Add("pkg.Func")
	f.Add("")
	f.Add("a.b.c.d.Fn")
	f.Add("no_dots")
	f.Add(".")
	f.Add("0.")

	f.Fuzz(func(t *testing.T, name string) {
		pkg, fn := ParseFunctionName(name)
		_ = pkg
		_ = fn
	})
}
