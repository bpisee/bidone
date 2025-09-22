package metrics

import (
	"testing"
	"time"
)

func TestMetricsCollector(t *testing.T) {
	collector := NewMetricsCollector(100)
	defer collector.Close()

	// Test counter metric
	collector.RecordCounter("test_counter", 1, map[string]string{"label": "value"})
	collector.RecordCounter("test_counter", 2, map[string]string{"label": "value"})

	// Test gauge metric
	collector.RecordGauge("test_gauge", 42.5, map[string]string{"type": "test"})

	// Test timing metric
	collector.RecordTiming("test_timing", 100*time.Millisecond, map[string]string{"operation": "test"})

	// Give goroutine time to process
	time.Sleep(10 * time.Millisecond)

	metrics := collector.GetMetrics()

	// Check counter was aggregated
	counterKey := collector.generateKey("test_counter", map[string]string{"label": "value"})
	if counter, exists := metrics[counterKey]; !exists {
		t.Error("Counter metric not found")
	} else if counter.Value != 3 {
		t.Errorf("Expected counter value 3, got %f", counter.Value)
	}

	// Check gauge
	gaugeKey := collector.generateKey("test_gauge", map[string]string{"type": "test"})
	if gauge, exists := metrics[gaugeKey]; !exists {
		t.Error("Gauge metric not found")
	} else if gauge.Value != 42.5 {
		t.Errorf("Expected gauge value 42.5, got %f", gauge.Value)
	}

	// Check timing
	timingKey := collector.generateKey("test_timing", map[string]string{"operation": "test"})
	if timing, exists := metrics[timingKey]; !exists {
		t.Error("Timing metric not found")
	} else if timing.Value != 100 {
		t.Errorf("Expected timing value 100ms, got %f", timing.Value)
	}
}

func TestMetricsSummary(t *testing.T) {
	collector := NewMetricsCollector(100)
	defer collector.Close()

	collector.RecordCounter("counter1", 1, nil)
	collector.RecordGauge("gauge1", 10, nil)
	collector.RecordTiming("timing1", time.Second, nil)

	// Give goroutine time to process
	time.Sleep(10 * time.Millisecond)

	summary := collector.GetMetricsSummary()

	if totalMetrics, ok := summary["total_metrics"].(int); !ok || totalMetrics != 3 {
		t.Errorf("Expected 3 total metrics, got %v", summary["total_metrics"])
	}

	byType := summary["by_type"].(map[MetricType]int)
	if byType[MetricTypeCounter] != 1 {
		t.Errorf("Expected 1 counter metric, got %d", byType[MetricTypeCounter])
	}
	if byType[MetricTypeGauge] != 1 {
		t.Errorf("Expected 1 gauge metric, got %d", byType[MetricTypeGauge])
	}
	if byType[MetricTypeTiming] != 1 {
		t.Errorf("Expected 1 timing metric, got %d", byType[MetricTypeTiming])
	}
}

func TestMetricsReset(t *testing.T) {
	collector := NewMetricsCollector(100)
	defer collector.Close()

	collector.RecordCounter("test", 1, nil)

	// Give goroutine time to process
	time.Sleep(10 * time.Millisecond)

	metrics := collector.GetMetrics()
	if len(metrics) != 1 {
		t.Errorf("Expected 1 metric before reset, got %d", len(metrics))
	}

	collector.Reset()
	metrics = collector.GetMetrics()
	if len(metrics) != 0 {
		t.Errorf("Expected 0 metrics after reset, got %d", len(metrics))
	}
}

func TestBusinessMetricsCollector(t *testing.T) {
	collector := NewMetricsCollector(100)
	defer collector.Close()

	businessMetrics := NewBusinessMetricsCollector(collector)

	// Test various business metrics
	businessMetrics.RecordProductCreated("Test Product")
	businessMetrics.RecordProductUpdated("test-id")
	businessMetrics.RecordProductDeleted("test-id")
	businessMetrics.RecordProductViewed("test-id")
	businessMetrics.RecordProductsListed(10, true)
	businessMetrics.RecordValidationError("create", "name")
	businessMetrics.RecordInventoryLevel("test-id", 100)
	businessMetrics.RecordServiceHealth(true, "api")

	// Give goroutine time to process
	time.Sleep(10 * time.Millisecond)

	metrics := collector.GetMetrics()

	// Should have multiple metrics recorded
	if len(metrics) < 5 {
		t.Errorf("Expected at least 5 metrics, got %d", len(metrics))
	}

	// Check that we have different types of metrics
	summary := collector.GetMetricsSummary()
	byType := summary["by_type"].(map[MetricType]int)

	if byType[MetricTypeCounter] == 0 {
		t.Error("Expected some counter metrics")
	}
	if byType[MetricTypeGauge] == 0 {
		t.Error("Expected some gauge metrics")
	}
}

func TestClose(t *testing.T) {
	t.Run("SingleClose", func(t *testing.T) {
		collector := NewMetricsCollector(100)

		// Record some metrics
		collector.RecordCounter("test", 1, nil)
		time.Sleep(10 * time.Millisecond)

		// Close should work without error
		err := collector.Close()
		if err != nil {
			t.Errorf("Expected no error on close, got %v", err)
		}

		// Verify metrics are still accessible after close
		metrics := collector.GetMetrics()
		if len(metrics) != 1 {
			t.Errorf("Expected 1 metric after close, got %d", len(metrics))
		}
	})

	t.Run("MultipleClose", func(t *testing.T) {
		collector := NewMetricsCollector(100)

		// First close should work
		err1 := collector.Close()
		if err1 != nil {
			t.Errorf("Expected no error on first close, got %v", err1)
		}

		// Second close should also work (idempotent)
		err2 := collector.Close()
		if err2 != nil {
			t.Errorf("Expected no error on second close, got %v", err2)
		}

		// Third close should also work
		err3 := collector.Close()
		if err3 != nil {
			t.Errorf("Expected no error on third close, got %v", err3)
		}
	})

	t.Run("RecordAfterClose", func(t *testing.T) {
		collector := NewMetricsCollector(100)

		// Close the collector
		err := collector.Close()
		if err != nil {
			t.Errorf("Expected no error on close, got %v", err)
		}

		// Recording after close should not panic (should be gracefully ignored)
		collector.RecordCounter("after_close", 1, nil)
		collector.RecordGauge("after_close_gauge", 42, nil)
		collector.RecordTiming("after_close_timing", time.Millisecond, nil)

		// Should not crash or cause issues
		time.Sleep(10 * time.Millisecond)
	})
}

func BenchmarkMetricsCollection(b *testing.B) {
	collector := NewMetricsCollector(1000)
	defer collector.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			collector.RecordCounter("benchmark_counter", 1, map[string]string{
				"worker": "test",
			})
		}
	})
}
