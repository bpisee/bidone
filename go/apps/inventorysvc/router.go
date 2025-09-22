package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/go-playground/validator/v10"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/lee/BidOne/shared/contracts"
	"github.com/lee/BidOne/shared/pkg/metrics"
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
	InventorySvc     contracts.InventoryService
	MetricsCollector *metrics.MetricsCollector
	BusinessMetrics  *metrics.BusinessMetricsCollector
	echo             *echo.Echo
}

// NewServer creates a new server instance with the provided inventory service
func NewServer(svc contracts.InventoryService) *Server {
	// Configure optimal worker count for high performance
	optimalWorkers := getOptimalWorkerCount()
	runtime.GOMAXPROCS(optimalWorkers)

	// Initialize metrics collector with larger buffer for high throughput
	bufferSize := getEnvInt("METRICS_BUFFER_SIZE", 10000)
	metricsCollector := metrics.NewMetricsCollector(bufferSize)
	businessMetrics := metrics.NewBusinessMetricsCollector(metricsCollector)

	e := echo.New()

	// High-performance Echo configuration
	e.HideBanner = true
	e.HidePort = true

	// Disable debug mode for production performance
	e.Debug = false

	// Disable HTTP/2 for maximum HTTP/1.1 performance if needed
	e.DisableHTTP2 = getEnvBool("DISABLE_HTTP2", false)

	s := &Server{
		InventorySvc:     svc,
		MetricsCollector: metricsCollector,
		BusinessMetrics:  businessMetrics,
		echo:             e,
	}

	// Configure underlying net/http server for high-throughput (100k+ RPS)
	readHeaderMs := getEnvInt("SERVER_READ_HEADER_TIMEOUT_MS", 100)    // Reduced for faster processing
	readMs := getEnvInt("SERVER_READ_TIMEOUT_MS", 1000)                // Reduced timeout
	writeMs := getEnvInt("SERVER_WRITE_TIMEOUT_MS", 2000)              // Reduced timeout
	idleMs := getEnvInt("SERVER_IDLE_TIMEOUT_MS", 30000)               // Reduced idle timeout
	maxHeaderBytes := getEnvInt("SERVER_MAX_HEADER_BYTES", 1<<16)      // Reduced header size (64KB)

	// High-performance server configuration
	e.Server = &http.Server{
		ReadHeaderTimeout: time.Duration(readHeaderMs) * time.Millisecond,
		ReadTimeout:       time.Duration(readMs) * time.Millisecond,
		WriteTimeout:      time.Duration(writeMs) * time.Millisecond,
		IdleTimeout:       time.Duration(idleMs) * time.Millisecond,
		MaxHeaderBytes:    maxHeaderBytes,

		// High-performance settings for 100k+ RPS
		ConnState: func(c net.Conn, cs http.ConnState) {
			// Optional: Track connection states for monitoring
		},
	}

	// Configure keep-alive settings for high throughput
	if !getEnvBool("ENABLE_KEEPALIVE", true) {
		e.Server.SetKeepAlivesEnabled(false)
	}

	// Set up custom validator
	s.echo.Validator = &CustomValidator{validator: validator.New()}

	s.setupRoutes()
	s.setupMiddleware()

	return s
}

// setupMiddleware configures the Echo middleware stack for high performance
func (s *Server) setupMiddleware() {
	// Pre-middleware: Normalize paths to reduce route misses (minimal overhead)
	s.echo.Pre(middleware.RemoveTrailingSlash())

	// High-performance middleware configuration

	// 1. Metrics middleware (lightweight, early in chain)
	if getEnvBool("ENABLE_METRICS", true) {
		s.echo.Use(metrics.HTTPMetricsMiddleware(s.MetricsCollector))
	}

	// 2. Essential middleware only for high throughput
	// Disable request logging in production for performance
	if getEnvBool("ENABLE_REQUEST_LOG", false) {
		s.echo.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
			Format: "${time_rfc3339_nano} ${method} ${uri} ${status} ${latency_human}\n",
			Output: os.Stdout,
		}))
	}

	// Recovery middleware (essential for stability)
	s.echo.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{
		DisableStackAll: true, // Disable stack traces for performance
		DisablePrintStack: true,
	}))

	// Request ID (lightweight)
	if getEnvBool("ENABLE_REQUEST_ID", true) {
		s.echo.Use(middleware.RequestID())
	}

	// 3. Optional middleware (disabled by default for max performance)

	// CORS middleware (only if needed)
	if getEnvBool("ENABLE_CORS", false) {
		s.echo.Use(middleware.CORSWithConfig(middleware.CORSConfig{
			AllowOrigins:     []string{"*"},
			AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
			AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
			AllowCredentials: false,
			MaxAge:           86400, // Cache preflight for 24 hours
		}))
	}

	// Security headers (disabled by default for max performance)
	if getEnvBool("ENABLE_SECURE_HEADERS", false) {
		s.echo.Use(middleware.SecureWithConfig(middleware.SecureConfig{
			XSSProtection:         "1; mode=block",
			ContentTypeNosniff:    "nosniff",
			XFrameOptions:         "DENY",
			HSTSMaxAge:            31536000,
			ContentSecurityPolicy: "default-src 'self'",
		}))
	}

	// Rate limiting (optional, for DDoS protection)
	// Note: Rate limiting is disabled by default for maximum performance
	// For production use, consider implementing custom rate limiting or using a reverse proxy
	if getEnvBool("ENABLE_RATE_LIMIT", false) {
		// Custom rate limiting implementation would go here
		// For now, we skip this to avoid dependency issues
		s.echo.Logger.Warn("Rate limiting requested but not implemented - use reverse proxy instead")
	}
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

	// Conditionally protect endpoints with JWT
	if os.Getenv("REQUIRE_AUTH") == "true" {
		jwtSecret := os.Getenv("JWT_SECRET")
		if jwtSecret == "" {
			jwtSecret = "dev-secret"
		}
		jwtConfig := echojwt.Config{
			SigningKey:  []byte(jwtSecret),
			TokenLookup: "header:Authorization",
		}
		products.Use(echojwt.WithConfig(jwtConfig))
	}

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

// Start starts the HTTP server on the specified address with high-performance configuration
func (s *Server) Start(address string) error {
	// For maximum compatibility, use Echo's built-in start method
	// The high-performance settings are already configured in the server
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

// getEnvInt retrieves an integer environment variable with a default value
func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

// getEnvBool retrieves a boolean environment variable with a default value
func getEnvBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		switch v {
		case "true", "1", "yes", "on":
			return true
		case "false", "0", "no", "off":
			return false
		}
	}
	return def
}

// getOptimalWorkerCount returns the optimal number of workers based on CPU cores
func getOptimalWorkerCount() int {
	cores := runtime.NumCPU()
	// For I/O bound workloads, use 2-4x CPU cores
	// For CPU bound workloads, use 1x CPU cores
	multiplier := getEnvInt("WORKER_MULTIPLIER", 2)
	return cores * multiplier
}
