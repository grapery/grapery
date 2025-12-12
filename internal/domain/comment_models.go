package domain

// Comment models threaded feedback
type Comment struct {
	ID         string `json:"id"`
	AuthorID   string `json:"authorId"`
	Content    string `json:"content"`
	TargetType string `json:"targetType"` // story, storyboard, character, comment
	TargetID   string `json:"targetId"`
	ParentID   string `json:"parentId"` // empty for top-level comments
	RootID     string `json:"rootId"`   // root comment ID for nested replies
	Likes      int    `json:"likes"`
	Dislikes   int    `json:"dislikes"`
	ReplyCount int    `json:"replyCount"` // 回复数量
	IsLiked    bool   `json:"isLiked"`    // 当前用户是否点赞 (非数据库字段)
	IsDisliked bool   `json:"isDisliked"` // 当前用户是否踩 (非数据库字段)
	CreatedAt  int64  `json:"createdAt"`
	UpdatedAt  int64  `json:"updatedAt"`

	// Relations
	Author  *User     `json:"author,omitempty"`
	Replies []Comment `json:"replies,omitempty"` // 嵌套回复（可选加载）
}

// CommentLike 评论点赞
type CommentLike struct {
	ID        string `json:"id"`
	UserID    string `json:"userId"`
	CommentID string `json:"commentId"`
	IsLike    bool   `json:"isLike"` // true=like, false=dislike
	CreatedAt int64  `json:"createdAt"`

	// Relations
	User    *User    `json:"user,omitempty"`
	Comment *Comment `json:"comment,omitempty"`
}
