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

// StoryboardImageGeneration - Step 3: Scene + refs → image
// Records the AI generation of images from scene descriptions and reference images
type StoryboardImageGeneration struct {
	ID                string   `json:"id"`
	StoryboardID      string   `json:"storyboardId"`
	SceneID           string   `json:"sceneId"`           // Reference to StoryScene
	SceneTitle        string   `json:"sceneTitle"`        // Snapshot of scene title
	SceneDescription  string   `json:"sceneDescription"`  // Snapshot of scene description
	ReferenceImages   []string `json:"referenceImages"`   // Character/scene reference image URLs
	GeneratedPrompt   string   `json:"generatedPrompt"`   // AI-generated image prompt
	GeneratedImageURL string   `json:"generatedImageUrl"` // Final generated image URL
	ImageWidth        int      `json:"imageWidth,omitempty"`
	ImageHeight       int      `json:"imageHeight,omitempty"`
	Status            string   `json:"status"` // pending, processing, completed, failed
	InputTokens       int      `json:"inputTokens"`
	OutputTokens      int      `json:"outputTokens"`
	TotalTokens       int      `json:"totalTokens"`
	ErrorMessage      string   `json:"errorMessage,omitempty"`
	CreatedAt         int64    `json:"createdAt"`
	CompletedAt       *int64   `json:"completedAt,omitempty"`

	// Relations
	Storyboard *Storyboard `json:"storyboard,omitempty"`
}

// StoryboardVideoGeneration - Step 4: Scene → video
// Records the AI generation of videos from scene descriptions
type StoryboardVideoGeneration struct {
	ID                string `json:"id"`
	StoryboardID      string `json:"storyboardId"`
	SceneID           string `json:"sceneId"`                  // Reference to StoryScene
	SceneTitle        string `json:"sceneTitle"`               // Snapshot of scene title
	InputDescription  string `json:"inputDescription"`         // Scene description for video
	ReferenceImageURL string `json:"referenceImageUrl"`        // Start keyframe image (reference image for video generation)
	EndFrameURL       string `json:"endFrameUrl"`              // End keyframe image for video transitions
	GeneratedPrompt   string `json:"generatedPrompt"`          // AI-generated video prompt
	GeneratedVideoURL string `json:"generatedVideoUrl"`        // Final generated video URL
	ProviderTaskID    string `json:"providerTaskId,omitempty"` // Provider's task ID for async video generation (for recovery)
	ProviderName      string `json:"providerName,omitempty"`   // Provider name (huoshan, hailuo, etc.) for recovery
	Duration          int    `json:"duration"`                 // Video duration in seconds
	Status            string `json:"status"`                   // pending, processing, completed, failed
	InputTokens       int    `json:"inputTokens"`
	OutputTokens      int    `json:"outputTokens"`
	TotalTokens       int    `json:"totalTokens"`
	ErrorMessage      string `json:"errorMessage,omitempty"`
	CreatedAt         int64  `json:"createdAt"`
	CompletedAt       *int64 `json:"completedAt,omitempty"`

	// Relations
	Storyboard *Storyboard `json:"storyboard,omitempty"`
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
}
