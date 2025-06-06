package vidgo

import (
	"context"

	"github.com/feitianbubu/vidgo/adapters"
)

// adapterWrapper wraps an adapters.Provider to implement the main package Provider interface
type adapterWrapper struct {
	provider adapters.Provider
}

// Name returns the provider name
func (w *adapterWrapper) Name() string {
	return w.provider.Name()
}

// CreateGeneration creates a new video generation task
func (w *adapterWrapper) CreateGeneration(ctx context.Context, req *GenerationRequest) (*GenerationResponse, error) {
	adapterReq := &adapters.GenerationRequest{
		Prompt:         req.Prompt,
		Image:          req.Image,
		Style:          req.Style,
		Duration:       req.Duration,
		FPS:            req.FPS,
		Width:          req.Width,
		Height:         req.Height,
		ResponseFormat: adapters.ResponseFormat(req.ResponseFormat),
		QualityLevel:   adapters.QualityLevel(req.QualityLevel),
		Seed:           req.Seed,
		Model:          req.Model,
		Metadata:       req.Metadata,
	}

	resp, err := w.provider.CreateGeneration(ctx, adapterReq)
	if err != nil {
		return nil, err
	}

	return &GenerationResponse{
		TaskID: resp.TaskID,
		Status: TaskStatus(resp.Status),
	}, nil
}

// GetGeneration retrieves the status and result of a generation task
func (w *adapterWrapper) GetGeneration(ctx context.Context, taskID string) (*TaskResult, error) {
	result, err := w.provider.GetGeneration(ctx, taskID)
	if err != nil {
		return nil, err
	}

	mainResult := &TaskResult{
		TaskID: result.TaskID,
		Status: TaskStatus(result.Status),
		URL:    result.URL,
		Format: result.Format,
	}

	if result.Metadata != nil {
		mainResult.Metadata = &Metadata{
			Duration: result.Metadata.Duration,
			FPS:      result.Metadata.FPS,
			Width:    result.Metadata.Width,
			Height:   result.Metadata.Height,
			Seed:     result.Metadata.Seed,
			Format:   result.Metadata.Format,
		}
	}

	if result.Error != nil {
		mainResult.Error = &TaskError{
			Code:    result.Error.Code,
			Message: result.Error.Message,
		}
	}

	return mainResult, nil
}

// SupportedModels returns a list of supported models for this provider
func (w *adapterWrapper) SupportedModels() []string {
	return w.provider.SupportedModels()
}

// ValidateRequest validates if the request is compatible with this provider
func (w *adapterWrapper) ValidateRequest(req *GenerationRequest) error {

	adapterReq := &adapters.GenerationRequest{
		Prompt:         req.Prompt,
		Image:          req.Image,
		Style:          req.Style,
		Duration:       req.Duration,
		FPS:            req.FPS,
		Width:          req.Width,
		Height:         req.Height,
		ResponseFormat: adapters.ResponseFormat(req.ResponseFormat),
		QualityLevel:   adapters.QualityLevel(req.QualityLevel),
		Seed:           req.Seed,
		Model:          req.Model,
		Metadata:       req.Metadata,
	}

	return w.provider.ValidateRequest(adapterReq)
}
