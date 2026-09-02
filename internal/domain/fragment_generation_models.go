package domain

import "strings"

const (
	// Fragment AI 任务类型
	AITaskGenerateFragment        AITaskType = "generate_fragment"
	AITaskGenerateFragmentContent AITaskType = "generate_fragment_content"
	AITaskGenerateFragmentImages  AITaskType = "generate_fragment_images"
)

// 碎片配图长宽比（与 Gemini / 客户端约定一致）
const (
	FragmentAspect1x1     = "1:1"
	FragmentAspect16x9    = "16:9"
	FragmentAspect9x16    = "9:16"
	FragmentAspect3x4     = "3:4"
	FragmentAspect4x3     = "4:3"
	FragmentAspectDefault = FragmentAspect16x9
	// FragmentComicPageAspectDefault 是故事碎片完整漫画页的默认画布；
	// 保留 FragmentAspectDefault 给历史单幅插图和其他复用调用方。
	FragmentComicPageAspectDefault = FragmentAspect3x4
)

// NormalizeFragmentAspectRatio 将输入规范为允许的比例；无法识别时返回空字符串。
func NormalizeFragmentAspectRatio(s string) string {
	switch strings.TrimSpace(s) {
	case FragmentAspect1x1, FragmentAspect16x9, FragmentAspect9x16, FragmentAspect3x4, FragmentAspect4x3:
		return strings.TrimSpace(s)
	default:
		return ""
	}
}

// FragmentImagePixelSizeForAspectRatio 将比例映射为火山等使用像素尺寸时的字符串（与角色立绘逻辑对齐）。
// 总像素须满足 Seedream 5.0（doubao-seedream-5-0）等组图接口下限（当前方舟返回至少 3686400 像素，约为 1920² 或 2560×1440）。
func FragmentImagePixelSizeForAspectRatio(ar string) string {
	switch NormalizeFragmentAspectRatio(ar) {
	case FragmentAspect1x1:
		return "1920x1920"
	case FragmentAspect16x9:
		return "2560x1440"
	case FragmentAspect9x16:
		return "1440x2560"
	case FragmentAspect4x3:
		return "2220x1665"
	case FragmentAspect3x4:
		return "1665x2220"
	default:
		return "2560x1440"
	}
}

// FragmentGenerationRequest 碎片故事生成请求
// 若将来增加统一 provider 字段，应仅作用于图片/视频；文案与规划仍由服务端按用户区域在 huoshan/gemini 间路由。
type FragmentGenerationRequest struct {
	UserInput  string   `json:"userInput"`  // 用户输入的描述
	ImageUrls  []string `json:"imageUrls"`  // 用户提供的参考图片（可选）
	ImageCount int      `json:"imageCount"` // 本次需要生成的图片数量 (1-8)；草稿可通过续写不断追加
	Style      string   `json:"style"`      // 风格：fantasy, realistic, anime, etc.
	Mood       string   `json:"mood"`       // 情绪：happy, sad, mysterious, etc.
	Length     string   `json:"length"`     // 内容长度：short, medium, long
	Language   string   `json:"language"`   // 语言：zh-Hans, en, ja
	Visibility string   `json:"visibility"` // 可见性：public, followers, private
	// AspectRatio 配图长宽比：1:1、16:9、9:16、3:4、4:3；空表示由多模态解析（有参考图时）或默认 16:9。
	AspectRatio             string                  `json:"aspectRatio,omitempty"`
	ConsistencyLevel        string                  `json:"consistencyLevel,omitempty"`       // off | standard | strong
	EnableReferenceAssets   *bool                   `json:"enableReferenceAssets,omitempty"`  // nil 时由 consistencyLevel 决定
	IncludeGenerationTrace  bool                    `json:"includeGenerationTrace,omitempty"` // 返回完整生成 trace
	ReferenceSlots          []FragmentReferenceSlot `json:"referenceSlots,omitempty"`         // 语义参考槽位（可选）
	TargetDraftFragmentID   string                  `json:"targetDraftFragmentId,omitempty"`  // 修改/续写时写回的现有草稿碎片
	ReplaceImageIndex       int                     `json:"replaceImageIndex,omitempty"`      // 单张重绘时替换的 1-based 图片位置
	ClientMessageID         string                  `json:"clientMessageId,omitempty"`        // 客户端消息幂等键，避免重复提交创建多个任务
	WorkflowReleaseID       string                  `json:"workflowReleaseId,omitempty"`
	WorkflowRunID           string                  `json:"workflowRunId,omitempty"`
	WorkflowSystemPrompt    string                  `json:"workflowSystemPrompt,omitempty"`
	WorkflowUserPrompt      string                  `json:"workflowUserPrompt,omitempty"`
	WorkflowModelConfig     map[string]any          `json:"workflowModelConfig,omitempty"`
	WorkflowOutputSchema    map[string]any          `json:"workflowOutputSchema,omitempty"`
	WorkflowPromptVersionID string                  `json:"workflowPromptVersionId,omitempty"`
}

