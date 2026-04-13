package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type ProductService struct {
	repo   ProductRepository
	logger *zap.Logger
}

func NewProductService(repo ProductRepository, logger *zap.Logger) *ProductService {
	return &ProductService{repo: repo, logger: logger}
}

func (s *ProductService) CreateProduct(ctx context.Context, req CreateProductRequest) (*Product, error) {
	now := time.Now().UTC()
	product := Product{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
		Category:    req.Category,
		Brand:       req.Brand,
		Price:       req.Price,
		Stock:       req.Stock,
		Tags:        req.Tags,
		IsActive:    req.IsActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.Create(ctx, product); err != nil {
		return nil, err
	}

	return &product, nil
}

func (s *ProductService) GetProduct(ctx context.Context, id string) (*Product, error) {
	return s.repo.Get(ctx, id)
}

func (s *ProductService) UpdateProduct(ctx context.Context, id string, req UpdateProductRequest) (*Product, error) {
	existing, err := s.GetProduct(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.Description != nil {
		existing.Description = *req.Description
	}
	if req.Category != nil {
		existing.Category = *req.Category
	}
	if req.Brand != nil {
		existing.Brand = *req.Brand
	}
	if req.Price != nil {
		existing.Price = *req.Price
	}
	if req.Stock != nil {
		existing.Stock = *req.Stock
	}
	if req.Tags != nil {
		existing.Tags = req.Tags
	}
	if req.IsActive != nil {
		existing.IsActive = *req.IsActive
	}
	existing.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, id, *existing); err != nil {
		return nil, err
	}

	return existing, nil
}

func (s *ProductService) DeleteProduct(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *ProductService) BulkIndex(ctx context.Context, req BulkIndexRequest) (int, error) {
	now := time.Now().UTC()
	products := make([]Product, 0, len(req.Products))

	for _, r := range req.Products {
		product := Product{
			ID:          uuid.New().String(),
			Name:        r.Name,
			Description: r.Description,
			Category:    r.Category,
			Brand:       r.Brand,
			Price:       r.Price,
			Stock:       r.Stock,
			Tags:        r.Tags,
			IsActive:    r.IsActive,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		products = append(products, product)
	}

	return s.repo.BulkIndex(ctx, products)
}

func (s *ProductService) Search(ctx context.Context, req SearchRequest) (*SearchResponse, error) {
	return s.repo.Search(ctx, req)
}

var ErrNotFound = fmt.Errorf("document not found")
