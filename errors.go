package vidgo

import (
	"errors"
	"fmt"
)

// Common errors
var (
	ErrUnsupportedProvider  = errors.New("unsupported provider")
	ErrInvalidConfiguration = errors.New("invalid configuration")
	ErrInvalidRequest       = errors.New("invalid request")
	ErrTaskNotFound         = errors.New("task not found")
	ErrProviderAPIError     = errors.New("provider API error")
	ErrNetworkError         = errors.New("network error")
	ErrAuthenticationFailed = errors.New("authentication failed")
	ErrRateLimitExceeded    = errors.New("rate limit exceeded")
	ErrInsufficientQuota    = errors.New("insufficient quota")
)

// APIError represents an error returned by the video generation API
type APIError struct {
	Code     int    `json:"code"`
	Message  string `json:"message"`
	Provider string `json:"provider,omitempty"`
}

func (e *APIError) Error() string {
	if e.Provider != "" {
		return fmt.Sprintf("[%s] API error %d: %s", e.Provider, e.Code, e.Message)
	}
	return fmt.Sprintf("API error %d: %s", e.Code, e.Message)
}

// ValidationError represents a request validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error for field '%s': %s", e.Field, e.Message)
}

// IsRetryableError determines if an error is retryable
func IsRetryableError(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		// Retry on server errors (5xx) and rate limiting (429)
		return apiErr.Code >= 500 || apiErr.Code == 429
	}

	// Retry on network errors
	return errors.Is(err, ErrNetworkError) || errors.Is(err, ErrRateLimitExceeded)
}
