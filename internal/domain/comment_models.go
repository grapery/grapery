package domain

import "github.com/grapestree/fgrapery/grapery/internal/common"

// Comment models threaded feedback
type Comment struct {
	// Base model fields
	common.BaseModel

	UserID     string `json:"authorId"` // 保持 JSON 标签为 authorId 以保持 API 兼容性
	Content    string `json:"content"`
	TargetType string `json:"targetType"` // story, storyboard, character, comment
	TargetID   string `json:"targetId"`
	ParentID   string `json:"parentId"` // empty for top-level comments
	RootID     string `json:"rootId"`   // root comment ID for nested replies

	// Partial engagement stats (using only Likes, Dislikes)
	Likes    int `json:"likes"`
	Dislikes int `json:"dislikes"`

	ReplyCount int  `json:"replyCount"` // 回复数量
	IsLiked    bool `json:"isLiked"`    // 当前用户是否点赞 (非数据库字段)
	IsDisliked bool `json:"isDisliked"` // 当前用户是否踩 (非数据库字段)

	// Relations
	Author  *User     `json:"author,omitempty"`
	Replies []Comment `json:"replies,omitempty"` // 嵌套回复（可选加载）
}

// CommentLike 评论点赞
type CommentLike struct {
	// Base model fields
	common.BaseModel

	UserID    string `json:"userId"`
	CommentID string `json:"commentId"`
	IsLike    bool   `json:"isLike"` // true=like, false=dislike

	// Relations
	User    *User    `json:"user,omitempty"`
	Comment *Comment `json:"comment,omitempty"`
}

// StoryComment 故事评论（增强版，包含用户信息）
type StoryComment struct {
	common.BaseModel

	StoryID  string `json:"storyId"`
	UserID   string `json:"authorId"`
	Content  string `json:"content"`
	ParentID string `json:"parentId,omitempty"`

	// 用户信息（反规范化，便于展示）
	UserName   string `json:"user"`
	UserAvatar string `json:"avatar,omitempty"`
	UserTag    string `json:"tag,omitempty"` // "作者"等标签

	Likes      int  `json:"likes"`
	ReplyCount int  `json:"replyCount"`
	IsLiked    bool `json:"isLiked"`

	Replies []StoryReply `json:"replies,omitempty"`
}

// StoryReply 故事评论回复
type StoryReply struct {
	common.BaseModel

	CommentID string `json:"commentId"`
	UserID    string `json:"authorId"`
	Content   string `json:"content"`

	UserName   string `json:"user"`
	UserAvatar string `json:"avatar,omitempty"`
	UserTag    string `json:"tag,omitempty"`

	Likes   int  `json:"likes"`
	IsLiked bool `json:"isLiked"`
}
