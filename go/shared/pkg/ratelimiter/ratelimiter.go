// Package ratelimiter provides a token bucket rate limiter implementation
// using Go channels and goroutines for controlling the rate of operations.
//
// The rate limiter uses a token bucket algorithm where tokens are added
// to a buffered channel at a specified rate. Operations consume tokens
// from the channel, blocking when no tokens are available.
//
// Example usage:
//
//	limiter := ratelimiter.NewRateLimiter(10) // 10 operations per second
//	defer limiter.Close()
//
//	for i := 0; i < 100; i++ {
//		if limiter.Allow() {
//			// Perform rate-limited operation
//			doWork()
//		}
//	}
package ratelimiter

import (
	"context"
	"sync/atomic"
	"time"
)

// RateLimiter implements a token bucket rate limiter that controls the rate
// of operations using channels and goroutines. It allows a specified number
// of operations per second and provides both blocking and non-blocking methods
// for acquiring tokens.
//
// The rate limiter is thread-safe and can be used concurrently from multiple
// goroutines. It uses a buffered channel to store tokens and a background
// goroutine to refill tokens at the specified rate.
type RateLimiter struct {
	tokens chan struct{}      // Buffered channel holding available tokens
	ctx    context.Context    // Context for graceful shutdown
	cancel context.CancelFunc // Cancel function to stop the refill goroutine
	closed int32              // Atomic flag to track if the rate limiter is closed
}

// NewRateLimiter creates a new rate limiter that allows the specified number
// of operations per second using a token bucket algorithm.
//
// The opsPerSecond parameter determines both the rate at which tokens are
// refilled and the maximum burst capacity (bucket size). If opsPerSecond
// is less than or equal to 0, it defaults to 1 operation per second.
//
// The rate limiter starts with a full bucket of tokens, allowing immediate
// burst operations up to the specified limit. A background goroutine
// continuously refills tokens at the specified rate.
//
// The caller must call Close() when done to clean up resources and stop
// the background goroutine.
//
// Example:
//	limiter := NewRateLimiter(5) // 5 ops/sec with burst capacity of 5
//	defer limiter.Close()
func NewRateLimiter(opsPerSecond int) *RateLimiter {
	// Ensure we have at least 1 operation per second to prevent division by zero
	// and provide meaningful rate limiting behavior
	if opsPerSecond <= 0 {
		opsPerSecond = 1
	}

	ctx, cancel := context.WithCancel(context.Background())
	rl := &RateLimiter{
		tokens: make(chan struct{}, opsPerSecond), // Buffered channel with capacity = rate
		ctx:    ctx,
		cancel: cancel,
	}

	// Pre-fill the token bucket to allow immediate burst operations
	// up to the specified capacity
	for i := 0; i < opsPerSecond; i++ {
		rl.tokens <- struct{}{}
	}

	// Start the background goroutine that continuously refills tokens
	// at the specified rate
	go rl.refillTokens(opsPerSecond)

	return rl
}

// refillTokens is a background goroutine that continuously adds tokens to the
// token bucket at the specified rate. It runs until the rate limiter is closed.
//
// The function uses a ticker to add one token every (1 second / opsPerSecond).
// For example, with 10 ops/sec, it adds a token every 100ms.
//
// If the token bucket is full when a token is ready to be added, the token
// is discarded (dropped on the floor) to maintain the rate limit. This prevents
// token accumulation beyond the bucket capacity.
func (rl *RateLimiter) refillTokens(opsPerSecond int) {
	// Create a ticker that fires every (1 second / opsPerSecond)
	// This ensures tokens are added at the correct rate
	ticker := time.NewTicker(time.Second / time.Duration(opsPerSecond))
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Time to add a token
			select {
			case rl.tokens <- struct{}{}:
				// Token successfully added to the bucket
			case <-rl.ctx.Done():
				// Rate limiter is being shut down
				return
			default:
				// Token bucket is full, discard this token to maintain rate limit.
				// This is expected behavior and prevents unlimited token accumulation.
			}
		case <-rl.ctx.Done():
			// Rate limiter is being shut down, exit the goroutine
			return
		}
	}
}

