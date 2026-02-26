package utils

import "testing"

func foo1() {}
func foo2(a, b string) string {
	return ""
}
func foo3(a, b string, c int) *testing.InternalExample {
	return nil
}

func TestGetFunctionName(t *testing.T) {
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
			if got := GetFunctionName(tt.args.i); got != tt.want {
				t.Errorf("GetFunctionName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseFunctionName(t *testing.T) {
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := ParseFunctionName(tt.args.name)
			if got != tt.want {
				t.Errorf("ParseFunctionName() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("ParseFunctionName() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