// FragmentReferenceSlot 描述用户可为某个故事实体提供的语义参考图。
type FragmentReferenceSlot struct {
	Key        string `json:"key"`
	Label      string `json:"label"`
	Kind       string `json:"kind"` // character | prop | location | scene | object
	Required   bool   `json:"required,omitempty"`
	InputType  string `json:"inputType,omitempty"` // image
	ImageURL   string `json:"imageUrl,omitempty"`
	HelperText string `json:"helperText,omitempty"`
}

// FragmentGenerationIntent 是分析阶段返回给客户端/agent 的标准生成意图。
type FragmentGenerationIntent struct {
	UserInput   string `json:"userInput"`
	ImageCount  int    `json:"imageCount"`
	Style       string `json:"style"`
	Mood        string `json:"mood,omitempty"`
	Length      string `json:"length,omitempty"`
	Language    string `json:"language"`
	Visibility  string `json:"visibility"`
	AspectRatio string `json:"aspectRatio"`
	Topic       string `json:"topic,omitempty"`
}

// FragmentAnalyzeRequest 是第一阶段文本分析请求。
type FragmentAnalyzeRequest struct {
	UserInput             string `json:"userInput"`
	Language              string `json:"language,omitempty"`
	AspectRatio           string `json:"aspectRatio,omitempty"`
	ImageCount            int    `json:"imageCount,omitempty"`
	Style                 string `json:"style,omitempty"`
	TargetDraftFragmentID string `json:"targetDraftFragmentId,omitempty"`
	EditOperation         string `json:"editOperation,omitempty"`      // replace | append, supplied by an explicit UI action.
	SelectedImageIndex    int    `json:"selectedImageIndex,omitempty"` // 1-based target for replace.
}

// FragmentRecommendedOptions 是客户端展示配置建议。
type FragmentRecommendedOptions struct {
	StyleCandidates []string `json:"styleCandidates,omitempty"`
	CanStart        bool     `json:"canStart"`
}

// FragmentAnalyzeResponse 是第一阶段文本分析响应。
type FragmentAnalyzeResponse struct {
	AssistantMessage   string                     `json:"assistantMessage"`
	IntentType         string                     `json:"intentType,omitempty"` // new_fragment | revise_current | adjust_options | chat_only | ask_clarification
	EditPlan           CreativeEditPlan           `json:"editPlan"`
	GenerationIntent   FragmentGenerationIntent   `json:"generationIntent"`
	StoryElements      []FragmentReferenceSlot    `json:"storyElements"`
	RecommendedOptions FragmentRecommendedOptions `json:"recommendedOptions"`
}

// FragmentVisualStyleBible 视觉圣经：全局画风锚点（英文描述为主，便于出图模型对齐）。
type FragmentVisualStyleBible struct {
	ArtStyle     string `json:"artStyle,omitempty"`
	LineQuality  string `json:"lineQuality,omitempty"`
	Palette      string `json:"palette,omitempty"`
	LightingMood string `json:"lightingMood,omitempty"`
}

// FragmentVisualCharacter 角色视觉资产（稳定 key + 不可变特征列表）。
type FragmentVisualCharacter struct {
	Key             string   `json:"key"`
	Name            string   `json:"name,omitempty"`
	ImmutableTraits []string `json:"immutableTraits,omitempty"`
	NegativeTraits  []string `json:"negativeTraits,omitempty"`
	Ownership       []string `json:"ownership,omitempty"`
	RoleImportance  string   `json:"roleImportance,omitempty"` // core | supporting | background
}

// FragmentVisualProp 道具视觉资产。
type FragmentVisualProp struct {
	Key             string   `json:"key"`
	Name            string   `json:"name,omitempty"`
	ImmutableTraits []string `json:"immutableTraits,omitempty"`
	NegativeTraits  []string `json:"negativeTraits,omitempty"`
	Ownership       string   `json:"ownership,omitempty"`      // owning/associated character key, if any
	RoleImportance  string   `json:"roleImportance,omitempty"` // core | supporting | background
}

