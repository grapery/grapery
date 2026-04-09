package kling

import "encoding/json"

// APIEnvelope is the standard Kling JSON wrapper.
type APIEnvelope struct {
	Code      int             `json:"code"`
	Message   string          `json:"message"`
	RequestID string          `json:"request_id"`
	Data      json.RawMessage `json:"data"`
}

// CreateTaskData is returned by async create endpoints.
type CreateTaskData struct {
	TaskID      string                 `json:"task_id"`
	TaskStatus  string                 `json:"task_status"`
	TaskInfo    map[string]interface{} `json:"task_info"`
	CreatedAt   int64                  `json:"created_at"`
	UpdatedAt   int64                  `json:"updated_at"`
}

// TaskResultVideo is a single generated video.
type TaskResultVideo struct {
	ID        string `json:"id"`
	URL       string `json:"url"`
	Duration  string `json:"duration"`
	CoverURL  string `json:"cover_url"`
	Thumbnail string `json:"thumbnail"`
}

// TaskResultImage is a single generated image.
type TaskResultImage struct {
	Index int    `json:"index"`
	URL   string `json:"url"`
}

// TaskResult holds media outputs when task succeeds.
type TaskResult struct {
	Videos []TaskResultVideo `json:"videos"`
	Images []TaskResultImage `json:"images"`
}

// QueryTaskData is returned by GET task endpoints.
type QueryTaskData struct {
	TaskID        string                 `json:"task_id"`
	TaskStatus    string                 `json:"task_status"`
	TaskStatusMsg string                 `json:"task_status_msg"`
	TaskInfo      map[string]interface{} `json:"task_info"`
	CreatedAt     int64                  `json:"created_at"`
	UpdatedAt     int64                  `json:"updated_at"`
	TaskResult    *TaskResult            `json:"task_result"`
}
