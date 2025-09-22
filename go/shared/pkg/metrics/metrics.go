package metrics

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

// MetricType represents the type of metric
type MetricType string

const (
	MetricTypeCounter   MetricType = "counter"
	MetricTypeGauge     MetricType = "gauge"
	MetricTypeHistogram MetricType = "histogram"
	MetricTypeTiming    MetricType = "timing"
)

// Metric represents a single metric data point
type Metric struct {
	Name      string                 `json:"name"`
	Type      MetricType             `json:"type"`
	Value     float64                `json:"value"`
	Labels    map[string]string      `json:"labels,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// MetricsCollector collects and manages metrics using goroutines and channels
type MetricsCollector struct {
	metricsChan chan Metric
	metrics     map[string]*Metric
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	closed      bool
	closeMu     sync.Mutex
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(bufferSize int) *MetricsCollector {
	ctx, cancel := context.WithCancel(context.Background())

	mc := &MetricsCollector{
		metricsChan: make(chan Metric, bufferSize),
		metrics:     make(map[string]*Metric),
		ctx:         ctx,
		cancel:      cancel,
	}

	// Start the metrics collection goroutine
	mc.wg.Add(1)
	go mc.collectMetrics()

	return mc
}

// collectMetrics runs in a goroutine to collect metrics from the channel
func (mc *MetricsCollector) collectMetrics() {
	defer mc.wg.Done()

	for {
		select {
		case metric := <-mc.metricsChan:
			mc.processMetric(metric)
		case <-mc.ctx.Done():
			// Drain remaining metrics before shutting down
			for {
				select {
				case metric := <-mc.metricsChan:
					mc.processMetric(metric)
				default:
					return
				}
			}
		}
	}
}

// processMetric processes a single metric
func (mc *MetricsCollector) processMetric(metric Metric) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	key := mc.generateKey(metric.Name, metric.Labels)

	switch metric.Type {
	case MetricTypeCounter:
		if existing, exists := mc.metrics[key]; exists {
			existing.Value += metric.Value
			existing.Timestamp = metric.Timestamp
		} else {
			mc.metrics[key] = &metric
		}
	case MetricTypeGauge:
		mc.metrics[key] = &metric
	case MetricTypeHistogram, MetricTypeTiming:
		// For histograms and timings, we'll store the latest value
		// In a production system, you'd want to maintain buckets/percentiles
		mc.metrics[key] = &metric
	}
}

// generateKey generates a unique key for a metric based on name and labels
func (mc *MetricsCollector) generateKey(name string, labels map[string]string) string {
	if len(labels) == 0 {
		return name
	}

	// Create a consistent key by serializing labels
	labelBytes, _ := json.Marshal(labels)
	return name + ":" + string(labelBytes)
}

// RecordCounter records a counter metric
func (mc *MetricsCollector) RecordCounter(name string, value float64, labels map[string]string) {
	mc.closeMu.Lock()
	if mc.closed {
		mc.closeMu.Unlock()
		return // Silently ignore if closed
	}
	mc.closeMu.Unlock()

	metric := Metric{
		Name:      name,
		Type:      MetricTypeCounter,
		Value:     value,
		Labels:    labels,
		Timestamp: time.Now(),
	}

	select {
	case mc.metricsChan <- metric:
	default:
		// Channel is full, drop the metric (or implement backpressure)
	}
}

// RecordGauge records a gauge metric
func (mc *MetricsCollector) RecordGauge(name string, value float64, labels map[string]string) {
	mc.closeMu.Lock()
	if mc.closed {
		mc.closeMu.Unlock()
		return // Silently ignore if closed
	}
	mc.closeMu.Unlock()

	metric := Metric{
		Name:      name,
		Type:      MetricTypeGauge,
		Value:     value,
		Labels:    labels,
		Timestamp: time.Now(),
	}

	select {
	case mc.metricsChan <- metric:
	default:
		// Channel is full, drop the metric
	}
}

// RecordTiming records a timing metric
func (mc *MetricsCollector) RecordTiming(name string, duration time.Duration, labels map[string]string) {
	mc.closeMu.Lock()
	if mc.closed {
		mc.closeMu.Unlock()
		return // Silently ignore if closed
	}
	mc.closeMu.Unlock()

	metric := Metric{
		Name:      name,
		Type:      MetricTypeTiming,
		Value:     float64(duration.Nanoseconds()) / 1e6, // Convert to milliseconds
		Labels:    labels,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"unit": "milliseconds",
		},
	}

	select {
	case mc.metricsChan <- metric:
	default:
		// Channel is full, drop the metric
	}
}

// GetMetrics returns all collected metrics
func (mc *MetricsCollector) GetMetrics() map[string]*Metric {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	// Create a copy to avoid race conditions
	result := make(map[string]*Metric, len(mc.metrics))
	for k, v := range mc.metrics {
		// Deep copy the metric
		metricCopy := *v
		if v.Labels != nil {
			metricCopy.Labels = make(map[string]string)
			for lk, lv := range v.Labels {
				metricCopy.Labels[lk] = lv
			}
		}
		if v.Metadata != nil {
			metricCopy.Metadata = make(map[string]interface{})
			for mk, mv := range v.Metadata {
				metricCopy.Metadata[mk] = mv
			}
		}
		result[k] = &metricCopy
	}

	return result
}

// GetMetricsSummary returns a summary of metrics grouped by type
func (mc *MetricsCollector) GetMetricsSummary() map[string]interface{} {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	summary := map[string]interface{}{
		"total_metrics": len(mc.metrics),
		"by_type":       make(map[MetricType]int),
		"last_updated":  time.Now(),
	}

	byType := summary["by_type"].(map[MetricType]int)
	for _, metric := range mc.metrics {
		byType[metric.Type]++
	}

	return summary
}

// Reset clears all collected metrics
func (mc *MetricsCollector) Reset() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.metrics = make(map[string]*Metric)
}

// Close gracefully shuts down the metrics collector
func (mc *MetricsCollector) Close() error {
	mc.closeMu.Lock()
	defer mc.closeMu.Unlock()

	// Check if already closed
	if mc.closed {
		return nil
	}

	// Mark as closed first
	mc.closed = true

	// Cancel context and wait for goroutine
	mc.cancel()
	mc.wg.Wait()

	// Close channel
	close(mc.metricsChan)

	return nil
}