// FragmentVisualLocation 场景/地点视觉资产。
type FragmentVisualLocation struct {
	Key             string   `json:"key"`
	Name            string   `json:"name,omitempty"`
	ImmutableTraits []string `json:"immutableTraits,omitempty"`
	NegativeTraits  []string `json:"negativeTraits,omitempty"`
	RoleImportance  string   `json:"roleImportance,omitempty"` // core | supporting | background
}

// FragmentVisualEvidenceEntity 是多模态模型从参考图中直接观察到的实体证据。
type FragmentVisualEvidenceEntity struct {
	Key        string   `json:"key,omitempty"`
	Kind       string   `json:"kind,omitempty"` // character | prop | location
	Name       string   `json:"name,omitempty"`
	Traits     []string `json:"traits,omitempty"`
	Position   string   `json:"position,omitempty"`
	OwnerKey   string   `json:"ownerKey,omitempty"`
	Confidence float64  `json:"confidence,omitempty"`
}

// FragmentVisualEvidence 保存一张参考图的视觉事实，避免把创作想象写成不可变特征。
type FragmentVisualEvidence struct {
	ImageURL        string                         `json:"imageUrl,omitempty"`
	Summary         string                         `json:"summary,omitempty"`
	Subjects        []string                       `json:"subjects,omitempty"`
	Entities        []FragmentVisualEvidenceEntity `json:"entities,omitempty"`
	Palette         []string                       `json:"palette,omitempty"`
	Lighting        string                         `json:"lighting,omitempty"`
	Composition     string                         `json:"composition,omitempty"`
	ImmutableTraits []string                       `json:"immutableTraits,omitempty"`
	Confidence      float64                        `json:"confidence,omitempty"`
	Provider        string                         `json:"provider,omitempty"`
	Model           string                         `json:"model,omitempty"`
}

// FragmentVisualBible Step1 结构化视觉设定，用于锚点图、参考图拼接与一致性检查。
type FragmentVisualBible struct {
	StyleBible     *FragmentVisualStyleBible `json:"styleBible,omitempty"`
	Characters     []FragmentVisualCharacter `json:"characters,omitempty"`
	Props          []FragmentVisualProp      `json:"props,omitempty"`
	Locations      []FragmentVisualLocation  `json:"locations,omitempty"`
	SourceEvidence []FragmentVisualEvidence  `json:"sourceEvidence,omitempty"`
}

// FragmentAnchorImage 锚点图记录（referenceKey -> 出图 URL）。
type FragmentAnchorImage struct {
	Key      string `json:"key"`
	Kind     string `json:"kind"` // character | prop | location
	ImageURL string `json:"imageUrl"`
}

// FragmentComicText 描述单格漫画文字层，供图片 prompt 与客户端精确叠字复用。
type FragmentComicText struct {
	ID           string   `json:"id,omitempty"`
	Type         string   `json:"type"` // narration | dialogue | thought | sfx | interjection
	Text         string   `json:"text"`
	Speaker      string   `json:"speaker,omitempty"`
	Target       string   `json:"target,omitempty"`
	Position     string   `json:"position,omitempty"`
	Tone         string   `json:"tone,omitempty"`
	Volume       string   `json:"volume,omitempty"` // whisper | normal | shout
	Rhythm       string   `json:"rhythm,omitempty"`
	BalloonStyle string   `json:"balloonStyle,omitempty"`
	TailTarget   string   `json:"tailTarget,omitempty"`
	Emphasis     []string `json:"emphasis,omitempty"`
	RenderMode   string   `json:"renderMode,omitempty"` // overlay | image
}

// FragmentSpatialRelation describes narrative staging rather than physical
// coordinates. The layout/render layer translates these relations into a
// concrete composition while preserving the dramatic intent.
type FragmentSpatialRelation struct {
	Subject          string   `json:"subject"`
	Relation         string   `json:"relation"` // dominates | looks_at | between | shields | follows | overlaps
	Object           []string `json:"object,omitempty"`
	VisualExpression string   `json:"visualExpression,omitempty"`
	Priority         string   `json:"priority,omitempty"` // required | preferred
}

