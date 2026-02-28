package domain

import "github.com/grapestree/fgrapery/grapery/internal/common"

// Story captures a creation project
type Story struct {
	// Base model fields
	common.BaseModel

	UserID      string `json:"authorId"` // 保持 JSON 标签为 authorId 以保持 API 兼容性
	Title       string `json:"title"`
	Description string `json:"description"`
	CoverImage  string `json:"coverImage"`

	// Engagement stats fields
	common.EngagementStats

	Followers           int          `json:"followers"`
	Panels              int          `json:"panels"`
	StoryboardCount     int          `json:"storyboardCount"`   // Number of storyboards in this story
	CharacterCount      int          `json:"characterCount"`    // Number of characters in the story
	DefaultSceneCount   int          `json:"defaultSceneCount"` // Default number of scenes for storyboards (2-8, default 3)
	Genre               string       `json:"genre"`
	Style               *StyleConfig `json:"style,omitempty"`            // AI生成风格配置（完整信息，可为空）
	Status              string       `json:"status"`                     // draft, published, rendering
	IsCollaborationOpen bool         `json:"isCollaborationOpen"`        // Whether collaboration is open: true=anyone can edit, false=only author can edit
	RootStoryboardID    string       `json:"rootStoryboardId,omitempty"` // 根故事板ID

	// AI 丰富相关字段
	OriginalDescription string `json:"originalDescription,omitempty"` // 用户原始描述（AI丰富前）
	EnrichedDescription string `json:"enrichedDescription,omitempty"` // AI 丰富后的描述
	IsAIEnriched        bool   `json:"isAIEnriched"`                  // 是否使用AI丰富过
	AIEnrichedAt        *int64 `json:"aiEnrichedAt,omitempty"`        // AI 丰富时间
	CoverGeneratedByAI  bool   `json:"coverGeneratedByAI"`            // 封面是否由AI生成
	PosterImage         string `json:"posterImage,omitempty"`         // AI 生成的海报图片
	BackgroundImage     string `json:"backgroundImage,omitempty"`     // AI 生成的背景图片

	// AI使用策略（故事级别开关，创建时确定，不可更改）
	UseAI               bool                 `json:"useAI"`                         // 是否使用AI辅助创作
	AIAssistanceOptions *AIAssistanceOptions `json:"aiAssistanceOptions,omitempty"` // AI辅助选项

	// 可见性设置
	Visibility string `json:"visibility"` // 可见性: public, followers, private

	// 评论设置
	AllowComments bool `json:"allowComments"` // 是否允许评论

	// AI 协作标签显示
	ShowAICollaborationLabel bool `json:"showAICollaborationLabel"` // 是否显示 AI 协作标签

	// 已废弃：使用 UseAI 替代
	AIEnabled bool `json:"aiEnabled"`

	// Token 消耗统计
	TokensUsed       int `json:"tokensUsed"`       // 故事创建过程中消耗的总token
	TextTokensUsed   int `json:"textTokensUsed"`   // 文本生成消耗的token
	ImageTokensUsed  int `json:"imageTokensUsed"`  // 图片生成消耗的token
	AIGenerationCost int `json:"aiGenerationCost"` // AI 生成费用（积分/点数）

	// Relations
	Author       *User               `json:"author,omitempty"`
	Characters   []*Character        `json:"characters,omitempty"`
	Scenes       []*StoryScene       `json:"scenes,omitempty"`
	Contributors []*StoryContributor `json:"contributors,omitempty"`

	// 来源追踪
	SourceFragmentID *string `json:"sourceFragmentId,omitempty" gorm:"column:source_fragment_id;type:varchar(36);index"` // 来源碎片ID（如果是从碎片转换而来），支持数据库级联表查询

	// 默认路径节点ID列表（按顺序）
	DefaultPathNodeIDs []string `json:"defaultPathNodeIds,omitempty" gorm:"type:json;default:'[]'"`

	// 路径更新时间
	DefaultPathUpdatedAt *int64 `json:"defaultPathUpdatedAt,omitempty"`

	// 路径类型: manual(手动设置) | auto(自动计算)
	DefaultPathType string `json:"defaultPathType,omitempty" gorm:"default:'manual'"`

	// MARK: - StoryCreationAppUI Alignment Fields
	Topic string `json:"topic,omitempty"` // 话题标签 (e.g., "灵感来了·非人类的地球生存报告")
	Forks int    `json:"forks"`            // 分支数量
}

// AIAssistanceOptions AI辅助选项
type AIAssistanceOptions struct {
	GenerateMetadata bool `json:"generateMetadata"` // 生成标题/描述
	GenerateVisuals  bool `json:"generateVisuals"`  // 生成背景图/封面
	AssistStoryboard bool `json:"assistStoryboard"` // 故事板 AI 辅助
	GenerateVideo    bool `json:"generateVideo"`    // 生成视频（可选）
}

// DefaultAIAssistanceOptions 返回默认的AI辅助选项
func DefaultAIAssistanceOptions() *AIAssistanceOptions {
	return &AIAssistanceOptions{
		GenerateMetadata: true,
		GenerateVisuals:  true,
		AssistStoryboard: true,
		GenerateVideo:    false,
	}
}

// DisabledAIAssistanceOptions 返回全部禁用的AI辅助选项
func DisabledAIAssistanceOptions() *AIAssistanceOptions {
	return &AIAssistanceOptions{
		GenerateMetadata: false,
		GenerateVisuals:  false,
		AssistStoryboard: false,
		GenerateVideo:    false,
	}
}

// TextPosition 文本位置类型
type TextPosition string

