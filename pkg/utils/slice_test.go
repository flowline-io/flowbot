package utils

import "testing"

func TestSameStringSlice(t *testing.T) {
	type args struct {
		x []string
		y []string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "equal",
			args: args{
				x: []string{"a", "b", "c", "d", "e"},
				y: []string{"d", "a", "e", "b", "c"},
			},
			want: true,
		},
		{
			name: "not-equal",
			args: args{
				x: []string{"a", "b", "c", "d", "e"},
				y: []string{"d", "a", "f", "b", "c"},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SameStringSlice(tt.args.x, tt.args.y); got != tt.want {
				t.Errorf("SameStringSlice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInStringSlice(t *testing.T) {
	type args struct {
		x    []string
		find string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "find",
			args: args{
				x:    []string{"a", "b", "c", "d", "e"},
				find: "c",
			},
			want: true,
		},
		{
			name: "not-find",
			args: args{
				x:    []string{"a", "b", "c", "d", "e"},
				find: "f",
			},
			want: false,
		},
		{
			name: "not-find",
			args: args{
				x:    []string{},
				find: "f",
			},
			want: false,
		},
		{
			name: "not-find",
			args: args{
				x:    []string{},
				find: "",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := InStringSlice(tt.args.x, tt.args.find); got != tt.want {
				t.Errorf("InStringSlice() = %v, want %v", got, tt.want)
			}
		})
	}
}
