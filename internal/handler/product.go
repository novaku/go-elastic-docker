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

	c.JSON(http.StatusOK, gin.H{"indexed": count})
}

func (h *ProductHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	product, err := h.service.GetProduct(c.Request.Context(), id)
	if err != nil {
		h.writeServiceError(c, err, "product not found", "failed to get product", "get product failed", zap.String("id", id))
		return
	}

	c.JSON(http.StatusOK, product)
}

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

func (h *ProductHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	err := h.service.DeleteProduct(c.Request.Context(), id)
	if err != nil {
		h.writeServiceError(c, err, "product not found", "failed to delete product", "delete product failed", zap.String("id", id))
		return
	}

	c.Status(http.StatusNoContent)
}
