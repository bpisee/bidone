package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/lee/BidOne/pkg/model"
)

// ListProductsResponse represents the response structure for listing products
type ListProductsResponse struct {
	Total    int             `json:"total"`
	Limit    int             `json:"limit"`
	Offset   int             `json:"offset"`
	Products []model.Product `json:"products"`
}

// ListProductsRequest represents the query parameters for listing products
type ListProductsRequest struct {
	Limit  int    `query:"limit"`
	Offset int    `query:"offset"`
	Name   string `query:"name"`
}

// handleHealthCheck provides a simple health check endpoint
func (s *Server) handleHealthCheck(c echo.Context) error {
	// Record health check metric
	s.BusinessMetrics.RecordServiceHealth(true, "api")

	return c.JSON(http.StatusOK, map[string]string{
		"status":  "healthy",
		"service": "inventory-service",
		"version": "1.0.0",
	})
}

// handleMetrics handles GET /metrics - returns all collected metrics
func (s *Server) handleMetrics(c echo.Context) error {
	metrics := s.MetricsCollector.GetMetrics()

	return c.JSON(http.StatusOK, map[string]interface{}{
		"metrics":   metrics,
		"timestamp": time.Now(),
		"count":     len(metrics),
	})
}

// handleMetricsSummary handles GET /metrics/summary - returns metrics summary
func (s *Server) handleMetricsSummary(c echo.Context) error {
	summary := s.MetricsCollector.GetMetricsSummary()

	return c.JSON(http.StatusOK, summary)
}

// handleMetricsReset handles POST /metrics/reset - resets all metrics
func (s *Server) handleMetricsReset(c echo.Context) error {
	s.MetricsCollector.Reset()

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":   "Metrics reset successfully",
		"timestamp": time.Now(),
	})
}

// handleListProducts handles GET /api/v1/products
func (s *Server) handleListProducts(c echo.Context) error {
	var req ListProductsRequest

	// Bind query parameters
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid query parameters",
		})
	}

	// Apply default values and validation
	if req.Limit <= 0 || req.Limit > 1000 {
		req.Limit = 100
	}
	if req.Offset < 0 {
		req.Offset = 0
	}
	req.Name = strings.TrimSpace(req.Name)

	// Get products from service
	products, err := s.InventorySvc.ListProducts(c.Request().Context(), req.Limit, req.Offset, req.Name)
	if err != nil {
		c.Logger().Error("Failed to list products: ", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to retrieve products",
		})
	}

	// Get total count for pagination info
	// Note: This is a simplified approach. In production, you might want
	// the service to return both results and total count in one call
	allProducts, err := s.InventorySvc.ListProducts(c.Request().Context(), 1000, 0, req.Name)
	if err != nil {
		c.Logger().Error("Failed to get total count: ", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to retrieve product count",
		})
	}

	response := ListProductsResponse{
		Total:    len(allProducts),
		Limit:    req.Limit,
		Offset:   req.Offset,
		Products: products,
	}

	// Record business metrics
	hasFilter := req.Name != ""
	s.BusinessMetrics.RecordProductsListed(len(products), hasFilter)

	return c.JSON(http.StatusOK, response)
}

// CreateProductRequest represents the request body for creating a product
type CreateProductRequest struct {
	Name        string `json:"name" validate:"required,min=1" example:"Product Name"`
	Description string `json:"description,omitempty" example:"Product description"`
	PriceCents  int64  `json:"price_cents" validate:"min=0" example:"2999"`
	Quantity    int64  `json:"quantity" validate:"min=0" example:"10"`
}

// formatValidationError formats validation errors into a user-friendly message
func formatValidationError(err error) string {
	var errors []string

	// Check if it's a validator.ValidationErrors
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, fieldError := range validationErrors {
			// Convert field name from PascalCase to snake_case for user-friendly messages
			field := convertFieldName(fieldError.Field())
			tag := fieldError.Tag()

			switch tag {
			case "required":
				errors = append(errors, fmt.Sprintf("%s is required", field))
			case "min":
				if fieldError.Kind().String() == "string" {
					errors = append(errors, fmt.Sprintf("%s is required and cannot be empty", field))
				} else {
					errors = append(errors, fmt.Sprintf("%s is required and must be >= %s", field, fieldError.Param()))
				}
			default:
				errors = append(errors, fmt.Sprintf("%s is invalid", field))
			}
		}
	} else {
		return err.Error()
	}

	if len(errors) > 0 {
		return fmt.Sprintf("validation failed: %s", strings.Join(errors, ", "))
	}

	return "validation failed"
}

// convertFieldName converts PascalCase field names to snake_case for user-friendly error messages
func convertFieldName(fieldName string) string {
	switch fieldName {
	case "Name":
		return "name"
	case "PriceCents":
		return "price_cents"
	case "Quantity":
		return "quantity"
	case "Description":
		return "description"
	default:
		return strings.ToLower(fieldName)
	}
}