// Allow blocks until a token is available from the rate limiter, then consumes
// the token and returns true. This method will block indefinitely until either
// a token becomes available or the rate limiter is closed.
//
// Returns true if a token was successfully acquired and consumed.
// Returns false only if the rate limiter has been closed via Close().
//
// This method is thread-safe and can be called concurrently from multiple
// goroutines.
//
// Example:
//	if limiter.Allow() {
//		// Perform rate-limited operation
//		doWork()
//	} else {
//		// Rate limiter was closed
//		log.Println("Rate limiter closed")
//	}
func (rl *RateLimiter) Allow() bool {
	// Check if closed first to ensure deterministic behavior
	if atomic.LoadInt32(&rl.closed) == 1 {
		return false
	}

	select {
	case <-rl.tokens:
		// Double-check if closed after acquiring token
		if atomic.LoadInt32(&rl.closed) == 1 {
			// Put the token back and return false
			select {
			case rl.tokens <- struct{}{}:
			default:
				// Channel is full, token is lost but that's okay
			}
			return false
		}
		// Successfully acquired a token
		return true
	case <-rl.ctx.Done():
		// Rate limiter has been closed
		return false
	}
}

// AllowWithTimeout blocks until a token is available or the specified timeout
// duration elapses. This provides a way to avoid indefinite blocking when
// tokens are not available.
//
// Returns true if a token was successfully acquired within the timeout period.
// Returns false if the timeout elapsed, the rate limiter was closed, or if
// the timeout duration is negative or zero.
//
// This method is thread-safe and can be called concurrently from multiple
// goroutines.
//
// Example:
//	if limiter.AllowWithTimeout(100 * time.Millisecond) {
//		// Got token within 100ms
//		doWork()
//	} else {
//		// Timeout or rate limiter closed
//		log.Println("Could not acquire token within timeout")
//	}
func (rl *RateLimiter) AllowWithTimeout(timeout time.Duration) bool {
	// Check if closed first to ensure deterministic behavior
	if atomic.LoadInt32(&rl.closed) == 1 {
		return false
	}

	select {
	case <-rl.tokens:
		// Double-check if closed after acquiring token
		if atomic.LoadInt32(&rl.closed) == 1 {
			// Put the token back and return false
			select {
			case rl.tokens <- struct{}{}:
			default:
				// Channel is full, token is lost but that's okay
			}
			return false
		}
		// Successfully acquired a token
		return true
	case <-time.After(timeout):
		// Timeout elapsed before token became available
		return false
	case <-rl.ctx.Done():
		// Rate limiter has been closed
		return false
	}
}

// Close gracefully shuts down the rate limiter and releases all associated
// resources. It stops the background token refill goroutine and prevents
// further token acquisition.
//
// After Close() is called:
// - The background refill goroutine will terminate
// - All pending and future calls to Allow() and AllowWithTimeout() will return false
// - Available() and Capacity() will continue to work until the RateLimiter is garbage collected
//
// Close() is safe to call multiple times and from multiple goroutines.
// It's recommended to defer Close() immediately after creating a rate limiter.
//
// Note: The token channel is intentionally not closed to prevent panics
// in concurrent goroutines that might still be accessing it.
//
// Example:
//	limiter := NewRateLimiter(10)
//	defer limiter.Close() // Always clean up resources
func (rl *RateLimiter) Close() {
	// Set the closed flag atomically first
	atomic.StoreInt32(&rl.closed, 1)

	// Cancel the context to signal the refill goroutine to stop
	// This is safe to call multiple times
	rl.cancel()
	// Note: We intentionally don't close the tokens channel here because:
	// 1. It might cause panics if other goroutines are still trying to read from it
	// 2. The channel will be garbage collected when the RateLimiter is collected
	// 3. The context cancellation is sufficient to stop the refill goroutine
}

// Available returns the current number of tokens available in the bucket.
// This represents how many operations can be performed immediately without
// blocking.
//
// The returned value is a snapshot at the time of the call and may change
// immediately due to concurrent token consumption or refill operations.
//
// This method is thread-safe and can be called concurrently.
//
// Example:
//	fmt.Printf("Tokens available: %d/%d\n", limiter.Available(), limiter.Capacity())
func (rl *RateLimiter) Available() int {
	return len(rl.tokens)
}

// Capacity returns the maximum number of tokens that the bucket can hold.
// This value is set when the rate limiter is created and represents both
// the maximum burst capacity and the rate (tokens per second).
//
// The capacity never changes during the lifetime of the rate limiter.
//
// This method is thread-safe and can be called concurrently.
//
// Example:
//	fmt.Printf("Bucket capacity: %d tokens\n", limiter.Capacity())
func (rl *RateLimiter) Capacity() int {
	return cap(rl.tokens)
}
