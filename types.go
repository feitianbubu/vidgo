package vidgo

import "time"

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

// ProviderType represents different video generation providers
type ProviderType string

const (
	ProviderKling  ProviderType = "kling"
	ProviderJimeng ProviderType = "jimeng"
	ProviderVidu   ProviderType = "vidu"
)
