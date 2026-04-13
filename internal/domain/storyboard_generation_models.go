package domain

// Generation status constants
const (
	GenerationStatusPending    = "pending"
	GenerationStatusProcessing = "processing"
	GenerationStatusCompleted  = "completed"
	GenerationStatusFailed     = "failed"
)

// Workflow status constants
const (
	WorkflowStatusDraft        = "draft"
	WorkflowStatusContentReady = "content_ready"
	WorkflowStatusImagesReady  = "images_ready"
	WorkflowStatusVideoReady   = "video_ready"
	WorkflowStatusPublished    = "published"
)

// StoryboardContentGeneration - Step 1: Raw input → AI content
// Records the AI generation of narrative content from user's raw input
type StoryboardContentGeneration struct {
	ID               string   `json:"id"`
	StoryboardID     string   `json:"storyboardId"`
	RawInput         string   `json:"rawInput"`         // User's original story input
	CharacterIDs     []string `json:"characterIds"`     // Selected character references
	SceneIDs         []string `json:"sceneIds"`         // Selected scene references
	Style            string   `json:"style"`            // action, drama, comedy, mystery
	GeneratedContent string   `json:"generatedContent"` // AI-generated narrative
	Status           string   `json:"status"`           // pending, processing, completed, failed
	InputTokens      int      `json:"inputTokens"`
	OutputTokens     int      `json:"outputTokens"`
	TotalTokens      int      `json:"totalTokens"`
	ErrorMessage     string   `json:"errorMessage,omitempty"`
	CreatedAt        int64    `json:"createdAt"`
	CompletedAt      *int64   `json:"completedAt,omitempty"`

	// Relations
	Storyboard *Storyboard `json:"storyboard,omitempty"`
}

// StoryboardSceneGeneration - Step 2: Scene → detailed description
// Records the AI enhancement of scene descriptions
type StoryboardSceneGeneration struct {
	ID               string `json:"id"`
	StoryboardID     string `json:"storyboardId"`
	SceneID          string `json:"sceneId"`          // Reference to StoryScene
	SceneTitle       string `json:"sceneTitle"`       // Snapshot of scene title
	SceneLocation    string `json:"sceneLocation"`    // Snapshot of scene location
	InputDescription string `json:"inputDescription"` // Original scene description
	GeneratedDetail  string `json:"generatedDetail"`  // AI-enhanced description
	Status           string `json:"status"`           // pending, processing, completed, failed
	InputTokens      int    `json:"inputTokens"`
	OutputTokens     int    `json:"outputTokens"`
	TotalTokens      int    `json:"totalTokens"`
	ErrorMessage     string `json:"errorMessage,omitempty"`
	CreatedAt        int64  `json:"createdAt"`
	CompletedAt      *int64 `json:"completedAt,omitempty"`

	// Relations
	Storyboard *Storyboard `json:"storyboard,omitempty"`
}

// ImagePromptDetails 结构化的图片生成提示词详情
type ImagePromptDetails struct {
	ArtStyle        string   `json:"artStyle"`                  // 艺术风格和媒介，如 "digital painting", "watercolor", "photorealistic"
	Lighting        string   `json:"lighting"`                  // 光照和氛围，如 "soft morning light", "dramatic shadows"
	ColorPalette    string   `json:"colorPalette"`              // 色彩 palette，如 "warm tones", "cool blues and grays"
	Composition     string   `json:"composition"`               // 构图细节，如 "wide angle", "close-up", "rule of thirds"
	KeyElements     []string `json:"keyElements"`               // 关键视觉元素列表
	Mood            string   `json:"mood"`                      // 情绪氛围，如 "peaceful", "tense", "mysterious"
	AdditionalNotes string   `json:"additionalNotes,omitempty"` // 其他补充说明
}

