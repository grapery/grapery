package genapi

import "time"

// MediaType describes the asset category targeted by a generation request.
type MediaType string

const (
	// MediaTypeImage indicates an image generation workflow.
	MediaTypeImage MediaType = "image"
	// MediaTypeVideo indicates a video generation workflow.
	MediaTypeVideo MediaType = "video"
)

// OperationType enumerates the high-level generation flows supported by the proxy.
type OperationType string

const (
	OperationUnknown           OperationType = ""
	OperationTextToImage       OperationType = "text_to_image"
	OperationImageToImage      OperationType = "image_to_image"
	OperationTextToVideo       OperationType = "text_to_video"
	OperationImageToVideo      OperationType = "image_to_video"
	OperationKeyframeToVideo   OperationType = "keyframe_to_video"
	OperationStoryboardToVideo OperationType = "storyboard_to_video"
)

// TaskStatus represents the normalized status of a generation task.
type TaskStatus string

const (
	// StatusUnknown indicates the task status could not be determined.
	StatusUnknown TaskStatus = ""
	// StatusPending indicates the task is queued but not yet started.
	StatusPending TaskStatus = "pending"
	// StatusProcessing indicates the task is currently being processed.
	StatusProcessing TaskStatus = "processing"
	// StatusCompleted indicates the task finished successfully.
	StatusCompleted TaskStatus = "completed"
	// StatusFailed indicates the task failed with an error.
	StatusFailed TaskStatus = "failed"
	// StatusCancelled indicates the task was cancelled before completion.
	StatusCancelled TaskStatus = "cancelled"
)

// GenerationMode indicates the high-level conditioning mode for a generation request.
type GenerationMode string

const (
	GenerationModeUnknown    GenerationMode = ""
	GenerationModeText       GenerationMode = "text"
	GenerationModeImage      GenerationMode = "image"
	GenerationModeKeyframe   GenerationMode = "keyframe"
	GenerationModeStoryboard GenerationMode = "storyboard"
)

// MediaType derives the media category targeted by the given operation.
func (op OperationType) MediaType() MediaType {
	switch op {
	case OperationTextToImage, OperationImageToImage:
		return MediaTypeImage
	case OperationTextToVideo, OperationImageToVideo, OperationKeyframeToVideo, OperationStoryboardToVideo:
		return MediaTypeVideo
	default:
		return MediaType("")
	}
}

// GenerateRequest captures the unified request parameters accepted by the proxy layer.
type GenerateRequest struct {
	Operation         OperationType
	Prompt            string
	NegativePrompt    string
	AspectRatio       string
	Resolution        string
	DurationSeconds   int
	Quality           string
	Style             string
	Model             string
	CallbackURL       string
	Metadata          map[string]interface{}
	Options           map[string]interface{}
	ReferenceImageURL string
	ReferenceImages   []string
	FirstFrameURL     string
	LastFrameURL      string
	Size              string
	Width             int
	Height            int
	Seed              int
	OutputCount       int
	ResponseFormat    string
	Watermark         *bool
	GuidanceScale     float64
	AudioURL          string
	Template          string
	Storyboard        map[string]interface{}
	UserID            int64
	Platform          string
	Mode              GenerationMode
	ImageData         []byte
	VideoData         []byte
	ImageMIMEType     string
	VideoMIMEType     string
}

// Clone returns a deep copy of the request to avoid mutating caller state.
func (r *GenerateRequest) Clone() *GenerateRequest {
	if r == nil {
		return nil
	}
	cp := *r
	if len(r.ReferenceImages) > 0 {
		cp.ReferenceImages = append([]string(nil), r.ReferenceImages...)
	}
	if r.Metadata != nil {
		cp.Metadata = cloneMap(r.Metadata)
	}
	if r.Options != nil {
		cp.Options = cloneMap(r.Options)
	}
	if r.Storyboard != nil {
		cp.Storyboard = cloneMap(r.Storyboard)
	}
	if len(r.ImageData) > 0 {
		cp.ImageData = append([]byte(nil), r.ImageData...)
	}
	if len(r.VideoData) > 0 {
		cp.VideoData = append([]byte(nil), r.VideoData...)
	}
	return &cp
}

// GenerateResponse provides a normalized view of provider outputs.
type GenerateResponse struct {
	Provider     string
	Operation    OperationType
	MediaType    MediaType
	TaskID       string
	Status       string
	Message      string
	Progress     int
	ImageURLs    []string
	VideoURL     string
	ThumbnailURL string
	Error        string
	ErrorCode    string
	Usage        *Usage
	Metadata     map[string]interface{}
	Raw          map[string]interface{}
	StartedAt    time.Time
	CompletedAt  time.Time
}

// Clone returns a deep copy of the response.
func (r *GenerateResponse) Clone() *GenerateResponse {
	if r == nil {
		return nil
	}
	cp := *r
	if len(r.ImageURLs) > 0 {
		cp.ImageURLs = append([]string(nil), r.ImageURLs...)
	}
	if r.Metadata != nil {
		cp.Metadata = cloneMap(r.Metadata)
	}
	if r.Raw != nil {
		cp.Raw = cloneMap(r.Raw)
	}
	if r.Usage != nil {
		cp.Usage = r.Usage.Clone()
	}
	return &cp
}

// Duration returns the measured generation duration when timestamps are available.
func (r *GenerateResponse) Duration() time.Duration {
	if r == nil {
		return 0
	}
	if r.StartedAt.IsZero() || r.CompletedAt.IsZero() {
		return 0
	}
	return r.CompletedAt.Sub(r.StartedAt)
}

// Usage captures normalized billing or resource consumption details.
type Usage struct {
	InputTokens     int
	OutputTokens    int
	TotalTokens     int
	ImageCount      int
	VideoCount      int
	DurationSeconds int
	Additional      map[string]interface{}
}

// Clone returns a deep copy of the usage payload.
func (u *Usage) Clone() *Usage {
	if u == nil {
		return nil
	}
	cp := *u
	if u.Additional != nil {
		cp.Additional = cloneMap(u.Additional)
	}
	return &cp
}

// MergeAdditional merges extra usage metadata into the Additional map.
func (u *Usage) MergeAdditional(values map[string]interface{}) {
	if u == nil || len(values) == 0 {
		return
	}
	if u.Additional == nil {
		u.Additional = make(map[string]interface{}, len(values))
	}
	for k, v := range values {
		u.Additional[k] = v
	}
}

// IsEmpty reports whether the usage payload contains meaningful counters.
func (u *Usage) IsEmpty() bool {
	if u == nil {
		return true
	}
	if u.InputTokens != 0 || u.OutputTokens != 0 || u.TotalTokens != 0 {
		return false
	}
	if u.ImageCount != 0 || u.VideoCount != 0 || u.DurationSeconds != 0 {
		return false
	}
	return len(u.Additional) == 0
}
