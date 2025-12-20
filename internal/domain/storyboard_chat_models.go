package domain

import "encoding/json"

// StoryboardChatMessageType enum constants
const (
	MsgTypeCharacterSelection = "character_selection"
	MsgTypeStoryContent       = "story_content"
	MsgTypeImagePrompts       = "image_prompts"
	MsgTypeImagesResult       = "images_result"
	MsgTypeVideoPrompt        = "video_prompt"
	MsgTypeVideoProcessing    = "video_processing"
	MsgTypeVideoResult        = "video_result"
	MsgTypeCompletion         = "completion"
	MsgTypeError              = "error"
	MsgTypeUserInput          = "user_input"
	MsgTypeProcessing         = "processing"
)

// StoryboardChatSessionStatus constants
const (
	SessionStatusActive    = "active"
	SessionStatusCompleted = "completed"
	SessionStatusCancelled = "cancelled"
	SessionStatusError     = "error"
)

// StoryboardChatStep constants
const (
	StepCharacterSelection = 1
	StepContentGeneration  = 2
	StepImagePrompts       = 3
	StepImageGeneration    = 4
	StepVideoChoice        = 5
	StepVideoGeneration    = 6
	StepCompletion         = 7
)

// StoryboardChatSession tracks a storyboard creation chat session
type StoryboardChatSession struct {
	ID                 string   `json:"id"`
	UserID             string   `json:"userId"`
	StoryID            string   `json:"storyId"`
	StoryboardID       string   `json:"storyboardId,omitempty"`
	CurrentStep        int      `json:"currentStep"`
	Status             string   `json:"status"`
	SelectedCharacters []string `json:"selectedCharacters,omitempty"`
	SelectedStyle      string   `json:"selectedStyle,omitempty"`
	CreatedAt          int64    `json:"createdAt"`
	UpdatedAt          int64    `json:"updatedAt"`

	// Relations
	Story      *Story      `json:"story,omitempty"`
	Storyboard *Storyboard `json:"storyboard,omitempty"`
}

// StoryboardChatMessage represents a message in the storyboard creation chat
type StoryboardChatMessage struct {
	ID          string          `json:"id"`
	SessionID   string          `json:"sessionId"`
	MessageType string          `json:"messageType"`
	Status      string          `json:"status"`
	Step        int             `json:"step"`
	Data        json.RawMessage `json:"data"`
	Actions     []ChatAction    `json:"actions,omitempty"`
	Timestamp   int64           `json:"timestamp"`
	IsUser      bool            `json:"isUser"`
	Content     string          `json:"content,omitempty"` // Text content for user messages
}

// ChatAction represents an available action button for the user
type ChatAction struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Type     string `json:"type"` // primary, secondary, danger
	Disabled bool   `json:"disabled,omitempty"`
}

// ========== Message Payload Types ==========

// CharacterSelectionData payload for character selection step
type CharacterSelectionData struct {
	Prompt       string                    `json:"prompt"`
	Characters   []CharacterSelectionItem  `json:"characters"`
	MinSelection int                       `json:"minSelection"`
	MaxSelection int                       `json:"maxSelection"`
	Styles       []StyleOption             `json:"styles,omitempty"`
}

// CharacterSelectionItem represents a character in the selection list
type CharacterSelectionItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Avatar      string `json:"avatar"`
	Role        string `json:"role,omitempty"`
	Description string `json:"description,omitempty"`
}

// StyleOption represents an available style for image generation
type StyleOption struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	PreviewURL  string `json:"previewUrl,omitempty"`
}

// StoryContentData payload for story content step
type StoryContentData struct {
	Title    string               `json:"title"`
	Content  string               `json:"content"`
	Scenes   []SceneContentItem   `json:"scenes"`
	Editable bool                 `json:"editable"`
}

// SceneContentItem represents a scene in the story content
type SceneContentItem struct {
	ID          string   `json:"id"`
	Sequence    int      `json:"sequence"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Location    string   `json:"location,omitempty"`
	TimeOfDay   string   `json:"timeOfDay,omitempty"`
	Characters  []string `json:"characters,omitempty"`
	Mood        string   `json:"mood,omitempty"`
}

// ImagePromptsData payload for image prompts step
type ImagePromptsData struct {
	Scenes   []ScenePromptItem `json:"scenes"`
	Editable bool              `json:"editable"`
	Style    string            `json:"style,omitempty"`
}

// ScenePromptItem represents a scene with its image prompt
type ScenePromptItem struct {
	ID          string `json:"id"`
	Sequence    int    `json:"sequence"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Prompt      string `json:"prompt"`
	Editable    bool   `json:"editable"`
}

// ImagesResultData payload for images result step
type ImagesResultData struct {
	StoryboardID string                   `json:"storyboardId"`
	Title        string                   `json:"title"`
	Style        string                   `json:"style"`
	Scenes       []SceneImageResultItem   `json:"scenes"`
	Characters   []CharacterSelectionItem `json:"characters"`
}

