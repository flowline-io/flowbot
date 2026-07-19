package utils

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntToUint32(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		in     int
		want   uint32
		wantOK bool
	}{
		{name: "zero", in: 0, want: 0, wantOK: true},
		{name: "max uint32", in: math.MaxUint32, want: math.MaxUint32, wantOK: true},
		{name: "negative", in: -1, wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := IntToUint32(tt.in)
			require.Equal(t, tt.wantOK, ok)
			if tt.wantOK {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestIntToInt32(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		in     int
		want   int32
		wantOK bool
	}{
		{name: "zero", in: 0, want: 0, wantOK: true},
		{name: "max int32", in: math.MaxInt32, want: math.MaxInt32, wantOK: true},
		{name: "overflow", in: math.MaxInt32 + 1, wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := IntToInt32(tt.in)
			require.Equal(t, tt.wantOK, ok)
			if tt.wantOK {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestInt64ToInt32(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		in     int64
		want   int32
		wantOK bool
	}{
		{name: "min int32", in: math.MinInt32, want: math.MinInt32, wantOK: true},
		{name: "max int32", in: math.MaxInt32, want: math.MaxInt32, wantOK: true},
		{name: "underflow", in: math.MinInt32 - 1, wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := Int64ToInt32(tt.in)
			require.Equal(t, tt.wantOK, ok)
			if tt.wantOK {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestUint64ToUint32(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		in     uint64
		want   uint32
		wantOK bool
	}{
		{name: "zero", in: 0, want: 0, wantOK: true},
		{name: "max uint32", in: math.MaxUint32, want: math.MaxUint32, wantOK: true},
		{name: "overflow", in: math.MaxUint32 + 1, wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := Uint64ToUint32(tt.in)
			require.Equal(t, tt.wantOK, ok)
			if tt.wantOK {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestUint64ToInt64(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		in     uint64
		want   int64
		wantOK bool
	}{
		{name: "zero", in: 0, want: 0, wantOK: true},
		{name: "max int64", in: math.MaxInt64, want: math.MaxInt64, wantOK: true},
		{name: "overflow", in: math.MaxInt64 + 1, wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := Uint64ToInt64(tt.in)
			require.Equal(t, tt.wantOK, ok)
			if tt.wantOK {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
