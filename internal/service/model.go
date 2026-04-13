package service

import "time"

// ──────────────────────────────────────────────
// Domain model
// ──────────────────────────────────────────────

// Product is the document stored in Elasticsearch.
type Product struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	Brand       string    `json:"brand"`
	Price       float64   `json:"price"`
	Stock       int       `json:"stock"`
	Tags        []string  `json:"tags"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ──────────────────────────────────────────────
// Request / Response DTOs
// ──────────────────────────────────────────────

// SearchRequest defines query parameters for a product search.
type SearchRequest struct {
	Query    string   `form:"q"         json:"q"`
	Category string   `form:"category"  json:"category"`
	Brand    string   `form:"brand"     json:"brand"`
	MinPrice *float64 `form:"min_price" json:"min_price"`
	MaxPrice *float64 `form:"max_price" json:"max_price"`
	Tags     []string `form:"tags"      json:"tags"`
	IsActive *bool    `form:"is_active" json:"is_active"`
	SortBy   string   `form:"sort_by"   json:"sort_by"`   // field name
	SortDir  string   `form:"sort_dir"  json:"sort_dir"`  // asc | desc
	Page     int      `form:"page"      json:"page"`
	PageSize int      `form:"page_size" json:"page_size"`
}

// SearchResponse wraps a paginated list of products with metadata.
type SearchResponse struct {
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
	Took     int         `json:"took_ms"`
	Products []Product   `json:"products"`
}

// CreateProductRequest is the payload for creating a new product.
type CreateProductRequest struct {
	Name        string   `json:"name"        binding:"required"`
	Description string   `json:"description"`
	Category    string   `json:"category"    binding:"required"`
	Brand       string   `json:"brand"`
	Price       float64  `json:"price"       binding:"required,gt=0"`
	Stock       int      `json:"stock"       binding:"gte=0"`
	Tags        []string `json:"tags"`
	IsActive    bool     `json:"is_active"`
}

// UpdateProductRequest is the partial-update payload.
type UpdateProductRequest struct {
	Name        *string  `json:"name"`
	Description *string  `json:"description"`
	Category    *string  `json:"category"`
	Brand       *string  `json:"brand"`
	Price       *float64 `json:"price"`
	Stock       *int     `json:"stock"`
	Tags        []string `json:"tags"`
	IsActive    *bool    `json:"is_active"`
}

// BulkIndexRequest wraps multiple products for a bulk-index operation.
type BulkIndexRequest struct {
	Products []CreateProductRequest `json:"products" binding:"required,min=1"`
}
