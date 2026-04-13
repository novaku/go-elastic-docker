package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/novaku/go-elastic-search/internal/service"
	"go.uber.org/zap"
)

type mockProductUseCase struct {
	getProductFn func(ctx context.Context, id string) (*service.Product, error)
}

func init() {
	gin.SetMode(gin.TestMode)
}

func (m mockProductUseCase) Search(context.Context, service.SearchRequest) (*service.SearchResponse, error) {
	return &service.SearchResponse{}, nil
}

func (m mockProductUseCase) CreateProduct(context.Context, service.CreateProductRequest) (*service.Product, error) {
	return &service.Product{ID: "p-1"}, nil
}

func (m mockProductUseCase) BulkIndex(context.Context, service.BulkIndexRequest) (int, error) {
	return 1, nil
}

func (m mockProductUseCase) GetProduct(ctx context.Context, id string) (*service.Product, error) {
	if m.getProductFn != nil {
		return m.getProductFn(ctx, id)
	}
	return &service.Product{ID: id}, nil
}

func (m mockProductUseCase) UpdateProduct(context.Context, string, service.UpdateProductRequest) (*service.Product, error) {
	return &service.Product{ID: "p-1"}, nil
}

func (m mockProductUseCase) DeleteProduct(context.Context, string) error {
	return nil
}

func TestProductHandler_GetByID_NotFound(t *testing.T) {
	r := gin.New()
	h := NewProductHandler(mockProductUseCase{
		getProductFn: func(context.Context, string) (*service.Product, error) {
			return nil, service.ErrNotFound
		},
	}, zap.NewNop())
	h.RegisterRoutes(r.Group("/v1"))

	req := httptest.NewRequest(http.MethodGet, "/v1/products/missing", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if body["message"] != "product not found" {
		t.Fatalf("unexpected message: %q", body["message"])
	}
}

func TestProductHandler_Create_BadRequest(t *testing.T) {
	r := gin.New()
	h := NewProductHandler(mockProductUseCase{}, zap.NewNop())
	h.RegisterRoutes(r.Group("/v1"))

	req := httptest.NewRequest(http.MethodPost, "/v1/products", bytes.NewBufferString(`{"name":`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if body["message"] != "invalid request body" {
		t.Fatalf("unexpected message: %v", body["message"])
	}
}

func TestProductHandler_GetByID_InternalError(t *testing.T) {
	r := gin.New()
	h := NewProductHandler(mockProductUseCase{
		getProductFn: func(context.Context, string) (*service.Product, error) {
			return nil, errors.New("db down")
		},
	}, zap.NewNop())
	h.RegisterRoutes(r.Group("/v1"))

	req := httptest.NewRequest(http.MethodGet, "/v1/products/p-1", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if body["message"] != "failed to get product" {
		t.Fatalf("unexpected message: %q", body["message"])
	}
}
