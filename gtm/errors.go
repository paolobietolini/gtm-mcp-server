package gtm

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"google.golang.org/api/googleapi"
)

var (
	ErrNotFound       = errors.New("resource not found")
	ErrConflict       = errors.New("resource conflict - fingerprint mismatch")
	ErrRateLimit      = errors.New("rate limit exceeded")
	ErrPermission     = errors.New("insufficient permissions")
	ErrInvalidRequest = errors.New("invalid request")
)

// retryWithBackoff executes fn with exponential backoff for rate limits.
// Returns the result or final error after maxRetries attempts.
func retryWithBackoff[T any](ctx context.Context, maxRetries int, fn func() (T, error)) (T, error) {
	var zero T
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Check context before executing
		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		default:
		}

		result, err := fn()
		if err == nil {
			return result, nil
		}

		// Check if it's a rate limit error
		var apiErr *googleapi.Error
		if errors.As(err, &apiErr) && isRateLimit(apiErr) {
			if attempt < maxRetries {
				waitTime := time.Duration(1<<uint(attempt)) * time.Second
				if waitTime > 32*time.Second {
					waitTime = 32 * time.Second
				}

				select {
				case <-time.After(waitTime):
					lastErr = err
					continue
				case <-ctx.Done():
					return zero, ctx.Err()
				}
			}
		}

		return zero, err
	}

	return zero, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// isRateLimit reports whether the API rejected a request for exceeding a quota.
// Google uses 403 as well as 429 for this, so a 403 needs a closer look: an
// ordinary permission error must not be retried or reported as a rate limit.
func isRateLimit(apiErr *googleapi.Error) bool {
	if apiErr.Code == 429 {
		return true
	}
	if apiErr.Code != 403 {
		return false
	}
	for _, item := range apiErr.Errors {
		switch item.Reason {
		case "rateLimitExceeded", "userRateLimitExceeded", "quotaExceeded":
			return true
		}
	}
	message := strings.ToLower(apiErr.Message)
	return strings.Contains(message, "rate limit") || strings.Contains(message, "quota exceeded")
}

// formatAPIErrorDetail extracts all available detail from a Google API error.
func formatAPIErrorDetail(apiErr *googleapi.Error) string {
	detail := apiErr.Message
	if len(apiErr.Errors) > 0 {
		for _, e := range apiErr.Errors {
			detail += fmt.Sprintf("\n  reason=%s: %s", e.Reason, e.Message)
		}
	}
	// Only fall back to the raw body when Google returned no structured
	// errors; otherwise it repeats the same message and wastes model tokens.
	if len(apiErr.Errors) == 0 && apiErr.Body != "" {
		detail += fmt.Sprintf("\n  body: %s", apiErr.Body)
	}
	return detail
}

// mapGoogleError converts Google API errors to our error types.
func mapGoogleError(err error) error {
	if err == nil {
		return nil
	}

	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) {
		detail := formatAPIErrorDetail(apiErr)
		if isRateLimit(apiErr) {
			return fmt.Errorf("%w: %s", ErrRateLimit, detail)
		}
		switch apiErr.Code {
		case 404:
			return fmt.Errorf("%w: %s", ErrNotFound, detail)
		case 409:
			return fmt.Errorf("%w: %s", ErrConflict, detail)
		case 403:
			return fmt.Errorf("%w: %s", ErrPermission, detail)
		case 400:
			return fmt.Errorf("%w: %s", ErrInvalidRequest, detail)
		default:
			return fmt.Errorf("API error %d: %s: %w", apiErr.Code, detail, apiErr)
		}
	}

	return err
}
