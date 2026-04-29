package domain

import "github.com/grapestree/fgrapery/grapery/internal/common"

// Notification 通知
type Notification struct {
	// Base model fields
	common.BaseModel

	UserID      string `json:"userId"`
	Type        string `json:"type"` // like, comment, follow, story_update, system
	Title       string `json:"title"`
	Content     string `json:"content"`
	Link        string `json:"link,omitempty"`
	Read        bool   `json:"read"`
	ActorID     string `json:"actorId,omitempty"` // 触发通知的用户ID
	ActorName   string `json:"actorName,omitempty"`
	ActorAvatar string `json:"actorAvatar,omitempty"`

	// Story context (for like, comment, story_update types)
	StoryTitle   string `json:"storyTitle,omitempty"`
	StoryCover   string `json:"storyCover,omitempty"`
	StoryID      string `json:"storyId,omitempty"`
	CommentText  string `json:"commentText,omitempty"` // 评论内容摘要
	RelatedCommentID string `json:"relatedCommentId,omitempty"`
	StoryboardID string `json:"storyboardId,omitempty"`
	// StoryboardTitle 分镜标题（用于站内信/推送展示，可与 StoryTitle 并存）
	StoryboardTitle string `json:"storyboardTitle,omitempty"`
	FragmentID string `json:"fragmentId,omitempty"`
	// TokensUsed AI 生成类通知的 token 消耗（约），0 表示未统计或不展示
	TokensUsed int `json:"tokensUsed,omitempty"`

	// System notification fields (for system type)
	SysTitle string `json:"sysTitle,omitempty"`
	SysBody  string `json:"sysBody,omitempty"`
	SysIcon  string `json:"sysIcon,omitempty"` // icon name: gift, star, trending, etc.

	// Relations
	User  *User `json:"user,omitempty"`
	Actor *User `json:"actor,omitempty"`
}
