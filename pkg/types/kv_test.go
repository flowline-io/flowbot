package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKV_String(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		kv      KV
		key     string
		wantVal string
		wantOk  bool
	}{
		{
			name:    "string key exists",
			kv:      KV{"key": "value", "num": 42},
			key:     "key",
			wantVal: "value",
			wantOk:  true,
		},
		{
			name:    "non-string value returns false",
			kv:      KV{"key": "value", "num": 42},
			key:     "num",
			wantVal: "",
			wantOk:  false,
		},
		{
			name:    "missing key returns false",
			kv:      KV{"key": "value", "num": 42},
			key:     "missing",
			wantVal: "",
			wantOk:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			val, ok := tt.kv.String(tt.key)
			assert.Equal(t, tt.wantOk, ok)
			if ok {
				assert.Equal(t, tt.wantVal, val)
			}
		})
	}
}

func TestKV_Int64(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
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
	t.Parallel()
	tests := []struct {
		name    string
		kv      KV
		key     string
		wantVal uint64
		wantOk  bool
	}{
		{
			name:    "float64 value converts to uint64",
			kv:      KV{"key": float64(42)},
			key:     "key",
			wantVal: uint64(42),
			wantOk:  true,
		},
		{
			name:    "missing key returns false",
			kv:      KV{"key": float64(42)},
			key:     "missing",
			wantVal: 0,
			wantOk:  false,
		},
		{
			name:    "string value returns false",
			kv:      KV{"key": float64(42), "str": "42"},
			key:     "str",
			wantVal: 0,
			wantOk:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			val, ok := tt.kv.Uint64(tt.key)
			assert.Equal(t, tt.wantOk, ok)
			if ok {
				assert.Equal(t, tt.wantVal, val)
			}
		})
	}
}

func TestKV_Float64(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		kv      KV
		key     string
		wantVal float64
		wantOk  bool
	}{
		{
			name:    "float64 value",
			kv:      KV{"key": float64(3.14)},
			key:     "key",
			wantVal: float64(3.14),
			wantOk:  true,
		},
		{
			name:    "missing key returns false",
			kv:      KV{"key": float64(3.14)},
			key:     "missing",
			wantVal: 0,
			wantOk:  false,
		},
		{
			name:    "int value returns false",
			kv:      KV{"key": float64(3.14), "int": 42},
			key:     "int",
			wantVal: 0,
			wantOk:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			val, ok := tt.kv.Float64(tt.key)
			assert.Equal(t, tt.wantOk, ok)
			if ok {
				assert.InEpsilon(t, tt.wantVal, val, 0.001)
			}
		})
	}
}

func TestKV_Map(t *testing.T) {
	t.Parallel()
	nested := map[string]any{"a": "b"}

	tests := []struct {
		name    string
		kv      KV
		key     string
		wantVal map[string]any
		wantOk  bool
	}{
		{
			name:    "map value",
			kv:      KV{"key": nested},
			key:     "key",
			wantVal: nested,
			wantOk:  true,
		},
		{
			name:    "missing key returns false",
			kv:      KV{"key": nested},
			key:     "missing",
			wantVal: nil,
			wantOk:  false,
		},
		{
			name:    "non-map value returns false",
			kv:      KV{"key": nested, "wrong": "string"},
			key:     "wrong",
			wantVal: nil,
			wantOk:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			val, ok := tt.kv.Map(tt.key)
			assert.Equal(t, tt.wantOk, ok)
			if ok {
				assert.Equal(t, tt.wantVal, val)
			}
		})
	}
}

func TestKV_Any(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		kv      KV
		key     string
		wantVal any
		wantOk  bool
	}{
		{
			name:    "string value",
			kv:      KV{"key": "value", "num": 42},
			key:     "key",
			wantVal: "value",
			wantOk:  true,
		},
		{
			name:    "int value",
			kv:      KV{"key": "value", "num": 42},
			key:     "num",
			wantVal: 42,
			wantOk:  true,
		},
		{
			name:    "missing key returns false",
			kv:      KV{"key": "value", "num": 42},
			key:     "missing",
			wantVal: nil,
			wantOk:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			val, ok := tt.kv.Any(tt.key)
			assert.Equal(t, tt.wantOk, ok)
			if ok {
				assert.Equal(t, tt.wantVal, val)
			}
		})
	}
}

