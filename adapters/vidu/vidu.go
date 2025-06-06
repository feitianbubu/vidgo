package vidu

import (
	"context"
	"fmt"

	"github.com/feitianbubu/vidgo/adapters"
)

// Provider implements the adapters.Provider interface for Vidu video generation
type Provider struct {
	config *adapters.ProviderConfig
}

// New creates a new Vidu provider instance
func New(config *adapters.ProviderConfig) (adapters.Provider, error) {
	if config == nil {
		return nil, fmt.Errorf("invalid configuration")
	}

	return &Provider{
		config: config,
	}, nil
}

// Name returns the provider name
func (p *Provider) Name() string {
	return "Vidu"
}

// SupportedModels returns supported models
func (p *Provider) SupportedModels() []string {
	return []string{"vidu-v1", "vidu-v2"}
}

// ValidateRequest validates the request for Vidu
func (p *Provider) ValidateRequest(req *adapters.GenerationRequest) error {
	// TODO: Implement Vidu-specific validation
	return nil
}

// CreateGeneration creates a video generation task
func (p *Provider) CreateGeneration(ctx context.Context, req *adapters.GenerationRequest) (*adapters.GenerationResponse, error) {
	// TODO: Implement Vidu API integration
	return nil, fmt.Errorf("Vidu provider not yet implemented")
}

// GetGeneration retrieves the task status
func (p *Provider) GetGeneration(ctx context.Context, taskID string) (*adapters.TaskResult, error) {
	// TODO: Implement Vidu API integration
	return nil, fmt.Errorf("Vidu provider not yet implemented")
}
