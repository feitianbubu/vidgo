package vidgo

import (
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	config := &ProviderConfig{
		BaseURL: "https://test.api.com",
		APIKey:  "test_access_key,test_secret_key",
		Timeout: 30 * time.Second,
	}

	client, err := NewClient(ProviderKling, config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if client == nil {
		t.Fatal("Client is nil")
	}

	if client.GetProviderName() != "Kling" {
		t.Errorf("Expected provider name 'Kling', got '%s'", client.GetProviderName())
	}

	models := client.GetSupportedModels()
	if len(models) == 0 {
		t.Error("Expected supported models, got empty list")
	}
}

func TestValidateRequest(t *testing.T) {
	config := &ProviderConfig{
		BaseURL: "https://test.api.com",
		APIKey:  "test_access_key,test_secret_key",
		Timeout: 30 * time.Second,
	}

	client, err := NewClient(ProviderKling, config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test valid request
	validReq := &GenerationRequest{
		Prompt:   "Test prompt",
		Duration: 5.0,
		Width:    512,
		Height:   512,
		Model:    "kling-v2-master",
	}

	err = client.validateRequest(validReq)
	if err != nil {
		t.Errorf("Valid request should not return error: %v", err)
	}

	// Test invalid request - no prompt or image
	invalidReq := &GenerationRequest{
		Duration: 5.0,
		Width:    512,
		Height:   512,
	}

	err = client.validateRequest(invalidReq)
	if err == nil {
		t.Error("Invalid request should return validation error")
	}

	// Test invalid duration
	invalidDurationReq := &GenerationRequest{
		Prompt:   "Test prompt",
		Duration: 0,
		Width:    512,
		Height:   512,
	}

	err = client.validateRequest(invalidDurationReq)
	if err == nil {
		t.Error("Request with zero duration should return validation error")
	}
}

func TestProviderTypes(t *testing.T) {
	// Test provider type constants
	if ProviderKling != "kling" {
		t.Errorf("Expected ProviderKling to be 'kling', got '%s'", ProviderKling)
	}

	if ProviderJimeng != "jimeng" {
		t.Errorf("Expected ProviderJimeng to be 'jimeng', got '%s'", ProviderJimeng)
	}

	if ProviderVidu != "vidu" {
		t.Errorf("Expected ProviderVidu to be 'vidu', got '%s'", ProviderVidu)
	}
}

func TestTaskStatus(t *testing.T) {
	// Test task status constants
	if TaskStatusQueued != "queued" {
		t.Errorf("Expected TaskStatusQueued to be 'queued', got '%s'", TaskStatusQueued)
	}

	if TaskStatusProcessing != "processing" {
		t.Errorf("Expected TaskStatusProcessing to be 'processing', got '%s'", TaskStatusProcessing)
	}

	if TaskStatusSucceeded != "succeeded" {
		t.Errorf("Expected TaskStatusSucceeded to be 'succeeded', got '%s'", TaskStatusSucceeded)
	}

	if TaskStatusFailed != "failed" {
		t.Errorf("Expected TaskStatusFailed to be 'failed', got '%s'", TaskStatusFailed)
	}
}

func TestErrors(t *testing.T) {
	// Test APIError
	apiErr := &APIError{
		Code:     400,
		Message:  "Bad Request",
		Provider: "TestProvider",
	}

	expected := "[TestProvider] API error 400: Bad Request"
	if apiErr.Error() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, apiErr.Error())
	}

	// Test ValidationError
	validationErr := &ValidationError{
		Field:   "duration",
		Message: "must be positive",
	}

	expected = "validation error for field 'duration': must be positive"
	if validationErr.Error() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, validationErr.Error())
	}
}

func TestIsRetryableError(t *testing.T) {
	// Test retryable errors
	retryableErr := &APIError{Code: 500, Message: "Internal Server Error"}
	if !IsRetryableError(retryableErr) {
		t.Error("500 error should be retryable")
	}

	rateLimitErr := &APIError{Code: 429, Message: "Rate limit exceeded"}
	if !IsRetryableError(rateLimitErr) {
		t.Error("429 error should be retryable")
	}

	// Test non-retryable errors
	clientErr := &APIError{Code: 400, Message: "Bad Request"}
	if IsRetryableError(clientErr) {
		t.Error("400 error should not be retryable")
	}
}