func TestKV_List(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		kv      KV
		key     string
		wantVal []any
		wantOk  bool
	}{
		{
			name:    "list value",
			kv:      KV{"key": []any{"a", "b"}},
			key:     "key",
			wantVal: []any{"a", "b"},
			wantOk:  true,
		},
		{
			name:    "missing key returns false",
			kv:      KV{"key": []any{"a", "b"}},
			key:     "missing",
			wantVal: nil,
			wantOk:  false,
		},
		{
			name:    "non-list value returns false",
			kv:      KV{"key": []any{"a", "b"}, "wrong": "string"},
			key:     "wrong",
			wantVal: nil,
			wantOk:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			val, ok := tt.kv.List(tt.key)
			assert.Equal(t, tt.wantOk, ok)
			if ok {
				assert.Equal(t, tt.wantVal, val)
			}
		})
	}
}

func TestKV_StringValue(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		kv      KV
		wantVal string
		wantOk  bool
	}{
		{
			name:    "value key exists",
			kv:      KV{"value": "hello"},
			wantVal: "hello",
			wantOk:  true,
		},
		{
			name:    "empty KV",
			kv:      KV{},
			wantVal: "",
			wantOk:  false,
		},
		{
			name:    "non-string value returns false",
			kv:      KV{"value": 42},
			wantVal: "",
			wantOk:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			val, ok := tt.kv.StringValue()
			assert.Equal(t, tt.wantOk, ok)
			if ok {
				assert.Equal(t, tt.wantVal, val)
			}
		})
	}
}

func TestKV_Int64Value(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		kv      KV
		wantVal int64
		wantOk  bool
	}{
		{
			name:    "value key exists with int64",
			kv:      KV{"value": int64(64)},
			wantVal: int64(64),
			wantOk:  true,
		},
		{
			name:    "empty KV",
			kv:      KV{},
			wantVal: 0,
			wantOk:  false,
		},
		{
			name:    "string value returns false",
			kv:      KV{"value": "64"},
			wantVal: 0,
			wantOk:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			val, ok := tt.kv.Int64Value()
			assert.Equal(t, tt.wantOk, ok)
			if ok {
				assert.Equal(t, tt.wantVal, val)
			}
		})
	}
}

func TestKV_Uint64Value(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		kv      KV
		wantVal uint64
		wantOk  bool
	}{
		{
			name:    "value key exists with float64",
			kv:      KV{"value": float64(100)},
			wantVal: uint64(100),
			wantOk:  true,
		},
		{
			name:    "empty KV",
			kv:      KV{},
			wantVal: 0,
			wantOk:  false,
		},
		{
			name:    "string value returns false",
			kv:      KV{"value": "100"},
			wantVal: 0,
			wantOk:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			val, ok := tt.kv.Uint64Value()
			assert.Equal(t, tt.wantOk, ok)
			if ok {
				assert.Equal(t, tt.wantVal, val)
			}
		})
	}
}

func TestKV_Float64Value(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		kv      KV
		wantVal float64
		wantOk  bool
	}{
		{
			name:    "value key exists with float64",
			kv:      KV{"value": float64(2.71)},
			wantVal: float64(2.71),
			wantOk:  true,
		},
		{
			name:    "empty KV",
			kv:      KV{},
			wantVal: 0,
			wantOk:  false,
		},
		{
			name:    "string value returns false",
			kv:      KV{"value": "3.14"},
			wantVal: 0,
			wantOk:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			val, ok := tt.kv.Float64Value()
			assert.Equal(t, tt.wantOk, ok)
			if ok {
				assert.InEpsilon(t, tt.wantVal, val, 0.001)
			}
		})
	}
}

func TestKV_Merge_Simple(t *testing.T) {
	t.Parallel()
	t.Run("simple merge", func(t *testing.T) {
		t.Parallel()
		a := KV{"x": "1"}
		b := KV{"y": "2"}
		result := a.Merge(b)
		assert.Equal(t, "1", result["x"])
		assert.Equal(t, "2", result["y"])
	})
}

