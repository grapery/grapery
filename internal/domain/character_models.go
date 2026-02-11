package domain

// Character represents a story character
type Character struct {
	ID      string `json:"id"`
	StoryID string `json:"storyId"`
	// AuthorID 角色创建者ID
	// 核心规则：只有故事创作者可以创建角色，所以 AuthorID 应该等于 Story.AuthorID
	AuthorID                 string  `json:"-"`
	Name                     string  `json:"name"`
	Description              string  `json:"description"`
	Avatar                   string  `json:"avatar,omitempty"`
	Poster                   string  `json:"poster,omitempty"`
	Portrait                 string  `json:"portrait,omitempty"`                 // 完整角色形象图URL（AI生成）
	NeedsPortrait            bool    `json:"needsPortrait"`                      // 是否需要生成形象
	ReferenceImage           string  `json:"referenceImage,omitempty"`           // 参考图URL
	PortraitGenerationStatus string  `json:"portraitGenerationStatus,omitempty"` // none/pending/generating/generated/failed
	Personality              string  `json:"personality,omitempty"`
	Background               string  `json:"background,omitempty"`
	ShortTermGoal            string  `json:"shortTermGoal,omitempty"`   // Immediate objectives in current story arc
	LongTermGoal             string  `json:"longTermGoal,omitempty"`    // Overarching ambitions
	HandlingStyle            string  `json:"handlingStyle,omitempty"`   // Approach to handling situations
	CognitionRange           string  `json:"cognitionRange,omitempty"`  // Knowledge and awareness of their world
	AbilityFeatures          string  `json:"abilityFeatures,omitempty"` // Special skills and capabilities
	Appearance               string  `json:"appearance,omitempty"`      // Physical appearance and features
	DressPreference          string  `json:"dressPreference,omitempty"` // Clothing preferences and style
	TraitsJSON               string  `json:"-"`                         // Internal storage for DB conversion
	SkillsJSON               string  `json:"-"`                         // Internal storage for DB conversion
	IsPublic                 bool    `json:"isPublic"`
	SourceType               string  `json:"sourceType,omitempty"`
	SourcePrompt             string  `json:"sourcePrompt,omitempty"`
	SourceImage              string  `json:"sourceImage,omitempty"`
	CreatedBy                string  `json:"createdBy,omitempty"`
	LastEditedBy             string  `json:"lastEditedBy,omitempty"`
	Likes                    int     `json:"likes"`
	Followers                int     `json:"followers"`
	Stories                  int     `json:"stories"`
	CreatedAt                int64   `json:"createdAt"`
	UpdatedAt                int64   `json:"updatedAt"`

	// 海报创建权限
	PosterCreationPermission string `json:"posterCreationPermission"` // creator_only, anyone (V1/V2 MVP - group_members removed)

	// Business fields
	Traits      []string `json:"traits,omitempty"`
	Skills      []string `json:"skills,omitempty"`
	Author      *User    `json:"author,omitempty"`
	Agent       *Agent   `json:"agent,omitempty"`
	IsFollowing *bool    `json:"isFollowing,omitempty"` // 当前用户是否关注此角色
}

// CharacterFollow 角色关注
type CharacterFollow struct {
	ID          string `json:"id"`
	UserID      string `json:"userId"`
	CharacterID string `json:"characterId"`
	CreatedAt   int64  `json:"createdAt"`

	// Relations
	User      *User      `json:"user,omitempty"`
	Character *Character `json:"character,omitempty"`
}

// PosterStatus 海报状态
type PosterStatus string

const (
	PosterStatusDraft      PosterStatus = "draft"      // 草稿，用户创建但未生成
	PosterStatusGenerating PosterStatus = "generating" // 正在生成
	PosterStatusGenerated  PosterStatus = "generated"  // 生成完成
	PosterStatusPublished  PosterStatus = "published"  // 已发布
	PosterStatusFailed     PosterStatus = "failed"     // 生成失败
)

