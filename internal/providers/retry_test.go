package providers

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/swarm-ai/swarm/pkg/api"
)

func TestRetryableProvider_SuccessOnFirstAttempt(t *testing.T) {
	mockProvider := &FailingProvider{
		failCount: 0,
		responses: []*api.ChatResponse{
			{ID: "success-1"},
		},
	}

	retryable := &RetryableProvider{
		provider: mockProvider,
		config:   DefaultRetryConfig,
	}

	resp, err := retryable.Chat(context.Background(), &api.ChatRequest{})
	assert.NoError(t, err)
	assert.Equal(t, "success-1", resp.ID)
	assert.Equal(t, 1, mockProvider.callCount)
}

func TestRetryableProvider_RetryOnFailure(t *testing.T) {
	mockProvider := &FailingProvider{
		failCount: 2,
		responses: []*api.ChatResponse{
			{ID: "success-after-retry"},
		},
	}

	retryable := &RetryableProvider{
		provider: mockProvider,
		config: RetryConfig{
			MaxAttempts:     3,
			InitialDelay:    10 * time.Millisecond,
			MaxDelay:        100 * time.Millisecond,
			BackoffFactor:   2.0,
			RetryableErrors: []ErrorType{ErrorTypeServer},
		},
	}

	resp, err := retryable.Chat(context.Background(), &api.ChatRequest{})
	assert.NoError(t, err)
	assert.Equal(t, "success-after-retry", resp.ID)
	assert.Equal(t, 3, mockProvider.callCount)
}

func TestRetryableProvider_MaxRetriesExceeded(t *testing.T) {
	mockProvider := &FailingProvider{
		failCount: 10,
		responses: []*api.ChatResponse{},
	}

	retryable := &RetryableProvider{
		provider: mockProvider,
		config: RetryConfig{
			MaxAttempts:     3,
			InitialDelay:    10 * time.Millisecond,
			MaxDelay:        100 * time.Millisecond,
			BackoffFactor:   2.0,
			RetryableErrors: []ErrorType{ErrorTypeServer},
		},
	}

	_, err := retryable.Chat(context.Background(), &api.ChatRequest{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max retries exceeded")
	assert.Equal(t, 3, mockProvider.callCount)
}

func TestRetryableProvider_NonRetryableError(t *testing.T) {
	mockProvider := &FailingProvider{
		failCount: 1,
		responses: []*api.ChatResponse{},
		errorType: ErrorTypeAuth,
	}

	retryable := &RetryableProvider{
		provider: mockProvider,
		config: RetryConfig{
			MaxAttempts:     3,
			InitialDelay:    10 * time.Millisecond,
			MaxDelay:        100 * time.Millisecond,
			BackoffFactor:   2.0,
			RetryableErrors: []ErrorType{ErrorTypeServer},
		},
	}

	_, err := retryable.Chat(context.Background(), &api.ChatRequest{})
	assert.Error(t, err)
	assert.Equal(t, 1, mockProvider.callCount)
}

func TestRetryableProvider_ContextCancellation(t *testing.T) {
	mockProvider := &FailingProvider{
		failCount: 10,
		responses: []*api.ChatResponse{},
	}

	retryable := &RetryableProvider{
		provider: mockProvider,
		config: RetryConfig{
			MaxAttempts:     10,
			InitialDelay:    100 * time.Millisecond,
			MaxDelay:        100 * time.Millisecond,
			BackoffFactor:   1.0,
			RetryableErrors: []ErrorType{ErrorTypeServer},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	_, err := retryable.Chat(ctx, &api.ChatRequest{})
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestRetryableProvider_Name(t *testing.T) {
	mockProvider := &FailingProvider{
		name: "test-provider",
	}

	retryable := &RetryableProvider{
		provider: mockProvider,
		config:   DefaultRetryConfig,
	}

	assert.Equal(t, "test-provider", retryable.Name())
}

func TestRetryableProvider_SupportsStreaming(t *testing.T) {
	mockProvider := &FailingProvider{
		supportsStreaming: true,
	}

	retryable := &RetryableProvider{
		provider: mockProvider,
		config:   DefaultRetryConfig,
	}

	assert.True(t, retryable.SupportsStreaming())
}

type FailingProvider struct {
	callCount         int
	name              string
	failCount         int
	responses         []*api.ChatResponse
	supportsStreaming bool
	errorType         ErrorType
}

func (f *FailingProvider) Name() string {
	if f.name != "" {
		return f.name
	}
	return "failing-provider"
}

func (f *FailingProvider) Models() []api.ModelInfo {
	return []api.ModelInfo{}
}

func (f *FailingProvider) Chat(ctx context.Context, req *api.ChatRequest) (*api.ChatResponse, error) {
	f.callCount++

	if f.callCount <= f.failCount {
		errType := f.errorType
		if errType == "" {
			errType = ErrorTypeServer
		}
		return nil, NewProviderError("test_error", "test failure", errType, errors.New("test error"))
	}

	if len(f.responses) > 0 {
		return f.responses[len(f.responses)-1], nil
	}

	return &api.ChatResponse{ID: "success"}, nil
}

func (f *FailingProvider) StreamChat(ctx context.Context, req *api.ChatRequest) (<-chan api.ChatStreamEvent, error) {
	ch := make(chan api.ChatStreamEvent)
	close(ch)
	return ch, nil
}

func (f *FailingProvider) SupportsStreaming() bool {
	return f.supportsStreaming
}

func (f *FailingProvider) SupportsFunctionCalling() bool {
	return false
}

func (f *FailingProvider) SupportsVision() bool {
	return false
}

func (f *FailingProvider) SupportsAudio() bool {
	return false
}

func (f *FailingProvider) MaxTokens(model string) int {
	return 4096
}

func (f *FailingProvider) Configure(config *api.ProviderConfig) error {
	return nil
}

func (f *FailingProvider) Metrics() *api.ProviderMetrics {
	return &api.ProviderMetrics{}
}
