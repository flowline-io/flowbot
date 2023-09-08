package utils

import (
	"testing"
)

func TestFileExist(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "file_exist",
			args: args{
				name: "./string.go",
			},
			want: true,
		},
		{
			name: "file_not_exist",
			args: args{
				name: "./not_exist.test",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FileExist(tt.args.name); got != tt.want {
				t.Errorf("FileExist() = %v, want %v", got, tt.want)
			}
		})
	}
}
