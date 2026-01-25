package domain

// Story captures a creation project
type Story struct {
	ID                  string       `json:"id"`
	AuthorID            string       `json:"-"`
	GroupID             string       `json:"groupId,omitempty"` // Group ID if the story belongs to a group
	Title               string       `json:"title"`
	Description         string       `json:"description"`
	CoverImage          string       `json:"coverImage"`
	Likes               int          `json:"likes"`
	Followers           int          `json:"followers"`
	Panels              int          `json:"panels"`
	StoryboardCount     int          `json:"storyboardCount"`   // Number of storyboards in this story
	CharacterCount      int          `json:"characterCount"`    // Number of characters in the story
	DefaultSceneCount   int          `json:"defaultSceneCount"` // Default number of scenes for storyboards (2-8, default 3)
	Genre               string       `json:"genre"`
	Style               *StyleConfig `json:"style,omitempty"`     // AI生成风格配置（完整信息，可为空）
	Status              string       `json:"status"`              // draft, published, rendering
	IsCollaborationOpen bool         `json:"isCollaborationOpen"` // Whether collaboration is open: true=anyone can edit, false=only author and group members can edit
	CreatedAt           int64        `json:"createdAt"`
	UpdatedAt           int64        `json:"updatedAt"`

	// AI 丰富相关字段
	OriginalDescription string `json:"originalDescription,omitempty"` // 用户原始描述（AI丰富前）
	EnrichedDescription string `json:"enrichedDescription,omitempty"` // AI 丰富后的描述
	IsAIEnriched        bool   `json:"isAIEnriched"`                  // 是否使用AI丰富过
	AIEnrichedAt        *int64 `json:"aiEnrichedAt,omitempty"`        // AI 丰富时间
	CoverGeneratedByAI  bool   `json:"coverGeneratedByAI"`            // 封面是否由AI生成
	PosterImage         string `json:"posterImage,omitempty"`         // AI 生成的海报图片
	BackgroundImage     string `json:"backgroundImage,omitempty"`     // AI 生成的背景图片

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
}

// Panel describes a storyboard panel
type Panel struct {
	ID        string `json:"id"`
	StoryID   string `json:"storyId"`
	Sequence  int    `json:"sequence"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	Image     string `json:"image,omitempty"`
	Likes     int    `json:"likes"`
	Published bool   `json:"isPublished"`
	CreatedAt int64  `json:"createdAt"`

	// Relations
	Story      *Story      `json:"story,omitempty"`
	Characters []Character `json:"characters,omitempty"`
}

// StoryLike 故事点赞
type StoryLike struct {
	ID        string `json:"id"`
	UserID    string `json:"userId"`
	StoryID   string `json:"storyId"`
	CreatedAt int64  `json:"createdAt"`

	// Relations
	User  *User  `json:"user,omitempty"`
	Story *Story `json:"story,omitempty"`
}

// StoryFollow 故事关注
type StoryFollow struct {
	ID        string `json:"id"`
	UserID    string `json:"userId"`
	StoryID   string `json:"storyId"`
	CreatedAt int64  `json:"createdAt"`

	// Relations
	User  *User  `json:"user,omitempty"`
	Story *Story `json:"story,omitempty"`
}

// StoryPublication 故事发布记录
type StoryPublication struct {
	ID            string `json:"id"`
	StoryID       string `json:"storyId"`
	Version       int    `json:"version"` // 发布版本号
	Status        string `json:"status"`  // published, unpublished
	RenderTaskID  string `json:"renderTaskId,omitempty"`
	PublishedAt   int64  `json:"publishedAt"`
	UpdatedAt     int64  `json:"updatedAt"`
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
	ID        string               `json:"id"`
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
