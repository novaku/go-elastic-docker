package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/novaku/go-elastic-search/internal/service"
	"go.uber.org/zap"
)

type ProductHandler struct {
	service service.ProductUseCase
	logger  *zap.Logger
}

type BulkIndexResponse struct {
	Indexed int `json:"indexed"`
}

func NewProductHandler(svc service.ProductUseCase, logger *zap.Logger) *ProductHandler {
	return &ProductHandler{service: svc, logger: logger}
}

func (h *ProductHandler) writeServiceError(c *gin.Context, err error, notFoundMessage string, internalMessage string, logMessage string, fields ...zap.Field) {
	if errors.Is(err, service.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"message": notFoundMessage})
		return
	}

	allFields := append([]zap.Field{zap.Error(err)}, fields...)
	h.logger.Error(logMessage, allFields...)
	c.JSON(http.StatusInternalServerError, gin.H{"message": internalMessage})
}

func (h *ProductHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/products", h.Search)
	rg.POST("/products", h.Create)
	rg.POST("/products/bulk", h.BulkIndex)
	rg.GET("/products/:id", h.GetByID)
	rg.PUT("/products/:id", h.Update)
	rg.DELETE("/products/:id", h.Delete)
}

// Search godoc
// @Summary Search products
// @Description Search products with full-text query and filters.
// @Tags products
// @Accept json
// @Produce json
// @Param q query string false "Full text query"
// @Param category query string false "Category filter"
// @Param brand query string false "Brand filter"
// @Param min_price query number false "Minimum price"
// @Param max_price query number false "Maximum price"
// @Param tags query []string false "Tags filter" collectionFormat(multi)
// @Param is_active query boolean false "Active status filter"
// @Param sort_by query string false "Sort field"
// @Param sort_dir query string false "Sort direction (asc|desc)"
// @Param page query int false "Page number"
// @Param page_size query int false "Page size"
// @Success 200 {object} service.SearchResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /v1/products [get]
func (h *ProductHandler) Search(c *gin.Context) {
	var req service.SearchRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid query parameters", "error": err.Error()})
		return
	}

	resp, err := h.service.Search(c.Request.Context(), req)
	if err != nil {
		h.writeServiceError(c, err, "product not found", "failed to search products", "search failed")
		return
	}

	c.JSON(http.StatusOK, resp)
}

// Create godoc
// @Summary Create product
// @Description Create a new product document.
// @Tags products
// @Accept json
// @Produce json
// @Param request body service.CreateProductRequest true "Create product payload"
// @Success 201 {object} service.Product
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /v1/products [post]
func (h *ProductHandler) Create(c *gin.Context) {
	var req service.CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body", "error": err.Error()})
		return
	}

	product, err := h.service.CreateProduct(c.Request.Context(), req)
	if err != nil {
		h.writeServiceError(c, err, "product not found", "failed to create product", "create product failed")
		return
	}

	c.JSON(http.StatusCreated, product)
}

// BulkIndex godoc
// @Summary Bulk index products
// @Description Create multiple products in one request.
// @Tags products
// @Accept json
// @Produce json
// @Param request body service.BulkIndexRequest true "Bulk index payload"
// @Success 200 {object} BulkIndexResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /v1/products/bulk [post]
func (h *ProductHandler) BulkIndex(c *gin.Context) {
	var req service.BulkIndexRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body", "error": err.Error()})
		return
	}

	count, err := h.service.BulkIndex(c.Request.Context(), req)
	if err != nil {
		h.writeServiceError(c, err, "product not found", "failed to bulk index products", "bulk index failed")
		return
	}

	c.JSON(http.StatusOK, BulkIndexResponse{Indexed: count})
}

// GetByID godoc
// @Summary Get product by ID
// @Description Retrieve one product by document ID.
// @Tags products
// @Accept json
// @Produce json
// @Param id path string true "Product ID"
// @Success 200 {object} service.Product
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /v1/products/{id} [get]
func (h *ProductHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	product, err := h.service.GetProduct(c.Request.Context(), id)
	if err != nil {
		h.writeServiceError(c, err, "product not found", "failed to get product", "get product failed", zap.String("id", id))
		return
	}

	c.JSON(http.StatusOK, product)
}

// Update godoc
// @Summary Update product
// @Description Partially update a product by ID.
// @Tags products
// @Accept json
// @Produce json
// @Param id path string true "Product ID"
// @Param request body service.UpdateProductRequest true "Update payload"
// @Success 200 {object} service.Product
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /v1/products/{id} [put]
func (h *ProductHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var req service.UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body", "error": err.Error()})
		return
	}

	updated, err := h.service.UpdateProduct(c.Request.Context(), id, req)
	if err != nil {
		h.writeServiceError(c, err, "product not found", "failed to update product", "update product failed", zap.String("id", id))
		return
	}

	c.JSON(http.StatusOK, updated)
}

// Delete godoc
// @Summary Delete product
// @Description Delete one product by ID.
// @Tags products
// @Accept json
// @Produce json
// @Param id path string true "Product ID"
// @Success 204
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /v1/products/{id} [delete]
func (h *ProductHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	err := h.service.DeleteProduct(c.Request.Context(), id)
	if err != nil {
		h.writeServiceError(c, err, "product not found", "failed to delete product", "delete product failed", zap.String("id", id))
		return
	}

	c.Status(http.StatusNoContent)
}