// PosterVisualConcept 海报视觉概念
type PosterVisualConcept struct {
	VisualSubject      string `json:"visualSubject"`      // 视觉主体描述（角色+动作）
	SceneEnvironment   string `json:"sceneEnvironment"`   // 场景环境（背景+天气+道具）
	CompositionCamera  string `json:"compositionCamera"`  // 构图与摄像角度
	LightingAtmosphere string `json:"lightingAtmosphere"` // 光照与氛围
	ArtStyle           string `json:"artStyle"`           // 艺术风格
}

// PosterTypography 海报排版指令
type PosterTypography struct {
	TitleContent    string `json:"titleContent"`    // 标题文字内容（大写）
	TitleStyle      string `json:"titleStyle"`      // 标题样式（字体+材质+颜色）
	TitlePosition   string `json:"titlePosition"`   // 标题位置
	SubtitleContent string `json:"subtitleContent"` // 副标题内容
	SubtitleStyle   string `json:"subtitleStyle"`   // 副标题样式
}

// PosterConceptDetails 结构化的海报概念详情
type PosterConceptDetails struct {
	VisualConcept *PosterVisualConcept `json:"posterConcept"`         // 视觉概念
	Typography    *PosterTypography    `json:"typographyInstruction"` // 排版指令
}

// CharacterPoster 角色海报
type CharacterPoster struct {
	ID          string       `json:"id"`
	CharacterID string       `json:"characterId"`
	AuthorID    string       `json:"-"`
	Type        string       `json:"type"` // image, video
	Title       string       `json:"title"`
	Image       string       `json:"image"`               // Poster image URL (for image type)
	Video       string       `json:"video,omitempty"`     // Video URL (for video type)
	Thumbnail   string       `json:"thumbnail,omitempty"` // Video thumbnail URL
	Duration    int          `json:"duration,omitempty"`  // Video duration in seconds
	Prompt      string       `json:"prompt,omitempty"`    // User's original prompt/description
	Status      PosterStatus `json:"status"`              // draft, generating, generated, published, failed

	// AI Generation fields
	ReferenceStoryEnabled bool                  `json:"referenceStoryEnabled,omitempty"` // Whether to reference recent story plots
	PosterConceptJSON     string                `json:"-"`                               // LLM generated poster concept JSON (internal storage)
	PosterConcept         *PosterConceptDetails `json:"posterConcept,omitempty"`         // Structured poster concept for client editing
	FinalImagePrompt      string                `json:"finalImagePrompt,omitempty"`      // Final assembled prompt for image generation
	ErrorMessage          string                `json:"errorMessage,omitempty"`          // Error message if generation failed

	// AI Generation Record IDs (for tracking both AI steps)
	ConceptGenerationID string `json:"conceptGenerationId,omitempty"` // AI record for concept generation (Step 1)
	ImageGenerationID   string `json:"imageGenerationId,omitempty"`   // AI record for image generation (Step 2)

	LikesCount int   `json:"likesCount"`
	Likes      int   `json:"likes"`
	Shares     int   `json:"shares"`
	CreatedAt  int64 `json:"createdAt"`
	UpdatedAt  int64 `json:"updatedAt,omitempty"`

	// Relations
	Character *Character `json:"character,omitempty"`
	Author    *User      `json:"author,omitempty"`
}

// CharacterAnalytics 角色分析数据
type CharacterAnalytics struct {
	ID                   string `json:"id"`
	CharacterID          string `json:"characterId"`
	UsersWhoChattedCount int    `json:"usersWhoChattedCount"`
	TotalMessagesSent    int    `json:"totalMessagesSent"`
	TotalTokensConsumed  int64  `json:"totalTokensConsumed"`
	UpdatedAt            int64  `json:"updatedAt"`

	// Relations
	Character *Character `json:"character,omitempty"`
}

// PosterCreationPermissionType 海报创建权限类型
type PosterCreationPermissionType string

const (
	PosterCreationPermissionCreatorOnly PosterCreationPermissionType = "creator_only"
	PosterCreationPermissionAnyone       PosterCreationPermissionType = "anyone"
)
