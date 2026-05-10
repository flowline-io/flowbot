package sets

import (
	"slices"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewInt(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		items []int
		want  Int
	}{
		{
			name:  "empty set",
			items: []int{},
			want:  Int{},
		},
		{
			name:  "single item",
			items: []int{1},
			want:  Int{1: struct{}{}},
		},
		{
			name:  "multiple items",
			items: []int{1, 2, 3},
			want:  Int{1: struct{}{}, 2: struct{}{}, 3: struct{}{}},
		},
		{
			name:  "duplicate items",
			items: []int{1, 2, 2, 3},
			want:  Int{1: struct{}{}, 2: struct{}{}, 3: struct{}{}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := NewInt(tt.items...)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIntKeySet(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		theMap any
		want   Int
	}{
		{
			name:   "valid int map",
			theMap: map[int]string{1: "one", 2: "two", 3: "three"},
			want:   Int{1: struct{}{}, 2: struct{}{}, 3: struct{}{}},
		},
		{
			name:   "empty map",
			theMap: map[int]string{},
			want:   Int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := IntKeySet(tt.theMap)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestInt_Insert(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		initial []int
		insert  []int
		want    Int
	}{
		{
			name:    "insert new items",
			initial: []int{1, 2},
			insert:  []int{3, 4, 2},
			want:    Int{1: struct{}{}, 2: struct{}{}, 3: struct{}{}, 4: struct{}{}},
		},
		{
			name:    "insert duplicate only",
			initial: []int{1, 2},
			insert:  []int{1},
			want:    Int{1: struct{}{}, 2: struct{}{}},
		},
		{
			name:    "insert into empty set",
			initial: []int{},
			insert:  []int{1, 2},
			want:    Int{1: struct{}{}, 2: struct{}{}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := NewInt(tt.initial...)
			got := s.Insert(tt.insert...)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestInt_Delete(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		initial []int
		delete  []int
		want    Int
	}{
		{
			name:    "delete existing items",
			initial: []int{1, 2, 3, 4},
			delete:  []int{2, 4, 5},
			want:    Int{1: struct{}{}, 3: struct{}{}},
		},
		{
			name:    "delete from empty set",
			initial: []int{},
			delete:  []int{1},
			want:    Int{},
		},
		{
			name:    "delete all items",
			initial: []int{1, 2},
			delete:  []int{1, 2},
			want:    Int{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := NewInt(tt.initial...)
			got := s.Delete(tt.delete...)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestInt_Has(t *testing.T) {
	t.Parallel()
	s := NewInt(1, 2, 3)

	tests := []struct {
		name string
		item int
		want bool
	}{
		{name: "existing 1", item: 1, want: true},
		{name: "existing 2", item: 2, want: true},
		{name: "existing 3", item: 3, want: true},
		{name: "missing 4", item: 4, want: false},
		{name: "zero", item: 0, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, s.Has(tt.item))
		})
	}
}

func TestInt_HasAll(t *testing.T) {
	t.Parallel()
	s := NewInt(1, 2, 3, 4)

	tests := []struct {
		name  string
		items []int
		want  bool
	}{
		{
			name:  "all present",
			items: []int{1, 2, 3},
			want:  true,
		},
		{
			name:  "some missing",
			items: []int{1, 2, 5},
			want:  false,
		},
		{
			name:  "empty items",
			items: []int{},
			want:  true,
		},
		{
			name:  "all missing",
			items: []int{5, 6, 7},
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

func TestInt_HasAny(t *testing.T) {
	t.Parallel()
	s := NewInt(1, 2, 3)

	tests := []struct {
		name  string
		items []int
		want  bool
	}{
		{
			name:  "some present",
			items: []int{1, 5, 6},
			want:  true,
		},
		{
			name:  "none present",
			items: []int{5, 6, 7},
			want:  false,
		},
		{
			name:  "empty items",
			items: []int{},
			want:  false,
		},
		{
			name:  "all present",
			items: []int{1, 2, 3},
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

func TestInt_Difference(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		s1   Int
		s2   Int
		want Int
	}{
		{
			name: "partial overlap",
			s1:   NewInt(1, 2, 3, 4),
			s2:   NewInt(3, 4, 5, 6),
			want: NewInt(1, 2),
		},
		{
			name: "no overlap",
			s1:   NewInt(1, 2),
			s2:   NewInt(3, 4),
			want: NewInt(1, 2),
		},
		{
			name: "complete overlap",
			s1:   NewInt(1, 2),
			s2:   NewInt(1, 2),
			want: NewInt(),
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

func TestInt_Union(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		s1   Int
		s2   Int
		want Int
	}{
		{
			name: "partial overlap",
			s1:   NewInt(1, 2, 3),
			s2:   NewInt(3, 4, 5),
			want: NewInt(1, 2, 3, 4, 5),
		},
		{
			name: "no overlap",
			s1:   NewInt(1),
			s2:   NewInt(2),
			want: NewInt(1, 2),
		},
		{
			name: "identical sets",
			s1:   NewInt(1, 2),
			s2:   NewInt(1, 2),
			want: NewInt(1, 2),
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

func TestInt_Intersection(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		s1   Int
		s2   Int
		want Int
	}{
		{
			name: "partial overlap",
			s1:   NewInt(1, 2, 3, 4),
			s2:   NewInt(3, 4, 5, 6),
			want: NewInt(3, 4),
		},
		{
			name: "no overlap",
			s1:   NewInt(1, 2),
			s2:   NewInt(3, 4),
			want: NewInt(),
		},
		{
			name: "identical sets",
			s1:   NewInt(1, 2),
			s2:   NewInt(1, 2),
			want: NewInt(1, 2),
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

func TestInt_IsSuperset(t *testing.T) {
	t.Parallel()
	s := NewInt(1, 2, 3, 4, 5)

	tests := []struct {
		name  string
		other Int
		want  bool
	}{
		{
			name:  "is superset",
			other: NewInt(2, 3, 4),
			want:  true,
		},
		{
			name:  "not superset",
			other: NewInt(2, 3, 6),
			want:  false,
		},
		{
			name:  "empty set",
			other: NewInt(),
			want:  true,
		},
		{
			name:  "equal sets",
			other: NewInt(1, 2, 3, 4, 5),
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

func TestInt_Equal(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		s1   Int
		s2   Int
		want bool
	}{
		{
			name: "equal different order",
			s1:   NewInt(1, 2, 3),
			s2:   NewInt(3, 2, 1),
			want: true,
		},
		{
			name: "not equal",
			s1:   NewInt(1, 2, 3),
			s2:   NewInt(1, 2, 4),
			want: false,
		},
		{
			name: "both empty",
			s1:   NewInt(),
			s2:   NewInt(),
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

func TestInt_List(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		s    Int
		want []int
	}{
		{
			name: "unsorted input returns sorted",
			s:    NewInt(3, 1, 4, 2),
			want: []int{1, 2, 3, 4},
		},
		{
			name: "empty set",
			s:    NewInt(),
			want: []int{},
		},
		{
			name: "single item",
			s:    NewInt(1),
			want: []int{1},
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

func TestInt_UnsortedList(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		s    Int
		want []int
	}{
		{
			name: "typical set",
			s:    NewInt(1, 2, 3),
			want: []int{1, 2, 3},
		},
		{
			name: "empty set",
			s:    NewInt(),
			want: []int{},
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

func TestInt_PopAny(t *testing.T) {
	t.Parallel()
	t.Run("non-empty set pops and shrinks", func(t *testing.T) {
		t.Parallel()
		s := NewInt(1, 2, 3)
		originalLen := s.Len()

		item, ok := s.PopAny()
		require.True(t, ok)
		assert.Equal(t, originalLen-1, s.Len())
		assert.False(t, s.Has(item))
	})

	t.Run("empty set returns false", func(t *testing.T) {
		t.Parallel()
		emptySet := NewInt()
		_, ok := emptySet.PopAny()
		assert.False(t, ok)
	})
}

func TestInt_Len(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		s    Int
		want int
	}{
		{
			name: "empty set",
			s:    NewInt(),
			want: 0,
		},
		{
			name: "single item",
			s:    NewInt(1),
			want: 1,
		},
		{
			name: "multiple items",
			s:    NewInt(1, 2, 3, 4, 5),
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

func FuzzIntSet(f *testing.F) {
	f.Add([]byte(`[]`), []byte(`[]`))
	f.Add([]byte(`[1,2,3]`), []byte(`[3,4,5]`))
	f.Add([]byte(`[1]`), []byte(`[1]`))

	f.Fuzz(func(t *testing.T, aData, bData []byte) {
		var a, b []int
		if err := sonic.Unmarshal(aData, &a); err != nil {
			t.Skip()
		}
		if err := sonic.Unmarshal(bData, &b); err != nil {
			t.Skip()
		}

		s1 := NewInt(a...)
		s2 := NewInt(b...)

		for _, v := range a {
			if !s1.Has(v) {
				t.Errorf("Set constructed from %v missing element %d", a, v)
			}
		}

		if !s1.HasAll(a...) {
			t.Errorf("HasAll failed for self-elements: %v", a)
		}

		u1 := s1.Union(s2)
		u2 := s2.Union(s1)
		if !u1.Equal(u2) {
			t.Errorf("Union not commutative: %v vs %v", u1, u2)
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
			t.Errorf("Intersection not commutative: %v vs %v", i1, i2)
		}

		if i1.Len() > min(s1.Len(), s2.Len()) {
			t.Errorf("Intersection size %d > min(%d, %d)", i1.Len(), s1.Len(), s2.Len())
		}

		diff := s1.Difference(s2)
		reconstructed := diff.Union(i1)
		if !reconstructed.Equal(s1) {
			t.Errorf("Difference+Intersection != original: %v + %v != %v", diff, i1, s1)
		}

		if !s1.Equal(s1) {
			t.Errorf("Set not equal to itself: %v", s1)
		}

		lst := s1.List()
		for i := 1; i < len(lst); i++ {
			if lst[i-1] > lst[i] {
				t.Errorf("List not sorted: %v", lst)
				break
			}
		}

		if s1.Len() == 0 {
			_, ok := s1.PopAny()
			if ok {
				t.Error("PopAny on empty set returned ok=true")
			}
		}

		if !s1.IsSuperset(NewInt()) {
			t.Errorf("Every set should be superset of empty: %v", s1)
		}

		sCopy := NewInt(a...)
		sCopy.Delete(a...)
		if sCopy.Len() > 0 {
			t.Errorf("Delete all elements left %d items", sCopy.Len())
		}
	})
}

func FuzzIntKeySet(f *testing.F) {
	f.Fuzz(func(t *testing.T, keysData []byte) {
		var keys []int
		if err := sonic.Unmarshal(keysData, &keys); err != nil {
			t.Skip()
		}
		theMap := make(map[int]string, len(keys))
		for _, k := range keys {
			theMap[k] = ""
		}
		result := IntKeySet(theMap)
		if result.Len() != len(theMap) {
			t.Errorf("IntKeySet size %d != map size %d", result.Len(), len(theMap))
		}
		for k := range theMap {
			if !result.Has(k) {
				t.Errorf("IntKeySet missing key %d", k)
			}
		}
	})
}