// FragmentReferenceAsset 是按策略生成或复用的实体参考资产。
type FragmentReferenceAsset struct {
	Key               string `json:"key"`
	Kind              string `json:"kind"` // character | prop | location | user_reference
	ImageURL          string `json:"imageUrl"`
	Source            string `json:"source,omitempty"` // generated | user | reused
	UsageCount        int    `json:"usageCount,omitempty"`
	GeneratedByPolicy bool   `json:"generatedByPolicy,omitempty"`
	TraitsHash        string `json:"traitsHash,omitempty"`
	TokensUsed        int    `json:"tokensUsed,omitempty"`
}

// FragmentEntityBinding 描述单格中实体的关系绑定，避免多人/多物品串位。
type FragmentEntityBinding struct {
	Key                    string  `json:"key"`
	Kind                   string  `json:"kind"` // character | prop | location
	Role                   string  `json:"role,omitempty"`
	Position               string  `json:"position,omitempty"`
	Action                 string  `json:"action,omitempty"`
	OwnerKey               string  `json:"ownerKey,omitempty"`
	ConsistencyNote        string  `json:"consistencyNote,omitempty"`
	NarrativeRole          string  `json:"narrativeRole,omitempty"`
	Region                 string  `json:"region,omitempty"`
	Depth                  string  `json:"depth,omitempty"`
	RelativeScale          string  `json:"relativeScale,omitempty"`
	Facing                 string  `json:"facing,omitempty"`
	GazeTarget             string  `json:"gazeTarget,omitempty"`
	Pose                   string  `json:"pose,omitempty"`
	Expression             string  `json:"expression,omitempty"`
	Emotion                string  `json:"emotion,omitempty"`
	EmotionIntensity       float64 `json:"emotionIntensity,omitempty"`
	StagingIntent          string  `json:"stagingIntent,omitempty"`
	AllowComicExaggeration bool    `json:"allowComicExaggeration,omitempty"`
}

// FragmentComicPanelPlan 描述一张故事碎片图片内部的单个漫画格。
// 图片槽位仍是一张最终图片；Panels 只描述这张图片内部的阅读结构。
type FragmentComicPanelPlan struct {
	Index          int                       `json:"index"`
	BeatPurpose    string                    `json:"beatPurpose,omitempty"`
	SceneDesc      string                    `json:"sceneDesc"`
	ImagePrompt    string                    `json:"imagePrompt"`
	ShotType       string                    `json:"shotType,omitempty"`
	CameraAngle    string                    `json:"cameraAngle,omitempty"`
	Composition    string                    `json:"composition,omitempty"`
	NewInformation string                    `json:"newInformation,omitempty"`
	DramaticIntent string                    `json:"dramaticIntent,omitempty"`
	SilentIntent   string                    `json:"silentIntent,omitempty"`
	ReferenceKeys  []string                  `json:"referenceKeys,omitempty"`
	EntityBindings []FragmentEntityBinding   `json:"entityBindings,omitempty"`
	Relations      []FragmentSpatialRelation `json:"relations,omitempty"`
	ComicTexts     []FragmentComicText       `json:"comicTexts,omitempty"`
}

type FragmentComicRect struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type FragmentComicPanelLayout struct {
	Index      int               `json:"index"`
	Rect       FragmentComicRect `json:"rect"`
	Importance string            `json:"importance,omitempty"`
}

type FragmentComicLayout struct {
	PageAspectRatio string                     `json:"pageAspectRatio"`
	Gutter          float64                    `json:"gutter"`
	ReadingOrder    []int                      `json:"readingOrder"`
	Panels          []FragmentComicPanelLayout `json:"panels"`
}

// FragmentComicPagePlan 描述一张最终图片内部的完整漫画页结构。
// 它不能改变外层 ImageCount：ImageCount 始终表示最终输出图片/漫画页数量。
type FragmentComicPagePlan struct {
	PanelCount        int                      `json:"panelCount"`
	LayoutPreset      string                   `json:"layoutPreset"`
	LayoutDescription string                   `json:"layoutDescription,omitempty"`
	ReadingOrder      string                   `json:"readingOrder,omitempty"`
	Panels            []FragmentComicPanelPlan `json:"panels"`
	Layout            *FragmentComicLayout     `json:"layout,omitempty"`
	PlanningStatus    string                   `json:"planningStatus,omitempty"` // planned | revised | planned_with_review_notes | fallback
	PlanningError     string                   `json:"planningError,omitempty"`
}

