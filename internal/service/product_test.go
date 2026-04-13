package service

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/zap"
)

type mockProductRepository struct {
	createFn func(ctx context.Context, product Product) error
	getFn    func(ctx context.Context, id string) (*Product, error)
	updateFn func(ctx context.Context, id string, product Product) error
}

func (m mockProductRepository) EnsureIndex(context.Context) error { return nil }

func (m mockProductRepository) Create(ctx context.Context, product Product) error {
	if m.createFn != nil {
		return m.createFn(ctx, product)
	}
	return nil
}

func (m mockProductRepository) Get(ctx context.Context, id string) (*Product, error) {
	if m.getFn != nil {
		return m.getFn(ctx, id)
	}
	return &Product{ID: id}, nil
}

func (m mockProductRepository) Update(ctx context.Context, id string, product Product) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, id, product)
	}
	return nil
}

func (m mockProductRepository) Delete(context.Context, string) error { return nil }

func (m mockProductRepository) BulkIndex(context.Context, []Product) (int, error) {
	return 0, nil
}

func (m mockProductRepository) Search(context.Context, SearchRequest) (*SearchResponse, error) {
	return &SearchResponse{}, nil
}

func TestProductService_CreateProduct(t *testing.T) {
	t.Parallel()

	createdCalled := false
	svc := NewProductService(mockProductRepository{
		createFn: func(_ context.Context, product Product) error {
			createdCalled = true
			if product.ID == "" {
				t.Fatalf("expected generated ID")
			}
			if product.Name != "Phone" {
				t.Fatalf("unexpected name: %q", product.Name)
			}
			if product.CreatedAt.IsZero() || product.UpdatedAt.IsZero() {
				t.Fatalf("timestamps must be set")
			}
			return nil
		},
	}, zap.NewNop())

	_, err := svc.CreateProduct(context.Background(), CreateProductRequest{
		Name:     "Phone",
		Category: "electronics",
		Price:    123.45,
	})
	if err != nil {
		t.Fatalf("create product failed: %v", err)
	}
	if !createdCalled {
		t.Fatalf("expected repository Create to be called")
	}
}

func TestProductService_UpdateProduct_NotFound(t *testing.T) {
	t.Parallel()

	svc := NewProductService(mockProductRepository{
		getFn: func(context.Context, string) (*Product, error) {
			return nil, ErrNotFound
		},
	}, zap.NewNop())

	_, err := svc.UpdateProduct(context.Background(), "missing", UpdateProductRequest{})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
