package utils

import "math"

// IntToUint32 converts n to uint32 when it is non-negative and within range.
func IntToUint32(n int) (uint32, bool) {
	if n < 0 || uint64(n) > math.MaxUint32 {
		return 0, false
	}
	return uint32(n), true
}

// IntToInt32 converts n to int32 when it fits in the int32 range.
func IntToInt32(n int) (int32, bool) {
	if n < math.MinInt32 || n > math.MaxInt32 {
		return 0, false
	}
	return int32(n), true
}

// Int64ToInt32 converts n to int32 when it fits in the int32 range.
func Int64ToInt32(n int64) (int32, bool) {
	if n < math.MinInt32 || n > math.MaxInt32 {
		return 0, false
	}
	return int32(n), true
}

// Uint64ToUint32 converts n to uint32 when it fits in the uint32 range.
func Uint64ToUint32(n uint64) (uint32, bool) {
	if n > math.MaxUint32 {
		return 0, false
	}
	return uint32(n), true
}

// Uint64ToInt64 converts n to int64 when it fits in the int64 range.
func Uint64ToInt64(n uint64) (int64, bool) {
	if n > math.MaxInt64 {
		return 0, false
	}
	return int64(n), true
}
