package vidgo

import (
	"fmt"
	"net/http"
)

// TaskAdaptorInterface defines the interface for task-based video generation
type TaskAdaptorInterface interface {
	// Init initializes the adaptor with relay information
	Init(info *TaskRelayInfo)

	// ValidateRequestAndSetAction validates the request and sets the action
	ValidateRequestAndSetAction(requestBody []byte, action string) (*VidgoSubmitReq, *TaskAdaptorError)

	// BuildRequestURL builds the request URL for the video generation API
	BuildRequestURL(info *TaskRelayInfo) (string, error)

	// BuildRequestHeader builds the request headers
	BuildRequestHeader(info *TaskRelayInfo) map[string]string

	// BuildRequestBody builds the request body for the API call
	BuildRequestBody(vidgoRequest *VidgoSubmitReq) ([]byte, error)

	// DoRequest performs the HTTP request to the video generation API
	DoRequest(url string, headers map[string]string, requestBody []byte) (*http.Response, error)

	// DoResponse processes the API response
	DoResponse(resp *http.Response) (taskID string, taskData []byte, taskErr *TaskAdaptorError)

	// FetchTask fetches the status of a video generation task
	FetchTask(baseUrl, key string, taskID string) (*http.Response, error)

	// GetModelList returns the list of supported models
	GetModelList() []string

	// GetChannelName returns the channel name
	GetChannelName() string
}

// TaskAdaptor is a factory that creates vendor-specific adaptors
type TaskAdaptor struct {
	vendor string
	impl   TaskAdaptorInterface
}

// TaskRelayInfo contains information needed for task relay
type TaskRelayInfo struct {
	ChannelType int
	BaseUrl     string
	ApiKey      string
	Action      string
}

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

