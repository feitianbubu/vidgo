package kling

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/feitianbubu/vidgo/adapters"
	"github.com/golang-jwt/jwt"
)

// TaskAdaptorError represents an error in task processing
type TaskAdaptorError struct {
	StatusCode int    `json:"status_code"`
	Code       string `json:"code"`
	Message    string `json:"message"`
	LocalError bool   `json:"local_error"`
}

func (e *TaskAdaptorError) Error() string {
	return e.Message
}

// TaskRelayInfo contains information needed for task relay
type TaskRelayInfo struct {
	ChannelType int
	BaseUrl     string
	ApiKey      string
	Action      string
}

// VidgoSubmitReq represents a video generation request
// For Kling: metadata.image or metadata.image_tail is required (cannot both be empty)
// Optional metadata fields: image_tail, camera_moving
type VidgoSubmitReq struct {
	Prompt   string                 `json:"prompt"`             // Required: 文本描述
	Model    string                 `json:"model,omitempty"`    // Optional: 模型名称
	Mode     string                 `json:"mode,omitempty"`     // Optional: 模式 "std" or "pro", defaults to "std"
	Image    string                 `json:"image,omitempty"`    // Optional: 图像URL，用于图生视频
	Size     string                 `json:"size,omitempty"`     // Optional: 画面尺寸，用于推断aspect_ratio
	Duration int                    `json:"duration,omitempty"` // Optional: 视频时长（秒），5或10，默认5
	Metadata map[string]interface{} `json:"metadata,omitempty"` // Optional: 额外的元数据
}

// TaskResponse represents a generic task response
type TaskResponse[T any] struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

func (t *TaskResponse[T]) IsSuccess() bool {
	return t.Code == "success"
}

// KlingAdaptor implements TaskAdaptorInterface for Kling video generation
type KlingAdaptor struct {
	ChannelType int
	provider    *Provider // Use the existing Provider implementation
}

// NewKlingAdaptor creates a new KlingAdaptor instance
func NewKlingAdaptor() *KlingAdaptor {
	return &KlingAdaptor{}
}

// Init initializes the Kling adaptor
func (k *KlingAdaptor) Init(info *TaskRelayInfo) {
	k.ChannelType = info.ChannelType

	// Create provider config
	providerConfig := &adapters.ProviderConfig{
		APIKey:  info.ApiKey,
		BaseURL: info.BaseUrl,
	}
	if providerConfig.BaseURL == "" {
		providerConfig.BaseURL = "https://api.klingai.com"
	}

	// Create the provider instance
	provider, err := New(providerConfig)
	if err != nil {
		// For now, we'll store the error and handle it later
		// In a real implementation, you might want to panic or handle this differently
		return
	}

	k.provider = provider.(*Provider)
}

// ValidateRequestAndSetAction validates the request and sets the action for Kling
func (k *KlingAdaptor) ValidateRequestAndSetAction(requestBody []byte, action string) (*VidgoSubmitReq, *TaskAdaptorError) {
	action = strings.ToLower(action)

	var vidgoRequest VidgoSubmitReq
	err := json.Unmarshal(requestBody, &vidgoRequest)
	if err != nil {
		return nil, &TaskAdaptorError{
			StatusCode: 400,
			Code:       "invalid_request",
			Message:    "Failed to parse request: " + err.Error(),
			LocalError: true,
		}
	}

	// Convert to adapters.GenerationRequest and validate using provider
	generationReq := k.convertToGenerationRequest(&vidgoRequest)
	if k.provider != nil {
		err = k.provider.ValidateRequest(generationReq)
		if err != nil {
			return nil, &TaskAdaptorError{
				StatusCode: 400,
				Code:       "invalid_request",
				Message:    err.Error(),
				LocalError: true,
			}
		}
	}

	return &vidgoRequest, nil
}

// convertToGenerationRequest converts VidgoSubmitReq to adapters.GenerationRequest
func (k *KlingAdaptor) convertToGenerationRequest(req *VidgoSubmitReq) *adapters.GenerationRequest {
	generationReq := &adapters.GenerationRequest{
		Prompt:   req.Prompt,
		Model:    req.Model, // modelName取自vidgo的model
		Image:    req.Image, // image取自vidgo的image
		Duration: float64(req.Duration),
		Metadata: req.Metadata, // 传递metadata用于获取mode
	}

	// Extract width and height from size
	if req.Size != "" {
		switch req.Size {
		case "1024x1024":
			generationReq.Width, generationReq.Height = 1024, 1024
		case "512x512":
			generationReq.Width, generationReq.Height = 512, 512
		case "1280x720":
			generationReq.Width, generationReq.Height = 1280, 720
		case "1920x1080":
			generationReq.Width, generationReq.Height = 1920, 1080
		case "720x1280":
			generationReq.Width, generationReq.Height = 720, 1280
		case "1080x1920":
			generationReq.Width, generationReq.Height = 1080, 1920
		}
	}

	// Extract image from metadata
	if req.Metadata != nil {
		if image, ok := req.Metadata["image"].(string); ok && image != "" {
			generationReq.Image = image
		}
	}

	return generationReq
}

