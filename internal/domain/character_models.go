package domain

import "github.com/grapestree/fgrapery/grapery/internal/common"

// Character represents a story character
type Character struct {
	// Base model fields
	common.BaseModel

	StoryID string `json:"storyId"`
	// UserID 角色创建者ID
	// 核心规则：只有故事创作者可以创建角色，所以 UserID 应该等于 Story.UserID
	UserID                   string `json:"authorId"` // 保持 JSON 标签为 authorId 以保持 API 兼容性
	Name                     string `json:"name"`
	Description              string `json:"description"`
	Avatar                   string `json:"avatar,omitempty"`
	Poster                   string `json:"poster,omitempty"`
	Portrait                 string `json:"portrait,omitempty"`                 // 完整角色形象图URL（AI生成）
	NeedsPortrait            bool   `json:"needsPortrait"`                      // 是否需要生成形象
	ReferenceImage           string `json:"referenceImage,omitempty"`           // 参考图URL
	PortraitGenerationStatus string `json:"portraitGenerationStatus,omitempty"` // none/pending/generating/generated/failed
	Personality              string `json:"personality,omitempty"`
	Background               string `json:"background,omitempty"`
	ShortTermGoal            string `json:"shortTermGoal,omitempty"`   // Immediate objectives in current story arc
	LongTermGoal             string `json:"longTermGoal,omitempty"`    // Overarching ambitions
	HandlingStyle            string `json:"handlingStyle,omitempty"`   // Approach to handling situations
	CognitionRange           string `json:"cognitionRange,omitempty"`  // Knowledge and awareness of their world
	AbilityFeatures          string `json:"abilityFeatures,omitempty"` // Special skills and capabilities
	Appearance               string `json:"appearance,omitempty"`      // Physical appearance and features
	DressPreference          string `json:"dressPreference,omitempty"` // Clothing preferences and style
	TraitsJSON               string `json:"-"`                         // Internal storage for DB conversion
	SkillsJSON               string `json:"-"`                         // Internal storage for DB conversion
	IsPublic                 bool   `json:"isPublic"`
	SourceType               string `json:"sourceType,omitempty"`
	SourcePrompt             string `json:"sourcePrompt,omitempty"`
	SourceImage              string `json:"sourceImage,omitempty"`
	CreatedBy                string `json:"createdBy,omitempty"`
	LastEditedBy             string `json:"lastEditedBy,omitempty"`

	// Engagement stats fields (partial - using Likes, Followers, Stories)
	// Note: We use individual fields instead of embedding EngagementStats
	// because Character has a custom 'Stories' field instead of 'Views'
	Likes     int `json:"likes"`
	Comments  int `json:"comments"`
	Shares    int `json:"shares"`
	Followers int `json:"followers"`
	Stories   int `json:"stories"` // Custom field: number of stories this character appears in

	// 海报创建权限
	PosterCreationPermission string `json:"posterCreationPermission"` // creator_only, anyone (V1/V2 MVP - group_members removed)

	// StoryCreationAppUI alignment fields
	Role         string `json:"role,omitempty"`         // 角色定位 (主角/配角/反派/导师/神秘人)
	AIStyle      string `json:"aiStyle,omitempty"`      // AI 生成风格
	AIPrompt     string `json:"aiPrompt,omitempty"`     // AI 生成提示词
	AIGenerated  bool   `json:"aiGenerated"`            // 是否由 AI 生成
	Backstory    string `json:"backstory,omitempty"`    // 角色背景故事 (alias for Background)

	// Business fields
	Traits      []string `json:"traits,omitempty"`
	Skills      []string `json:"skills,omitempty"`
	Author      *User    `json:"author,omitempty"`
	Agent       *Agent   `json:"agent,omitempty"`
	IsFollowing *bool    `json:"isFollowing,omitempty"` // 当前用户是否关注此角色
}

// CharacterFollow 角色关注
type CharacterFollow struct {
	// Base model fields
	common.BaseModel

	UserID      string `json:"userId"`
	CharacterID string `json:"characterId"`

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
	// Base model fields
	common.BaseModel

	CharacterID string       `json:"characterId"`
	UserID      string       `json:"authorId"` // 保持 JSON 标签为 authorId 以保持 API 兼容性
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

	// Engagement stats fields
	common.EngagementStats

	// Relations
	Character *Character `json:"character,omitempty"`
	Author    *User      `json:"author,omitempty"`
}

// GetLikesCount returns the like count (alias for API compatibility)
func (cp *CharacterPoster) GetLikesCount() int {
	return cp.Likes
}

// CharacterAnalytics 角色分析数据
type CharacterAnalytics struct {
	// Base model fields
	common.BaseModel

	CharacterID          string `json:"characterId"`
	UsersWhoChattedCount int    `json:"usersWhoChattedCount"`
	TotalMessagesSent    int    `json:"totalMessagesSent"`
	TotalTokensConsumed  int64  `json:"totalTokensConsumed"`

	// Relations
	Character *Character `json:"character,omitempty"`
}

// PosterCreationPermissionType 海报创建权限类型
type PosterCreationPermissionType string

const (
	PosterCreationPermissionCreatorOnly PosterCreationPermissionType = "creator_only"
	PosterCreationPermissionAnyone      PosterCreationPermissionType = "anyone"
)

// CharacterViewType 角色视图类型
type CharacterViewType string

const (
	CharacterViewFront CharacterViewType = "front"
	CharacterViewSide  CharacterViewType = "side"
	CharacterViewBack  CharacterViewType = "back"
)

// CharacterViewStatus 角色视图生成状态
type CharacterViewStatus string

const (
	CharacterViewStatusPending    CharacterViewStatus = "pending"
	CharacterViewStatusGenerating CharacterViewStatus = "generating"
	CharacterViewStatusCompleted  CharacterViewStatus = "completed"
	CharacterViewStatusFailed     CharacterViewStatus = "failed"
)

// CharacterView 角色三视图
type CharacterView struct {
	common.BaseModel

	CharacterID   string             `json:"characterId"`
	ViewType      CharacterViewType  `json:"viewType"` // front, side, back
	ImageURL      string             `json:"imageUrl"`

	IsAIGenerated bool               `json:"isAIGenerated"`
	Prompt        string             `json:"prompt,omitempty"`
	Status        CharacterViewStatus `json:"status"` // pending, generating, completed, failed
	ErrorMessage  string             `json:"errorMessage,omitempty"`

	// Relations
	Character *Character `json:"character,omitempty"`
}

// GenerateCharacterViewsRequest 生成角色三视图请求
type GenerateCharacterViewsRequest struct {
	ViewTypes    []CharacterViewType `json:"viewTypes,omitempty"`    // 要生成的视图类型，默认全部
	CustomPrompt string              `json:"customPrompt,omitempty"` // 自定义提示词
}

// GenerateCharacterViewsResponse 生成角色三视图响应
type GenerateCharacterViewsResponse struct {
	Views      []CharacterView `json:"views"`
	TaskID     string          `json:"taskId,omitempty"`
	EstimatedTime int           `json:"estimatedTime,omitempty"` // 预估完成时间（秒）
}