type FragmentCreativeFact struct {
	Content string `json:"content"`
	Source  string `json:"source"` // user_text | user_image | existing_story | agent_expansion
	Mutable bool   `json:"mutable"`
}

type FragmentInputAsset struct {
	URL          string  `json:"url"`
	Role         string  `json:"role"` // character_reference | style_reference | scene_reference | prop_reference | existing_story_page | general_inspiration
	ReferenceKey string  `json:"referenceKey,omitempty"`
	UserDeclared bool    `json:"userDeclared,omitempty"`
	Confidence   float64 `json:"confidence,omitempty"`
}

type FragmentCreativeContext struct {
	UserText string                 `json:"userText"`
	Language string                 `json:"language,omitempty"`
	Style    string                 `json:"style,omitempty"`
	Mood     string                 `json:"mood,omitempty"`
	Inputs   []FragmentInputAsset   `json:"inputs,omitempty"`
	Facts    []FragmentCreativeFact `json:"facts,omitempty"`
}

type FragmentCharacterState struct {
	CharacterKey       string            `json:"characterKey"`
	CurrentLocationKey string            `json:"currentLocationKey,omitempty"`
	CurrentClothing    []string          `json:"currentClothing,omitempty"`
	Holding            []string          `json:"holding,omitempty"`
	Injuries           []string          `json:"injuries,omitempty"`
	Knowledge          []string          `json:"knowledge,omitempty"`
	Emotion            string            `json:"emotion,omitempty"`
	Relationships      map[string]string `json:"relationships,omitempty"`
}

type FragmentStoryState struct {
	Characters      []FragmentCharacterState `json:"characters,omitempty"`
	CurrentLocation string                   `json:"currentLocation,omitempty"`
	LastPageResult  string                   `json:"lastPageResult,omitempty"`
}

type FragmentComicPageDocument struct {
	PageIndex          int                   `json:"pageIndex"`
	PageIntent         string                `json:"pageIntent"`
	Plan               FragmentComicPagePlan `json:"plan"`
	TextLayers         []FragmentComicText   `json:"textLayers,omitempty"`
	PanelImageURLs     []string              `json:"panelImageUrls,omitempty"`
	BackgroundImageURL string                `json:"backgroundImageUrl,omitempty"`
	FlattenedImageURL  string                `json:"flattenedImageUrl,omitempty"`
	Status             string                `json:"status"`
}

// FragmentComicDocument is the versioned source of truth for generation,
// continuation and editing. It is persisted inside the generation result so
// older fragments remain readable without a schema migration.
type FragmentComicDocument struct {
	SchemaVersion   int                         `json:"schemaVersion"`
	Revision        int                         `json:"revision"`
	FragmentID      string                      `json:"fragmentId,omitempty"`
	CreativeContext FragmentCreativeContext     `json:"creativeContext"`
	VisualBible     *FragmentVisualBible        `json:"visualBible,omitempty"`
	StoryState      FragmentStoryState          `json:"storyState"`
	Pages           []FragmentComicPageDocument `json:"pages"`
	ReferenceAssets []FragmentReferenceAsset    `json:"referenceAssets,omitempty"`
}

// FragmentScenePlan 是可持久化的单张图片（漫画页）生成计划。
// 历史任务没有 ComicPage 时仍按旧的单幅插图逻辑回放。
type FragmentScenePlan struct {
	Index             int                     `json:"index"`
	SceneDesc         string                  `json:"sceneDesc"`
	ImagePrompt       string                  `json:"imagePrompt"`
	FinalImagePrompt  string                  `json:"finalImagePrompt,omitempty"`
	ReferenceKeys     []string                `json:"referenceKeys,omitempty"`
	EntityBindings    []FragmentEntityBinding `json:"entityBindings,omitempty"`
	ComicTexts        []FragmentComicText     `json:"comicTexts,omitempty"`
	ComicPage         *FragmentComicPagePlan  `json:"comicPage,omitempty"`
	Seed              int                     `json:"seed,omitempty"`
	ProviderOptions   map[string]interface{}  `json:"providerOptions,omitempty"`
	GeneratedImageURL string                  `json:"generatedImageUrl,omitempty"`
	PanelImageURLs    []string                `json:"panelImageUrls,omitempty"`
}