// VidgoSubmitReq represents a video generation request
type VidgoSubmitReq struct {
	Prompt   string                 `json:"prompt"`
	Model    string                 `json:"model,omitempty"`
	Mode     string                 `json:"mode,omitempty"`  // Mode: "std" or "pro", defaults to "std"
	Image    string                 `json:"image,omitempty"` // Image URL for image-to-video
	Size     string                 `json:"size,omitempty"`
	Duration int                    `json:"duration,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
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

// NewTaskAdaptor creates a new TaskAdaptor with default vendor (Kling)
func NewTaskAdaptor() *TaskAdaptor {
	return NewTaskAdaptorWithVendor("kling")
}

// NewTaskAdaptorWithVendor creates a new TaskAdaptor with specified vendor
func NewTaskAdaptorWithVendor(vendor string) *TaskAdaptor {
	var impl TaskAdaptorInterface

	switch vendor {
	case "kling":
		impl = NewKlingAdaptor()
	default:
		impl = NewKlingAdaptor() // Default to Kling
	}

	return &TaskAdaptor{
		vendor: vendor,
		impl:   impl,
	}
}

// ===== High-level workflow methods =====

// ProcessVideoGeneration handles the complete video generation workflow
func (a *TaskAdaptor) ProcessVideoGeneration(info *TaskRelayInfo, requestBody []byte) (taskID string, responseData []byte, taskErr *TaskAdaptorError) {
	// Ensure impl is initialized
	if a.impl == nil {
		switch a.vendor {
		case "kling":
			a.impl = NewKlingAdaptor()
		default:
			a.impl = NewKlingAdaptor()
		}
	}

	// Initialize the vendor-specific adaptor
	a.impl.Init(info)

	// Validate request and set action
	vidgoRequest, taskErr := a.impl.ValidateRequestAndSetAction(requestBody, info.Action)
	if taskErr != nil {
		return
	}

	// Build request URL
	requestUrl, err := a.impl.BuildRequestURL(info)
	if err != nil {
		taskErr = &TaskAdaptorError{
			StatusCode: 500,
			Code:       "build_url_failed",
			Message:    err.Error(),
			LocalError: true,
		}
		return
	}

	// Build headers
	headers := a.impl.BuildRequestHeader(info)

	// Build request body
	requestBodyBytes, err := a.impl.BuildRequestBody(vidgoRequest)
	if err != nil {
		taskErr = &TaskAdaptorError{
			StatusCode: 500,
			Code:       "build_body_failed",
			Message:    err.Error(),
			LocalError: true,
		}
		return
	}

	// Make the request
	resp, err := a.impl.DoRequest(requestUrl, headers, requestBodyBytes)
	if err != nil {
		taskErr = &TaskAdaptorError{
			StatusCode: 500,
			Code:       "request_failed",
			Message:    err.Error(),
			LocalError: true,
		}
		return
	}
	defer resp.Body.Close()

	// Process response
	return a.impl.DoResponse(resp)
}

// ProcessTaskFetch handles the complete task status fetch workflow
func (a *TaskAdaptor) ProcessTaskFetch(info *TaskRelayInfo, taskID string) (*http.Response, error) {
	// Ensure impl is initialized
	if a.impl == nil {
		switch a.vendor {
		case "kling":
			a.impl = NewKlingAdaptor()
		default:
			a.impl = NewKlingAdaptor()
		}
	}

	// Initialize the vendor-specific adaptor
	a.impl.Init(info)

	// Fetch task status
	return a.impl.FetchTask(info.BaseUrl, info.ApiKey, taskID)
}

// ===== Delegate methods for backward compatibility =====

// Delegate all methods to the implementation
func (a *TaskAdaptor) Init(info *TaskRelayInfo) {
	// Ensure impl is initialized before calling Init
	if a.impl == nil {
		switch a.vendor {
		case "kling":
			a.impl = NewKlingAdaptor()
		default:
			a.impl = NewKlingAdaptor()
		}
	}
	a.impl.Init(info)
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(requestBody []byte, action string) (*VidgoSubmitReq, *TaskAdaptorError) {
	a.ensureImpl()
	return a.impl.ValidateRequestAndSetAction(requestBody, action)
}

func (a *TaskAdaptor) BuildRequestURL(info *TaskRelayInfo) (string, error) {
	a.ensureImpl()
	return a.impl.BuildRequestURL(info)
}

func (a *TaskAdaptor) BuildRequestHeader(info *TaskRelayInfo) map[string]string {
	a.ensureImpl()
	return a.impl.BuildRequestHeader(info)
}

func (a *TaskAdaptor) BuildRequestBody(vidgoRequest *VidgoSubmitReq) ([]byte, error) {
	a.ensureImpl()
	return a.impl.BuildRequestBody(vidgoRequest)
}

func (a *TaskAdaptor) DoRequest(url string, headers map[string]string, requestBody []byte) (*http.Response, error) {
	a.ensureImpl()
	return a.impl.DoRequest(url, headers, requestBody)
}

func (a *TaskAdaptor) DoResponse(resp *http.Response) (taskID string, taskData []byte, taskErr *TaskAdaptorError) {
	a.ensureImpl()
	return a.impl.DoResponse(resp)
}

func (a *TaskAdaptor) FetchTask(baseUrl, key string, taskID string) (*http.Response, error) {
	a.ensureImpl()
	return a.impl.FetchTask(baseUrl, key, taskID)
}

func (a *TaskAdaptor) GetModelList() []string {
	a.ensureImpl()
	return a.impl.GetModelList()
}

func (a *TaskAdaptor) GetChannelName() string {
	a.ensureImpl()
	return a.impl.GetChannelName()
}

// ensureImpl ensures that the implementation is initialized
func (a *TaskAdaptor) ensureImpl() {
	if a.impl == nil {
		switch a.vendor {
		case "kling":
			a.impl = NewKlingAdaptor()
		default:
			a.impl = NewKlingAdaptor()
		}
	}
}

// actionValidate validates the action and request parameters
func (a *TaskAdaptor) actionValidate(vidgoRequest *VidgoSubmitReq, action string) error {
	switch action {
	case "generate":
		if vidgoRequest.Prompt == "" {
			return fmt.Errorf("prompt_empty")
		}
		if vidgoRequest.Model == "" {
			vidgoRequest.Model = "kling-v1" // 默认模型
		}
	default:
		return fmt.Errorf("invalid_action")
	}
	return nil
}
