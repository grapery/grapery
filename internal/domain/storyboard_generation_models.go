package domain

// Generation status constants
const (
	GenerationStatusPending    = "pending"
	GenerationStatusProcessing = "processing"
	GenerationStatusCompleted  = "completed"
	GenerationStatusFailed     = "failed"
)

// Storyboard image pipeline kind (persisted on storyboard_image_generations.pipeline_kind).
const (
	StoryboardImagePipelineScene     = "scene"
	StoryboardImagePipelineComicPage = "comic_page"
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

// StoryboardComicText 描述单格漫画文字层（对白气泡、思想泡、拟声/语气词、旁白框）。
// 与 FragmentComicText 同构，便于 image prompt 生成器将其翻译为英文视觉指令直接绘入图片。
type StoryboardComicText struct {
	// Type: narration | dialogue | thought | sfx
	//   narration  — 旁白框（方形，通常置于格顶/底，叙述时间/地点或第三人称视角）
	//   dialogue   — 对白气泡（圆形/椭圆，尾巴指向说话角色）
	//   thought    — 内心独白气泡（云朵形/虚线椭圆，尾巴为小圆泡）
	//   sfx        — 拟声词/语气词（夸张字体，如「砰！」「啊？」「……」，无固定气泡形状）
	Type     string `json:"type"`
	Text     string `json:"text"`               // 实际要绘入图中的精确短句（建议 ≤12 汉字）
	Speaker  string `json:"speaker,omitempty"`  // 气泡归属角色（dialogue/thought 时填写，其余可留空）
	Position string `json:"position,omitempty"` // 建议排版区：top-left / top-right / bottom-left / bottom-right / mid-frame / speech-bubble / thought-bubble
	// PanelIndex 仅漫画页管线使用：零基 panel 索引。单格故事板文字层可留空。
	PanelIndex *int `json:"panelIndex,omitempty"`
}

// ImagePromptDetails 结构化的图片生成提示词详情
type ImagePromptDetails struct {
	ArtStyle        string                `json:"artStyle"`                  // 艺术风格和媒介，如 "digital painting", "watercolor", "photorealistic"
	Lighting        string                `json:"lighting"`                  // 光照和氛围，如 "soft morning light", "dramatic shadows"
	ColorPalette    string                `json:"colorPalette"`              // 色彩 palette，如 "warm tones", "cool blues and grays"
	Composition     string                `json:"composition"`               // 构图细节，如 "wide angle", "close-up", "rule of thirds"
	KeyElements     []string              `json:"keyElements"`               // 关键视觉元素列表
	Mood            string                `json:"mood"`                      // 情绪氛围，如 "peaceful", "tense", "mysterious"
	AdditionalNotes string                `json:"additionalNotes,omitempty"` // 其他补充说明（微表情、大气特效等）
	ComicTexts      []StoryboardComicText `json:"comicTexts,omitempty"`      // 漫画文字层：对白/思想泡/拟声/旁白；图片模型须直接绘入
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
	// PipelineKind scene vs comic_page（单图插画 vs 多格漫画页）；空为历史数据。
	PipelineKind string `json:"pipelineKind,omitempty"`
	// SkipPeerFailureGate 用户主动重试时跳过「兄弟分镜已失败则本任务放弃」闸门（仅内存传递，不入库）。
	SkipPeerFailureGate bool `json:"-"`

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
	PipelinePhaseAudit   = "audit"

	PipelineStepPending   = "pending"
	PipelineStepRunning   = "running"
	PipelineStepCompleted = "completed"
	PipelineStepFailed    = "failed"
	PipelineStepSkipped   = "skipped"

	SuggestedResumeNone              = "none"
	SuggestedResumeRetryFailedImages = "retry_failed_images"
	SuggestedResumeRegenerateContent = "regenerate_content"
	SuggestedResumeRegenerateScenes  = "regenerate_scenes"
)

const (
	StoryboardGenerationStepContext     = "context"
	StoryboardGenerationStepBiblePlan   = "bible_plan"
	StoryboardGenerationStepScenePlan   = "scene_plan"
	StoryboardGenerationStepImage       = "image"
	StoryboardGenerationStepConsistency = "consistency_audit"

	StoryboardGenerationAssetCharacterTurnaround = "character_turnaround"
	StoryboardGenerationAssetCharacterAnchor     = "character_anchor"
	StoryboardGenerationAssetLocationAnchor      = "location_anchor"
	StoryboardGenerationAssetPropAnchor          = "prop_anchor"
	StoryboardGenerationAssetPreviousScene       = "previous_scene"
	StoryboardGenerationAssetParentTailScene     = "parent_tail_scene"

	PromptKindContext           = "context_prompt"
	PromptKindBiblePlanSystem   = "bible_plan_system_prompt"
	PromptKindBiblePlanUser     = "bible_plan_user_prompt"
	PromptKindSceneWriterSystem = "scene_writer_system_prompt"
	PromptKindSceneWriterUser   = "scene_writer_user_prompt"
	PromptKindImagePrompt       = "image_prompt"
	PromptKindConsistencyAudit  = "consistency_audit_prompt"
)

// StoryboardGenerationRun is the durable envelope for one storyboard AI pipeline execution.
type StoryboardGenerationRun struct {
	ID                    string                       `json:"id"`
	StoryboardID          string                       `json:"storyboardId"`
	StoryID               string                       `json:"storyId"`
	UserID                string                       `json:"userId"`
	Status                string                       `json:"status"`
	Progress              int                          `json:"progress"`
	CurrentStep           string                       `json:"currentStep"`
	RequestJSON           string                       `json:"requestJson,omitempty"`
	ContextSnapshotJSON   string                       `json:"contextSnapshotJson,omitempty"`
	AlignmentSnapshotJSON string                       `json:"alignmentSnapshotJson,omitempty"`
	StoryboardBibleJSON   string                       `json:"storyboardBibleJson,omitempty"`
	BeatsJSON             string                       `json:"beatsJson,omitempty"`
	ScenePlanJSON         string                       `json:"scenePlanJson,omitempty"`
	ConsistencyIssuesJSON string                       `json:"consistencyIssuesJson,omitempty"`
	MetricsJSON           string                       `json:"metricsJson,omitempty"`
	ErrorCode             string                       `json:"errorCode,omitempty"`
	ErrorMessage          string                       `json:"errorMessage,omitempty"`
	CreatedAt             int64                        `json:"createdAt"`
	UpdatedAt             int64                        `json:"updatedAt"`
	CompletedAt           *int64                       `json:"completedAt,omitempty"`
	Assets                []*StoryboardGenerationAsset `json:"assets,omitempty"`
	PromptAuditRecords    []*AIPromptAuditRecord       `json:"promptAuditRecords,omitempty"`
}

type StoryboardGenerationAsset struct {
	ID           string `json:"id"`
	RunID        string `json:"runId"`
	Kind         string `json:"kind"`
	AssetKey     string `json:"assetKey"`
	EntityID     string `json:"entityId,omitempty"`
	ImageURL     string `json:"imageUrl"`
	Source       string `json:"source,omitempty"`
	MetadataJSON string `json:"metadataJson,omitempty"`
	CreatedAt    int64  `json:"createdAt"`
}

type AIPromptAuditRecord struct {
	ID                    string   `json:"id"`
	RunID                 string   `json:"runId,omitempty"`
	RelatedEntityType     string   `json:"relatedEntityType,omitempty"`
	RelatedEntityID       string   `json:"relatedEntityId,omitempty"`
	Step                  string   `json:"step"`
	PromptKind            string   `json:"promptKind"`
	PromptTemplateVersion string   `json:"promptTemplateVersion"`
	AlignmentSnapshotHash string   `json:"alignmentSnapshotHash,omitempty"`
	FullPromptHash        string   `json:"fullPromptHash,omitempty"`
	Provider              string   `json:"provider,omitempty"`
	Model                 string   `json:"model,omitempty"`
	Temperature           float64  `json:"temperature,omitempty"`
	MaxTokens             int      `json:"maxTokens,omitempty"`
	SystemPrompt          string   `json:"systemPrompt,omitempty"`
	UserPrompt            string   `json:"userPrompt,omitempty"`
	AlignmentPrompt       string   `json:"alignmentPrompt,omitempty"`
	ReferencePreamble     string   `json:"referencePreamble,omitempty"`
	FinalPrompt           string   `json:"finalPrompt,omitempty"`
	ReferenceImageURLs    []string `json:"referenceImageUrls,omitempty"`
	Output                string   `json:"output,omitempty"`
	TokenUsageJSON        string   `json:"tokenUsageJson,omitempty"`
	MetadataJSON          string   `json:"metadataJson,omitempty"`
	CreatedAt             int64    `json:"createdAt"`
}

type StoryboardVisualStyleBible struct {
	ArtStyle      string `json:"artStyle,omitempty"`
	LineQuality   string `json:"lineQuality,omitempty"`
	Palette       string `json:"palette,omitempty"`
	LightingMood  string `json:"lightingMood,omitempty"`
	CameraGrammar string `json:"cameraGrammar,omitempty"`
}

type StoryboardVisualCharacter struct {
	Key                  string               `json:"key"`
	ID                   string               `json:"id,omitempty"`
	Name                 string               `json:"name,omitempty"`
	NarrativeRole        string               `json:"narrativeRole,omitempty"`
	ImmutableTraits      []string             `json:"immutableTraits,omitempty"`
	CurrentState         string               `json:"currentState,omitempty"`
	TurnaroundAssetKeys  []string             `json:"turnaroundAssetKeys,omitempty"`
	TurnaroundImageURLs  *CharacterThreeViews `json:"turnaroundImageUrls,omitempty"`
	PrimaryIdentityImage string               `json:"primaryIdentityImage,omitempty"`
}

type StoryboardVisualLocation struct {
	Key             string   `json:"key"`
	ID              string   `json:"id,omitempty"`
	Name            string   `json:"name,omitempty"`
	ImmutableTraits []string `json:"immutableTraits,omitempty"`
	CurrentState    string   `json:"currentState,omitempty"`
}

type StoryboardVisualProp struct {
	Key             string   `json:"key"`
	Name            string   `json:"name,omitempty"`
	ImmutableTraits []string `json:"immutableTraits,omitempty"`
	CurrentState    string   `json:"currentState,omitempty"`
}

type StoryboardVisualBible struct {
	StyleBible      *StoryboardVisualStyleBible `json:"styleBible,omitempty"`
	Characters      []StoryboardVisualCharacter `json:"characters,omitempty"`
	Locations       []StoryboardVisualLocation  `json:"locations,omitempty"`
	Props           []StoryboardVisualProp      `json:"props,omitempty"`
	ContinuityRules []string                    `json:"continuityRules,omitempty"`
}

type StoryboardBeat struct {
	Index          int      `json:"index"`
	BeatID         string   `json:"beatId,omitempty"`
	Purpose        string   `json:"purpose"`
	Summary        string   `json:"summary"`
	ComicFunction  string   `json:"comicFunction,omitempty"` // establish | dialogue | action_impact | reaction | transition | atmosphere
	LayoutHint     string   `json:"layoutHint,omitempty"`    // short layout hint for scene writer
	Characters     []string `json:"characters,omitempty"`
	LocationKey    string   `json:"locationKey,omitempty"`
	ReferenceKeys  []string `json:"referenceKeys,omitempty"`
	ContinuityNote string   `json:"continuityNote,omitempty"`
}

type StoryboardBiblePlan struct {
	StoryboardBible StoryboardVisualBible `json:"storyboardBible"`
	Beats           []StoryboardBeat      `json:"beats"`
}

type StoryboardScenePlanItem struct {
	Sequence       int            `json:"sequence"`
	Title          string         `json:"title"`
	Description    string         `json:"description"`
	Location       string         `json:"location,omitempty"`
	TimeOfDay      string         `json:"timeOfDay,omitempty"`
	Characters     []string       `json:"characters,omitempty"`
	Mood           string         `json:"mood,omitempty"`
	ReferenceKeys  []string       `json:"referenceKeys,omitempty"`
	ContinuityNote string         `json:"continuityNote,omitempty"`
	BeatPurpose    string         `json:"beatPurpose,omitempty"`
	ImagePrompt    string         `json:"imagePrompt,omitempty"`
	VisualState    map[string]any `json:"visualState,omitempty"`
	// ComicTexts 本格漫画文字层（对白/思想泡/拟声/旁白）。
	// AI 规划时产出，随后透传至图片 prompt 让图片模型直接绘入，无需 App 侧叠加。
	ComicTexts []StoryboardComicText `json:"comicTexts,omitempty"`
	// 漫画版式规划（与碎片多格规划字段语义对齐）
	LayoutIntent    string `json:"layoutIntent,omitempty"`
	CompositionPlan string `json:"compositionPlan,omitempty"`
	ShotType        string `json:"shotType,omitempty"`
	VisualHierarchy string `json:"visualHierarchy,omitempty"`
	// PanelShape AI 根据剧情情绪为本格选定的裁切形状，与 StoryboardScene.PanelShape 保持一致。
	// 允许值：full | diagonal_left | diagonal_right |
	//         trapezoid_leading | trapezoid_trailing |
	//         triangle_tl | triangle_tr | triangle_bl | triangle_br |
	//         wide_panorama
	PanelShape string `json:"panelShape,omitempty"`
}

type StoryboardScenePlan struct {
	Content string                    `json:"content"`
	Scenes  []StoryboardScenePlanItem `json:"scenes"`
}

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

// StoryboardStructureGenerationResponse is returned by POST .../generate/structure.
// When scenes already exist the server returns the full storyboard synchronously (asyncAccepted=false).
// When scenes must be rebuilt, work runs in the background (asyncAccepted=true) and clients should poll GET .../generation-progress.
type StoryboardStructureGenerationResponse struct {
	AsyncAccepted        bool                         `json:"asyncAccepted"`
	Storyboard           *Storyboard                  `json:"storyboard,omitempty"`
	GenerationProgress   *StoryboardGenerationProgress `json:"generationProgress,omitempty"`
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
	SuggestedResumeAction string                   `json:"suggestedResumeAction,omitempty"`
	LatestRun             *StoryboardGenerationRun `json:"latestRun,omitempty"`
	ConsistencyIssuesJSON string                   `json:"consistencyIssuesJson,omitempty"`
	PromptAuditRecordIDs  []string                 `json:"promptAuditRecordIds,omitempty"`
}
