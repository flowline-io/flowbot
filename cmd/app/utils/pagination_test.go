package utils

import (
	"testing"
)

func TestPagination_New(t *testing.T) {
	p := NewPagination()
	if p == nil {
		t.Fatal("NewPagination returned nil")
	}
	if p.Page != 1 {
		t.Errorf("Page = %d, want 1", p.Page)
	}
	if p.PageSize != 10 {
		t.Errorf("PageSize = %d, want 10", p.PageSize)
	}
}

func TestPagination_SetPageSize(t *testing.T) {
	p := NewPagination()
	p.SetPageSize(20)
	if p.PageSize != 20 {
		t.Errorf("PageSize = %d, want 20", p.PageSize)
	}
}

func TestPagination_Next(t *testing.T) {
	p := NewPagination()
	p.TotalPages = 5

	if !p.Next() {
		t.Error("Next() should return true when more pages exist")
	}
	if p.Page != 2 {
		t.Errorf("Page = %d, want 2", p.Page)
	}
}

func TestPagination_NextAtEnd(t *testing.T) {
	p := NewPagination()
	p.Page = 5
	p.TotalPages = 5

	if p.Next() {
		t.Error("Next() should return false at last page")
	}
	if p.Page != 5 {
		t.Errorf("Page = %d, want 5", p.Page)
	}
}

func TestPagination_Prev(t *testing.T) {
	p := NewPagination()
	p.Page = 3

	if !p.Prev() {
		t.Error("Prev() should return true when not at first page")
	}
	if p.Page != 2 {
		t.Errorf("Page = %d, want 2", p.Page)
	}
}

func TestPagination_PrevAtStart(t *testing.T) {
	p := NewPagination()
	p.Page = 1

	if p.Prev() {
		t.Error("Prev() should return false at first page")
	}
	if p.Page != 1 {
		t.Errorf("Page = %d, want 1", p.Page)
	}
}

func TestPagination_Reset(t *testing.T) {
	p := NewPagination()
	p.Page = 5
	p.Reset()
	if p.Page != 1 {
		t.Errorf("Page = %d, want 1 after Reset", p.Page)
	}
}

func TestPagination_HasPages(t *testing.T) {
	p := NewPagination()
	if p.HasPages() {
		t.Error("HasPages() should return false with 0 total pages")
	}

	p.TotalPages = 1
	if p.HasPages() {
		t.Error("HasPages() should return false with 1 total page")
	}

	p.TotalPages = 3
	if !p.HasPages() {
		t.Error("HasPages() should return true with 3 total pages")
	}
}

func TestPagination_Offset(t *testing.T) {
	p := NewPagination()
	p.Page = 1
	if p.Offset() != 0 {
		t.Errorf("Offset = %d, want 0 for page 1", p.Offset())
	}

	p.Page = 2
	if p.Offset() != 10 {
		t.Errorf("Offset = %d, want 10 for page 2 with size 10", p.Offset())
	}

	p.PageSize = 20
	if p.Offset() != 20 {
		t.Errorf("Offset = %d, want 20 for page 2 with size 20", p.Offset())
	}
}

func TestPagination_VisiblePages(t *testing.T) {
	tests := []struct {
		name       string
		page       int
		totalPages int
		want       []int
	}{
		{
			name:       "single page",
			page:       1,
			totalPages: 1,
			want:       []int{1},
		},
		{
			name:       "three pages on first",
			page:       1,
			totalPages: 3,
			want:       []int{1, 2, 3},
		},
		{
			name:       "five pages in middle",
			page:       3,
			totalPages: 5,
			want:       []int{1, 2, 3, 4, 5},
		},
		{
			name:       "many pages in middle",
			page:       5,
			totalPages: 10,
			want:       []int{3, 4, 5, 6, 7},
		},
		{
			name:       "many pages near end",
			page:       9,
			totalPages: 10,
			want:       []int{7, 8, 9, 10},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPagination()
			p.Page = tt.page
			p.TotalPages = tt.totalPages

			got := p.VisiblePages()
			if len(got) != len(tt.want) {
				t.Errorf("VisiblePages() = %v, want %v", got, tt.want)
				return
			}
			for i, v := range got {
				if v != tt.want[i] {
					t.Errorf("VisiblePages()[%d] = %d, want %d", i, v, tt.want[i])
				}
			}
		})
	}
}
