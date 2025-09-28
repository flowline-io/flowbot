package sets

import (
	"reflect"
	"testing"
)

// TestNewInt tests the NewInt function
func TestNewInt(t *testing.T) {
	tests := []struct {
		name  string
		items []int
		want  Int
	}{
		{
			name:  "empty_set",
			items: []int{},
			want:  Int{},
		},
		{
			name:  "single_item",
			items: []int{1},
			want:  Int{1: struct{}{}},
		},
		{
			name:  "multiple_items",
			items: []int{1, 2, 3},
			want:  Int{1: struct{}{}, 2: struct{}{}, 3: struct{}{}},
		},
		{
			name:  "duplicate_items",
			items: []int{1, 2, 2, 3},
			want:  Int{1: struct{}{}, 2: struct{}{}, 3: struct{}{}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewInt(tt.items...)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewInt() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIntKeySet tests the IntKeySet function
func TestIntKeySet(t *testing.T) {
	tests := []struct {
		name    string
		theMap  interface{}
		want    Int
		wantErr bool
	}{
		{
			name:   "valid_int_map",
			theMap: map[int]string{1: "one", 2: "two", 3: "three"},
			want:   Int{1: struct{}{}, 2: struct{}{}, 3: struct{}{}},
		},
		{
			name:   "empty_map",
			theMap: map[int]string{},
			want:   Int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IntKeySet(tt.theMap)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("IntKeySet() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestInt_Insert tests the Insert method
func TestInt_Insert(t *testing.T) {
	s := NewInt(1, 2)
	result := s.Insert(3, 4, 2) // 2 is duplicate

	expected := Int{1: struct{}{}, 2: struct{}{}, 3: struct{}{}, 4: struct{}{}}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Insert() = %v, want %v", result, expected)
	}
}

// TestInt_Delete tests the Delete method
func TestInt_Delete(t *testing.T) {
	s := NewInt(1, 2, 3, 4)
	result := s.Delete(2, 4, 5) // 5 doesn't exist

	expected := Int{1: struct{}{}, 3: struct{}{}}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Delete() = %v, want %v", result, expected)
	}
}

// TestInt_Has tests the Has method
func TestInt_Has(t *testing.T) {
	s := NewInt(1, 2, 3)

	tests := []struct {
		item int
		want bool
	}{
		{1, true},
		{2, true},
		{3, true},
		{4, false},
		{0, false},
	}

	for _, tt := range tests {
		if got := s.Has(tt.item); got != tt.want {
			t.Errorf("Has(%v) = %v, want %v", tt.item, got, tt.want)
		}
	}
}

// TestInt_HasAll tests the HasAll method
func TestInt_HasAll(t *testing.T) {
	s := NewInt(1, 2, 3, 4)

	tests := []struct {
		name  string
		items []int
		want  bool
	}{
		{
			name:  "all_present",
			items: []int{1, 2, 3},
			want:  true,
		},
		{
			name:  "some_missing",
			items: []int{1, 2, 5},
			want:  false,
		},
		{
			name:  "empty_items",
			items: []int{},
			want:  true,
		},
		{
			name:  "all_missing",
			items: []int{5, 6, 7},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := s.HasAll(tt.items...); got != tt.want {
				t.Errorf("HasAll() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestInt_HasAny tests the HasAny method
func TestInt_HasAny(t *testing.T) {
	s := NewInt(1, 2, 3)

	tests := []struct {
		name  string
		items []int
		want  bool
	}{
		{
			name:  "some_present",
			items: []int{1, 5, 6},
			want:  true,
		},
		{
			name:  "none_present",
			items: []int{5, 6, 7},
			want:  false,
		},
		{
			name:  "empty_items",
			items: []int{},
			want:  false,
		},
		{
			name:  "all_present",
			items: []int{1, 2, 3},
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := s.HasAny(tt.items...); got != tt.want {
				t.Errorf("HasAny() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestInt_Difference tests the Difference method
func TestInt_Difference(t *testing.T) {
	s1 := NewInt(1, 2, 3, 4)
	s2 := NewInt(3, 4, 5, 6)

	result := s1.Difference(s2)
	expected := NewInt(1, 2)

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Difference() = %v, want %v", result, expected)
	}
}

// TestInt_Union tests the Union method
func TestInt_Union(t *testing.T) {
	s1 := NewInt(1, 2, 3)
	s2 := NewInt(3, 4, 5)

	result := s1.Union(s2)
	expected := NewInt(1, 2, 3, 4, 5)

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Union() = %v, want %v", result, expected)
	}
}

// TestInt_Intersection tests the Intersection method
func TestInt_Intersection(t *testing.T) {
	s1 := NewInt(1, 2, 3, 4)
	s2 := NewInt(3, 4, 5, 6)

	result := s1.Intersection(s2)
	expected := NewInt(3, 4)

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Intersection() = %v, want %v", result, expected)
	}
}

// TestInt_IsSuperset tests the IsSuperset method
func TestInt_IsSuperset(t *testing.T) {
	s1 := NewInt(1, 2, 3, 4, 5)
	s2 := NewInt(2, 3, 4)
	s3 := NewInt(2, 3, 6)

	if !s1.IsSuperset(s2) {
		t.Error("IsSuperset() should return true when s1 contains all elements of s2")
	}

	if s1.IsSuperset(s3) {
		t.Error("IsSuperset() should return false when s1 doesn't contain all elements of s3")
	}
}

// TestInt_Equal tests the Equal method
func TestInt_Equal(t *testing.T) {
	s1 := NewInt(1, 2, 3)
	s2 := NewInt(3, 2, 1) // Same elements, different order
	s3 := NewInt(1, 2, 4) // Different elements

	if !s1.Equal(s2) {
		t.Error("Equal() should return true for sets with same elements")
	}

	if s1.Equal(s3) {
		t.Error("Equal() should return false for sets with different elements")
	}
}

// TestInt_List tests the List method
func TestInt_List(t *testing.T) {
	s := NewInt(3, 1, 4, 2)
	result := s.List()
	expected := []int{1, 2, 3, 4}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("List() = %v, want %v", result, expected)
	}
}

// TestInt_UnsortedList tests the UnsortedList method
func TestInt_UnsortedList(t *testing.T) {
	s := NewInt(1, 2, 3)
	result := s.UnsortedList()

	// Should have all elements
	if len(result) != 3 {
		t.Errorf("UnsortedList() length = %v, want 3", len(result))
	}

	// Should contain all original elements
	for _, item := range []int{1, 2, 3} {
		found := false
		for _, resultItem := range result {
			if item == resultItem {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("UnsortedList() missing item %v", item)
		}
	}
}

// TestInt_PopAny tests the PopAny method
func TestInt_PopAny(t *testing.T) {
	// Test with non-empty set
	s := NewInt(1, 2, 3)
	originalLen := s.Len()

	item, ok := s.PopAny()
	if !ok {
		t.Error("PopAny() should return true for non-empty set")
	}

	if s.Len() != originalLen-1 {
		t.Errorf("PopAny() should reduce set size by 1")
	}

	if s.Has(item) {
		t.Errorf("PopAny() should remove the returned item from set")
	}

	// Test with empty set
	emptySet := NewInt()
	_, ok = emptySet.PopAny()
	if ok {
		t.Error("PopAny() should return false for empty set")
	}
}

// TestInt_Len tests the Len method
func TestInt_Len(t *testing.T) {
	tests := []struct {
		name string
		s    Int
		want int
	}{
		{
			name: "empty_set",
			s:    NewInt(),
			want: 0,
		},
		{
			name: "single_item",
			s:    NewInt(1),
			want: 1,
		},
		{
			name: "multiple_items",
			s:    NewInt(1, 2, 3, 4, 5),
			want: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.Len(); got != tt.want {
				t.Errorf("Len() = %v, want %v", got, tt.want)
			}
		})
	}
}
