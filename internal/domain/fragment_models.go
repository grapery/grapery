package domain

import (
	"encoding/json"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/common"
)

// FragmentSourceType 片段来源类型
type FragmentSourceType string

const (
	FragmentSourceOriginal       FragmentSourceType = "original"
	FragmentSourceStoryExcerpt   FragmentSourceType = "story_excerpt"
	FragmentSourceStoryboardNode FragmentSourceType = "storyboard_node"
)

// FragmentStatus 片段状态 - 使用 common.BaseStatus 作为类型别名以保持向后兼容
type FragmentStatus = common.BaseStatus

const (
	FragmentStatusActive  FragmentStatus = common.StatusActive
	FragmentStatusDeleted FragmentStatus = common.StatusDeleted
)

// FragmentVisibility 片段可见性（使用无类型字符串常量以保持兼容性）
const (
	FragmentVisibilityPublic    = "public"
	FragmentVisibilityFollowers = "followers"
	// FragmentVisibilityFollowersLegacy 保留兼容旧数据/旧客户端
	FragmentVisibilityFollowersLegacy = "followers_only"
	FragmentVisibilityPrivate   = "private"
)

// Fragment represents a fragment story - short, complete stories shared by users
type Fragment struct {
	// Base model fields (ID, CreatedAt, UpdatedAt as int64)
	common.BaseModel

	UserID     string   `json:"authorId"`   // 作者ID - 保持 JSON 标签为 authorId 以保持 API 兼容性
	Content    string   `json:"content"`    // 内容
	MediaURLs  []string `json:"mediaUrls"`  // 媒体URL列表
	SourceType string   `json:"sourceType"` // 来源类型: original, story_excerpt, storyboard_node
	SourceID   string   `json:"sourceId"`   // 来源ID
	Visibility string   `json:"visibility"` // 可见性: public, followers_only, private
	Status     string   `json:"status"`     // 状态: active, deleted

	// Engagement stats fields
	common.EngagementStats

	// MARK: - StoryCreationAppUI Alignment Fields
	Saves      int    `json:"saves"`                // 收藏/保存数
	Topic      string `json:"topic,omitempty"`      // 话题标签
	Caption    string `json:"caption,omitempty"`    // 标题/简介文字
	IsDraft    bool   `json:"isDraft" gorm:"column:is_draft;type:tinyint(1);default:0;index"`       // 是否为草稿 (StoryCreationAppUI alignment)
	DraftCount int    `json:"draftCount" gorm:"column:draft_count;type:int;default:0"`             // 草稿数量 (StoryCreationAppUI alignment)

	ConvertedToStoryID *string `json:"convertedToStoryId,omitempty" gorm:"column:converted_to_story_id;type:varchar(36);index"` // 转换为的故事ID
	IsConverted        bool    `json:"isConverted" gorm:"column:is_converted;type:tinyint(1);default:0;index"`                  // 是否已转换为故事

	// 向后兼容字段（内部使用）
	CreatorID     string    `json:"-" gorm:"column:creator_id;type:varchar(36);not null;index"`              // 兼容旧代码 - 不在 JSON 中暴露
	ImageUrls     string    `json:"imageUrls" gorm:"column:image_urls;type:text"`                            // 兼容旧代码
	Style         *string   `json:"style,omitempty" gorm:"column:style;type:varchar(50)"`                    // 兼容旧代码
	FragmentCount *int      `json:"fragmentCount,omitempty" gorm:"column:fragment_count;type:int;default:1"` // 兼容旧代码
	IsLiked       *bool     `json:"isLiked,omitempty" gorm:"-"`                                              // 兼容旧代码
	CreatedAtTime time.Time `json:"-" gorm:"column:created_at;type:datetime;autoCreateTime"`                 // 兼容旧代码
	UpdatedAtTime time.Time `json:"-" gorm:"column:updated_at;type:datetime;autoUpdateTime"`                 // 兼容旧代码

	// 非持久化字段
	IsLikedNew *bool `json:"isLikedNew,omitempty" gorm:"-"` // 当前用户是否点赞
	Author     *User `json:"author,omitempty"`              // 作者信息
}

