package common

type Page struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

func NewPage(limit, offset int) Page {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	return Page{Limit: limit, Offset: offset}
}

type PageResult[T any] struct {
	Items      []T `json:"items"`
	Total      int `json:"total"`
	Limit      int `json:"limit"`
	Offset     int `json:"offset"`
	NextOffset int `json:"next_offset,omitempty"`
}

func NewPageResult[T any](items []T, total, limit, offset int) PageResult[T] {
	next := offset + len(items)
	if next >= total {
		next = 0
	}
	return PageResult[T]{Items: items, Total: total, Limit: limit, Offset: offset, NextOffset: next}
}
