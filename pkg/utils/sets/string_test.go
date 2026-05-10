package sets

import (
	"slices"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		items []string
		want  String
	}{
		{
			name:  "empty set",
			items: []string{},
			want:  String{},
		},
		{
			name:  "single item",
			items: []string{"a"},
			want:  String{"a": struct{}{}},
		},
		{
			name:  "multiple items",
			items: []string{"a", "b", "c"},
			want:  String{"a": struct{}{}, "b": struct{}{}, "c": struct{}{}},
		},
		{
			name:  "duplicate items",
			items: []string{"a", "b", "b", "c"},
			want:  String{"a": struct{}{}, "b": struct{}{}, "c": struct{}{}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := NewString(tt.items...)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStringKeySet(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		theMap any
		want   String
	}{
		{
			name:   "valid string map",
			theMap: map[string]int{"a": 1, "b": 2, "c": 3},
			want:   String{"a": struct{}{}, "b": struct{}{}, "c": struct{}{}},
		},
		{
			name:   "empty map",
			theMap: map[string]int{},
			want:   String{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := StringKeySet(tt.theMap)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestString_Insert(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		initial []string
		insert  []string
		want    String
	}{
		{
			name:    "insert new items",
			initial: []string{"a", "b"},
			insert:  []string{"c", "d", "b"},
			want:    String{"a": struct{}{}, "b": struct{}{}, "c": struct{}{}, "d": struct{}{}},
		},
		{
			name:    "insert duplicate only",
			initial: []string{"a", "b"},
			insert:  []string{"a"},
			want:    String{"a": struct{}{}, "b": struct{}{}},
		},
		{
			name:    "insert into empty set",
			initial: []string{},
			insert:  []string{"a", "b"},
			want:    String{"a": struct{}{}, "b": struct{}{}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := NewString(tt.initial...)
			got := s.Insert(tt.insert...)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestString_Delete(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		initial []string
		delete  []string
		want    String
	}{
		{
			name:    "delete existing items",
			initial: []string{"a", "b", "c", "d"},
			delete:  []string{"b", "d", "e"},
			want:    String{"a": struct{}{}, "c": struct{}{}},
		},
		{
			name:    "delete from empty set",
			initial: []string{},
			delete:  []string{"a"},
			want:    String{},
		},
		{
			name:    "delete all items",
			initial: []string{"a", "b"},
			delete:  []string{"a", "b"},
			want:    String{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := NewString(tt.initial...)
			got := s.Delete(tt.delete...)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestString_Has(t *testing.T) {
	t.Parallel()
	s := NewString("a", "b", "c")

	tests := []struct {
		name string
		item string
		want bool
	}{
		{name: "existing a", item: "a", want: true},
		{name: "existing b", item: "b", want: true},
		{name: "existing c", item: "c", want: true},
		{name: "missing d", item: "d", want: false},
		{name: "empty string", item: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, s.Has(tt.item))
		})
	}
}

func TestString_HasAll(t *testing.T) {
	t.Parallel()
	s := NewString("a", "b", "c", "d")

	tests := []struct {
		name  string
		items []string
		want  bool
	}{
		{
			name:  "all present",
			items: []string{"a", "b", "c"},
			want:  true,
		},
		{
			name:  "some missing",
			items: []string{"a", "b", "e"},
			want:  false,
		},
		{
			name:  "empty items",
			items: []string{},
			want:  true,
		},
		{
			name:  "all missing",
			items: []string{"e", "f", "g"},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, s.HasAll(tt.items...))
		})
	}
}

func TestString_HasAny(t *testing.T) {
	t.Parallel()
	s := NewString("a", "b", "c")

	tests := []struct {
		name  string
		items []string
		want  bool
	}{
		{
			name:  "some present",
			items: []string{"a", "e", "f"},
			want:  true,
		},
		{
			name:  "none present",
			items: []string{"e", "f", "g"},
			want:  false,
		},
		{
			name:  "empty items",
			items: []string{},
			want:  false,
		},
		{
			name:  "all present",
			items: []string{"a", "b", "c"},
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, s.HasAny(tt.items...))
		})
	}
}

func TestString_Difference(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		s1   String
		s2   String
		want String
	}{
		{
			name: "partial overlap",
			s1:   NewString("a", "b", "c", "d"),
			s2:   NewString("c", "d", "e", "f"),
			want: NewString("a", "b"),
		},
		{
			name: "no overlap",
			s1:   NewString("a", "b"),
			s2:   NewString("c", "d"),
			want: NewString("a", "b"),
		},
		{
			name: "complete overlap",
			s1:   NewString("a", "b"),
			s2:   NewString("a", "b"),
			want: NewString(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.s1.Difference(tt.s2)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestString_Union(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		s1   String
		s2   String
		want String
	}{
		{
			name: "partial overlap",
			s1:   NewString("a", "b", "c"),
			s2:   NewString("c", "d", "e"),
			want: NewString("a", "b", "c", "d", "e"),
		},
		{
			name: "no overlap",
			s1:   NewString("a"),
			s2:   NewString("b"),
			want: NewString("a", "b"),
		},
		{
			name: "identical sets",
			s1:   NewString("a", "b"),
			s2:   NewString("a", "b"),
			want: NewString("a", "b"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.s1.Union(tt.s2)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestString_Intersection(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		s1   String
		s2   String
		want String
	}{
		{
			name: "partial overlap",
			s1:   NewString("a", "b", "c", "d"),
			s2:   NewString("c", "d", "e", "f"),
			want: NewString("c", "d"),
		},
		{
			name: "no overlap",
			s1:   NewString("a", "b"),
			s2:   NewString("c", "d"),
			want: NewString(),
		},
		{
			name: "identical sets",
			s1:   NewString("a", "b"),
			s2:   NewString("a", "b"),
			want: NewString("a", "b"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.s1.Intersection(tt.s2)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestString_IsSuperset(t *testing.T) {
	t.Parallel()
	s := NewString("a", "b", "c", "d", "e")

	tests := []struct {
		name  string
		other String
		want  bool
	}{
		{
			name:  "is superset",
			other: NewString("b", "c", "d"),
			want:  true,
		},
		{
			name:  "not superset",
			other: NewString("b", "c", "f"),
			want:  false,
		},
		{
			name:  "empty set",
			other: NewString(),
			want:  true,
		},
		{
			name:  "equal sets",
			other: NewString("a", "b", "c", "d", "e"),
			want:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, s.IsSuperset(tt.other))
		})
	}
}

func TestString_Equal(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		s1   String
		s2   String
		want bool
	}{
		{
			name: "equal different order",
			s1:   NewString("a", "b", "c"),
			s2:   NewString("c", "b", "a"),
			want: true,
		},
		{
			name: "not equal",
			s1:   NewString("a", "b", "c"),
			s2:   NewString("a", "b", "d"),
			want: false,
		},
		{
			name: "both empty",
			s1:   NewString(),
			s2:   NewString(),
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.s1.Equal(tt.s2))
		})
	}
}

func TestString_List(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		s    String
		want []string
	}{
		{
			name: "unsorted input returns sorted",
			s:    NewString("c", "a", "d", "b"),
			want: []string{"a", "b", "c", "d"},
		},
		{
			name: "empty set",
			s:    NewString(),
			want: []string{},
		},
		{
			name: "single item",
			s:    NewString("a"),
			want: []string{"a"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.s.List()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestString_UnsortedList(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		s    String
		want []string
	}{
		{
			name: "typical set",
			s:    NewString("a", "b", "c"),
			want: []string{"a", "b", "c"},
		},
		{
			name: "empty set",
			s:    NewString(),
			want: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.s.UnsortedList()
			assert.Len(t, got, len(tt.want))
			for _, item := range tt.want {
				assert.True(t, slices.Contains(got, item), "missing item %v", item)
			}
		})
	}
}

func TestString_PopAny(t *testing.T) {
	t.Parallel()
	t.Run("non-empty set pops and shrinks", func(t *testing.T) {
		t.Parallel()
		s := NewString("a", "b", "c")
		originalLen := s.Len()

		item, ok := s.PopAny()
		require.True(t, ok)
		assert.Equal(t, originalLen-1, s.Len())
		assert.False(t, s.Has(item))
	})

	t.Run("empty set returns false", func(t *testing.T) {
		t.Parallel()
		emptySet := NewString()
		_, ok := emptySet.PopAny()
		assert.False(t, ok)
	})
}

func TestString_Len(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		s    String
		want int
	}{
		{
			name: "empty set",
			s:    NewString(),
			want: 0,
		},
		{
			name: "single item",
			s:    NewString("a"),
			want: 1,
		},
		{
			name: "multiple items",
			s:    NewString("a", "b", "c", "d", "e"),
			want: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.s.Len())
		})
	}
}

func FuzzStringSet(f *testing.F) {
	f.Add([]byte(`[]`), []byte(`[]`))
	f.Add([]byte(`["a","b","c"]`), []byte(`["c","d","e"]`))
	f.Add([]byte(`["a"]`), []byte(`["a"]`))

	f.Fuzz(func(t *testing.T, aData, bData []byte) {
		var a, b []string
		if err := sonic.Unmarshal(aData, &a); err != nil {
			t.Skip()
		}
		if err := sonic.Unmarshal(bData, &b); err != nil {
			t.Skip()
		}

		s1 := NewString(a...)
		s2 := NewString(b...)

		for _, v := range a {
			if !s1.Has(v) {
				t.Errorf("Set constructed from %v missing element %q", a, v)
			}
		}

		if !s1.HasAll(a...) {
			t.Errorf("HasAll failed for self-elements: %v", a)
		}

		u1 := s1.Union(s2)
		u2 := s2.Union(s1)
		if !u1.Equal(u2) {
			t.Errorf("Union not commutative")
		}

		if u1.Len() > s1.Len()+s2.Len() {
			t.Errorf("Union size %d > sum of sizes %d+%d", u1.Len(), s1.Len(), s2.Len())
		}
		if u1.Len() < max(s1.Len(), s2.Len()) {
			t.Errorf("Union size %d < max(%d, %d)", u1.Len(), s1.Len(), s2.Len())
		}

		i1 := s1.Intersection(s2)
		i2 := s2.Intersection(s1)
		if !i1.Equal(i2) {
			t.Errorf("Intersection not commutative")
		}

		if i1.Len() > min(s1.Len(), s2.Len()) {
			t.Errorf("Intersection size %d > min(%d, %d)", i1.Len(), s1.Len(), s2.Len())
		}

		diff := s1.Difference(s2)
		reconstructed := diff.Union(i1)
		if !reconstructed.Equal(s1) {
			t.Errorf("Difference+Intersection != original")
		}

		if !s1.Equal(s1) {
			t.Errorf("Set not equal to itself")
		}

		lst := s1.List()
		for i := 1; i < len(lst); i++ {
			if lst[i-1] > lst[i] {
				t.Errorf("List not sorted")
				break
			}
		}

		if s1.Len() == 0 {
			_, ok := s1.PopAny()
			if ok {
				t.Error("PopAny on empty set returned ok=true")
			}
		}

		if !s1.IsSuperset(NewString()) {
			t.Errorf("Every set should be superset of empty")
		}

		sCopy := NewString(a...)
		sCopy.Delete(a...)
		if sCopy.Len() > 0 {
			t.Errorf("Delete all elements left %d items", sCopy.Len())
		}
	})
}

func FuzzStringKeySet(f *testing.F) {
	f.Fuzz(func(t *testing.T, keysData []byte) {
		var keys []string
		if err := sonic.Unmarshal(keysData, &keys); err != nil {
			t.Skip()
		}
		theMap := make(map[string]int, len(keys))
		for _, k := range keys {
			theMap[k] = 0
		}
		result := StringKeySet(theMap)
		if result.Len() != len(theMap) {
			t.Errorf("StringKeySet size %d != map size %d", result.Len(), len(theMap))
		}
		for k := range theMap {
			if !result.Has(k) {
				t.Errorf("StringKeySet missing key %q", k)
			}
		}
	})
}
