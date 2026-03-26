package utils

type Pagination struct {
	Page       int
	PageSize   int
	Total      int64
	TotalPages int
}

func NewPagination() *Pagination {
	return &Pagination{
		Page:     1,
		PageSize: 10,
	}
}

func (p *Pagination) SetPageSize(size int) {
	if size > 0 {
		p.PageSize = size
	}
}

func (p *Pagination) GoTo(page int) {
	if page >= 1 && page <= p.TotalPages {
		p.Page = page
	}
}

func (p *Pagination) Next() bool {
	if p.Page < p.TotalPages {
		p.Page++
		return true
	}
	return false
}

func (p *Pagination) Prev() bool {
	if p.Page > 1 {
		p.Page--
		return true
	}
	return false
}

func (p *Pagination) Reset() {
	p.Page = 1
}

func (p *Pagination) HasPages() bool {
	return p.TotalPages > 1
}

func (p *Pagination) Start() int {
	if p.TotalPages <= 5 {
		return 1
	}
	start := max(p.Page-2, 1)
	return start
}

func (p *Pagination) End() int {
	if p.TotalPages <= 5 {
		return p.TotalPages
	}
	end := min(p.Page+2, p.TotalPages)
	return end
}

func (p *Pagination) VisiblePages() []int {
	pages := make([]int, 0)
	start := p.Start()
	end := p.End()
	for i := start; i <= end; i++ {
		pages = append(pages, i)
	}
	return pages
}

func (p *Pagination) Offset() int {
	return (p.Page - 1) * p.PageSize
}
