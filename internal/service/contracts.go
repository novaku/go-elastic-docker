package service

import "context"

// ProductUseCase defines product operations required by transport layers.
type ProductUseCase interface {
	Search(ctx context.Context, req SearchRequest) (*SearchResponse, error)
	CreateProduct(ctx context.Context, req CreateProductRequest) (*Product, error)
	BulkIndex(ctx context.Context, req BulkIndexRequest) (int, error)
	GetProduct(ctx context.Context, id string) (*Product, error)
	UpdateProduct(ctx context.Context, id string, req UpdateProductRequest) (*Product, error)
	DeleteProduct(ctx context.Context, id string) error
}

// ProductRepository defines persistence operations for products.
type ProductRepository interface {
	EnsureIndex(ctx context.Context) error
	Create(ctx context.Context, product Product) error
	Get(ctx context.Context, id string) (*Product, error)
	Update(ctx context.Context, id string, product Product) error
	Delete(ctx context.Context, id string) error
	BulkIndex(ctx context.Context, products []Product) (int, error)
	Search(ctx context.Context, req SearchRequest) (*SearchResponse, error)
}

// SearchQueryBuilder builds backend-specific query payloads.
type SearchQueryBuilder interface {
	Build(req SearchRequest) map[string]interface{}
}

// HealthChecker checks service health readiness.
type HealthChecker interface {
	Check(ctx context.Context) error
}
