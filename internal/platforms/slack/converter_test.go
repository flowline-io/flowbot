package slack

import (
	"testing"
)

func TestSafeStr(t *testing.T) {
	tests := []struct {
		name string
		data map[string]any
		key  string
		want string
	}{
		{name: "existing string key", data: map[string]any{"k": "val"}, key: "k", want: "val"},
		{name: "missing key returns empty", data: map[string]any{"k": "val"}, key: "x", want: ""},
		{name: "non-string value returns empty", data: map[string]any{"k": 42}, key: "k", want: ""},
		{name: "nil map", data: nil, key: "k", want: ""},
		{name: "empty map", data: map[string]any{}, key: "k", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := safeStr(tt.data, tt.key); got != tt.want {
				t.Errorf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestToStringSlice(t *testing.T) {
	tests := []struct {
		name string
		v    any
		want []string
	}{
		{name: "[]string", v: []string{"a", "b"}, want: []string{"a", "b"}},
		{name: "[]any with strings", v: []any{"x", "y", "z"}, want: []string{"x", "y", "z"}},
		{name: "[]any mixed", v: []any{"hello", 42, "world"}, want: []string{"hello", "world"}},
		{name: "nil input", v: nil, want: nil},
		{name: "int input", v: 123, want: nil},
		{name: "string input", v: "not a slice", want: nil},
		{name: "empty []string", v: []string{}, want: []string{}},
		{name: "empty []any", v: []any{}, want: []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toStringSlice(tt.v)
			if len(got) != len(tt.want) {
				t.Fatalf("expected len %d, got %d", len(tt.want), len(got))
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("index %d: expected %q, got %q", i, tt.want[i], got[i])
				}
			}
		})
	}
}

func TestToFloat64Slice(t *testing.T) {
	tests := []struct {
		name string
		v    any
		want []float64
	}{
		{name: "[]float64", v: []float64{1.0, 2.5, 3.0}, want: []float64{1.0, 2.5, 3.0}},
		{name: "[]any with float64", v: []any{1.0, 2.0}, want: []float64{1.0, 2.0}},
		{name: "[]any with int", v: []any{1, 2, 3}, want: []float64{1.0, 2.0, 3.0}},
		{name: "[]any with int64", v: []any{int64(100)}, want: []float64{100.0}},
		{name: "[]any mixed numbers", v: []any{float64(1.5), 2, int64(3)}, want: []float64{1.5, 2.0, 3.0}},
		{name: "nil input", v: nil, want: nil},
		{name: "string input", v: "not numbers", want: nil},
		{name: "empty slice", v: []float64{}, want: []float64{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toFloat64Slice(tt.v)
			if len(got) != len(tt.want) {
				t.Fatalf("expected len %d, got %d", len(tt.want), len(got))
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("index %d: expected %v, got %v", i, tt.want[i], got[i])
				}
			}
		})
	}
}

func TestToRowSlice(t *testing.T) {
	tests := []struct {
		name string
		v    any
		want [][]any
	}{
		{
			name: "[][]any",
			v:    [][]any{{"a", 1}, {"b", 2}},
			want: [][]any{{"a", 1}, {"b", 2}},
		},
		{
			name: "[]any with []any children",
			v:    []any{[]any{"x", "y"}, []any{"z"}},
			want: [][]any{{"x", "y"}, {"z"}},
		},
		{
			name: "[]any with mixed children",
			v:    []any{[]any{"ok"}, "not ok", []any{1, 2}},
			want: [][]any{{"ok"}, {1, 2}},
		},
		{
			name: "nil input",
			v:    nil,
			want: nil,
		},
		{
			name: "int input",
			v:    42,
			want: nil,
		},
		{
			name: "empty slice",
			v:    [][]any{},
			want: [][]any{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toRowSlice(tt.v)
			if len(got) != len(tt.want) {
				t.Fatalf("expected len %d, got %d", len(tt.want), len(got))
			}
		})
	}
}

func TestToStringMap(t *testing.T) {
	tests := []struct {
		name string
		v    any
		want map[string]string
	}{
		{
			name: "map[string]string",
			v:    map[string]string{"k": "v"},
			want: map[string]string{"k": "v"},
		},
		{
			name: "map[string]any",
			v:    map[string]any{"k": "v", "n": 42},
			want: map[string]string{"k": "v", "n": "42"},
		},
		{
			name: "nil input",
			v:    nil,
			want: nil,
		},
		{
			name: "int input",
			v:    42,
			want: map[string]string{},
		},
		{
			name: "empty map",
			v:    map[string]string{},
			want: map[string]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toStringMap(tt.v)
			if tt.want == nil && got != nil {
				t.Errorf("expected nil, got %v", got)
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("expected len %d, got %d", len(tt.want), len(got))
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("key %q: expected %q, got %q", k, v, got[k])
				}
			}
		})
	}
}

