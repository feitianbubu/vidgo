package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Type definitions to avoid circular imports

// TaskStatus represents the status of a video generation task
type TaskStatus string

const (
	TaskStatusQueued     TaskStatus = "queued"
	TaskStatusProcessing TaskStatus = "processing"
	TaskStatusSucceeded  TaskStatus = "succeeded"
	TaskStatusFailed     TaskStatus = "failed"
)

// ResponseFormat represents the format of the response
type ResponseFormat string

const (
	ResponseFormatURL     ResponseFormat = "url"
	ResponseFormatB64JSON ResponseFormat = "b64_json"
)

// QualityLevel represents the quality level of the video
type QualityLevel string

const (
	QualityLevelLow      QualityLevel = "low"
	QualityLevelStandard QualityLevel = "standard"
	QualityLevelHigh     QualityLevel = "high"
)

// GenerationRequest represents a video generation request
type GenerationRequest struct {
	Prompt         string                 `json:"prompt,omitempty"`
	Image          string                 `json:"image,omitempty"`
	Style          string                 `json:"style,omitempty"`
	Duration       float64                `json:"duration"`
	FPS            int                    `json:"fps,omitempty"`
	Width          int                    `json:"width"`
	Height         int                    `json:"height"`
	ResponseFormat ResponseFormat         `json:"response_format,omitempty"`
	QualityLevel   QualityLevel           `json:"quality_level,omitempty"`
	Seed           *int                   `json:"seed,omitempty"`
	Model          string                 `json:"model,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// GenerationResponse represents the response from creating a generation task
type GenerationResponse struct {
	TaskID string     `json:"task_id"`
	Status TaskStatus `json:"status"`
}

// TaskResult represents the result of a video generation task
type TaskResult struct {
	TaskID   string     `json:"task_id"`
	Status   TaskStatus `json:"status"`
	URL      string     `json:"url,omitempty"`
	Format   string     `json:"format,omitempty"`
	Metadata *Metadata  `json:"metadata,omitempty"`
	Error    *TaskError `json:"error,omitempty"`
}

// Metadata contains video metadata information
type Metadata struct {
	Duration float64 `json:"duration,omitempty"`
	FPS      int     `json:"fps,omitempty"`
	Width    int     `json:"width,omitempty"`
	Height   int     `json:"height,omitempty"`
	Seed     *int    `json:"seed,omitempty"`
	Format   string  `json:"format,omitempty"`
}

// TaskError represents an error in task execution
type TaskError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ProviderConfig holds configuration for a specific provider
type ProviderConfig struct {
	BaseURL    string            `json:"base_url"`
	APIKey     string            `json:"api_key"`
	SecretKey  string            `json:"secret_key,omitempty"`
	Timeout    time.Duration     `json:"timeout"`
	RetryCount int               `json:"retry_count"`
	Extra      map[string]string `json:"extra,omitempty"`
}

// ValidationError represents a request validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error for field '%s': %s", e.Field, e.Message)
}

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

// Provider interface (minimal for adapters)
type Provider interface {
	Name() string
	CreateGeneration(ctx context.Context, req *GenerationRequest) (*GenerationResponse, error)
	GetGeneration(ctx context.Context, taskID string) (*TaskResult, error)
	SupportedModels() []string
	ValidateRequest(req *GenerationRequest) error
}

// KlingProvider implements the Provider interface for Kling video generation
type KlingProvider struct {
	config    *ProviderConfig
	client    *http.Client
	baseURL   string
	accessKey string
	secretKey string
}

// KlingGenerationRequest represents Kling-specific request format
type KlingGenerationRequest struct {
	Prompt       string  `json:"prompt,omitempty"`
	Image        string  `json:"image,omitempty"`
	Mode         string  `json:"mode,omitempty"`
	Duration     string  `json:"duration,omitempty"`
	AspectRatio  string  `json:"aspect_ratio,omitempty"`
	CameraMoving *string `json:"camera_moving,omitempty"`
	Model        string  `json:"model,omitempty"`
}

// KlingGenerationResponse represents Kling's response format
type KlingGenerationResponse struct {
	Code    int               `json:"code"`
	Message string            `json:"message"`
	Data    KlingResponseData `json:"data"`
}

type KlingResponseData struct {
	TaskID string `json:"task_id"`
}

// KlingTaskResponse represents Kling's task status response
type KlingTaskResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    KlingTaskResult `json:"data"`
}

type KlingTaskResult struct {
	ID         string               `json:"id"`
	Status     string               `json:"status"`
	CreatedAt  int64                `json:"created_at"`
	UpdatedAt  int64                `json:"updated_at"`
	Task       KlingTaskDetails     `json:"task"`
	TaskResult *KlingTaskResultData `json:"task_result,omitempty"`
}

type KlingTaskDetails struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	Status string `json:"status"`
}

type KlingTaskResultData struct {
	Videos []KlingVideo `json:"videos,omitempty"`
}

type KlingVideo struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Duration string `json:"duration"`
}

var klingModels = []string{
	"kling-v1",
	"kling-v1-6",
	"kling-v2-master",
}

// NewKlingProvider creates a new Kling provider instance
func NewKlingProvider(config *ProviderConfig) (Provider, error) {
	if config == nil {
		return nil, fmt.Errorf("invalid configuration")
	}

	keyParts := strings.Split(config.APIKey, ",")
	if len(keyParts) != 2 {
		return nil, fmt.Errorf("invalid API key format for Kling, expected 'access_key,secret_key'")
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.kuaishou.com"
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &KlingProvider{
		config:    config,
		client:    &http.Client{Timeout: timeout},
		baseURL:   baseURL,
		accessKey: strings.TrimSpace(keyParts[0]),
		secretKey: strings.TrimSpace(keyParts[1]),
	}, nil
}

// Name returns the provider name
func (p *KlingProvider) Name() string {
	return "Kling"
}

// SupportedModels returns supported models
func (p *KlingProvider) SupportedModels() []string {
	return append([]string{}, klingModels...)
}

// ValidateRequest validates the request for Kling
func (p *KlingProvider) ValidateRequest(req *GenerationRequest) error {
	if req.Model != "" {
		found := false
		for _, model := range klingModels {
			if model == req.Model {
				found = true
				break
			}
		}
		if !found {
			return &ValidationError{
				Field:   "model",
				Message: fmt.Sprintf("unsupported model: %s", req.Model),
			}
		}
	}

	if req.Duration != 5.0 && req.Duration != 10.0 {
		return &ValidationError{
			Field:   "duration",
			Message: "Kling only supports 5s or 10s duration",
		}
	}

	return nil
}

// CreateGeneration creates a video generation task
func (p *KlingProvider) CreateGeneration(ctx context.Context, req *GenerationRequest) (*GenerationResponse, error) {
	klingReq := p.convertToKlingRequest(req)

	token, err := p.createJWTToken()
	if err != nil {
		return nil, fmt.Errorf("failed to create JWT token: %w", err)
	}
	url := fmt.Sprintf("%s/api/open/v1/video/generation", p.baseURL)
	resp, err := p.makeRequest(ctx, "POST", url, token, klingReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var klingResp KlingGenerationResponse
	if err := json.NewDecoder(resp.Body).Decode(&klingResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if klingResp.Code != 0 {
		return nil, &APIError{
			Code:     klingResp.Code,
			Message:  klingResp.Message,
			Provider: "Kling",
		}
	}

	return &GenerationResponse{
		TaskID: klingResp.Data.TaskID,
		Status: TaskStatusQueued,
	}, nil
}

// GetGeneration retrieves the task status
func (p *KlingProvider) GetGeneration(ctx context.Context, taskID string) (*TaskResult, error) {
	token, err := p.createJWTToken()
	if err != nil {
		return nil, fmt.Errorf("failed to create JWT token: %w", err)
	}
	url := fmt.Sprintf("%s/api/open/v1/video/generation/%s", p.baseURL, taskID)
	resp, err := p.makeRequest(ctx, "GET", url, token, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var klingResp KlingTaskResponse
	if err := json.NewDecoder(resp.Body).Decode(&klingResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if klingResp.Code != 0 {
		return nil, &APIError{
			Code:     klingResp.Code,
			Message:  klingResp.Message,
			Provider: "Kling",
		}
	}

	return p.convertToTaskResult(&klingResp.Data), nil
}

// convertToKlingRequest converts standard request to Kling format
func (p *KlingProvider) convertToKlingRequest(req *GenerationRequest) *KlingGenerationRequest {
	klingReq := &KlingGenerationRequest{
		Prompt: req.Prompt,
		Image:  req.Image,
	}

	if req.Image != "" {
		klingReq.Mode = "img2video"
	} else {
		klingReq.Mode = "txt2video"
	}

	if req.Duration == 10.0 {
		klingReq.Duration = "10"
	} else {
		klingReq.Duration = "5"
	}

	aspectRatio := p.getAspectRatio(req.Width, req.Height)
	klingReq.AspectRatio = aspectRatio

	if req.Model != "" {
		klingReq.Model = req.Model
	} else {
		klingReq.Model = "kling-v2-master"
	}

	return klingReq
}

// getAspectRatio determines aspect ratio from width and height
func (p *KlingProvider) getAspectRatio(width, height int) string {
	ratio := float64(width) / float64(height)

	switch {
	case ratio > 1.5:
		return "16:9"
	case ratio < 0.7:
		return "9:16"
	default:
		return "1:1"
	}
}

// convertToTaskResult converts Kling task result to standard format
func (p *KlingProvider) convertToTaskResult(data *KlingTaskResult) *TaskResult {
	result := &TaskResult{
		TaskID: data.ID,
		Status: p.convertStatus(data.Status),
	}

	if data.TaskResult != nil && len(data.TaskResult.Videos) > 0 {
		video := data.TaskResult.Videos[0]
		result.URL = video.URL
		result.Format = "mp4"

		if duration, err := strconv.ParseFloat(video.Duration, 64); err == nil {
			result.Metadata = &Metadata{
				Duration: duration,
				Format:   "mp4",
			}
		}
	}

	return result
}

// convertStatus converts Kling status to standard status
func (p *KlingProvider) convertStatus(status string) TaskStatus {
	switch status {
	case "submitted", "queued":
		return TaskStatusQueued
	case "processing":
		return TaskStatusProcessing
	case "succeed":
		return TaskStatusSucceeded
	case "failed":
		return TaskStatusFailed
	default:
		return TaskStatusQueued
	}
}

// createJWTToken creates JWT token for Kling API
func (p *KlingProvider) createJWTToken() (string, error) {

	return fmt.Sprintf("%s:%s", p.accessKey, p.secretKey), nil
}

// makeRequest makes HTTP request with proper authentication
func (p *KlingProvider) makeRequest(ctx context.Context, method, url, token string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", "vidgo-sdk/1.0")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}

	return resp, nil
}