// SceneImageResultItem represents a scene with its generated image
type SceneImageResultItem struct {
	ID          string `json:"id"`
	Sequence    int    `json:"sequence"`
	Title       string `json:"title"`
	Description string `json:"description"`
	ImageURL    string `json:"imageUrl"`
	Prompt      string `json:"prompt"`
}

// VideoPromptData payload for video generation choice
type VideoPromptData struct {
	Message     string                 `json:"message"`
	Scenes      []SceneImageResultItem `json:"scenes"`
	EstimatedTime string               `json:"estimatedTime,omitempty"`
}

// VideoProcessingData payload for video processing status
type VideoProcessingData struct {
	Message        string             `json:"message"`
	Progress       int                `json:"progress"` // 0-100
	CurrentScene   int                `json:"currentScene,omitempty"`
	TotalScenes    int                `json:"totalScenes,omitempty"`
	EstimatedTime  string             `json:"estimatedTime,omitempty"`
	SceneStatuses  []SceneVideoStatus `json:"sceneStatuses,omitempty"`
}

// SceneVideoStatus represents video generation status for a scene
type SceneVideoStatus struct {
	ID       string `json:"id"`
	Sequence int    `json:"sequence"`
	Title    string `json:"title"`
	Status   string `json:"status"` // pending, processing, completed, failed
	VideoURL string `json:"videoUrl,omitempty"`
}

// VideoResultData payload for video generation result
type VideoResultData struct {
	StoryboardID string                   `json:"storyboardId"`
	Title        string                   `json:"title"`
	Scenes       []SceneVideoResultItem   `json:"scenes"`
	Characters   []CharacterSelectionItem `json:"characters"`
}

// SceneVideoResultItem represents a scene with its generated video
type SceneVideoResultItem struct {
	ID          string `json:"id"`
	Sequence    int    `json:"sequence"`
	Title       string `json:"title"`
	Description string `json:"description"`
	ImageURL    string `json:"imageUrl"`
	VideoURL    string `json:"videoUrl"`
}

// CompletionData payload for completion step
type CompletionData struct {
	StoryboardID   string                   `json:"storyboardId"`
	Title          string                   `json:"title"`
	Content        string                   `json:"content"`
	CoverImage     string                   `json:"coverImage,omitempty"`
	Scenes         []SceneCompletionItem    `json:"scenes"`
	Characters     []CharacterSelectionItem `json:"characters"`
	WorkflowStatus string                   `json:"workflowStatus"`
	HasVideo       bool                     `json:"hasVideo"`
}

// SceneCompletionItem represents a scene in the completion view
type SceneCompletionItem struct {
	ID          string `json:"id"`
	Sequence    int    `json:"sequence"`
	Title       string `json:"title"`
	Description string `json:"description"`
	ImageURL    string `json:"imageUrl,omitempty"`
	VideoURL    string `json:"videoUrl,omitempty"`
}

// ErrorData payload for error messages
type ErrorData struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	Details    string `json:"details,omitempty"`
	Retryable  bool   `json:"retryable"`
	RetryAfter int    `json:"retryAfter,omitempty"` // seconds
}

// ProcessingData payload for processing status messages
type ProcessingData struct {
	Message       string `json:"message"`
	Progress      int    `json:"progress,omitempty"` // 0-100
	EstimatedTime string `json:"estimatedTime,omitempty"`
}

// ========== Request Types ==========

// StartSessionRequest request to start a new storyboard chat session
type StartSessionRequest struct {
	StoryID string `json:"storyId"`
}

// SendMessageRequest request to send a message in the storyboard chat
type SendMessageRequest struct {
	ActionID string          `json:"actionId,omitempty"` // Action button clicked
	Content  string          `json:"content,omitempty"`  // Text content (for user input)
	Data     json.RawMessage `json:"data,omitempty"`     // Action-specific data
}

// CharacterSelectionInput user's character selection input
type CharacterSelectionInput struct {
	CharacterIDs []string `json:"characterIds"`
	StyleID      string   `json:"styleId,omitempty"`
}

// ContentConfirmationInput user's content confirmation/edit input
type ContentConfirmationInput struct {
	Title   string             `json:"title,omitempty"`
	Content string             `json:"content,omitempty"`
	Scenes  []SceneContentItem `json:"scenes,omitempty"`
	Edited  bool               `json:"edited"`
}

// ImagePromptConfirmationInput user's image prompt confirmation/edit input
type ImagePromptConfirmationInput struct {
	Scenes []ScenePromptItem `json:"scenes,omitempty"`
	Edited bool              `json:"edited"`
}

// VideoChoiceInput user's video generation choice
type VideoChoiceInput struct {
	GenerateVideo bool     `json:"generateVideo"`
	SceneIDs      []string `json:"sceneIds,omitempty"` // Specific scenes to generate video for
}

// PublishChoiceInput user's publish/draft choice
type PublishChoiceInput struct {
	Publish bool   `json:"publish"`
	Title   string `json:"title,omitempty"` // Optional title update
}

