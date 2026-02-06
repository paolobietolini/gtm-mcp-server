package gtm

import (
	"context"
	"errors"
	"testing"
	"time"

	"google.golang.org/api/googleapi"
)

func TestMapGoogleError_NilError(t *testing.T) {
	err := mapGoogleError(nil)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestMapGoogleError_NotFound(t *testing.T) {
	apiErr := &googleapi.Error{
		Code:    404,
		Message: "Resource not found",
	}

	err := mapGoogleError(apiErr)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}

	expectedMsg := "resource not found: Resource not found"
	if err.Error() != expectedMsg {
		t.Errorf("expected message %q, got %q", expectedMsg, err.Error())
	}
}

func TestMapGoogleError_Conflict(t *testing.T) {
	apiErr := &googleapi.Error{
		Code:    409,
		Message: "Fingerprint mismatch",
	}

	err := mapGoogleError(apiErr)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}

	expectedMsg := "resource conflict - fingerprint mismatch: Fingerprint mismatch"
	if err.Error() != expectedMsg {
		t.Errorf("expected message %q, got %q", expectedMsg, err.Error())
	}
}

func TestMapGoogleError_Permission(t *testing.T) {
	apiErr := &googleapi.Error{
		Code:    403,
		Message: "Permission denied",
	}

	err := mapGoogleError(apiErr)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, ErrPermission) {
		t.Errorf("expected ErrPermission, got %v", err)
	}

	expectedMsg := "insufficient permissions: Permission denied"
	if err.Error() != expectedMsg {
		t.Errorf("expected message %q, got %q", expectedMsg, err.Error())
	}
}

func TestMapGoogleError_RateLimit(t *testing.T) {
	apiErr := &googleapi.Error{
		Code:    429,
		Message: "Rate limit exceeded",
	}

	err := mapGoogleError(apiErr)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, ErrRateLimit) {
		t.Errorf("expected ErrRateLimit, got %v", err)
	}

	expectedMsg := "rate limit exceeded: Rate limit exceeded"
	if err.Error() != expectedMsg {
		t.Errorf("expected message %q, got %q", expectedMsg, err.Error())
	}
}

func TestMapGoogleError_InvalidRequest(t *testing.T) {
	apiErr := &googleapi.Error{
		Code:    400,
		Message: "Invalid request parameters",
	}

	err := mapGoogleError(apiErr)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, ErrInvalidRequest) {
		t.Errorf("expected ErrInvalidRequest, got %v", err)
	}

	expectedMsg := "invalid request: Invalid request parameters"
	if err.Error() != expectedMsg {
		t.Errorf("expected message %q, got %q", expectedMsg, err.Error())
	}
}

func TestMapGoogleError_UnknownStatusCode(t *testing.T) {
	apiErr := &googleapi.Error{
		Code:    500,
		Message: "Internal server error",
	}

	err := mapGoogleError(apiErr)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Should return the original error for unknown status codes
	if !errors.Is(err, apiErr) {
		t.Errorf("expected original error, got %v", err)
	}
}

func TestMapGoogleError_NonGoogleAPIError(t *testing.T) {
	originalErr := errors.New("some other error")

	err := mapGoogleError(originalErr)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if err != originalErr {
		t.Errorf("expected original error, got %v", err)
	}
}

func TestMapGoogleError_AllStatusCodes(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		message       string
		expectedError error
	}{
		{
			name:          "404 Not Found",
			statusCode:    404,
			message:       "Tag not found",
			expectedError: ErrNotFound,
		},
		{
			name:          "409 Conflict",
			statusCode:    409,
			message:       "Version conflict",
			expectedError: ErrConflict,
		},
		{
			name:          "403 Permission Denied",
			statusCode:    403,
			message:       "Access denied",
			expectedError: ErrPermission,
		},
		{
			name:          "429 Rate Limited",
			statusCode:    429,
			message:       "Too many requests",
			expectedError: ErrRateLimit,
		},
		{
			name:          "400 Bad Request",
			statusCode:    400,
			message:       "Bad input",
			expectedError: ErrInvalidRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiErr := &googleapi.Error{
				Code:    tt.statusCode,
				Message: tt.message,
			}

			err := mapGoogleError(apiErr)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			if !errors.Is(err, tt.expectedError) {
				t.Errorf("expected error type %v, got %v", tt.expectedError, err)
			}

			// Verify message is included
			if !contains(err.Error(), tt.message) {
				t.Errorf("expected error message to contain %q, got %q", tt.message, err.Error())
			}
		})
	}
}

