package ratelimiter

import (
	"fmt"
	"testing"
	"time"
)

// TestExampleAllow demonstrates basic usage of the Allow() method.
// This test shows how rate limiting works by performing 10 operations
// with a rate limiter configured for 5 operations per second.
// The first 5 operations should execute immediately (burst capacity),
// while the remaining 5 will be rate-limited.
func TestExampleAllow(t *testing.T) {
	// Create a rate limiter that allows 5 operations per second
	// with a burst capacity of 5 tokens
	limiter := NewRateLimiter(5)
	defer limiter.Close()

	fmt.Println("Starting rate-limited operations (5 ops/sec):")

	// Perform 10 operations - first 5 should be immediate,
	// remaining 5 will be spaced out over time
	for i := 0; i < 10; i++ {
		if limiter.Allow() {
			fmt.Printf("Operation %d completed at %s\n", i+1, time.Now().Format("15:04:05.000"))
		} else {
			fmt.Printf("Operation %d failed - rate limiter closed\n", i+1)
		}
	}
}

// TestExampleAllowWithTimeout demonstrates usage of AllowWithTimeout() method.
// This test shows how to avoid indefinite blocking when tokens might not
// be available within a reasonable time frame.
func TestExampleAllowWithTimeout(t *testing.T) {
	// Create a rate limiter with a low rate to demonstrate timeout behavior
	limiter := NewRateLimiter(2) // 2 operations per second
	defer limiter.Close()

	fmt.Println("Testing timeout behavior:")

	// First attempt should succeed immediately (using burst capacity)
	if limiter.AllowWithTimeout(100 * time.Millisecond) {
		fmt.Println("✓ First token acquired immediately")
	} else {
		fmt.Println("✗ Unexpected timeout on first token")
	}

	// Second attempt should also succeed immediately
	if limiter.AllowWithTimeout(100 * time.Millisecond) {
		fmt.Println("✓ Second token acquired immediately")
	} else {
		fmt.Println("✗ Unexpected timeout on second token")
	}

	// Third attempt should timeout since bucket is empty and 100ms < 500ms refill time
	if limiter.AllowWithTimeout(100 * time.Millisecond) {
		fmt.Println("✗ Unexpected success - should have timed out")
	} else {
		fmt.Println("✓ Third token request timed out as expected")
	}
}

// TestNewRateLimiter verifies that NewRateLimiter correctly initializes
// rate limiters with valid and invalid parameters.
func TestNewRateLimiter(t *testing.T) {
	t.Run("ValidRate", func(t *testing.T) {
		// Test creating a rate limiter with a valid rate
		rl := NewRateLimiter(5)
		defer rl.Close()

		// Verify the capacity matches the requested rate
		if rl.Capacity() != 5 {
			t.Errorf("Expected capacity 5, got %d", rl.Capacity())
		}

		// Verify initial tokens are available (burst capacity)
		if rl.Available() != 5 {
			t.Errorf("Expected 5 initial tokens, got %d", rl.Available())
		}
	})

	t.Run("InvalidRate", func(t *testing.T) {
		// Test that invalid rates (≤ 0) default to 1 operation per second
		testCases := []int{0, -1, -10}

		for _, rate := range testCases {
			t.Run(fmt.Sprintf("Rate_%d", rate), func(t *testing.T) {
				rl := NewRateLimiter(rate)
				defer rl.Close()

				if rl.Capacity() != 1 {
					t.Errorf("Expected capacity 1 for invalid rate %d, got %d", rate, rl.Capacity())
				}

				if rl.Available() != 1 {
					t.Errorf("Expected 1 initial token for invalid rate %d, got %d", rate, rl.Available())
				}
			})
		}
	})
}

// TestAllow verifies the basic functionality of the Allow() method,
// including immediate token consumption and token refill behavior.
func TestAllow(t *testing.T) {
	// Create a rate limiter with 2 operations per second
	rl := NewRateLimiter(2)
	defer rl.Close()

	t.Run("ImmediateTokenConsumption", func(t *testing.T) {
		// Should be able to consume all initial tokens immediately (burst capacity)
		for i := 0; i < 2; i++ {
			if !rl.Allow() {
				t.Errorf("Expected to acquire token %d immediately from burst capacity", i+1)
			}
		}

		// Verify no tokens are left
		if rl.Available() != 0 {
			t.Errorf("Expected 0 tokens after consuming burst capacity, got %d", rl.Available())
		}
	})

	t.Run("TokenRefill", func(t *testing.T) {
		// Wait for at least one token to be refilled
		// At 2 ops/sec, tokens are added every 500ms
		time.Sleep(600 * time.Millisecond)

		// Should be able to get at least one more token after refill
		if !rl.Allow() {
			t.Error("Expected to acquire token after refill period")
		}
	})
}