const (
	TextPosTop        TextPosition = "top"
	TextPosBottom     TextPosition = "bottom"
	TextPosLeft       TextPosition = "left"
	TextPosBottomLeft TextPosition = "bottom-left"
)

// Panel describes a storyboard panel
type Panel struct {
	// Base model fields
	common.BaseModel

	StoryID      string `json:"storyId"`
	StoryboardID string `json:"storyboardId,omitempty"` // 关联故事板ID（可选）
	Sequence     int    `json:"sequence"`

	// 内容字段
	Image     string `json:"img,omitempty"`  // API 字段名为 img
	Text      string `json:"text"`
	Title     string `json:"title,omitempty"`     // 保留兼容
	Content   string `json:"content,omitempty"`   // 保留兼容
	TextPos   string `json:"textPos,omitempty"`   // "top", "bottom", "left", "bottom-left"
	TextRight string `json:"textRight,omitempty"` // 分屏布局右侧文本

	// Use only Likes from EngagementStats
	Likes int `json:"likes"`

	Published bool `json:"isPublished"`

	// AI 生成相关
	IsAIGenerated bool   `json:"isAIGenerated,omitempty"`
	Prompt       string `json:"prompt,omitempty"`

	// Relations
	Story      *Story      `json:"story,omitempty"`
	Characters []Character `json:"characters,omitempty"`
}

// StoryLike 故事点赞
type StoryLike struct {
	// Base model fields
	common.BaseModel

	UserID  string `json:"userId"`
	StoryID string `json:"storyId"`

	// Relations
	User  *User  `json:"user,omitempty"`
	Story *Story `json:"story,omitempty"`
}

// StoryFollow 故事关注
type StoryFollow struct {
	// Base model fields
	common.BaseModel

	UserID  string `json:"userId"`
	StoryID string `json:"storyId"`

	// Relations
	User  *User  `json:"user,omitempty"`
	Story *Story `json:"story,omitempty"`
}

// StoryPublication 故事发布记录
type StoryPublication struct {
	// Base model fields
	common.BaseModel

	StoryID       string `json:"storyId"`
	Version       int    `json:"version"` // 发布版本号
	Status        string `json:"status"`  // published, unpublished
	RenderTaskID  string `json:"renderTaskId,omitempty"`
	PublishedAt   int64  `json:"publishedAt"`
	UnpublishedAt *int64 `json:"unpublishedAt,omitempty"`

	// Relations
	Story *Story `json:"story,omitempty"`
}

// StoryContributorRole 故事贡献者角色
type StoryContributorRole string

const (
	StoryRoleOwner        StoryContributorRole = "owner"
	StoryRoleCollaborator StoryContributorRole = "collaborator"
	StoryRoleContributor  StoryContributorRole = "contributor"
)

// StoryContributor 故事贡献者（创建故事板或初始化故事的参与者）
type StoryContributor struct {
	// Base model fields
	common.BaseModel

	StoryID   string               `json:"storyId"`
	UserID    string               `json:"userId"`
	Role      StoryContributorRole `json:"role"`
	InvitedBy string               `json:"invitedBy"`
	JoinedAt  int64                `json:"joinedAt"`

	// Flattened fields for client display (populated from User relation)
	Name       string               `json:"name,omitempty"`
	Avatar     string               `json:"avatar,omitempty"`
	BadgeStyle StoryContributorRole `json:"badge_style,omitempty"`

	// Relations
	User    *User  `json:"user,omitempty"`
	Inviter *User  `json:"inviter,omitempty"`
	Story   *Story `json:"story,omitempty"`
}

// StoryVisibility 故事可见性 - 注意：使用 VisibilityType from user_models.go
type StoryVisibility string

const (
	StoryVisibilityPublic    StoryVisibility = "public"    // 公开 - 所有人可见
	StoryVisibilityFollowers StoryVisibility = "followers" // 仅关注者 - 只有关注你的人可见
	StoryVisibilityPrivate   StoryVisibility = "private"   // 私密 - 仅自己可见
	// Deprecated: Use StoryVisibilityFollowers instead
	StoryVisibilityUnlisted StoryVisibility = "unlisted"
)

// ValidStoryVisibility returns true if the visibility string is valid
func ValidStoryVisibility(visibility string) bool {
	switch visibility {
	case string(StoryVisibilityPublic), string(StoryVisibilityFollowers), string(StoryVisibilityPrivate):
		return true
	default:
		return false
	}
}

// StoryStatus 故事状态 - 使用 common.ContentStatus 作为类型别名以保持向后兼容
type StoryStatus = common.ContentStatus

const (
	StoryStatusDraft     StoryStatus = common.ContentStatusDraft
	StoryStatusPublished StoryStatus = common.ContentStatusPublished
	StoryStatusRendering StoryStatus = common.ContentStatusRendering
)

// MARK: - StoryCreationAppUI Alignment Types

// StoryCharacter 故事角色简要信息 (用于故事卡片显示)
// Aligns with StoryCreationAppUI StoryCharacter interface
type StoryCharacter struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar,omitempty"`
	Role   string `json:"role,omitempty"`
}

// StoryForkItem 故事分支项 (用于分支列表显示)
// Aligns with StoryCreationAppUI StoryForkItem interface
type StoryForkItem struct {
	ID           string `json:"id"`
	AuthorName   string `json:"authorName"`
	AuthorAvatar string `json:"authorAvatar,omitempty"`
	Direction    string `json:"direction"`             // 分支走向描述
	CoverImg     string `json:"coverImg,omitempty"`
	PanelCount   int    `json:"panelCount,omitempty"`
	Likes        int    `json:"likes,omitempty"`
	CreatedAt    int64  `json:"createdAt,omitempty"`
}
