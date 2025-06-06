package vidgo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/golang-jwt/jwt"
)

// KlingAdaptor implements TaskAdaptorInterface for Kling video generation
type KlingAdaptor struct {
	ChannelType int
	accessKey   string
	secretKey   string
	baseURL     string
}

// NewKlingAdaptor creates a new KlingAdaptor instance
func NewKlingAdaptor() *KlingAdaptor {
	return &KlingAdaptor{}
}

// Init initializes the Kling adaptor
func (k *KlingAdaptor) Init(info *TaskRelayInfo) {
	k.ChannelType = info.ChannelType

	// Set default official URL if baseUrl is empty
	if info.BaseUrl == "" {
		info.BaseUrl = "https://api.klingai.com"
	}
	k.baseURL = info.BaseUrl

	// Parse API key in format "access_key,secret_key"
	keyParts := strings.Split(info.ApiKey, ",")
	if len(keyParts) == 2 {
		k.accessKey = strings.TrimSpace(keyParts[0])
		k.secretKey = strings.TrimSpace(keyParts[1])
	}
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

	err = k.actionValidate(&vidgoRequest, action)
	if err != nil {
		return nil, &TaskAdaptorError{
			StatusCode: 400,
			Code:       "invalid_request",
			Message:    err.Error(),
			LocalError: true,
		}
	}

	return &vidgoRequest, nil
}

// BuildRequestURL builds the request URL for Kling video generation API
func (k *KlingAdaptor) BuildRequestURL(info *TaskRelayInfo) (string, error) {
	fullRequestURL := fmt.Sprintf("%s%s", k.baseURL, "/v1/videos/image2video")
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

// KlingRequest represents Kling-specific request format
type KlingRequest struct {
	Prompt      string  `json:"prompt,omitempty"`
	Image       string  `json:"image,omitempty"`
	Mode        string  `json:"mode,omitempty"`
	Duration    string  `json:"duration,omitempty"`
	AspectRatio string  `json:"aspect_ratio,omitempty"`
	Model       string  `json:"model,omitempty"`
	ModelName   string  `json:"model_name,omitempty"`
	CfgScale    float64 `json:"cfg_scale,omitempty"`
}

// BuildRequestBody builds the request body for Kling API call
func (k *KlingAdaptor) BuildRequestBody(vidgoRequest *VidgoSubmitReq) ([]byte, error) {
	// Convert to Kling format
	klingReq := k.convertToKlingRequest(vidgoRequest)

	data, err := json.Marshal(klingReq)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// convertToKlingRequest converts standard request to Kling format
func (k *KlingAdaptor) convertToKlingRequest(req *VidgoSubmitReq) *KlingRequest {
	klingReq := &KlingRequest{
		Prompt:    req.Prompt,
		ModelName: req.Model, // 1. modelName取自vidgo的model
		Model:     req.Model,
		CfgScale:  0.5, // Default cfg_scale
	}

	// 2. image取自vidgo的image
	klingReq.Image = req.Image

	// 3. mode取自metadata的mode，如果没取到默认为std
	klingReq.Mode = "std" // 默认为std
	if req.Metadata != nil {
		if mode, ok := req.Metadata["mode"].(string); ok && mode != "" {
			klingReq.Mode = mode
		}
	}

	// Convert duration
	if req.Duration == 10 {
		klingReq.Duration = "10"
	} else {
		klingReq.Duration = "5"
	}

	// Set aspect ratio based on size
	klingReq.AspectRatio = k.getAspectRatio(req.Size)

	// Set default model if not specified
	if klingReq.Model == "" {
		klingReq.ModelName = "kling-v1"
	}

	return klingReq
}

// getAspectRatio determines aspect ratio from size string
func (k *KlingAdaptor) getAspectRatio(size string) string {
	switch size {
	case "1024x1024", "512x512":
		return "1:1"
	case "1280x720", "1920x1080":
		return "16:9"
	case "720x1280", "1080x1920":
		return "9:16"
	default:
		return "1:1"
	}
}

// DoRequest performs the HTTP request to Kling video generation API
func (k *KlingAdaptor) DoRequest(url string, headers map[string]string, requestBody []byte) (*http.Response, error) {
	req, err := http.NewRequest("POST", url, bytes.NewReader(requestBody))
	if err != nil {
		return nil, err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	return client.Do(req)
}

// KlingResponse represents Kling's response format
type KlingResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		TaskID string `json:"task_id"`
	} `json:"data"`
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
	var klingResponse KlingResponse
	err = json.Unmarshal(responseBody, &klingResponse)
	if err == nil && klingResponse.Code == 0 {
		// Success response from Kling
		return klingResponse.Data.TaskID, responseBody, nil
	}

	// If not Kling format, try standard format
	var vidgoResponse TaskResponse[string]
	err = json.Unmarshal(responseBody, &vidgoResponse)
	if err != nil {
		// warn log
		fmt.Printf("unmarshal Kling response fail: %s, body: %s\n", err.Error(), responseBody)
		taskErr = &TaskAdaptorError{
			StatusCode: 500,
			Code:       "unmarshal_response_body_failed",
			Message:    errors.Wrapf(err, "body: %s", responseBody).Error(),
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

// FetchTask fetches the status of a Kling video generation task
func (k *KlingAdaptor) FetchTask(baseUrl, key string, taskID string) (*http.Response, error) {
	// Set default official URL if baseUrl is empty
	if baseUrl == "" {
		baseUrl = "https://api.klingai.com"
	}
	// Use Kling's actual API endpoint for task status
	requestUrl := fmt.Sprintf("%s/v1/videos/image2video/%s", baseUrl, taskID)

	req, err := http.NewRequest("GET", requestUrl, nil)
	if err != nil {
		return nil, err
	}

	// Create JWT token for authentication
	token, err := k.createJWTTokenWithKey(key)
	if err != nil {
		token = key // Fallback to provided key
	}

	// 设置超时时间
	timeout := time.Second * 15
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// 使用带有超时的 context 创建新的请求
	req = req.WithContext(ctx)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", "vidgo-sdk/1.0")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
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

// actionValidate validates the action and request for Kling
func (k *KlingAdaptor) actionValidate(vidgoRequest *VidgoSubmitReq, action string) error {
	if action != "generate" {
		return fmt.Errorf("unsupported action: %s", action)
	}

	if vidgoRequest.Prompt == "" {
		return fmt.Errorf("prompt is required")
	}

	// Validate model if specified
	if vidgoRequest.Model != "" {
		validModels := k.GetModelList()
		isValidModel := false
		for _, model := range validModels {
			if vidgoRequest.Model == model {
				isValidModel = true
				break
			}
		}
		if !isValidModel {
			return fmt.Errorf("unsupported model: %s", vidgoRequest.Model)
		}
	}

	return nil
}

// createJWTToken creates JWT token for Kling API with proper JWT signature
func (k *KlingAdaptor) createJWTToken() (string, error) {
	return k.createJWTTokenWithKeys(k.accessKey, k.secretKey)
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
