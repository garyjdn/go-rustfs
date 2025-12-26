package utils

import (
	"context"
	"math"
	"strings"
	"time"

	"github.com/garyjdn/go-rustfs/types"
)

// RetryableFunc represents a function that can be retried
type RetryableFunc func() error

// RetryableFuncWithContext represents a function that can be retried with context
type RetryableFuncWithContext func(ctx context.Context) error

// RetryResult represents the result of a retry operation
type RetryResult struct {
	Success    bool
	Attempts   int
	Duration   time.Duration
	LastError  error
	TotalDelay time.Duration
}

// RetryWithBackoff executes a function with exponential backoff retry
func RetryWithBackoff(fn RetryableFunc, config *types.RetryConfig) *RetryResult {
	return RetryWithBackoffWithContext(context.Background(), func(ctx context.Context) error {
		return fn()
	}, config)
}

// RetryWithBackoffWithContext executes a function with exponential backoff retry and context
func RetryWithBackoffWithContext(ctx context.Context, fn RetryableFuncWithContext, config *types.RetryConfig) *RetryResult {
	if config == nil {
		config = &types.RetryConfig{
			MaxAttempts: 3,
			Delay:       time.Second,
			Backoff:     2.0,
		}
	}

	startTime := time.Now()
	var lastError error
	totalDelay := time.Duration(0)

	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		// Check if context is cancelled
		if ctx.Err() != nil {
			return &RetryResult{
				Success:    false,
				Attempts:   attempt,
				Duration:   time.Since(startTime),
				LastError:  ctx.Err(),
				TotalDelay: totalDelay,
			}
		}

		// Execute the function
		err := fn(ctx)
		if err == nil {
			return &RetryResult{
				Success:    true,
				Attempts:   attempt + 1,
				Duration:   time.Since(startTime),
				LastError:  nil,
				TotalDelay: totalDelay,
			}
		}

		lastError = err

		// Don't wait on the last attempt
		if attempt < config.MaxAttempts-1 {
			// Calculate delay with exponential backoff
			delay := calculateDelay(attempt, config.Delay, config.Backoff)
			totalDelay += delay

			// Wait for the delay or context cancellation
			select {
			case <-time.After(delay):
				// Continue to next attempt
			case <-ctx.Done():
				return &RetryResult{
					Success:    false,
					Attempts:   attempt + 1,
					Duration:   time.Since(startTime),
					LastError:  ctx.Err(),
					TotalDelay: totalDelay,
				}
			}
		}
	}

	return &RetryResult{
		Success:    false,
		Attempts:   config.MaxAttempts,
		Duration:   time.Since(startTime),
		LastError:  lastError,
		TotalDelay: totalDelay,
	}
}

// RetryWithFixedDelay executes a function with fixed delay retry
func RetryWithFixedDelay(fn RetryableFunc, maxAttempts int, delay time.Duration) *RetryResult {
	config := &types.RetryConfig{
		MaxAttempts: maxAttempts,
		Delay:       delay,
		Backoff:     1.0, // No backoff
	}
	return RetryWithBackoff(fn, config)
}

// RetryWithFixedDelayWithContext executes a function with fixed delay retry and context
func RetryWithFixedDelayWithContext(ctx context.Context, fn RetryableFuncWithContext, maxAttempts int, delay time.Duration) *RetryResult {
	config := &types.RetryConfig{
		MaxAttempts: maxAttempts,
		Delay:       delay,
		Backoff:     1.0, // No backoff
	}
	return RetryWithBackoffWithContext(ctx, fn, config)
}

// IsRetryableError checks if an error should trigger a retry
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Common retryable error patterns
	retryablePatterns := []string{
		"connection refused",
		"connection reset",
		"timeout",
		"temporary failure",
		"service unavailable",
		"rate limit",
		"too many requests",
		"network unreachable",
		"connection timed out",
		"read timeout",
		"write timeout",
	}

	errStr := err.Error()
	for _, pattern := range retryablePatterns {
		if contains(errStr, pattern) {
			return true
		}
	}

	// Check for specific error types
	switch {
	case isNetworkError(err):
		return true
	case isTimeoutError(err):
		return true
	case isTemporaryError(err):
		return true
	default:
		return false
	}
}

