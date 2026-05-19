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
	ImageCount int      `json:"imageCount"` // 需要生成的图片数量 (1-10)
	Style      string   `json:"style"`      // 风格：fantasy, realistic, anime, etc.
	Mood       string   `json:"mood"`       // 情绪：happy, sad, mysterious, etc.
	Length     string   `json:"length"`     // 内容长度：short, medium, long
	Language   string   `json:"language"`   // 语言：zh-Hans, en, ja
	Visibility string   `json:"visibility"` // 可见性：public, followers, private
	// AspectRatio 配图长宽比：1:1、16:9、9:16、3:4、4:3；空表示由多模态解析（有参考图时）或默认 16:9。
	AspectRatio            string `json:"aspectRatio,omitempty"`
	ConsistencyLevel       string `json:"consistencyLevel,omitempty"`       // off | standard | strong
	EnableReferenceAssets  *bool  `json:"enableReferenceAssets,omitempty"`  // nil 时由 consistencyLevel 决定
	IncludeGenerationTrace bool   `json:"includeGenerationTrace,omitempty"` // 返回完整生成 trace
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
	Type     string `json:"type"` // narration | dialogue | thought | sfx
	Text     string `json:"text"`
	Speaker  string `json:"speaker,omitempty"`
	Position string `json:"position,omitempty"`
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
	Key             string `json:"key"`
	Kind            string `json:"kind"` // character | prop | location
	Role            string `json:"role,omitempty"`
	Position        string `json:"position,omitempty"`
	Action          string `json:"action,omitempty"`
	OwnerKey        string `json:"ownerKey,omitempty"`
	ConsistencyNote string `json:"consistencyNote,omitempty"`
}

// FragmentScenePlan 是可持久化的单格生成计划。
type FragmentScenePlan struct {
	Index             int                     `json:"index"`
	SceneDesc         string                  `json:"sceneDesc"`
	ImagePrompt       string                  `json:"imagePrompt"`
	FinalImagePrompt  string                  `json:"finalImagePrompt,omitempty"`
	ReferenceKeys     []string                `json:"referenceKeys,omitempty"`
	EntityBindings    []FragmentEntityBinding `json:"entityBindings,omitempty"`
	ComicTexts        []FragmentComicText     `json:"comicTexts,omitempty"`
	Seed              int                     `json:"seed,omitempty"`
	ProviderOptions   map[string]interface{}  `json:"providerOptions,omitempty"`
	GeneratedImageURL string                  `json:"generatedImageUrl,omitempty"`
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
	Content           string                     `json:"content"`                     // 生成的文字内容
	ImageUrls         []string                   `json:"imageUrls"`                   // 生成的图片URL列表
	AspectRatio       string                     `json:"aspectRatio,omitempty"`       // 实际使用的配图长宽比
	TokensUsed        int                        `json:"tokensUsed"`                  // 使用的token数量
	DraftFragmentID   string                     `json:"draftFragmentId,omitempty"`   // 服务端为该次生成落库的草稿碎片 ID（客户端发布时 PUT 同一条，避免重复创建）
	VisualBible       *FragmentVisualBible       `json:"visualBible,omitempty"`       // 结构化视觉设定（方案 B）
	VisualEvidence    []FragmentVisualEvidence   `json:"visualEvidence,omitempty"`    // 多模态参考图事实
	AnchorImages      []FragmentAnchorImage      `json:"anchorImages,omitempty"`      // 锚点参考图
	ReferenceAssets   []FragmentReferenceAsset   `json:"referenceAssets,omitempty"`   // 按需参考资产
	ScenePlan         []FragmentScenePlan        `json:"scenePlan,omitempty"`         // 可追踪场景计划
	ConsistencyPolicy *FragmentConsistencyPolicy `json:"consistencyPolicy,omitempty"` // 一致性策略
	GenerationTrace   *FragmentGenerationTrace   `json:"generationTrace,omitempty"`   // 完整生成 trace
	ConsistencyIssues []FragmentConsistencyIssue `json:"consistencyIssues,omitempty"` // 一致性检查（best-effort）
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
