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
	deleteFn func(ctx context.Context, id string) error
	bulkFn   func(ctx context.Context, products []Product) (int, error)
	searchFn func(ctx context.Context, req SearchRequest) (*SearchResponse, error)
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

func (m mockProductRepository) Delete(ctx context.Context, id string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m mockProductRepository) BulkIndex(ctx context.Context, products []Product) (int, error) {
	if m.bulkFn != nil {
		return m.bulkFn(ctx, products)
	}
	return 0, nil
}

func (m mockProductRepository) Search(ctx context.Context, req SearchRequest) (*SearchResponse, error) {
	if m.searchFn != nil {
		return m.searchFn(ctx, req)
	}
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

func TestProductService_UpdateProduct_Success(t *testing.T) {
	t.Parallel()

	name := "Updated"
	price := 42.0
	updatedCalled := false

	svc := NewProductService(mockProductRepository{
		getFn: func(context.Context, string) (*Product, error) {
			return &Product{ID: "p-1", Name: "Old", Price: 10}, nil
		},
		updateFn: func(_ context.Context, id string, p Product) error {
			updatedCalled = true
			if id != "p-1" || p.Name != "Updated" || p.Price != 42 {
				t.Fatalf("unexpected update payload: %+v", p)
			}
			return nil
		},
	}, zap.NewNop())

	got, err := svc.UpdateProduct(context.Background(), "p-1", UpdateProductRequest{Name: &name, Price: &price})
	if err != nil {
		t.Fatalf("update product failed: %v", err)
	}
	if !updatedCalled {
		t.Fatalf("expected repository Update to be called")
	}
	if got.Name != "Updated" || got.Price != 42 {
		t.Fatalf("unexpected updated product: %+v", got)
	}
}

func TestProductService_Delete_Bulk_Search(t *testing.T) {
	t.Parallel()

	svc := NewProductService(mockProductRepository{
		deleteFn: func(context.Context, string) error { return nil },
		bulkFn: func(_ context.Context, products []Product) (int, error) {
			if len(products) != 2 {
				t.Fatalf("expected 2 products, got %d", len(products))
			}
			if products[0].ID == "" || products[1].ID == "" {
				t.Fatalf("expected generated ids")
			}
			return 2, nil
		},
		searchFn: func(_ context.Context, req SearchRequest) (*SearchResponse, error) {
			if req.Query != "phone" {
				t.Fatalf("unexpected query: %q", req.Query)
			}
			return &SearchResponse{Total: 1}, nil
		},
	}, zap.NewNop())

	if err := svc.DeleteProduct(context.Background(), "p-1"); err != nil {
		t.Fatalf("delete product: %v", err)
	}

	count, err := svc.BulkIndex(context.Background(), BulkIndexRequest{Products: []CreateProductRequest{
		{Name: "A", Category: "cat", Price: 1},
		{Name: "B", Category: "cat", Price: 2},
	}})
	if err != nil || count != 2 {
		t.Fatalf("bulk index failed: count=%d err=%v", count, err)
	}

	resp, err := svc.Search(context.Background(), SearchRequest{Query: "phone"})
	if err != nil || resp.Total != 1 {
		t.Fatalf("search failed: resp=%+v err=%v", resp, err)
	}
}