// BuildRequestURL builds the request URL for Kling video generation API
func (k *KlingAdaptor) BuildRequestURL(info *TaskRelayInfo) (string, error) {
	baseURL := info.BaseUrl
	// If baseURL is still empty, use default
	if baseURL == "" {
		baseURL = "https://api.klingai.com"
	}
	// Use Kling's actual API endpoint
	fullRequestURL := fmt.Sprintf("%s%s", baseURL, "/v1/videos/image2video")
	return fullRequestURL, nil
}

// BuildRequestHeader builds the request headers for Kling
func (k *KlingAdaptor) BuildRequestHeader(info *TaskRelayInfo) map[string]string {
	// Create JWT token for authentication
	token, err := k.createJWTToken()
	if err != nil {
		// Fallback to basic auth if JWT fails
		token = info.ApiKey
	}

	return map[string]string{
		"Content-Type":  "application/json",
		"Accept":        "application/json",
		"Authorization": "Bearer " + token,
		"User-Agent":    "vidgo-sdk/1.0",
	}
}

// BuildRequestBody builds the request body for Kling API call
func (k *KlingAdaptor) BuildRequestBody(vidgoRequest *VidgoSubmitReq) ([]byte, error) {
	// We don't need to build actual request body since we're using provider directly
	// Just marshal the vidgoRequest for consistency
	data, err := json.Marshal(vidgoRequest)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// DoRequest performs video generation using the provider (bypasses HTTP)
func (k *KlingAdaptor) DoRequest(url string, headers map[string]string, requestBody []byte) (*http.Response, error) {
	if k.provider == nil {
		return nil, fmt.Errorf("provider not initialized")
	}

	// Parse the request body to VidgoSubmitReq
	var vidgoRequest VidgoSubmitReq
	err := json.Unmarshal(requestBody, &vidgoRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to parse request: %w", err)
	}

	// Convert to GenerationRequest and call provider
	generationReq := k.convertToGenerationRequest(&vidgoRequest)
	generationResp, err := k.provider.CreateGeneration(context.Background(), generationReq)
	if err != nil {
		return nil, fmt.Errorf("video generation failed: %w", err)
	}

	// Create a proper response with the generation result including taskID and status
	responseData := map[string]interface{}{
		"code":    0,
		"message": "success",
		"data": map[string]interface{}{
			"task_id": generationResp.TaskID,
			"status":  string(generationResp.Status), // Include status in response
		},
	}

	responseBytes, err := json.Marshal(responseData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	// Create a mock HTTP response
	mockResp := &http.Response{
		StatusCode: 200,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader(responseBytes)),
	}
	mockResp.Header.Set("Content-Type", "application/json")

	return mockResp, nil
}

// Video represents a single video in the response
type Video struct {
	ID       string `json:"id"`
	Url      string `json:"url"`
	Duration string `json:"duration"`
}

// TaskResult represents the task result containing videos
type TaskResult struct {
	Videos []Video `json:"videos,omitempty"`
}

// TaskData represents the data field in the response
type TaskData struct {
	TaskID     string     `json:"task_id"`
	Status     string     `json:"status,omitempty"`
	TaskResult TaskResult `json:"task_result,omitempty"`
}

// Response represents Kling's response format
type Response struct {
	Code    int      `json:"code"`
	Message string   `json:"message"`
	Data    TaskData `json:"data"`
	Url     string   `json:"url,omitempty"`
}

// DoResponse processes the Kling API response
func (k *KlingAdaptor) DoResponse(resp *http.Response) (taskID string, taskData []byte, taskErr *TaskAdaptorError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		taskErr = &TaskAdaptorError{
			StatusCode: 500,
			Code:       "read_response_body_failed",
			Message:    err.Error(),
			LocalError: true,
		}
		return
	}

	// Try to parse as Kling response first
	var klingResponse Response
	err = json.Unmarshal(responseBody, &klingResponse)
	if err == nil && klingResponse.Code == 0 {
		// Success response from Kling with taskID and status
		if klingResponse.Data.TaskID != "" {
			return klingResponse.Data.TaskID, responseBody, nil
		}
	}

	// If not Kling format, try standard format
	var vidgoResponse TaskResponse[string]
	err = json.Unmarshal(responseBody, &vidgoResponse)
	if err != nil {
		taskErr = &TaskAdaptorError{
			StatusCode: 500,
			Code:       "unmarshal_response_body_failed",
			Message:    err.Error(),
			LocalError: true,
		}
		return
	}

	if !vidgoResponse.IsSuccess() {
		taskErr = &TaskAdaptorError{
			StatusCode: resp.StatusCode,
			Code:       vidgoResponse.Code,
			Message:    vidgoResponse.Message,
			LocalError: false,
		}
		return
	}

	// Handle error responses
	if klingResponse.Code != 0 {
		taskErr = &TaskAdaptorError{
			StatusCode: resp.StatusCode,
			Code:       fmt.Sprintf("kling_error_%d", klingResponse.Code),
			Message:    klingResponse.Message,
			LocalError: false,
		}
		return
	}

	return vidgoResponse.Data, responseBody, nil
}

