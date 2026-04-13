package service

// ElasticsearchQueryBuilder generates Elasticsearch DSL payloads.
type ElasticsearchQueryBuilder struct{}

func NewElasticsearchQueryBuilder() *ElasticsearchQueryBuilder {
	return &ElasticsearchQueryBuilder{}
}

func applySearchDefaults(req SearchRequest) SearchRequest {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 100 {
		req.PageSize = 20
	}
	if req.SortDir != "asc" && req.SortDir != "desc" {
		req.SortDir = "desc"
	}
	return req
}

func (b *ElasticsearchQueryBuilder) Build(req SearchRequest) map[string]interface{} {
	mustClauses := []interface{}{}
	filterClauses := []interface{}{}

	if req.Query != "" {
		mustClauses = append(mustClauses, map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":     req.Query,
				"fields":    []string{"name^3", "name.autocomplete^2", "description", "tags"},
				"type":      "best_fields",
				"fuzziness": "AUTO",
			},
		})
	}

	if req.Category != "" {
		filterClauses = append(filterClauses, map[string]interface{}{
			"term": map[string]interface{}{"category": req.Category},
		})
	}
	if req.Brand != "" {
		filterClauses = append(filterClauses, map[string]interface{}{
			"term": map[string]interface{}{"brand": req.Brand},
		})
	}
	if req.IsActive != nil {
		filterClauses = append(filterClauses, map[string]interface{}{
			"term": map[string]interface{}{"is_active": *req.IsActive},
		})
	}
	if len(req.Tags) > 0 {
		filterClauses = append(filterClauses, map[string]interface{}{
			"terms": map[string]interface{}{"tags": req.Tags},
		})
	}

	if req.MinPrice != nil || req.MaxPrice != nil {
		priceRange := map[string]interface{}{}
		if req.MinPrice != nil {
			priceRange["gte"] = *req.MinPrice
		}
		if req.MaxPrice != nil {
			priceRange["lte"] = *req.MaxPrice
		}
		filterClauses = append(filterClauses, map[string]interface{}{
			"range": map[string]interface{}{"price": priceRange},
		})
	}

	boolQuery := map[string]interface{}{}
	if len(mustClauses) > 0 {
		boolQuery["must"] = mustClauses
	} else {
		boolQuery["must"] = []interface{}{map[string]interface{}{"match_all": map[string]interface{}{}}}
	}
	if len(filterClauses) > 0 {
		boolQuery["filter"] = filterClauses
	}

	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": boolQuery,
		},
	}

	if req.SortBy != "" {
		query["sort"] = []interface{}{
			map[string]interface{}{
				req.SortBy: map[string]interface{}{"order": req.SortDir},
			},
		}
	} else if req.Query == "" {
		query["sort"] = []interface{}{
			map[string]interface{}{
				"created_at": map[string]interface{}{"order": "desc"},
			},
		}
	}

	return query
}
