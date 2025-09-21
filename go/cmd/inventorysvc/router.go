package main

import (
	"context"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/lee/BidOne/internal/services"
	"github.com/lee/BidOne/pkg/metrics"
)

// CustomValidator wraps the validator instance
type CustomValidator struct {
	validator *validator.Validate
}

// Validate validates the struct using the validator instance
func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

// Server represents the HTTP server with its dependencies
type Server struct {
	InventorySvc     services.InventoryService
	MetricsCollector *metrics.MetricsCollector
	BusinessMetrics  *metrics.BusinessMetricsCollector
	echo             *echo.Echo
}

// NewServer creates a new server instance with the provided inventory service
func NewServer(svc services.InventoryService) *Server {
	// Initialize metrics collector with buffer size of 1000
	metricsCollector := metrics.NewMetricsCollector(1000)
	businessMetrics := metrics.NewBusinessMetricsCollector(metricsCollector)

	s := &Server{
		InventorySvc:     svc,
		MetricsCollector: metricsCollector,
		BusinessMetrics:  businessMetrics,
		echo:             echo.New(),
	}

	// Set up custom validator
	s.echo.Validator = &CustomValidator{validator: validator.New()}

	s.setupRoutes()
	s.setupMiddleware()

	return s
}

// setupMiddleware configures the Echo middleware stack
func (s *Server) setupMiddleware() {
	// Metrics middleware (should be first to capture all requests)
	s.echo.Use(metrics.HTTPMetricsMiddleware(s.MetricsCollector))

	// Basic middleware
	s.echo.Use(middleware.Logger())
	s.echo.Use(middleware.Recover())
	s.echo.Use(middleware.RequestID())

	// CORS middleware for API access
	s.echo.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
	}))

	// Security headers
	s.echo.Use(middleware.Secure())

	// Note: Timeout middleware removed to avoid issues with long-running requests
}

// setupRoutes configures the API routes
func (s *Server) setupRoutes() {
	// Health check endpoint
	s.echo.GET("/health", s.handleHealthCheck)

	// Metrics endpoints
	s.echo.GET("/metrics", s.handleMetrics)
	s.echo.GET("/metrics/summary", s.handleMetricsSummary)
	s.echo.POST("/metrics/reset", s.handleMetricsReset)

	// API versioning
	v1 := s.echo.Group("/api/v1")

	// Products endpoints
	products := v1.Group("/products")
	products.GET("", s.handleListProducts)
	products.POST("", s.handleCreateProduct)
	products.GET("/:id", s.handleGetProduct)
	products.PUT("/:id", s.handleUpdateProduct)
	products.DELETE("/:id", s.handleDeleteProduct)
}

// Echo returns the underlying Echo instance
func (s *Server) Echo() *echo.Echo {
	return s.echo
}

// Start starts the HTTP server on the specified address
func (s *Server) Start(address string) error {
	return s.echo.Start(address)
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	// Close metrics collector first
	if err := s.MetricsCollector.Close(); err != nil {
		// Log error but don't fail shutdown
	}

	return s.echo.Shutdown(ctx)
}
