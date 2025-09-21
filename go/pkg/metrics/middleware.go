package metrics

import (
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

// HTTPMetricsMiddleware creates middleware for collecting HTTP metrics
func HTTPMetricsMiddleware(collector *MetricsCollector) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			// Record request count
			collector.RecordCounter("http_requests_total", 1, map[string]string{
				"method": c.Request().Method,
				"path":   c.Path(),
			})

			// Execute the handler
			err := next(c)

			// Calculate duration
			duration := time.Since(start)

			// Get status code
			status := c.Response().Status
			statusClass := strconv.Itoa(status/100) + "xx"

			// Record response metrics
			labels := map[string]string{
				"method":       c.Request().Method,
				"path":         c.Path(),
				"status":       strconv.Itoa(status),
				"status_class": statusClass,
			}

			// Record response time
			collector.RecordTiming("http_request_duration", duration, labels)

			// Record response count by status
			collector.RecordCounter("http_responses_total", 1, labels)

			// Record response size
			collector.RecordGauge("http_response_size_bytes", float64(c.Response().Size), labels)

			return err
		}
	}
}

// BusinessMetricsCollector provides methods for collecting business-specific metrics
type BusinessMetricsCollector struct {
	collector *MetricsCollector
}

// NewBusinessMetricsCollector creates a new business metrics collector
func NewBusinessMetricsCollector(collector *MetricsCollector) *BusinessMetricsCollector {
	return &BusinessMetricsCollector{
		collector: collector,
	}
}

// RecordProductCreated records when a product is created
func (bmc *BusinessMetricsCollector) RecordProductCreated(productName string) {
	bmc.collector.RecordCounter("products_created_total", 1, map[string]string{
		"operation": "create",
	})

	bmc.collector.RecordGauge("last_product_created", float64(time.Now().Unix()), map[string]string{
		"product_name": productName,
	})
}

// RecordProductUpdated records when a product is updated
func (bmc *BusinessMetricsCollector) RecordProductUpdated(productID string) {
	bmc.collector.RecordCounter("products_updated_total", 1, map[string]string{
		"operation": "update",
	})

	bmc.collector.RecordGauge("last_product_updated", float64(time.Now().Unix()), map[string]string{
		"product_id": productID,
	})
}

// RecordProductDeleted records when a product is deleted
func (bmc *BusinessMetricsCollector) RecordProductDeleted(productID string) {
	bmc.collector.RecordCounter("products_deleted_total", 1, map[string]string{
		"operation": "delete",
	})
}

// RecordProductViewed records when a product is viewed
func (bmc *BusinessMetricsCollector) RecordProductViewed(productID string) {
	bmc.collector.RecordCounter("products_viewed_total", 1, map[string]string{
		"operation": "view",
	})
}

// RecordProductsListed records when products are listed
func (bmc *BusinessMetricsCollector) RecordProductsListed(count int, hasFilter bool) {
	labels := map[string]string{
		"operation": "list",
		"filtered":  strconv.FormatBool(hasFilter),
	}

	bmc.collector.RecordCounter("products_list_requests_total", 1, labels)
	bmc.collector.RecordGauge("products_list_count", float64(count), labels)
}

// RecordValidationError records validation errors
func (bmc *BusinessMetricsCollector) RecordValidationError(operation, field string) {
	bmc.collector.RecordCounter("validation_errors_total", 1, map[string]string{
		"operation": operation,
		"field":     field,
	})
}

// RecordInventoryLevel records current inventory levels
func (bmc *BusinessMetricsCollector) RecordInventoryLevel(productID string, quantity int64) {
	bmc.collector.RecordGauge("inventory_level", float64(quantity), map[string]string{
		"product_id": productID,
	})
}

// RecordServiceHealth records service health metrics
func (bmc *BusinessMetricsCollector) RecordServiceHealth(healthy bool, component string) {
	value := float64(0)
	if healthy {
		value = 1
	}

	bmc.collector.RecordGauge("service_health", value, map[string]string{
		"component": component,
	})
}