// TestAllowWithTimeout verifies the timeout functionality of AllowWithTimeout(),
// ensuring it properly handles both timeout and successful acquisition scenarios.
func TestAllowWithTimeout(t *testing.T) {
	// Create a rate limiter with 1 operation per second (slow rate for testing timeouts)
	rl := NewRateLimiter(1)
	defer rl.Close()

	t.Run("ImmediateSuccess", func(t *testing.T) {
		// First call should succeed immediately using burst capacity
		if !rl.AllowWithTimeout(100 * time.Millisecond) {
			t.Error("Expected immediate success with burst capacity token")
		}
	})

	t.Run("TimeoutBehavior", func(t *testing.T) {
		// Now the bucket is empty, so this should timeout
		// At 1 ops/sec, next token won't be available for 1000ms
		start := time.Now()
		if rl.AllowWithTimeout(100 * time.Millisecond) {
			t.Error("Expected timeout after 100ms, but got token")
		}
		elapsed := time.Since(start)

		// Verify the timeout was respected (should be ~100ms, allow some variance)
		if elapsed < 90*time.Millisecond || elapsed > 200*time.Millisecond {
			t.Errorf("Expected timeout around 100ms, got %v", elapsed)
		}
	})

	t.Run("SuccessWithLongerTimeout", func(t *testing.T) {
		// This should succeed because we wait long enough for token refill
		// At 1 ops/sec, token should be available within 1200ms
		if !rl.AllowWithTimeout(1200 * time.Millisecond) {
			t.Error("Expected to acquire token with sufficient timeout")
		}
	})
}

// TestRateLimiting verifies that the rate limiter actually enforces the
// specified rate by measuring the time taken for multiple operations.
func TestRateLimiting(t *testing.T) {
	// Create a rate limiter with 2 operations per second
	rl := NewRateLimiter(2)
	defer rl.Close()

	t.Run("RateEnforcement", func(t *testing.T) {
		start := time.Now()
		operations := 0

		// Perform 5 operations:
		// - Operations 1-2: immediate (burst capacity)
		// - Operation 3: available after 500ms (first refill)
		// - Operation 4: available after 1000ms (second refill)
		// - Operation 5: available after 1500ms (third refill)
		for i := 0; i < 5; i++ {
			if rl.Allow() {
				operations++
				t.Logf("Operation %d completed at %v", i+1, time.Since(start))
			} else {
				t.Errorf("Operation %d failed - rate limiter closed unexpectedly", i+1)
			}
		}

		elapsed := time.Since(start)

		// Verify all operations completed
		if operations != 5 {
			t.Errorf("Expected 5 operations, got %d", operations)
		}

		// Should take at least 1.5 seconds for 5 operations at 2 ops/sec:
		// - 2 ops immediate (0ms)
		// - 1 op at 500ms
		// - 1 op at 1000ms
		// - 1 op at 1500ms
		expectedMinDuration := 1400 * time.Millisecond // Allow 100ms tolerance
		if elapsed < expectedMinDuration {
			t.Errorf("Expected at least %v for 5 operations at 2 ops/sec, got %v", expectedMinDuration, elapsed)
		}

		// Should not take excessively long (max 2 seconds with reasonable tolerance)
		expectedMaxDuration := 2000 * time.Millisecond
		if elapsed > expectedMaxDuration {
			t.Errorf("Operations took too long: %v (expected max %v)", elapsed, expectedMaxDuration)
		}
	})
}

