package providers

import (
	"fmt"
	"time"

	"github.com/swarm-ai/swarm/pkg/api"
)

// Normalizer converts between SWARM and provider formats
type Normalizer interface {
	NormalizeRequest(req *api.ChatRequest) (any, error)
	NormalizeResponse(resp any) (*api.ChatResponse, error)
	NormalizeStreamEvent(event any) (*api.ChatStreamEvent, error)
	NormalizeError(err error) *ProviderError
}

// ProviderError with actionable info
type ProviderError struct {
	Code       string
	Message    string
	Retryable  bool
	RetryAfter time.Duration
	Type       ErrorType
	Err        error
}

func (e *ProviderError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *ProviderError) Unwrap() error {
	return e.Err
}

type ErrorType string

const (
	ErrorTypeRateLimit    ErrorType = "rate_limit"
	ErrorTypeAuth         ErrorType = "authentication"
	ErrorTypeInvalid      ErrorType = "invalid_request"
	ErrorTypeNotFound     ErrorType = "not_found"
	ErrorTypeContextLimit ErrorType = "context_limit"
	ErrorTypeServer       ErrorType = "server_error"
)

// NewProviderError creates a new provider error
func NewProviderError(code, message string, errType ErrorType, err error) *ProviderError {
	return &ProviderError{
		Code:      code,
		Message:   message,
		Type:      errType,
		Retryable: errType == ErrorTypeRateLimit || errType == ErrorTypeServer,
		Err:       err,
	}
}
