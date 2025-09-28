package sets

import (
	"reflect"
	"testing"
)

// TestNewString tests the NewString function
func TestNewString(t *testing.T) {
	tests := []struct {
		name  string
		items []string
		want  String
	}{
		{
			name:  "empty_set",
			items: []string{},
			want:  String{},
		},
		{
			name:  "single_item",
			items: []string{"a"},
			want:  String{"a": struct{}{}},
		},
		{
			name:  "multiple_items",
			items: []string{"a", "b", "c"},
			want:  String{"a": struct{}{}, "b": struct{}{}, "c": struct{}{}},
		},
		{
			name:  "duplicate_items",
			items: []string{"a", "b", "b", "c"},
			want:  String{"a": struct{}{}, "b": struct{}{}, "c": struct{}{}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewString(tt.items...)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewString() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestStringKeySet tests the StringKeySet function
func TestStringKeySet(t *testing.T) {
	tests := []struct {
		name   string
		theMap interface{}
		want   String
	}{
		{
			name:   "valid_string_map",
			theMap: map[string]int{"a": 1, "b": 2, "c": 3},
			want:   String{"a": struct{}{}, "b": struct{}{}, "c": struct{}{}},
		},
		{
			name:   "empty_map",
			theMap: map[string]int{},
			want:   String{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StringKeySet(tt.theMap)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("StringKeySet() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestString_Insert tests the Insert method
func TestString_Insert(t *testing.T) {
	s := NewString("a", "b")
	result := s.Insert("c", "d", "b") // "b" is duplicate

	expected := String{"a": struct{}{}, "b": struct{}{}, "c": struct{}{}, "d": struct{}{}}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Insert() = %v, want %v", result, expected)
	}
}

// TestString_Delete tests the Delete method
func TestString_Delete(t *testing.T) {
	s := NewString("a", "b", "c", "d")
	result := s.Delete("b", "d", "e") // "e" doesn't exist

	expected := String{"a": struct{}{}, "c": struct{}{}}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Delete() = %v, want %v", result, expected)
	}
}

// TestString_Has tests the Has method
func TestString_Has(t *testing.T) {
	s := NewString("a", "b", "c")

	tests := []struct {
		item string
		want bool
	}{
		{"a", true},
		{"b", true},
		{"c", true},
		{"d", false},
		{"", false},
	}

	for _, tt := range tests {
		if got := s.Has(tt.item); got != tt.want {
			t.Errorf("Has(%v) = %v, want %v", tt.item, got, tt.want)
		}
	}
}

// TestString_HasAll tests the HasAll method
func TestString_HasAll(t *testing.T) {
	s := NewString("a", "b", "c", "d")

	tests := []struct {
		name  string
		items []string
		want  bool
	}{
		{
			name:  "all_present",
			items: []string{"a", "b", "c"},
			want:  true,
		},
		{
			name:  "some_missing",
			items: []string{"a", "b", "e"},
			want:  false,
		},
		{
			name:  "empty_items",
			items: []string{},
			want:  true,
		},
		{
			name:  "all_missing",
			items: []string{"e", "f", "g"},
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

// TestString_HasAny tests the HasAny method
func TestString_HasAny(t *testing.T) {
	s := NewString("a", "b", "c")

	tests := []struct {
		name  string
		items []string
		want  bool
	}{
		{
			name:  "some_present",
			items: []string{"a", "e", "f"},
			want:  true,
		},
		{
			name:  "none_present",
			items: []string{"e", "f", "g"},
			want:  false,
		},
		{
			name:  "empty_items",
			items: []string{},
			want:  false,
		},
		{
			name:  "all_present",
			items: []string{"a", "b", "c"},
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

// TestString_Difference tests the Difference method
func TestString_Difference(t *testing.T) {
	s1 := NewString("a", "b", "c", "d")
	s2 := NewString("c", "d", "e", "f")

	result := s1.Difference(s2)
	expected := NewString("a", "b")

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Difference() = %v, want %v", result, expected)
	}
}

// TestString_Union tests the Union method
func TestString_Union(t *testing.T) {
	s1 := NewString("a", "b", "c")
	s2 := NewString("c", "d", "e")

	result := s1.Union(s2)
	expected := NewString("a", "b", "c", "d", "e")

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Union() = %v, want %v", result, expected)
	}
}

// TestString_Intersection tests the Intersection method
func TestString_Intersection(t *testing.T) {
	s1 := NewString("a", "b", "c", "d")
	s2 := NewString("c", "d", "e", "f")

	result := s1.Intersection(s2)
	expected := NewString("c", "d")

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Intersection() = %v, want %v", result, expected)
	}
}

// TestString_IsSuperset tests the IsSuperset method
func TestString_IsSuperset(t *testing.T) {
	s1 := NewString("a", "b", "c", "d", "e")
	s2 := NewString("b", "c", "d")
	s3 := NewString("b", "c", "f")

	if !s1.IsSuperset(s2) {
		t.Error("IsSuperset() should return true when s1 contains all elements of s2")
	}

	if s1.IsSuperset(s3) {
		t.Error("IsSuperset() should return false when s1 doesn't contain all elements of s3")
	}
}

// TestString_Equal tests the Equal method
func TestString_Equal(t *testing.T) {
	s1 := NewString("a", "b", "c")
	s2 := NewString("c", "b", "a") // Same elements, different order
	s3 := NewString("a", "b", "d") // Different elements

	if !s1.Equal(s2) {
		t.Error("Equal() should return true for sets with same elements")
	}

	if s1.Equal(s3) {
		t.Error("Equal() should return false for sets with different elements")
	}
}

// TestString_List tests the List method
func TestString_List(t *testing.T) {
	s := NewString("c", "a", "d", "b")
	result := s.List()
	expected := []string{"a", "b", "c", "d"}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("List() = %v, want %v", result, expected)
	}
}

// TestString_UnsortedList tests the UnsortedList method
func TestString_UnsortedList(t *testing.T) {
	s := NewString("a", "b", "c")
	result := s.UnsortedList()

	// Should have all elements
	if len(result) != 3 {
		t.Errorf("UnsortedList() length = %v, want 3", len(result))
	}

	// Should contain all original elements
	for _, item := range []string{"a", "b", "c"} {
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

// TestString_PopAny tests the PopAny method
func TestString_PopAny(t *testing.T) {
	// Test with non-empty set
	s := NewString("a", "b", "c")
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
	emptySet := NewString()
	_, ok = emptySet.PopAny()
	if ok {
		t.Error("PopAny() should return false for empty set")
	}
}

// TestString_Len tests the Len method
func TestString_Len(t *testing.T) {
	tests := []struct {
		name string
		s    String
		want int
	}{
		{
			name: "empty_set",
			s:    NewString(),
			want: 0,
		},
		{
			name: "single_item",
			s:    NewString("a"),
			want: 1,
		},
		{
			name: "multiple_items",
			s:    NewString("a", "b", "c", "d", "e"),
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
