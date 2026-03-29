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

	// StoryCreationAppUI alignment fields
	Role        string               `json:"role,omitempty"`      // 角色定位 (主角/配角/反派/导师/神秘人)
	AIStyle     string               `json:"aiStyle,omitempty"`   // AI 生成风格
	AIPrompt    string               `json:"aiPrompt,omitempty"`  // AI 生成提示词
	AIGenerated bool                 `json:"aiGenerated"`         // 是否由 AI 生成
	Backstory   string               `json:"backstory,omitempty"` // 角色背景故事 (alias for Background)
	Views       *CharacterThreeViews `json:"views,omitempty"`     // 三视图：sheet=单张合成图，或 front/side/back 分图（旧）

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

// REMOVED: CharacterPoster, PosterStatus, PosterVisualConcept, PosterTypography, PosterConceptDetails, PosterCreationPermissionType - not in StoryCreationAppUI design

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

// REMOVED: PosterCreationPermissionType - not in StoryCreationAppUI design

// CharacterThreeViews 角色三视图 URL（与客户端 CharacterThreeViews 对齐）
type CharacterThreeViews struct {
	Sheet string `json:"sheet,omitempty"` // 单张横向正/侧/背合一参考图（优先）
	Front string `json:"front,omitempty"`
	Side  string `json:"side,omitempty"`
	Back  string `json:"back,omitempty"`
}

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

	CharacterID string            `json:"characterId"`
	ViewType    CharacterViewType `json:"viewType"` // front, side, back
	ImageURL    string            `json:"imageUrl"`

	IsAIGenerated bool                `json:"isAIGenerated"`
	Prompt        string              `json:"prompt,omitempty"`
	Status        CharacterViewStatus `json:"status"` // pending, generating, completed, failed
	ErrorMessage  string              `json:"errorMessage,omitempty"`

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
	Views         []CharacterView `json:"views"`
	TaskID        string          `json:"taskId,omitempty"`
	EstimatedTime int             `json:"estimatedTime,omitempty"` // 预估完成时间（秒）
}
