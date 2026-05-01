package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKV_String(t *testing.T) {
	kv := KV{"key": "value", "num": 42}
	val, ok := kv.String("key")
	assert.True(t, ok)
	assert.Equal(t, "value", val)

	_, ok = kv.String("num")
	assert.False(t, ok)

	_, ok = kv.String("missing")
	assert.False(t, ok)
}

func TestKV_Int64(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  int64
		ok    bool
	}{
		{"int", int(42), 42, true},
		{"int8", int8(8), 8, true},
		{"int32", int32(32), 32, true},
		{"int64", int64(100), 100, true},
		{"uint", uint(10), 10, true},
		{"float32", float32(3.0), 3, true},
		{"float64", float64(99.9), 99, true},
		{"string", "42", 0, false},
		{"nil", nil, 0, false},
		{"missing", nil, 0, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			kv := KV{}
			if tc.name != "missing" {
				kv["key"] = tc.value
			}
			val, ok := kv.Int64("key")
			assert.Equal(t, tc.ok, ok)
			if ok {
				assert.Equal(t, tc.want, val)
			}
		})
	}
}

func TestKV_Uint64(t *testing.T) {
	kv := KV{"key": float64(42)}
	val, ok := kv.Uint64("key")
	assert.True(t, ok)
	assert.Equal(t, uint64(42), val)

	_, ok = kv.Uint64("missing")
	assert.False(t, ok)

	kv["str"] = "42"
	_, ok = kv.Uint64("str")
	assert.False(t, ok)
}

func TestKV_Float64(t *testing.T) {
	kv := KV{"key": float64(3.14)}
	val, ok := kv.Float64("key")
	assert.True(t, ok)
	assert.Equal(t, float64(3.14), val)

	_, ok = kv.Float64("missing")
	assert.False(t, ok)

	kv["int"] = 42
	_, ok = kv.Float64("int")
	assert.False(t, ok)
}

func TestKV_Map(t *testing.T) {
	nested := map[string]any{"a": "b"}
	kv := KV{"key": nested}
	val, ok := kv.Map("key")
	assert.True(t, ok)
	assert.Equal(t, nested, val)

	_, ok = kv.Map("missing")
	assert.False(t, ok)

	kv["wrong"] = "string"
	_, ok = kv.Map("wrong")
	assert.False(t, ok)
}

func TestKV_Any(t *testing.T) {
	kv := KV{"key": "value", "num": 42}
	val, ok := kv.Any("key")
	assert.True(t, ok)
	assert.Equal(t, "value", val)

	val, ok = kv.Any("num")
	assert.True(t, ok)
	assert.Equal(t, 42, val)

	_, ok = kv.Any("missing")
	assert.False(t, ok)
}

func TestKV_List(t *testing.T) {
	kv := KV{"key": []any{"a", "b"}}
	val, ok := kv.List("key")
	assert.True(t, ok)
	assert.Equal(t, []any{"a", "b"}, val)

	_, ok = kv.List("missing")
	assert.False(t, ok)

	kv["wrong"] = "string"
	_, ok = kv.List("wrong")
	assert.False(t, ok)
}

func TestKV_StringValue(t *testing.T) {
	kv := KV{"value": "hello"}
	val, ok := kv.StringValue()
	assert.True(t, ok)
	assert.Equal(t, "hello", val)

	_, ok = KV{}.StringValue()
	assert.False(t, ok)
}

func TestKV_Int64Value(t *testing.T) {
	kv := KV{"value": int64(64)}
	val, ok := kv.Int64Value()
	assert.True(t, ok)
	assert.Equal(t, int64(64), val)

	_, ok = KV{}.Int64Value()
	assert.False(t, ok)
}

func TestKV_Uint64Value(t *testing.T) {
	kv := KV{"value": float64(100)}
	val, ok := kv.Uint64Value()
	assert.True(t, ok)
	assert.Equal(t, uint64(100), val)

	_, ok = KV{}.Uint64Value()
	assert.False(t, ok)
}

func TestKV_Float64Value(t *testing.T) {
	kv := KV{"value": float64(2.71)}
	val, ok := kv.Float64Value()
	assert.True(t, ok)
	assert.Equal(t, float64(2.71), val)

	_, ok = KV{}.Float64Value()
	assert.False(t, ok)
}

func TestKV_Merge_Simple(t *testing.T) {
	a := KV{"x": "1"}
	b := KV{"y": "2"}
	result := a.Merge(b)
	assert.Equal(t, "1", result["x"])
	assert.Equal(t, "2", result["y"])
}

func TestKV_Merge_Override(t *testing.T) {
	a := KV{"x": "old"}
	b := KV{"x": "new"}
	result := a.Merge(b)
	assert.Equal(t, "new", result["x"])
}

func TestKV_Merge_Nested(t *testing.T) {
	a := KV{"nested": map[string]any{"a": 1}}
	b := KV{"nested": map[string]any{"b": 2}}
	result := a.Merge(b)
	m, ok := result["nested"].(KV)
	require.True(t, ok)
	assert.Equal(t, 1, m["a"])
	assert.Equal(t, 2, m["b"])
}

func TestKV_Merge_Lists(t *testing.T) {
	a := KV{"items": []any{"a", "b"}}
	b := KV{"items": []any{"c"}}
	result := a.Merge(b)
	list := result["items"].([]any)
	assert.Equal(t, []any{"a", "b", "c"}, list)
}

func TestKV_Merge_ListNil(t *testing.T) {
	a := KV{"items": nil}
	b := KV{"items": []any{"a"}}
	result := a.Merge(b)
	list := result["items"].([]any)
	assert.Equal(t, []any{"a"}, list)
}

func TestKV_Merge_TypeMismatch(t *testing.T) {
	a := KV{"x": "string"}
	b := KV{"x": []any{"a"}}
	result := a.Merge(b)
	assert.Equal(t, "string", result["x"])
}

func TestKV_Merge_TypeMismatchMap(t *testing.T) {
	a := KV{"x": "string"}
	b := KV{"x": map[string]any{"a": 1}}
	result := a.Merge(b)
	assert.Equal(t, "string", result["x"])
}

func TestKV_Scan_ValidJSON(t *testing.T) {
	var kv KV
	err := kv.Scan([]byte(`{"key": "value", "num": 42}`))
	require.NoError(t, err)
	assert.Equal(t, "value", kv["key"])
}

func TestKV_Scan_InvalidJSON(t *testing.T) {
	var kv KV
	err := kv.Scan([]byte(`{invalid`))
	assert.Error(t, err)
}

func TestKV_Scan_MapType(t *testing.T) {
	var kv KV
	err := kv.Scan(map[string]any{"key": "value"})
	require.NoError(t, err)
	assert.Equal(t, "value", kv["key"])
}

func TestKV_Scan_UnknownType(t *testing.T) {
	var kv KV
	err := kv.Scan(42)
	assert.Error(t, err)
}

func TestKV_Value_Empty(t *testing.T) {
	v, err := (KV{}).Value()
	require.NoError(t, err)
	assert.Nil(t, v)
}

func TestKV_Value_Populated(t *testing.T) {
	kv := KV{"key": "value"}
	v, err := kv.Value()
	require.NoError(t, err)
	assert.NotNil(t, v)
	assert.Contains(t, string(v.([]byte)), "key")
}