// StoryboardImageGeneration - Step 3: Scene + refs → image
// Records the AI generation of images from scene descriptions and reference images
type StoryboardImageGeneration struct {
	ID                string              `json:"id"`
	StoryboardID      string              `json:"storyboardId"`
	SceneID           string              `json:"sceneId"`                 // Reference to StoryScene
	SceneTitle        string              `json:"sceneTitle"`              // Snapshot of scene title
	SceneDescription  string              `json:"sceneDescription"`        // Snapshot of scene description
	ReferenceImages   []string            `json:"referenceImages"`         // Character/scene reference image URLs
	GeneratedPrompt   string              `json:"generatedPrompt"`         // AI-generated image prompt (final combined text)
	PromptDetails     *ImagePromptDetails `json:"promptDetails,omitempty"` // Structured prompt details for client editing
	GeneratedImageURL string              `json:"generatedImageUrl"`       // Final generated image URL
	ImageWidth        int                 `json:"imageWidth,omitempty"`
	ImageHeight       int                 `json:"imageHeight,omitempty"`
	Status            string              `json:"status"` // pending, processing, completed, failed
	InputTokens       int                 `json:"inputTokens"`
	OutputTokens      int                 `json:"outputTokens"`
	TotalTokens       int                 `json:"totalTokens"`
	ErrorMessage      string              `json:"errorMessage,omitempty"`
	CreatedAt         int64               `json:"createdAt"`
	CompletedAt       *int64              `json:"completedAt,omitempty"`

	// 场景角色和风格配置（用于 AI 生成时的上下文）
	SceneCharacters          []string     `json:"sceneCharacters,omitempty"`          // 场景中的角色名称列表
	CharacterReferenceImages []string     `json:"characterReferenceImages,omitempty"` // 角色参考图片 URL 列表
	StoryStyle               *StyleConfig `json:"storyStyle,omitempty"`               // 故事的风格配置
	IsTransitionScene        bool         `json:"isTransitionScene"`                  // 是否为过渡场景（无角色出现）
	// ComicStyle 续写或请求指定的漫画/视觉风格 slug（不入库列，仅随当次生成记录传递）
	ComicStyle string `json:"comicStyle,omitempty"`

	// Relations
	Storyboard *Storyboard `json:"storyboard,omitempty"`
}

// VideoPromptDetails 结构化的视频生成提示词详情
type VideoPromptDetails struct {
	CameraMovement  string   `json:"cameraMovement"`            // 摄像机运动，如 "slow pan left", "zoom in", "static"
	SubjectMotion   string   `json:"subjectMotion"`             // 主体动作，如 "walking slowly", "turning around"
	Atmosphere      string   `json:"atmosphere"`                // 氛围，如 "peaceful morning", "tense night"
	TransitionStyle string   `json:"transitionStyle"`           // 转场风格，如 "fade", "cut", "crossfade"
	Duration        string   `json:"duration"`                  // 时长描述，如 "5 seconds"
	KeyMoments      []string `json:"keyMoments"`                // 关键时刻列表
	AdditionalNotes string   `json:"additionalNotes,omitempty"` // 其他补充说明
}

// StoryboardVideoGeneration - Step 4: Scene → video
// Records the AI generation of videos from scene descriptions
type StoryboardVideoGeneration struct {
	ID                string              `json:"id"`
	StoryboardID      string              `json:"storyboardId"`
	SceneID           string              `json:"sceneId"`                  // Reference to StoryScene
	SceneTitle        string              `json:"sceneTitle"`               // Snapshot of scene title
	InputDescription  string              `json:"inputDescription"`         // Scene description for video
	ReferenceImageURL string              `json:"referenceImageUrl"`        // Start keyframe image (reference image for video generation)
	EndFrameURL       string              `json:"endFrameUrl"`              // End keyframe image for video transitions
	GeneratedPrompt   string              `json:"generatedPrompt"`          // AI-generated video prompt (final combined text)
	PromptDetails     *VideoPromptDetails `json:"promptDetails,omitempty"`  // Structured prompt details for client editing
	GeneratedVideoURL string              `json:"generatedVideoUrl"`        // Final generated video URL
	ProviderTaskID    string              `json:"providerTaskId,omitempty"` // Provider's task ID for async video generation (for recovery)
	ProviderName      string              `json:"providerName,omitempty"`   // Provider name (huoshan, hailuo, etc.) for recovery
	Duration          int                 `json:"duration"`                 // Video duration in seconds
	Status            string              `json:"status"`                   // pending, processing, completed, failed
	InputTokens       int                 `json:"inputTokens"`
	OutputTokens      int                 `json:"outputTokens"`
	TotalTokens       int                 `json:"totalTokens"`
	ErrorMessage      string              `json:"errorMessage,omitempty"`
	CreatedAt         int64               `json:"createdAt"`
	CompletedAt       *int64              `json:"completedAt,omitempty"`

	// Keyframe Subdivision fields
	IsSubdivided        bool               `json:"isSubdivided"`              // Whether keyframe subdivision was applied
	VideoSegmentsJSON   string             `json:"-"`                         // JSON storage for video segments
	MiddleFrameURLsJSON string             `json:"-"`                         // JSON storage for middle frame URLs
	VideoSegments       []VideoSegmentInfo `json:"videoSegments,omitempty"`   // Parsed video segments
	MiddleFrameURLs     []string           `json:"middleFrameUrls,omitempty"` // Parsed middle frame URLs

	// Relations
	Storyboard *Storyboard `json:"storyboard,omitempty"`
}