func TestKV_Merge_Override(t *testing.T) {
	t.Parallel()
	t.Run("override merge", func(t *testing.T) {
		t.Parallel()
		a := KV{"x": "old"}
		b := KV{"x": "new"}
		result := a.Merge(b)
		assert.Equal(t, "new", result["x"])
	})
}

func TestKV_Merge_Nested(t *testing.T) {
	t.Parallel()
	t.Run("nested merge", func(t *testing.T) {
		t.Parallel()
		a := KV{"nested": map[string]any{"a": 1}}
		b := KV{"nested": map[string]any{"b": 2}}
		result := a.Merge(b)
		m, ok := result["nested"].(KV)
		require.True(t, ok)
		assert.Equal(t, 1, m["a"])
		assert.Equal(t, 2, m["b"])
	})
}

func TestKV_Merge_Lists(t *testing.T) {
	t.Parallel()
	t.Run("list merge", func(t *testing.T) {
		t.Parallel()
		a := KV{"items": []any{"a", "b"}}
		b := KV{"items": []any{"c"}}
		result := a.Merge(b)
		list := result["items"].([]any)
		assert.Equal(t, []any{"a", "b", "c"}, list)
	})
}

func TestKV_Merge_ListNil(t *testing.T) {
	t.Parallel()
	t.Run("nil list merge", func(t *testing.T) {
		t.Parallel()
		a := KV{"items": nil}
		b := KV{"items": []any{"a"}}
		result := a.Merge(b)
		list := result["items"].([]any)
		assert.Equal(t, []any{"a"}, list)
	})
}

func TestKV_Merge_TypeMismatch(t *testing.T) {
	t.Parallel()
	t.Run("type mismatch merge", func(t *testing.T) {
		t.Parallel()
		a := KV{"x": "string"}
		b := KV{"x": []any{"a"}}
		result := a.Merge(b)
		assert.Equal(t, "string", result["x"])
	})
}

func TestKV_Merge_TypeMismatchMap(t *testing.T) {
	t.Parallel()
	t.Run("type mismatch map merge", func(t *testing.T) {
		t.Parallel()
		a := KV{"x": "string"}
		b := KV{"x": map[string]any{"a": 1}}
		result := a.Merge(b)
		assert.Equal(t, "string", result["x"])
	})
}

func TestKV_Scan_ValidJSON(t *testing.T) {
	t.Parallel()
	t.Run("valid JSON", func(t *testing.T) {
		t.Parallel()
		var kv KV
		err := kv.Scan([]byte(`{"key": "value", "num": 42}`))
		require.NoError(t, err)
		assert.Equal(t, "value", kv["key"])
	})
}

func TestKV_Scan_InvalidJSON(t *testing.T) {
	t.Parallel()
	t.Run("invalid JSON", func(t *testing.T) {
		t.Parallel()
		var kv KV
		err := kv.Scan([]byte(`{invalid`))
		assert.Error(t, err)
	})
}

func TestKV_Scan_MapType(t *testing.T) {
	t.Parallel()
	t.Run("map type", func(t *testing.T) {
		t.Parallel()
		var kv KV
		err := kv.Scan(map[string]any{"key": "value"})
		require.NoError(t, err)
		assert.Equal(t, "value", kv["key"])
	})
}

func TestKV_Scan_UnknownType(t *testing.T) {
	t.Parallel()
	t.Run("unknown type", func(t *testing.T) {
		t.Parallel()
		var kv KV
		err := kv.Scan(42)
		assert.Error(t, err)
	})
}

func TestKV_Value_Empty(t *testing.T) {
	t.Parallel()
	t.Run("empty KV value", func(t *testing.T) {
		t.Parallel()
		v, err := (KV{}).Value()
		require.NoError(t, err)
		assert.Nil(t, v)
	})
}

func TestKV_Value_Populated(t *testing.T) {
	t.Parallel()
	t.Run("populated KV value", func(t *testing.T) {
		t.Parallel()
		kv := KV{"key": "value"}
		v, err := kv.Value()
		require.NoError(t, err)
		assert.NotNil(t, v)
		assert.Contains(t, string(v.([]byte)), "key")
	})
}
