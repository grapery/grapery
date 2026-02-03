package domain

import "time"

// FragmentSourceType 片段来源类型
type FragmentSourceType string

const (
	FragmentSourceOriginal       FragmentSourceType = "original"
	FragmentSourceStoryExcerpt   FragmentSourceType = "story_excerpt"
	FragmentSourceStoryboardNode FragmentSourceType = "storyboard_node"
)

// FragmentStatus 片段状态
type FragmentStatus string

const (
	FragmentStatusActive  FragmentStatus = "active"
	FragmentStatusDeleted FragmentStatus = "deleted"
)

// FragmentVisibility 片段可见性（使用无类型字符串常量以保持兼容性）
const (
	FragmentVisibilityPublic    = "public"
	FragmentVisibilityFollowers = "followers_only"
	FragmentVisibilityPrivate   = "private"
)

// Fragment represents a fragment story - short, complete stories shared by users
type Fragment struct {
	ID            string   `json:"id"`
	AuthorID      string   `json:"authorId"`      // 作者ID
	Content       string   `json:"content"`       // 内容
	MediaURLs     []string `json:"mediaUrls"`     // 媒体URL列表
	SourceType    string   `json:"sourceType"`    // 来源类型: original, story_excerpt, storyboard_node
	SourceID      string   `json:"sourceId"`      // 来源ID
	Visibility    string   `json:"visibility"`    // 可见性: public, followers_only, private
	Status        string   `json:"status"`        // 状态: active, deleted
	ViewsCount    int      `json:"viewsCount"`    // 浏览数
	LikesCount    int      `json:"likesCount"`    // 点赞数
	CommentsCount int      `json:"commentsCount"` // 评论数
	SharesCount   int      `json:"sharesCount"`   // 分享数
	CreatedAt     int64    `json:"createdAt"`     // 创建时间
	UpdatedAt     int64    `json:"updatedAt"`     // 更新时间

	// 向后兼容字段（内部使用）
	CreatorID string    `json:"creatorId" gorm:"column:creator_id;type:varchar(36);not null;index"` // 兼容旧代码
	ImageUrls string    `json:"imageUrls" gorm:"column:image_urls;type:text"`                      // 兼容旧代码
	Style     *string   `json:"style,omitempty" gorm:"column:style;type:varchar(50)"`              // 兼容旧代码
	FragmentCount *int  `json:"fragmentCount,omitempty" gorm:"column:fragment_count;type:int;default:1"` // 兼容旧代码
	Likes     int       `json:"likes,omitempty" gorm:"column:likes;type:int;default:0"`            // 兼容旧代码
	Comments  int       `json:"comments,omitempty" gorm:"column:comments;type:int;default:0"`      // 兼容旧代码
	Shares    int       `json:"shares,omitempty" gorm:"column:shares;type:int;default:0"`          // 兼容旧代码
	Views     int       `json:"views,omitempty" gorm:"column:views;type:int;default:0"`            // 兼容旧代码
	IsLiked   *bool     `json:"isLiked,omitempty" gorm:"-"`                                        // 兼容旧代码
	CreatedAtTime time.Time `json:"-" gorm:"column:created_at;type:datetime;autoCreateTime"`       // 兼容旧代码
	UpdatedAtTime time.Time `json:"-" gorm:"column:updated_at;type:datetime;autoUpdateTime"`       // 兼容旧代码

	// 非持久化字段
	IsLikedNew *bool `json:"isLiked,omitempty" gorm:"-"` // 当前用户是否点赞
	Author     *User `json:"author,omitempty"`           // 作者信息
}

// TableName specifies the table name for Fragment
func (Fragment) TableName() string {
	return "fragments"
}

// ValidFragmentVisibility returns true if the visibility string is valid
func ValidFragmentVisibility(visibility string) bool {
	switch visibility {
	case string(FragmentVisibilityPublic), string(FragmentVisibilityFollowers), string(FragmentVisibilityPrivate):
		return true
	default:
		return false
	}
}

// ConvertFragmentRequest 碎片转故事请求
type ConvertFragmentRequest struct {
	Title             string `json:"title" binding:"required"`           // 故事标题
	Description       string `json:"description,omitempty"`              // 故事描述
	Genre             string `json:"genre,omitempty"`                    // 故事类型
	CoverImage        string `json:"coverImage,omitempty"`               // 封面图片
	SceneCount        int    `json:"sceneCount,omitempty"`               // 场景数量 (2-8, 默认3)
	IsAIEnabled       bool   `json:"isAIEnabled,omitempty"`              // 是否启用AI
	CollaborationType string `json:"collaborationType,omitempty"`        // 协作类型: open, restricted, closed
}

// ConvertFragmentResponse 碎片转故事响应
type ConvertFragmentResponse struct {
	Story      *Story      `json:"story"`      // 创建的故事
	Storyboard *Storyboard `json:"storyboard"` // 创建的故事板
	FragmentID string      `json:"fragmentId"` // 原碎片ID
}