// TestAvailable verifies that the Available() method correctly reports
// the current number of tokens in the bucket as tokens are consumed and refilled.
func TestAvailable(t *testing.T) {
	// Create a rate limiter with 3 operations per second (3 token capacity)
	rl := NewRateLimiter(3)
	defer rl.Close()

	t.Run("InitialTokens", func(t *testing.T) {
		// Initially should have full burst capacity available
		if rl.Available() != 3 {
			t.Errorf("Expected 3 available tokens initially, got %d", rl.Available())
		}

		// Verify capacity is also correct
		if rl.Capacity() != 3 {
			t.Errorf("Expected capacity of 3, got %d", rl.Capacity())
		}
	})

	t.Run("TokenConsumption", func(t *testing.T) {
		// Consume one token and verify count decreases
		if !rl.Allow() {
			t.Fatal("Failed to consume first token")
		}
		if rl.Available() != 2 {
			t.Errorf("Expected 2 available tokens after consuming one, got %d", rl.Available())
		}

		// Consume remaining tokens
		if !rl.Allow() {
			t.Fatal("Failed to consume second token")
		}
		if rl.Available() != 1 {
			t.Errorf("Expected 1 available token after consuming two, got %d", rl.Available())
		}

		if !rl.Allow() {
			t.Fatal("Failed to consume third token")
		}
		if rl.Available() != 0 {
			t.Errorf("Expected 0 available tokens after consuming all, got %d", rl.Available())
		}
	})

	t.Run("TokenRefill", func(t *testing.T) {
		// Wait for token refill (at 3 ops/sec, tokens are added every ~333ms)
		time.Sleep(400 * time.Millisecond)

		// Should have at least one token available now
		available := rl.Available()
		if available < 1 {
			t.Errorf("Expected at least 1 token after refill period, got %d", available)
		}

		// Should not exceed capacity
		if available > 3 {
			t.Errorf("Available tokens (%d) should not exceed capacity (3)", available)
		}
	})
}

// TestClose verifies that Close() properly shuts down the rate limiter
// and prevents further token acquisition.
func TestClose(t *testing.T) {
	t.Run("NormalOperation", func(t *testing.T) {
		rl := NewRateLimiter(5)
		defer rl.Close()

		// Should work normally before closing
		if !rl.Allow() {
			t.Error("Expected token acquisition before close")
		}
	})

	t.Run("AfterClose", func(t *testing.T) {
		rl := NewRateLimiter(2)

		// Consume all available tokens first
		rl.Allow() // consume token 1
		rl.Allow() // consume token 2

		// Now close the rate limiter
		rl.Close()

		// Allow() should return false after close (no tokens left and refill stopped)
		if rl.Allow() {
			t.Error("Expected Allow() to return false after Close() with no tokens available")
		}

		// AllowWithTimeout() should also return false
		if rl.AllowWithTimeout(100 * time.Millisecond) {
			t.Error("Expected AllowWithTimeout() to return false after Close() with no tokens available")
		}
	})

	t.Run("CloseStopsOperationsEventually", func(t *testing.T) {
		rl := NewRateLimiter(3)

		// Consume only one token, leaving 2 in the bucket
		if !rl.Allow() {
			t.Fatal("Failed to consume initial token")
		}

		// Verify there are still tokens available before closing
		if rl.Available() != 2 {
			t.Errorf("Expected 2 tokens remaining, got %d", rl.Available())
		}

		// Close the rate limiter
		rl.Close()

		// After close, Allow() should eventually return false.
		// Due to the non-deterministic nature of Go's select statement,
		// it might consume remaining tokens first, but eventually it
		// should return false when the context is cancelled.
		tokensConsumed := 0
		for i := 0; i < 10; i++ { // Try up to 10 times
			if !rl.Allow() {
				// Good, Allow() returned false as expected
				break
			}
			tokensConsumed++
			if tokensConsumed > 2 {
				t.Fatal("Consumed more tokens than should be available after close")
			}
		}

		// At this point, Allow() should definitely return false
		if rl.Allow() {
			t.Error("Expected Allow() to return false after multiple attempts post-Close()")
		}

		// AllowWithTimeout should also return false
		if rl.AllowWithTimeout(10 * time.Millisecond) {
			t.Error("Expected AllowWithTimeout() to return false after Close()")
		}

		// Available() should still work
		if rl.Available() < 0 || rl.Available() > 2 {
			t.Errorf("Available() should still work after close, got %d", rl.Available())
		}
	})

	t.Run("MultipleClose", func(t *testing.T) {
		rl := NewRateLimiter(1)

		// Multiple calls to Close() should be safe
		rl.Close()
		rl.Close()
		rl.Close()
		// If we get here without panic, the test passes
	})
}