// GetRetryDelay calculates delay for a specific attempt
func GetRetryDelay(attempt int, baseDelay time.Duration, backoff float64) time.Duration {
	return calculateDelay(attempt, baseDelay, backoff)
}

// calculateDelay calculates delay using exponential backoff with jitter
func calculateDelay(attempt int, baseDelay time.Duration, backoff float64) time.Duration {
	// Exponential backoff: delay = baseDelay * backoff^attempt
	delay := float64(baseDelay) * math.Pow(backoff, float64(attempt))

	// Add jitter to prevent thundering herd (±25%)
	jitter := delay * 0.25 * (2*randomFloat64() - 1)
	delay += jitter

	// Ensure minimum delay
	if delay < float64(baseDelay) {
		delay = float64(baseDelay)
	}

	return time.Duration(delay)
}

// isNetworkError checks if error is network-related
func isNetworkError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	networkPatterns := []string{
		"connection refused",
		"connection reset",
		"network unreachable",
		"no route to host",
		"host unreachable",
		"connection timed out",
		"read timeout",
		"write timeout",
	}

	for _, pattern := range networkPatterns {
		if contains(errStr, pattern) {
			return true
		}
	}

	return false
}

// isTimeoutError checks if error is timeout-related
func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	timeoutPatterns := []string{
		"timeout",
		"deadline exceeded",
		"context canceled",
		"context deadline exceeded",
	}

	for _, pattern := range timeoutPatterns {
		if contains(errStr, pattern) {
			return true
		}
	}

	return false
}

// isTemporaryError checks if error is temporary
func isTemporaryError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	temporaryPatterns := []string{
		"temporary",
		"transient",
		"retry later",
		"service unavailable",
		"rate limit",
		"too many requests",
	}

	for _, pattern := range temporaryPatterns {
		if contains(errStr, pattern) {
			return true
		}
	}

	return false
}

// contains checks if string contains substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			(s[:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				containsIgnoreCase(s, substr))))
}

// containsIgnoreCase checks if string contains substring (case-insensitive)
func containsIgnoreCase(s, substr string) bool {
	if len(s) < len(substr) {
		return false
	}

	sLower := strings.ToLower(s)
	substrLower := strings.ToLower(substr)

	return strings.Contains(sLower, substrLower)
}

// randomFloat64 generates a random float64 between 0 and 1
func randomFloat64() float64 {
	return float64(time.Now().UnixNano()%1000) / 1000.0
}

// RetryConfigBuilder helps build retry configurations
type RetryConfigBuilder struct {
	config *types.RetryConfig
}

// NewRetryConfigBuilder creates a new retry configuration builder
func NewRetryConfigBuilder() *RetryConfigBuilder {
	return &RetryConfigBuilder{
		config: &types.RetryConfig{
			MaxAttempts: 3,
			Delay:       time.Second,
			Backoff:     2.0,
		},
	}
}

// WithMaxAttempts sets the maximum number of attempts
func (b *RetryConfigBuilder) WithMaxAttempts(attempts int) *RetryConfigBuilder {
	b.config.MaxAttempts = attempts
	return b
}

// WithDelay sets the base delay between attempts
func (b *RetryConfigBuilder) WithDelay(delay time.Duration) *RetryConfigBuilder {
	b.config.Delay = delay
	return b
}

// WithBackoff sets the backoff multiplier
func (b *RetryConfigBuilder) WithBackoff(backoff float64) *RetryConfigBuilder {
	b.config.Backoff = backoff
	return b
}

// Build creates the retry configuration
func (b *RetryConfigBuilder) Build() *types.RetryConfig {
	return b.config
}

// DefaultRetryConfig returns a default retry configuration
func DefaultRetryConfig() *types.RetryConfig {
	return NewRetryConfigBuilder().Build()
}

// FastRetryConfig returns a retry configuration for fast operations
func FastRetryConfig() *types.RetryConfig {
	return NewRetryConfigBuilder().
		WithMaxAttempts(5).
		WithDelay(100 * time.Millisecond).
		WithBackoff(1.5).
		Build()
}

// SlowRetryConfig returns a retry configuration for slow operations
func SlowRetryConfig() *types.RetryConfig {
	return NewRetryConfigBuilder().
		WithMaxAttempts(3).
		WithDelay(5 * time.Second).
		WithBackoff(3.0).
		Build()
}