// handleCreateProduct handles POST /api/v1/products
func (s *Server) handleCreateProduct(c echo.Context) error {
	var req CreateProductRequest

	// Bind JSON request body
	if err := c.Bind(&req); err != nil {
		c.Logger().Error("Failed to bind product data: ", err)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid JSON format. Please check your request body.",
		})
	}

	// Validate using Echo's built-in validator
	if err := c.Validate(req); err != nil {
		c.Logger().Error("Product validation failed: ", err)

		// Record validation error metrics
		s.BusinessMetrics.RecordValidationError("create", "multiple")

		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": formatValidationError(err),
		})
	}

	// Convert request to product model
	// Note: ID will be auto-generated by the service, so it should NOT be provided in the request
	product := model.Product{
		Name:        strings.TrimSpace(req.Name),
		Description: strings.TrimSpace(req.Description),
		PriceCents:  req.PriceCents,
		Quantity:    req.Quantity,
		// ID, CreatedAt, UpdatedAt will be set by the service/store
	}

	// Create product via service
	created, err := s.InventorySvc.CreateProduct(c.Request().Context(), product)
	if err != nil {
		c.Logger().Error("Failed to create product: ", err)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	// Record business metrics
	s.BusinessMetrics.RecordProductCreated(created.Name)
	s.BusinessMetrics.RecordInventoryLevel(created.ID, created.Quantity)

	return c.JSON(http.StatusCreated, created)
}

// handleGetProduct handles GET /api/v1/products/:id
func (s *Server) handleGetProduct(c echo.Context) error {
	id := c.Param("id")

	// Validate ID parameter
	if strings.TrimSpace(id) == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Product ID is required",
		})
	}

	// Get product from service
	product, err := s.InventorySvc.GetProduct(c.Request().Context(), id)
	if err != nil {
		c.Logger().Error("Failed to get product: ", err)
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Product not found",
		})
	}

	// Record business metrics
	s.BusinessMetrics.RecordProductViewed(product.ID)

	return c.JSON(http.StatusOK, product)
}

// UpdateProductRequest represents the request body for updating a product
type UpdateProductRequest struct {
	Name        string `json:"name" validate:"required,min=1" example:"Updated Product Name"`
	Description string `json:"description,omitempty" example:"Updated product description"`
	PriceCents  int64  `json:"price_cents" validate:"min=0" example:"3999"`
	Quantity    int64  `json:"quantity" validate:"min=0" example:"15"`
}

// handleUpdateProduct handles PUT /api/v1/products/:id
func (s *Server) handleUpdateProduct(c echo.Context) error {
	id := c.Param("id")

	// Validate ID parameter
	if strings.TrimSpace(id) == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Product ID is required",
		})
	}

	var req UpdateProductRequest

	// Bind JSON request body
	if err := c.Bind(&req); err != nil {
		c.Logger().Error("Failed to bind product data: ", err)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid JSON format. Please check your request body.",
		})
	}

	// Validate using Echo's built-in validator
	if err := c.Validate(req); err != nil {
		c.Logger().Error("Product validation failed: ", err)

		// Record validation error metrics
		s.BusinessMetrics.RecordValidationError("update", "multiple")

		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": formatValidationError(err),
		})
	}

	// Convert request to product model
	product := model.Product{
		Name:        strings.TrimSpace(req.Name),
		Description: strings.TrimSpace(req.Description),
		PriceCents:  req.PriceCents,
		Quantity:    req.Quantity,
		// ID, CreatedAt, UpdatedAt will be handled by the service/store
	}

	// Update product via service
	updated, err := s.InventorySvc.UpdateProduct(c.Request().Context(), id, product)
	if err != nil {
		c.Logger().Error("Failed to update product: ", err)

		// Determine appropriate HTTP status code
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}

		return c.JSON(status, map[string]string{
			"error": err.Error(),
		})
	}

	// Record business metrics
	s.BusinessMetrics.RecordProductUpdated(updated.ID)
	s.BusinessMetrics.RecordInventoryLevel(updated.ID, updated.Quantity)

	return c.JSON(http.StatusOK, updated)
}

// handleDeleteProduct handles DELETE /api/v1/products/:id
func (s *Server) handleDeleteProduct(c echo.Context) error {
	id := c.Param("id")

	// Validate ID parameter
	if strings.TrimSpace(id) == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Product ID is required",
		})
	}

	// Delete product via service
	if err := s.InventorySvc.DeleteProduct(c.Request().Context(), id); err != nil {
		c.Logger().Error("Failed to delete product: ", err)
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Product not found",
		})
	}

	// Record business metrics
	s.BusinessMetrics.RecordProductDeleted(id)

	return c.NoContent(http.StatusNoContent)
}