// MarshalJSON customizes JSON output for iOS/Voyager compatibility:
// - Adds creatorId (alias of authorId)
// - Adds creatorName and creatorAvatar from Author
// - Ensures imageUrls is []string, not a JSON string
func (f *Fragment) MarshalJSON() ([]byte, error) {
	type fragmentAlias Fragment
	imageUrls := f.MediaURLs
	if len(imageUrls) == 0 && f.ImageUrls != "" {
		_ = json.Unmarshal([]byte(f.ImageUrls), &imageUrls)
	}

	// 提取创建者信息
	var creatorName, creatorAvatar string
	if f.Author != nil {
		creatorName = f.Author.DisplayName
		creatorAvatar = f.Author.Avatar
	}

	return json.Marshal(struct {
		*fragmentAlias
		CreatorID     string   `json:"creatorId"`
		CreatorName   string   `json:"creatorName,omitempty"`
		CreatorAvatar string   `json:"creatorAvatar,omitempty"`
		ImageUrls     []string `json:"imageUrls"`
	}{
		fragmentAlias:  (*fragmentAlias)(f),
		CreatorID:      f.UserID,
		CreatorName:    creatorName,
		CreatorAvatar:  creatorAvatar,
		ImageUrls:      imageUrls,
	})
}

// GetLikesCount returns the like count (alias for API compatibility)
func (f *Fragment) GetLikesCount() int {
	return f.Likes
}

// GetCommentsCount returns the comment count (alias for API compatibility)
func (f *Fragment) GetCommentsCount() int {
	return f.Comments
}

// GetSharesCount returns the share count (alias for API compatibility)
func (f *Fragment) GetSharesCount() int {
	return f.Shares
}

// GetViewsCount returns the view count (alias for API compatibility)
func (f *Fragment) GetViewsCount() int {
	return f.Views
}

// TableName specifies the table name for Fragment
func (Fragment) TableName() string {
	return "fragments"
}

// ValidFragmentVisibility returns true if the visibility string is valid
func ValidFragmentVisibility(visibility string) bool {
	switch visibility {
	case string(FragmentVisibilityPublic),
		string(FragmentVisibilityFollowers),
		string(FragmentVisibilityFollowersLegacy),
		string(FragmentVisibilityPrivate):
		return true
	default:
		return false
	}
}

// NormalizeFragmentVisibility normalizes legacy visibility values.
func NormalizeFragmentVisibility(visibility string) string {
	switch visibility {
	case FragmentVisibilityFollowersLegacy:
		return FragmentVisibilityFollowers
	default:
		return visibility
	}
}

// ConvertFragmentRequest 碎片转故事请求（与 iOS FragmentConvertToStorySheet / creation 客户端 JSON 对齐）
type ConvertFragmentRequest struct {
	Title             string `json:"title" binding:"required"`    // 故事标题
	Description       string `json:"description,omitempty"`       // 故事描述（可为 caption + 正文合并）
	Genre             string `json:"genre,omitempty"`             // 故事类型（客户端可用碎片 topic）
	CoverImage        string `json:"coverImage,omitempty"`        // 封面图片 URL
	SceneCount        int    `json:"sceneCount,omitempty"`        // 场景数量 (2-8, 默认3)
	UseAI             bool   `json:"useAI,omitempty"`             // AI 一键续写入口应传 true，写入 Story.UseAI
	CollaborationType string `json:"collaborationType,omitempty"` // 协作类型: open, restricted, closed
}

// ConvertFragmentResponse 碎片转故事响应
// 故事板由用户在故事内自行创建；转换接口仅创建 Story，不再自动创建根故事板。
type ConvertFragmentResponse struct {
	Story      *Story      `json:"story"`                 // 创建的故事
	Storyboard *Storyboard `json:"storyboard,omitempty"` // 已废弃：始终为空，保留字段仅兼容旧客户端
	FragmentID string      `json:"fragmentId"`           // 原碎片ID
}
