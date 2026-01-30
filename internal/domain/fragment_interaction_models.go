package domain

// FragmentLike 碎片故事点赞
type FragmentLike struct {
	ID        string `json:"id" gorm:"primaryKey;size:36"`
	FragmentID string `json:"fragmentId" gorm:"size:36;not null;index:idx_fragment_likes_fragment_user"`
	UserID    string `json:"userId" gorm:"size:36;not null;index:idx_fragment_likes_fragment_user;index:idx_fragment_likes_user"`
	CreatedAt int64  `json:"createdAt" gorm:"type:bigint;autoCreateTime"`
}

// FragmentComment 碎片故事评论
type FragmentComment struct {
	ID        string `json:"id" gorm:"primaryKey;size:36"`
	FragmentID string `json:"fragmentId" gorm:"size:36;not null;index:idx_fragment_comments_fragment"`
	UserID    string `json:"userId" gorm:"size:36;not null;index:idx_fragment_comments_user"`
	Content   string `json:"content" gorm:"type:text;not null"`
	ParentID  *string `json:"parentId,omitempty" gorm:"size:36;index:idx_fragment_comments_parent"` // 支持回复评论
	CreatedAt int64  `json:"createdAt" gorm:"type:bigint;autoCreateTime"`
	UpdatedAt int64  `json:"updatedAt" gorm:"type:bigint;autoUpdateTime"`

	// 关联数据（不存储在数据库）
	User       *User                `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Fragment   *Fragment            `json:"fragment,omitempty" gorm:"foreignKey:FragmentID"`
	Replies    []*FragmentComment   `json:"replies,omitempty" gorm:"-"` // 子评论列表
	ReplyCount int                  `json:"replyCount,omitempty" gorm:"-"` // 子评论数量
	IsLiked    bool                 `json:"isLiked,omitempty" gorm:"-"` // 当前用户是否点赞了该评论
	LikeCount  int                  `json:"likeCount,omitempty" gorm:"-"` // 评论点赞数
}

// TableName 指定表名
func (FragmentLike) TableName() string {
	return "fragment_likes"
}

// TableName 指定表名
func (FragmentComment) TableName() string {
	return "fragment_comments"
}

// FragmentStats 碎片故事统计信息
type FragmentStats struct {
	FragmentID     string `json:"fragmentId"`
	LikesCount     int    `json:"likesCount"`
	CommentsCount  int    `json:"commentsCount"`
	SharesCount    int    `json:"sharesCount"`
	IsLikedByUser  bool   `json:"isLikedByUser"`
}

// FragmentShare 碎片故事分享记录
type FragmentShare struct {
	ID         string `json:"id" gorm:"primaryKey;size:36"`
	FragmentID string `json:"fragmentId" gorm:"size:36;not null;index:idx_fragment_shares_fragment"`
	UserID     string `json:"userId" gorm:"size:36;not null;index:idx_fragment_shares_user"`
	Platform   string `json:"platform" gorm:"size:20;not null"` // wechat, twitter, local, etc.
	CreatedAt  int64  `json:"createdAt" gorm:"type:bigint;autoCreateTime"`
}

// TableName 指定表名
func (FragmentShare) TableName() string {
	return "fragment_shares"
}