// FetchTask fetches the status of a Kling video generation task using provider
func (k *KlingAdaptor) FetchTask(baseUrl, key string, taskID string) (*http.Response, error) {
	if k.provider == nil {
		return nil, fmt.Errorf("provider not initialized")
	}

	// Use provider to get task status
	taskResult, err := k.provider.GetGeneration(context.Background(), taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task status: %w", err)
	}

	// Convert TaskResult to Kling response format
	var responseData interface{}

	if taskResult.Status == adapters.TaskStatusSucceeded && taskResult.URL != "" {
		// Task completed with video URL
		responseData = Response{
			Code:    0,
			Message: "success",
			Data: TaskData{
				TaskID: taskResult.TaskID,
				Status: "succeeded",
				TaskResult: TaskResult{
					Videos: []Video{
						{
							ID:  taskResult.TaskID,
							Url: taskResult.URL,
							Duration: func() string {
								if taskResult.Metadata != nil {
									return fmt.Sprintf("%.0f", taskResult.Metadata.Duration)
								}
								return "5"
							}(),
						},
					},
				},
			},
			Url: taskResult.URL,
		}
	} else {
		// Task still in progress or failed
		status := "submitted"
		switch taskResult.Status {
		case adapters.TaskStatusQueued:
			status = "submitted"
		case adapters.TaskStatusProcessing:
			status = "processing"
		case adapters.TaskStatusFailed:
			status = "failed"
		}

		responseData = Response{
			Code:    0,
			Message: "success",
			Data: TaskData{
				TaskID: taskResult.TaskID,
				Status: status,
			},
		}
	}

	responseBytes, err := json.Marshal(responseData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	// Create a mock HTTP response
	mockResp := &http.Response{
		StatusCode: 200,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader(responseBytes)),
	}
	mockResp.Header.Set("Content-Type", "application/json")

	return mockResp, nil
}

// GetModelList returns the list of supported Kling models
func (k *KlingAdaptor) GetModelList() []string {
	return []string{
		"kling-v1", "kling-v1-6", "kling-v2-master",
	}
}

// GetChannelName returns the channel name for Kling
func (k *KlingAdaptor) GetChannelName() string {
	return "kling"
}

// createJWTToken creates JWT token for Kling API with proper JWT signature
func (k *KlingAdaptor) createJWTToken() (string, error) {
	if k.provider == nil {
		return "", fmt.Errorf("provider not initialized")
	}
	return k.provider.createJWTToken()
}

// createJWTTokenWithKey creates JWT token using provided key (access_key,secret_key format)
func (k *KlingAdaptor) createJWTTokenWithKey(apiKey string) (string, error) {
	keyParts := strings.Split(apiKey, ",")
	if len(keyParts) != 2 {
		return "", fmt.Errorf("invalid API key format for Kling, expected 'access_key,secret_key'")
	}

	accessKey := strings.TrimSpace(keyParts[0])
	secretKey := strings.TrimSpace(keyParts[1])

	return k.createJWTTokenWithKeys(accessKey, secretKey)
}

// createJWTTokenWithKeys creates JWT token with specific access and secret keys
func (k *KlingAdaptor) createJWTTokenWithKeys(accessKey, secretKey string) (string, error) {
	if accessKey == "" || secretKey == "" {
		return "", fmt.Errorf("access key and secret key are required")
	}

	now := time.Now().Unix()
	claims := jwt.MapClaims{
		"iss": accessKey,
		"exp": now + 1800, // 30分钟
		"nbf": now - 5,    // 提前5秒生效
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["typ"] = "JWT"
	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", err
	}
	return tokenString, nil
}
