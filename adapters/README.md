# Video Generation Adapters

This package contains adapters for different video generation providers. Each provider is implemented as a separate sub-package.

## Package Structure

```
adapters/
├── types.go           # Common types and interfaces
├── kling/            # Kling provider implementation
├── jimeng/           # Jimeng provider implementation (placeholder)
├── vidu/             # Vidu provider implementation (placeholder)
```

## Implemented Providers

### Kling (`adapters/kling`)
- ✅ Fully implemented
- Models: `kling-v1`, `kling-v1-6`, `kling-v2-master`
- Features: Text-to-video, Image-to-video
- Duration: 5s, 10s

### Jimeng (`adapters/jimeng`) 
- 🚧 Placeholder implementation
- TODO: Implement API integration

### Vidu (`adapters/vidu`)
- 🚧 Placeholder implementation  
- TODO: Implement API integration

## Adding New Providers

To add a new provider:

1. Create a new sub-package: `adapters/newprovider/`
2. Implement the `adapters.Provider` interface
3. Export a `New(config *adapters.ProviderConfig) (adapters.Provider, error)` function
4. Add provider type to main package types
5. Update the client to support the new provider

## Interface

All providers must implement:

```go
type Provider interface {
    Name() string
    CreateGeneration(ctx context.Context, req *GenerationRequest) (*GenerationResponse, error)
    GetGeneration(ctx context.Context, taskID string) (*TaskResult, error)
    SupportedModels() []string
    ValidateRequest(req *GenerationRequest) error
}
``` 