// Generation pipeline (wizard-oriented): coarse phases for client UI.
const (
	PipelinePhaseContent = "content"
	PipelinePhaseScenes  = "scenes"
	PipelinePhaseImages  = "images"

	PipelineStepPending   = "pending"
	PipelineStepRunning   = "running"
	PipelineStepCompleted = "completed"
	PipelineStepFailed    = "failed"
	PipelineStepSkipped   = "skipped"

	SuggestedResumeNone = "none"
	SuggestedResumeRetryFailedImages  = "retry_failed_images"
	SuggestedResumeRegenerateContent  = "regenerate_content"
	SuggestedResumeRegenerateScenes   = "regenerate_scenes"
)

// GenerationPipelineSceneItem is per-scene status within a pipeline step (e.g. image failures).
type GenerationPipelineSceneItem struct {
	SceneID      string `json:"sceneId"`
	SceneTitle   string `json:"sceneTitle,omitempty"`
	Status       string `json:"status"` // pending, running, completed, failed
	ErrorMessage string `json:"errorMessage,omitempty"`
}

// GenerationPipelineStep is one wizard-visible stage derived from DB state (no extra tables).
type GenerationPipelineStep struct {
	Phase        string                        `json:"phase"` // content | scenes | images
	Status       string                        `json:"status"`
	Order        int                           `json:"order"`
	Title        string                        `json:"title"`
	Summary      string                        `json:"summary,omitempty"`
	ErrorMessage string                        `json:"errorMessage,omitempty"`
	SceneItems   []GenerationPipelineSceneItem `json:"sceneItems,omitempty"`
}

// StoryboardGenerationProgress aggregates all generation records for a storyboard
type StoryboardGenerationProgress struct {
	StoryboardID      string                       `json:"storyboardId"`
	WorkflowStatus    string                       `json:"workflowStatus"`
	CurrentStep       int                          `json:"currentStep"`
	TotalTokens       int                          `json:"totalTokens"`
	IsGenerating      bool                         `json:"isGenerating"`      // True if any generation is in progress
	HasPendingTasks   bool                         `json:"hasPendingTasks"`   // True if there are pending tasks
	GenerationMessage string                       `json:"generationMessage"` // Human-readable status message
	ContentGeneration *StoryboardContentGeneration `json:"contentGeneration,omitempty"`
	SceneGenerations  []*StoryboardSceneGeneration `json:"sceneGenerations,omitempty"`
	ImageGenerations  []*StoryboardImageGeneration `json:"imageGenerations,omitempty"`
	VideoGenerations  []*StoryboardVideoGeneration `json:"videoGenerations,omitempty"`
	// PipelineSteps 面向客户端「生成步骤详情」；由服务端从现有记录派生。
	PipelineSteps []GenerationPipelineStep `json:"pipelineSteps,omitempty"`
	// SuggestedResumeAction 建议的下一步恢复动作（与现有 POST 接口对齐）。
	SuggestedResumeAction string `json:"suggestedResumeAction,omitempty"`
}
