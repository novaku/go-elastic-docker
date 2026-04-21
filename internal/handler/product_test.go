package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/novaku/go-elastic-search/config"
	"github.com/novaku/go-elastic-search/internal/service"
	"go.uber.org/zap"
)

type mockProductUseCase struct {
	searchFn        func(context.Context, service.SearchRequest) (*service.SearchResponse, error)
	createFn        func(context.Context, service.CreateProductRequest) (*service.Product, error)
	bulkIndexFn     func(context.Context, service.BulkIndexRequest) (int, error)
	getProductFn    func(context.Context, string) (*service.Product, error)
	updateFn        func(context.Context, string, service.UpdateProductRequest) (*service.Product, error)
	deleteProductFn func(context.Context, string) error
}

func init() {
	gin.SetMode(gin.TestMode)
}

func (m mockProductUseCase) Search(context.Context, service.SearchRequest) (*service.SearchResponse, error) {
	if m.searchFn != nil {
		return m.searchFn(context.Background(), service.SearchRequest{})
	}
	return &service.SearchResponse{}, nil
}

func (m mockProductUseCase) CreateProduct(context.Context, service.CreateProductRequest) (*service.Product, error) {
	if m.createFn != nil {
		return m.createFn(context.Background(), service.CreateProductRequest{})
	}
	return &service.Product{ID: "p-1"}, nil
}

func (m mockProductUseCase) BulkIndex(context.Context, service.BulkIndexRequest) (int, error) {
	if m.bulkIndexFn != nil {
		return m.bulkIndexFn(context.Background(), service.BulkIndexRequest{})
	}
	return 1, nil
}

func (m mockProductUseCase) GetProduct(ctx context.Context, id string) (*service.Product, error) {
	if m.getProductFn != nil {
		return m.getProductFn(ctx, id)
	}
	return &service.Product{ID: id}, nil
}

func (m mockProductUseCase) UpdateProduct(context.Context, string, service.UpdateProductRequest) (*service.Product, error) {
	if m.updateFn != nil {
		return m.updateFn(context.Background(), "", service.UpdateProductRequest{})
	}
	return &service.Product{ID: "p-1"}, nil
}

func (m mockProductUseCase) DeleteProduct(context.Context, string) error {
	if m.deleteProductFn != nil {
		return m.deleteProductFn(context.Background(), "")
	}
	return nil
}

func TestProductHandler_Search_Create_Bulk_Update_Delete_Success(t *testing.T) {
	r := gin.New()
	h := NewProductHandler(mockProductUseCase{}, zap.NewNop())
	h.RegisterRoutes(r.Group("/v1"))

	req1 := httptest.NewRequest(http.MethodGet, "/v1/products?q=phone", nil)
	rec1 := httptest.NewRecorder()
	r.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Fatalf("search expected 200, got %d", rec1.Code)
	}

	req2 := httptest.NewRequest(http.MethodPost, "/v1/products", bytes.NewBufferString(`{"name":"Phone","category":"electronics","price":10}`))
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusCreated {
		t.Fatalf("create expected 201, got %d", rec2.Code)
	}

	req3 := httptest.NewRequest(http.MethodPost, "/v1/products/bulk", bytes.NewBufferString(`{"products":[{"name":"Phone","category":"electronics","price":10}]}`))
	req3.Header.Set("Content-Type", "application/json")
	rec3 := httptest.NewRecorder()
	r.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusOK {
		t.Fatalf("bulk expected 200, got %d", rec3.Code)
	}

	req4 := httptest.NewRequest(http.MethodPut, "/v1/products/p-1", bytes.NewBufferString(`{"name":"Updated"}`))
	req4.Header.Set("Content-Type", "application/json")
	rec4 := httptest.NewRecorder()
	r.ServeHTTP(rec4, req4)
	if rec4.Code != http.StatusOK {
		t.Fatalf("update expected 200, got %d", rec4.Code)
	}

	req5 := httptest.NewRequest(http.MethodDelete, "/v1/products/p-1", nil)
	rec5 := httptest.NewRecorder()
	r.ServeHTTP(rec5, req5)
	if rec5.Code != http.StatusNoContent {
		t.Fatalf("delete expected 204, got %d", rec5.Code)
	}
}

func TestProductHandler_ErrorPaths(t *testing.T) {
	r := gin.New()
	h := NewProductHandler(mockProductUseCase{
		searchFn: func(context.Context, service.SearchRequest) (*service.SearchResponse, error) {
			return nil, errors.New("search down")
		},
		bulkIndexFn: func(context.Context, service.BulkIndexRequest) (int, error) {
			return 0, errors.New("bulk down")
		},
		updateFn: func(context.Context, string, service.UpdateProductRequest) (*service.Product, error) {
			return nil, errors.New("update down")
		},
		deleteProductFn: func(context.Context, string) error {
			return errors.New("delete down")
		},
	}, zap.NewNop())
	h.RegisterRoutes(r.Group("/v1"))

	rec1 := httptest.NewRecorder()
	r.ServeHTTP(rec1, httptest.NewRequest(http.MethodGet, "/v1/products", nil))
	if rec1.Code != http.StatusInternalServerError {
		t.Fatalf("search expected 500, got %d", rec1.Code)
	}

	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/v1/products/bulk", bytes.NewBufferString(`{"products":[{"name":"Phone","category":"electronics","price":10}]}`))
	req2.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusInternalServerError {
		t.Fatalf("bulk expected 500, got %d", rec2.Code)
	}

	rec3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodPut, "/v1/products/p-1", bytes.NewBufferString(`{"name":"Updated"}`))
	req3.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusInternalServerError {
		t.Fatalf("update expected 500, got %d", rec3.Code)
	}

	rec4 := httptest.NewRecorder()
	r.ServeHTTP(rec4, httptest.NewRequest(http.MethodDelete, "/v1/products/p-1", nil))
	if rec4.Code != http.StatusInternalServerError {
		t.Fatalf("delete expected 500, got %d", rec4.Code)
	}
}

func TestAuthHandler_Login(t *testing.T) {
	r := gin.New()
	a := NewAuthHandler(&config.JWTConfig{
		Secret:         "secret",
		ExpiryDuration: time.Hour,
		AdminUsername:  "admin",
		AdminPassword:  "pass",
	})
	r.POST("/auth/login", a.Login)

	rec1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(`{"username":"admin"}`))
	req1.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec1.Code)
	}

	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(`{"username":"admin","password":"wrong"}`))
	req2.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec2.Code)
	}

	rec3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(`{"username":"admin","password":"pass"}`))
	req3.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec3.Code)
	}
	if !bytes.Contains(rec3.Body.Bytes(), []byte("token")) {
		t.Fatalf("expected token in response")
	}
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