func TestToFormFieldDefs(t *testing.T) {
	tests := []struct {
		name  string
		v     any
		want  int
		check func(*testing.T, FormFieldDef)
	}{
		{
			name: "[]any with form field maps",
			v: []any{
				map[string]any{"label": "Name", "key": "name", "type": "text", "placeholder": "Enter", "initial_value": "Alice"},
			},
			want: 1,
			check: func(t *testing.T, f FormFieldDef) {
				if f.Label != "Name" || f.Key != "name" || f.InitialVal != "Alice" {
					t.Errorf("unexpected field: %+v", f)
				}
			},
		},
		{
			name: "[]FormFieldDef",
			v: []FormFieldDef{
				{Label: "Email", Key: "email", Type: "email"},
			},
			want: 1,
		},
		{
			name: "multiple fields",
			v: []any{
				map[string]any{"label": "A", "key": "a"},
				map[string]any{"label": "B", "key": "b", "options": []any{"x", "y"}},
			},
			want: 2,
		},
		{
			name: "nil input",
			v:    nil,
			want: 0,
		},
		{
			name: "int input",
			v:    123,
			want: 0,
		},
		{
			name: "empty slice",
			v:    []any{},
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toFormFieldDefs(tt.v)
			if len(got) != tt.want {
				t.Fatalf("expected %d field defs, got %d", tt.want, len(got))
			}
			if tt.check != nil && len(got) > 0 {
				tt.check(t, got[0])
			}
		})
	}
}

func TestToButtonDefs(t *testing.T) {
	tests := []struct {
		name  string
		v     any
		want  int
		check func(*testing.T, ButtonDef)
	}{
		{
			name: "[]any with button maps",
			v: []any{
				map[string]any{"text": "OK", "action_id": "act", "value": "v", "style": "primary", "url": "https://a.com"},
			},
			want: 1,
			check: func(t *testing.T, b ButtonDef) {
				if b.Text != "OK" || b.ActionID != "act" || b.Style != "primary" {
					t.Errorf("unexpected button: %+v", b)
				}
				if b.URL != "https://a.com" {
					t.Errorf("expected URL, got %q", b.URL)
				}
			},
		},
		{
			name: "danger style button",
			v: []any{
				map[string]any{"text": "Del", "action_id": "del", "value": "x", "style": "danger"},
			},
			want: 1,
			check: func(t *testing.T, b ButtonDef) {
				if b.Style != "danger" {
					t.Errorf("expected danger style, got %s", b.Style)
				}
			},
		},
		{
			name: "[]ButtonDef",
			v:    []ButtonDef{{Text: "B", ActionID: "bid", Value: "bv"}},
			want: 1,
		},
		{
			name: "nil input",
			v:    nil,
			want: 0,
		},
		{
			name: "string input",
			v:    "not buttons",
			want: 0,
		},
		{
			name: "empty slice",
			v:    []any{},
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toButtonDefs(tt.v)
			if len(got) != tt.want {
				t.Fatalf("expected %d button defs, got %d", tt.want, len(got))
			}
			if tt.check != nil && len(got) > 0 {
				tt.check(t, got[0])
			}
		})
	}
}
