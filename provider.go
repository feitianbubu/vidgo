package vidgo

import "context"

// Provider defines the interface that all video generation providers must implement
type Provider interface {
	// Name returns the provider name
	Name() string

	// CreateGeneration creates a new video generation task
	CreateGeneration(ctx context.Context, req *GenerationRequest) (*GenerationResponse, error)

	// GetGeneration retrieves the status and result of a generation task
	GetGeneration(ctx context.Context, taskID string) (*TaskResult, error)

	// SupportedModels returns a list of supported models for this provider
	SupportedModels() []string

	// ValidateRequest validates if the request is compatible with this provider
	ValidateRequest(req *GenerationRequest) error
}

// ProviderFactory creates provider instances
type ProviderFactory interface {
	CreateProvider(providerType ProviderType, config *ProviderConfig) (Provider, error)
}