// FragmentConsistencyPolicy 记录一次任务实际采用的一致性策略。
type FragmentConsistencyPolicy struct {
	Level                 string                 `json:"level"` // off | standard | strong
	SeriesSeed            int                    `json:"seriesSeed,omitempty"`
	EnableReferenceAssets bool                   `json:"enableReferenceAssets"`
	MaxCharacterAssets    int                    `json:"maxCharacterAssets,omitempty"`
	MaxPropAssets         int                    `json:"maxPropAssets,omitempty"`
	MaxLocationAssets     int                    `json:"maxLocationAssets,omitempty"`
	ProviderOptions       map[string]interface{} `json:"providerOptions,omitempty"`
	Capabilities          []string               `json:"capabilities,omitempty"`
}

// FragmentGenerationStepMetric 记录各阶段成本与耗时。
type FragmentGenerationStepMetric struct {
	Name       string `json:"name"`
	Tokens     int    `json:"tokens,omitempty"`
	DurationMs int64  `json:"durationMs,omitempty"`
	Provider   string `json:"provider,omitempty"`
	Model      string `json:"model,omitempty"`
	Note       string `json:"note,omitempty"`
}

// FragmentConsistencyIssue 一致性检查问题（仅记录，不阻断任务）。
type FragmentConsistencyIssue struct {
	SceneIndex int     `json:"sceneIndex,omitempty"`
	Severity   string  `json:"severity,omitempty"` // low | medium | high
	Detail     string  `json:"detail"`
	EntityKey  string  `json:"entityKey,omitempty"`
	ImageURL   string  `json:"imageUrl,omitempty"`
	Expected   string  `json:"expected,omitempty"`
	Observed   string  `json:"observed,omitempty"`
	Confidence float64 `json:"confidence,omitempty"`
}

// FragmentGenerationTrace 保存完整生成计划、策略、成本和审计信息。
type FragmentGenerationTrace struct {
	ComicDocument       *FragmentComicDocument         `json:"comicDocument,omitempty"`
	VisualBible         *FragmentVisualBible           `json:"visualBible,omitempty"`
	VisualEvidence      []FragmentVisualEvidence       `json:"visualEvidence,omitempty"`
	Scenes              []FragmentScenePlan            `json:"scenes,omitempty"`
	EntityUsage         map[string]int                 `json:"entityUsage,omitempty"`
	ConsistencyPolicy   *FragmentConsistencyPolicy     `json:"consistencyPolicy,omitempty"`
	ProviderOptions     map[string]interface{}         `json:"providerOptions,omitempty"`
	ReferenceAssets     []FragmentReferenceAsset       `json:"referenceAssets,omitempty"`
	ConsistencyIssues   []FragmentConsistencyIssue     `json:"consistencyIssues,omitempty"`
	VisionAuditProvider string                         `json:"visionAuditProvider,omitempty"`
	AuditedImageCount   int                            `json:"auditedImageCount,omitempty"`
	SkippedAuditReason  string                         `json:"skippedAuditReason,omitempty"`
	Metrics             []FragmentGenerationStepMetric `json:"metrics,omitempty"`
}

// FragmentGenerationResult 碎片故事生成结果
type FragmentGenerationResult struct {
	Content            string                           `json:"content"`                      // 生成的文字内容
	ImageUrls          []string                         `json:"imageUrls"`                    // 生成的图片URL列表
	AspectRatio        string                           `json:"aspectRatio,omitempty"`        // 实际使用的配图长宽比
	TokensUsed         int                              `json:"tokensUsed"`                   // 使用的token数量
	DraftFragmentID    string                           `json:"draftFragmentId,omitempty"`    // 服务端为该次生成落库的草稿碎片 ID（客户端发布时 PUT 同一条，避免重复创建）
	ExpectedImageCount int                              `json:"expectedImageCount,omitempty"` // 用户/客户端请求的权威目标张数
	ImageSlots         []FragmentGenerationImageSlot    `json:"imageSlots,omitempty"`         // 逐张图片任务槽位，作为生成状态事实源
	ImageProgress      *FragmentGenerationImageProgress `json:"imageProgress,omitempty"`      // 基于 slot 的进度
	VisualBible        *FragmentVisualBible             `json:"visualBible,omitempty"`        // 结构化视觉设定（方案 B）
	VisualEvidence     []FragmentVisualEvidence         `json:"visualEvidence,omitempty"`     // 多模态参考图事实
	AnchorImages       []FragmentAnchorImage            `json:"anchorImages,omitempty"`       // 锚点参考图
	ReferenceAssets    []FragmentReferenceAsset         `json:"referenceAssets,omitempty"`    // 按需参考资产
	ScenePlan          []FragmentScenePlan              `json:"scenePlan,omitempty"`          // 可追踪场景计划
	ConsistencyPolicy  *FragmentConsistencyPolicy       `json:"consistencyPolicy,omitempty"`  // 一致性策略
	GenerationTrace    *FragmentGenerationTrace         `json:"generationTrace,omitempty"`    // 完整生成 trace
	ConsistencyIssues  []FragmentConsistencyIssue       `json:"consistencyIssues,omitempty"`  // 一致性检查（best-effort）
	StoryElements      []FragmentReferenceSlot          `json:"storyElements,omitempty"`      // 分析/生成时使用的语义元素槽位
	ComicDocument      *FragmentComicDocument           `json:"comicDocument,omitempty"`
}

