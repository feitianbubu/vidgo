package vidgo

import (
	"context"
	"fmt"
	"time"

	"github.com/feitianbubu/vidgo/adapters"
	"github.com/feitianbubu/vidgo/adapters/kling"
)

// Client is the main client for video generation
type Client struct {
	provider Provider
	config   *ClientConfig
}

// ClientConfig holds configuration for the client
type ClientConfig struct {
	Timeout    time.Duration
	MaxRetries int
	RetryDelay time.Duration
	Debug      bool
}

// DefaultClientConfig returns default client configuration
func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		RetryDelay: time.Second,
		Debug:      false,
	}
}

// NewClient creates a new video generation client
func NewClient(providerType ProviderType, providerConfig *ProviderConfig, clientConfig ...*ClientConfig) (*Client, error) {
	provider, err := createProvider(providerType, providerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	config := DefaultClientConfig()
	if len(clientConfig) > 0 && clientConfig[0] != nil {
		config = clientConfig[0]
	}

	return &Client{
		provider: provider,
		config:   config,
	}, nil
}

// NewClientWithProvider creates a new client with a custom provider
func NewClientWithProvider(provider Provider, config ...*ClientConfig) *Client {
	clientConfig := DefaultClientConfig()
	if len(config) > 0 && config[0] != nil {
		clientConfig = config[0]
	}

	return &Client{
		provider: provider,
		config:   clientConfig,
	}
}

// CreateGeneration creates a new video generation task
func (c *Client) CreateGeneration(ctx context.Context, req *GenerationRequest) (*GenerationResponse, error) {
	if err := c.validateRequest(req); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	var lastErr error
	for i := 0; i <= c.config.MaxRetries; i++ {
		if i > 0 {
			select {
			case <-time.After(c.config.RetryDelay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		resp, err := c.provider.CreateGeneration(ctx, req)
		if err == nil {
			return resp, nil
		}

		lastErr = err
		if !IsRetryableError(err) {
			break
		}

		if c.config.Debug {
			fmt.Printf("Attempt %d failed: %v, retrying...\n", i+1, err)
		}
	}

	return nil, lastErr
}

// GetGeneration retrieves the status and result of a generation task
func (c *Client) GetGeneration(ctx context.Context, taskID string) (*TaskResult, error) {
	if taskID == "" {
		return nil, &ValidationError{Field: "task_id", Message: "task ID cannot be empty"}
	}

	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	var lastErr error
	for i := 0; i <= c.config.MaxRetries; i++ {
		if i > 0 {
			select {
			case <-time.After(c.config.RetryDelay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		result, err := c.provider.GetGeneration(ctx, taskID)
		if err == nil {
			return result, nil
		}

		lastErr = err
		if !IsRetryableError(err) {
			break
		}

		if c.config.Debug {
			fmt.Printf("Attempt %d failed: %v, retrying...\n", i+1, err)
		}
	}

	return nil, lastErr
}

// WaitForCompletion waits for a generation task to complete
func (c *Client) WaitForCompletion(ctx context.Context, taskID string, pollInterval time.Duration) (*TaskResult, error) {
	if pollInterval <= 0 {
		pollInterval = 5 * time.Second
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			result, err := c.GetGeneration(ctx, taskID)
			if err != nil {
				return nil, err
			}

			switch result.Status {
			case TaskStatusSucceeded, TaskStatusFailed:
				return result, nil
			case TaskStatusQueued, TaskStatusProcessing:
				continue
			default:
				return result, nil
			}
		}
	}
}

// GetProviderName returns the name of the current provider
func (c *Client) GetProviderName() string {
	return c.provider.Name()
}

// GetSupportedModels returns supported models for the current provider
func (c *Client) GetSupportedModels() []string {
	return c.provider.SupportedModels()
}

// createProvider creates a provider instance based on the provider type
func createProvider(providerType ProviderType, config *ProviderConfig) (Provider, error) {

	adapterConfig := &adapters.ProviderConfig{
		BaseURL:    config.BaseURL,
		APIKey:     config.APIKey,
		SecretKey:  config.SecretKey,
		Timeout:    config.Timeout,
		RetryCount: config.RetryCount,
		Extra:      config.Extra,
	}

	switch providerType {
	case ProviderKling:
		adapterProvider, err := kling.New(adapterConfig)
		if err != nil {
			return nil, err
		}
		return &adapterWrapper{provider: adapterProvider}, nil
	default:
		return nil, ErrUnsupportedProvider
	}
}

// validateRequest validates the generation request
func (c *Client) validateRequest(req *GenerationRequest) error {
	if req == nil {
		return &ValidationError{Field: "request", Message: "request cannot be nil"}
	}

	if req.Prompt == "" && req.Image == "" {
		return &ValidationError{Field: "prompt/image", Message: "at least one of prompt or image must be provided"}
	}

	if req.Duration <= 0 {
		return &ValidationError{Field: "duration", Message: "duration must be positive"}
	}

	if req.Width <= 0 {
		return &ValidationError{Field: "width", Message: "width must be positive"}
	}

	if req.Height <= 0 {
		return &ValidationError{Field: "height", Message: "height must be positive"}
	}
	return c.provider.ValidateRequest(req)
}
