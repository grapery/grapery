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

// ReferenceImageAsset holds image bytes and MIME type for provider use (e.g. video reference images).
type ReferenceImageAsset struct {
	Data     []byte
	MIMEType string
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
	// ReferenceImagesData holds inline image bytes for video reference images (up to 3 for Veo 3.1).
	ReferenceImagesData []ReferenceImageAsset
	FirstFrameURL       string
	LastFrameURL        string
	// FirstFrameData, LastFrameData for keyframe-to-video. When set, used instead of FirstFrameURL/LastFrameURL.
	FirstFrameData     []byte
	LastFrameData      []byte
	FirstFrameMIMEType string
	LastFrameMIMEType  string
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
	if len(r.ReferenceImagesData) > 0 {
		cp.ReferenceImagesData = make([]ReferenceImageAsset, len(r.ReferenceImagesData))
		for i, a := range r.ReferenceImagesData {
			cp.ReferenceImagesData[i] = ReferenceImageAsset{
				Data:     append([]byte(nil), a.Data...),
				MIMEType: a.MIMEType,
			}
		}
	}
	if len(r.FirstFrameData) > 0 {
		cp.FirstFrameData = append([]byte(nil), r.FirstFrameData...)
	}
	if len(r.LastFrameData) > 0 {
		cp.LastFrameData = append([]byte(nil), r.LastFrameData...)
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

// ============== Keyframe Subdivision Types ==============

// VideoSegment represents a single video segment generated from keyframe subdivision.
type VideoSegment struct {
	Index        int    `json:"index"`        // Segment sequence number (0-based)
	VideoURL     string `json:"videoUrl"`     // Video URL
	StartFrame   string `json:"startFrame"`   // Start keyframe URL
	EndFrame     string `json:"endFrame"`     // End keyframe URL
	DurationSecs int    `json:"durationSecs"` // Duration in seconds
}

// KeyframeSubdivisionResult holds the result of keyframe subdivision video generation.
type KeyframeSubdivisionResult struct {
	Segments      []VideoSegment `json:"segments"`      // Video segments in order
	MiddleFrames  []string       `json:"middleFrames"`  // Generated middle frame URLs
	TotalDuration int            `json:"totalDuration"` // Total duration in seconds
	IsSubdivided  bool           `json:"isSubdivided"`  // Whether subdivision was applied
}

// FrameGapEvaluation holds the VLM evaluation result for frame gap assessment.
type FrameGapEvaluation struct {
	Feasible     bool   `json:"feasible"`     // Whether the transition is feasible in one clip
	Reason       string `json:"reason"`       // Reason for the evaluation
	MiddleAction string `json:"middleAction"` // Suggested middle action if not feasible
}

// SubdivisionVideoRequest represents a request for video generation with automatic subdivision.
type SubdivisionVideoRequest struct {
	FirstFrameURL   string                 `json:"firstFrameUrl"`   // Start keyframe URL
	LastFrameURL    string                 `json:"lastFrameUrl"`    // End keyframe URL
	Prompt          string                 `json:"prompt"`          // Motion/action prompt
	DurationSeconds int                    `json:"durationSeconds"` // Target duration per segment
	MaxDepth        int                    `json:"maxDepth"`        // Maximum recursion depth (default 3)
	Provider        string                 `json:"provider"`        // Video provider name
	AspectRatio     string                 `json:"aspectRatio"`     // Aspect ratio
	Metadata        map[string]interface{} `json:"metadata"`        // Additional metadata
}

// MiddleFrameRequest represents a request to generate a middle frame image.
type MiddleFrameRequest struct {
	StartFrameURL string `json:"startFrameUrl"` // Reference start frame
	EndFrameURL   string `json:"endFrameUrl"`   // Reference end frame (optional)
	MiddleAction  string `json:"middleAction"`  // Description of the middle state
	Provider      string `json:"provider"`      // Image provider name
	AspectRatio   string `json:"aspectRatio"`   // Aspect ratio
}