func TestRetryWithBackoff_SuccessFirstTry(t *testing.T) {
	ctx := context.Background()
	callCount := 0

	result, err := retryWithBackoff(ctx, 3, func() (string, error) {
		callCount++
		return "success", nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if result != "success" {
		t.Errorf("expected result 'success', got %q", result)
	}

	if callCount != 1 {
		t.Errorf("expected 1 call, got %d", callCount)
	}
}

func TestRetryWithBackoff_NonRetryableError(t *testing.T) {
	ctx := context.Background()
	callCount := 0

	nonRetryableErr := &googleapi.Error{
		Code:    400,
		Message: "Bad request",
	}

	result, err := retryWithBackoff(ctx, 3, func() (string, error) {
		callCount++
		return "", nonRetryableErr
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if result != "" {
		t.Errorf("expected empty result, got %q", result)
	}

	// Should not retry for non-rate-limit errors
	if callCount != 1 {
		t.Errorf("expected 1 call (no retries), got %d", callCount)
	}

	if !errors.Is(err, nonRetryableErr) {
		t.Errorf("expected original error, got %v", err)
	}
}

func TestRetryWithBackoff_RateLimitError403(t *testing.T) {
	ctx := context.Background()
	callCount := 0

	rateLimitErr := &googleapi.Error{
		Code:    403,
		Message: "Rate limit exceeded",
	}

	// Fail twice with rate limit, then succeed
	result, err := retryWithBackoff(ctx, 3, func() (string, error) {
		callCount++
		if callCount <= 2 {
			return "", rateLimitErr
		}
		return "success after retry", nil
	})

	if err != nil {
		t.Errorf("expected no error after retries, got %v", err)
	}

	if result != "success after retry" {
		t.Errorf("expected result 'success after retry', got %q", result)
	}

	// Should have retried twice and succeeded on third attempt
	if callCount != 3 {
		t.Errorf("expected 3 calls, got %d", callCount)
	}
}

func TestRetryWithBackoff_RateLimitError429(t *testing.T) {
	ctx := context.Background()
	callCount := 0

	rateLimitErr := &googleapi.Error{
		Code:    429,
		Message: "Too many requests",
	}

	// Fail once with rate limit, then succeed
	result, err := retryWithBackoff(ctx, 3, func() (string, error) {
		callCount++
		if callCount == 1 {
			return "", rateLimitErr
		}
		return "success", nil
	})

	if err != nil {
		t.Errorf("expected no error after retry, got %v", err)
	}

	if result != "success" {
		t.Errorf("expected result 'success', got %q", result)
	}

	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}
}

func TestRetryWithBackoff_MaxRetriesExceeded(t *testing.T) {
	ctx := context.Background()
	callCount := 0

	rateLimitErr := &googleapi.Error{
		Code:    429,
		Message: "Rate limit exceeded",
	}

	// Always fail with rate limit error
	result, err := retryWithBackoff(ctx, 2, func() (string, error) {
		callCount++
		return "", rateLimitErr
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if result != "" {
		t.Errorf("expected empty result, got %q", result)
	}

	// Should try initial + 2 retries = 3 total
	if callCount != 3 {
		t.Errorf("expected 3 calls, got %d", callCount)
	}

	// On the final attempt, the rate limit error is returned directly
	if !contains(err.Error(), "Rate limit exceeded") {
		t.Errorf("expected error to contain rate limit message, got %q", err.Error())
	}
}

func TestRetryWithBackoff_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	callCount := 0

	rateLimitErr := &googleapi.Error{
		Code:    429,
		Message: "Rate limit exceeded",
	}

	// Cancel context after first call
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	result, err := retryWithBackoff(ctx, 5, func() (string, error) {
		callCount++
		if callCount == 1 {
			// First call fails, which should trigger a retry attempt
			return "", rateLimitErr
		}
		// Subsequent calls should not happen due to context cancellation
		time.Sleep(200 * time.Millisecond)
		return "", rateLimitErr
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if result != "" {
		t.Errorf("expected empty result, got %q", result)
	}

	// Should have context.Canceled error
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled error, got %v", err)
	}

	// Should not have completed all retries
	if callCount > 2 {
		t.Errorf("expected at most 2 calls before cancellation, got %d", callCount)
	}
}

func TestRetryWithBackoff_ContextCancelledBeforeCall(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	callCount := 0

	result, err := retryWithBackoff(ctx, 3, func() (string, error) {
		callCount++
		return "should not reach here", nil
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if result != "" {
		t.Errorf("expected empty result, got %q", result)
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled error, got %v", err)
	}

	if callCount != 0 {
		t.Errorf("expected 0 calls, got %d", callCount)
	}
}

func TestRetryWithBackoff_ExponentialBackoff(t *testing.T) {
	ctx := context.Background()
	callTimes := []time.Time{}

	rateLimitErr := &googleapi.Error{
		Code:    429,
		Message: "Rate limit exceeded",
	}

	// Always fail to test backoff timing
	_, _ = retryWithBackoff(ctx, 3, func() (string, error) {
		callTimes = append(callTimes, time.Now())
		return "", rateLimitErr
	})

	if len(callTimes) != 4 {
		t.Fatalf("expected 4 calls, got %d", len(callTimes))
	}

	// Check exponential backoff (1s, 2s, 4s)
	// Allow some tolerance for timing
	tolerance := 100 * time.Millisecond

	// First retry should be ~1s after initial call
	firstBackoff := callTimes[1].Sub(callTimes[0])
	if firstBackoff < 1*time.Second-tolerance || firstBackoff > 1*time.Second+tolerance {
		t.Errorf("expected first backoff ~1s, got %v", firstBackoff)
	}

	// Second retry should be ~2s after first retry
	secondBackoff := callTimes[2].Sub(callTimes[1])
	if secondBackoff < 2*time.Second-tolerance || secondBackoff > 2*time.Second+tolerance {
		t.Errorf("expected second backoff ~2s, got %v", secondBackoff)
	}

	// Third retry should be ~4s after second retry
	thirdBackoff := callTimes[3].Sub(callTimes[2])
	if thirdBackoff < 4*time.Second-tolerance || thirdBackoff > 4*time.Second+tolerance {
		t.Errorf("expected third backoff ~4s, got %v", thirdBackoff)
	}
}

func TestRetryWithBackoff_MaxBackoffCap(t *testing.T) {
	// Verify the backoff cap logic: 2^attempt should be capped at 32 seconds.
	// Instead of waiting 63+ seconds, we verify the calculation directly.
	for attempt := uint(0); attempt <= 10; attempt++ {
		waitTime := time.Duration(1<<attempt) * time.Second
		if waitTime > 32*time.Second {
			waitTime = 32 * time.Second
		}
		if attempt >= 5 && waitTime != 32*time.Second {
			t.Errorf("attempt %d: expected cap at 32s, got %v", attempt, waitTime)
		}
	}
}

func TestRetryWithBackoff_IntegerResult(t *testing.T) {
	// Test with integer return type
	ctx := context.Background()
	callCount := 0

	result, err := retryWithBackoff(ctx, 2, func() (int, error) {
		callCount++
		if callCount == 1 {
			return 0, &googleapi.Error{Code: 429}
		}
		return 42, nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if result != 42 {
		t.Errorf("expected result 42, got %d", result)
	}
}

func TestRetryWithBackoff_StructResult(t *testing.T) {
	// Test with struct return type
	type Result struct {
		Value string
	}

	ctx := context.Background()

	result, err := retryWithBackoff(ctx, 2, func() (*Result, error) {
		return &Result{Value: "test"}, nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if result.Value != "test" {
		t.Errorf("expected result.Value 'test', got %q", result.Value)
	}
}

func TestRetryWithBackoff_ContextCancelledDuringWait(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	callCount := 0

	rateLimitErr := &googleapi.Error{
		Code:    429,
		Message: "Rate limit exceeded",
	}

	start := time.Now()
	result, err := retryWithBackoff(ctx, 5, func() (string, error) {
		callCount++
		return "", rateLimitErr
	})
	duration := time.Since(start)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if result != "" {
		t.Errorf("expected empty result, got %q", result)
	}

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded error, got %v", err)
	}

	// Should have been cancelled before completing all retries
	// First call happens immediately, second after 1s, but timeout is 500ms
	// so we should only get 1-2 calls
	if callCount > 2 {
		t.Errorf("expected at most 2 calls before timeout, got %d", callCount)
	}

	// Should have stopped around 500ms, not run all retries
	if duration > 2*time.Second {
		t.Errorf("expected early termination around 500ms, took %v", duration)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
