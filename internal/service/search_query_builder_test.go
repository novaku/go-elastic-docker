package service

import "testing"

func TestApplySearchDefaults(t *testing.T) {
	req := applySearchDefaults(SearchRequest{Page: 0, PageSize: 1000, SortDir: "invalid"})
	if req.Page != 1 {
		t.Fatalf("expected page 1, got %d", req.Page)
	}
	if req.PageSize != 20 {
		t.Fatalf("expected page size 20, got %d", req.PageSize)
	}
	if req.SortDir != "desc" {
		t.Fatalf("expected sort desc, got %q", req.SortDir)
	}
}

func TestElasticsearchQueryBuilder_Build(t *testing.T) {
	qb := NewElasticsearchQueryBuilder()
	active := true
	min, max := 10.0, 100.0

	q := qb.Build(SearchRequest{
		Query:    "phone",
		Category: "electronics",
		Brand:    "nova",
		IsActive: &active,
		Tags:     []string{"sale", "new"},
		MinPrice: &min,
		MaxPrice: &max,
		SortBy:   "price",
		SortDir:  "asc",
	})

	query, ok := q["query"].(map[string]interface{})
	if !ok {
		t.Fatalf("query missing")
	}
	boolQ, ok := query["bool"].(map[string]interface{})
	if !ok {
		t.Fatalf("bool query missing")
	}
	if _, ok := boolQ["must"]; !ok {
		t.Fatalf("must clause missing")
	}
	filters, ok := boolQ["filter"].([]interface{})
	if !ok || len(filters) < 5 {
		t.Fatalf("expected multiple filters")
	}
	if _, ok := q["sort"]; !ok {
		t.Fatalf("sort missing")
	}
}

func TestElasticsearchQueryBuilder_DefaultSortWhenNoQuery(t *testing.T) {
	qb := NewElasticsearchQueryBuilder()
	q := qb.Build(SearchRequest{})
	if _, ok := q["sort"]; !ok {
		t.Fatalf("expected default created_at sort")
	}
}
