package kling

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

	"github.com/feitianbubu/vidgo/adapters"
	"github.com/golang-jwt/jwt"
)

// Provider implements the adapters.Provider interface for Kling video generation
type Provider struct {
	config    *adapters.ProviderConfig
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
	ModelName    string  `json:"model_name,omitempty"`
	CfgScale     float64 `json:"cfg_scale,omitempty"`
	StaticMask   string  `json:"static_mask,omitempty"`
	DynamicMasks []struct {
		Mask         string `json:"mask"`
		Trajectories []struct {
			X int `json:"x"`
			Y int `json:"y"`
		} `json:"trajectories"`
	} `json:"dynamic_masks,omitempty"`
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

var supportedModels = []string{
	"kling-v1",
	"kling-v1-6",
	"kling-v2-master",
}

// New creates a new Kling provider instance
func New(config *adapters.ProviderConfig) (adapters.Provider, error) {
	if config == nil {
		return nil, fmt.Errorf("invalid configuration")
	}

	keyParts := strings.Split(config.APIKey, ",")
	if len(keyParts) != 2 {
		return nil, fmt.Errorf("invalid API key format for Kling, expected 'access_key,secret_key'")
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.klingai.com"
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &Provider{
		config:    config,
		client:    &http.Client{Timeout: timeout},
		baseURL:   baseURL,
		accessKey: strings.TrimSpace(keyParts[0]),
		secretKey: strings.TrimSpace(keyParts[1]),
	}, nil
}

// Name returns the provider name
func (p *Provider) Name() string {
	return "Kling"
}

// SupportedModels returns supported models
func (p *Provider) SupportedModels() []string {
	return append([]string{}, supportedModels...)
}

// ValidateRequest validates the request for Kling
func (p *Provider) ValidateRequest(req *adapters.GenerationRequest) error {
	if req.Model != "" {
		found := false
		for _, model := range supportedModels {
			if model == req.Model {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("unsupported model: %s", req.Model)
		}
	}

	if req.Duration != 5.0 && req.Duration != 10.0 {
		return fmt.Errorf("Kling only supports 5s or 10s duration")
	}

	return nil
}

// CreateGeneration creates a video generation task
func (p *Provider) CreateGeneration(ctx context.Context, req *adapters.GenerationRequest) (*adapters.GenerationResponse, error) {
	klingReq := p.convertToKlingRequest(req)

	token, err := p.createJWTToken()
	if err != nil {
		return nil, fmt.Errorf("failed to create JWT token: %w", err)
	}

	url := fmt.Sprintf("%s/v1/videos/image2video", p.baseURL)
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
		return nil, fmt.Errorf("API error %d: %s", klingResp.Code, klingResp.Message)
	}

	return &adapters.GenerationResponse{
		TaskID: klingResp.Data.TaskID,
		Status: adapters.TaskStatusQueued,
	}, nil
}

// GetGeneration retrieves the task status
func (p *Provider) GetGeneration(ctx context.Context, taskID string) (*adapters.TaskResult, error) {
	token, err := p.createJWTToken()
	if err != nil {
		return nil, fmt.Errorf("failed to create JWT token: %w", err)
	}

	url := fmt.Sprintf("%s/v1/videos/image2video/%s", p.baseURL, taskID)
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
		return nil, fmt.Errorf("API error %d: %s", klingResp.Code, klingResp.Message)
	}

	return p.convertToTaskResult(&klingResp.Data), nil
}

// convertToKlingRequest converts standard request to Kling format
func (p *Provider) convertToKlingRequest(req *adapters.GenerationRequest) *KlingGenerationRequest {
	klingReq := &KlingGenerationRequest{
		Prompt:    req.Prompt,
		Image:     req.Image,
		ModelName: req.Model,
		Model:     req.Model,
	}

	// mode取自metadata的mode，如果没取到默认为std
	klingReq.Mode = "std" // 默认为std
	if req.Metadata != nil {
		if mode, ok := req.Metadata["mode"].(string); ok && mode != "" {
			klingReq.Mode = mode
		}
	}

	if req.Duration == 10.0 {
		klingReq.Duration = "10"
	} else {
		klingReq.Duration = "5"
	}

	aspectRatio := p.getAspectRatio(req.Width, req.Height)
	klingReq.AspectRatio = aspectRatio

	if req.Model == "" {
		klingReq.Model = "kling-v2-master"
		klingReq.ModelName = "kling-v2-master"
	}

	// 设置默认的cfg_scale
	klingReq.CfgScale = 0.5

	return klingReq
}

// getAspectRatio determines aspect ratio from width and height
func (p *Provider) getAspectRatio(width, height int) string {
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
func (p *Provider) convertToTaskResult(data *KlingTaskResult) *adapters.TaskResult {
	result := &adapters.TaskResult{
		TaskID: data.ID,
		Status: p.convertStatus(data.Status),
	}

	if data.TaskResult != nil && len(data.TaskResult.Videos) > 0 {
		video := data.TaskResult.Videos[0]
		result.URL = video.URL
		result.Format = "mp4"

		if duration, err := strconv.ParseFloat(video.Duration, 64); err == nil {
			result.Metadata = &adapters.Metadata{
				Duration: duration,
				Format:   "mp4",
			}
		}
	}

	return result
}

// convertStatus converts Kling status to standard status
func (p *Provider) convertStatus(status string) adapters.TaskStatus {
	switch status {
	case "submitted", "queued":
		return adapters.TaskStatusQueued
	case "processing":
		return adapters.TaskStatusProcessing
	case "succeed":
		return adapters.TaskStatusSucceeded
	case "failed":
		return adapters.TaskStatusFailed
	default:
		return adapters.TaskStatusQueued
	}
}

// createJWTToken creates JWT token for Kling API with proper JWT signature
func (p *Provider) createJWTToken() (string, error) {
	now := time.Now().Unix()
	claims := jwt.MapClaims{
		"iss": p.accessKey,
		"exp": now + 1800, // 30分钟
		"nbf": now - 5,    // 提前5秒生效
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["typ"] = "JWT"
	tokenString, err := token.SignedString([]byte(p.secretKey))
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

// makeRequest makes HTTP request with proper authentication
func (p *Provider) makeRequest(ctx context.Context, method, url, token string, body interface{}) (*http.Response, error) {
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