// FragmentGenerationImageSlot 是故事碎片生成的单张图片槽位。
// 后端把 slots 作为事实源：只有所有槽位 completed，任务才可以 completed/发布。
type FragmentGenerationImageSlot struct {
	ID           string `json:"id,omitempty"`
	TaskID       string `json:"taskId,omitempty"`
	FragmentID   string `json:"fragmentId,omitempty"`
	Index        int    `json:"index"`
	Title        string `json:"title,omitempty"`
	Caption      string `json:"caption,omitempty"`
	Status       string `json:"status,omitempty"` // planning | planned | generating | completed | needs_review | failed
	ImageURL     string `json:"imageUrl,omitempty"`
	AssetID      string `json:"assetId,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

// FragmentGenerationImageProgress 汇总图片槽位完成情况。
type FragmentGenerationImageProgress struct {
	CompletedCount int `json:"completedCount"`
	TotalCount     int `json:"totalCount"`
}

// FragmentContentGenerationRequest 碎片故事内容生成请求
type FragmentContentGenerationRequest struct {
	UserInput string `json:"userInput"` // 用户输入
	Style     string `json:"style"`     // 风格
	Mood      string `json:"mood"`      // 情绪
	Length    string `json:"length"`    // 长度
	Language  string `json:"language"`  // 语言
	Context   string `json:"context"`   // 额外上下文
}

// FragmentContentGenerationResult 碎片故事内容生成结果
type FragmentContentGenerationResult struct {
	Content     string `json:"content"`               // 生成的文字内容
	AspectRatio string `json:"aspectRatio,omitempty"` // 解析后的配图比例（已含默认值）
	TokensUsed  int    `json:"tokensUsed"`            // 使用的token数量
}

// FragmentImageGenerationRequest 碎片故事图片生成请求
type FragmentImageGenerationRequest struct {
	Content    string `json:"content"`    // 文字内容（用于生成图片）
	ImageCount int    `json:"imageCount"` // 生成图片数量
	Style      string `json:"style"`      // 图片风格
	Size       string `json:"size"`       // 图片尺寸
	Quality    string `json:"quality"`    // 图片质量
}

// FragmentImageGenerationResult 碎片故事图片生成结果
type FragmentImageGenerationResult struct {
	ImageUrls  []string `json:"imageUrls"`  // 生成的图片URL列表
	TokensUsed int      `json:"tokensUsed"` // 使用的token数量
}

// FragmentGenerationTask 碎片故事生成任务
type FragmentGenerationTask struct {
	ID           string                    `json:"id"`
	UserID       string                    `json:"userId"`
	Status       string                    `json:"status"` // pending, processing, completed, failed
	Request      FragmentGenerationRequest `json:"request"`
	Result       *FragmentGenerationResult `json:"result,omitempty"`
	Progress     int                       `json:"progress"`    // 0-100
	CurrentStep  string                    `json:"currentStep"` // 当前步骤
	ErrorMessage string                    `json:"errorMessage,omitempty"`
	TokensUsed   int                       `json:"tokensUsed"`
	CreatedAt    int64                     `json:"createdAt"`
	StartedAt    *int64                    `json:"startedAt,omitempty"`
	CompletedAt  *int64                    `json:"completedAt,omitempty"`
	UpdatedAt    int64                     `json:"updatedAt"`

	// Relations
	User *User `json:"user,omitempty"`
}

// TableName specifies the table name for FragmentGenerationTask
func (FragmentGenerationTask) TableName() string {
	return "fragment_generation_tasks"
}
