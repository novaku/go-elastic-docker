package router

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/novaku/go-elastic-search/config"
	"github.com/novaku/go-elastic-search/internal/service"
	"go.uber.org/zap"
)

type stubUseCase struct{}

func (s stubUseCase) Search(context.Context, service.SearchRequest) (*service.SearchResponse, error) {
	return &service.SearchResponse{}, nil
}
func (s stubUseCase) CreateProduct(context.Context, service.CreateProductRequest) (*service.Product, error) {
	return &service.Product{}, nil
}
func (s stubUseCase) BulkIndex(context.Context, service.BulkIndexRequest) (int, error) {
	return 0, nil
}
func (s stubUseCase) GetProduct(context.Context, string) (*service.Product, error) {
	return &service.Product{}, nil
}
func (s stubUseCase) UpdateProduct(context.Context, string, service.UpdateProductRequest) (*service.Product, error) {
	return &service.Product{}, nil
}
func (s stubUseCase) DeleteProduct(context.Context, string) error { return nil }

type stubHealthChecker struct {
	err error
}

func (s stubHealthChecker) Check(context.Context) error { return s.err }

func TestRouter_Ready(t *testing.T) {
	t.Parallel()

	r := New(&config.Config{CORSAllowedOrigins: "*"}, stubUseCase{}, stubHealthChecker{}, zap.NewNop())
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if body["status"] != "ready" {
		t.Fatalf("unexpected status: %q", body["status"])
	}
}

func TestRouter_Health_Unhealthy(t *testing.T) {
	t.Parallel()

	r := New(&config.Config{CORSAllowedOrigins: "*"}, stubUseCase{}, stubHealthChecker{err: errors.New("es down")}, zap.NewNop())
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if body["status"] != "unhealthy" {
		t.Fatalf("unexpected status: %q", body["status"])
	}
	if body["elasticsearch"] != "es down" {
		t.Fatalf("unexpected elasticsearch message: %q", body["elasticsearch"])
	}
}
