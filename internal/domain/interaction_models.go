package domain

// Follow 关注关系（多态关联）
type Follow struct {
	ID                   string         `json:"id"`
	FollowerID           string         `json:"followerId"`
	FollowableType       FollowableType `json:"followableType"` // story, user, character
	FollowableID         string         `json:"followableId"`
	NotificationsEnabled bool           `json:"notificationsEnabled"`
	CreatedAt            int64          `json:"createdAt"`
}

// FollowableType 可关注对象类型
type FollowableType string

const (
	FollowableTypeStory     FollowableType = "story"
	FollowableTypeUser      FollowableType = "user"
	FollowableTypeCharacter FollowableType = "character"
)

// Like 点赞关系（多态关联）
type Like struct {
	ID           string       `json:"id"`
	UserID       string       `json:"userId"`
	LikeableType LikeableType `json:"likeableType"` // story, character, storyboard_node, fragment, character_poster
	LikeableID   string       `json:"likeableId"`
	CreatedAt    int64        `json:"createdAt"`
}

// LikeableType 可点赞对象类型
type LikeableType string

const (
	LikeableTypeStory           LikeableType = "story"
	LikeableTypeCharacter       LikeableType = "character"
	LikeableTypeStoryboardNode  LikeableType = "storyboard_node"
	LikeableTypeFragment        LikeableType = "fragment"
	LikeableTypeCharacterPoster LikeableType = "character_poster"
)

// FollowCount 关注统计（冗余表）
type FollowCount struct {
	FollowableType FollowableType `json:"followableType"`
	FollowableID   string         `json:"followableId"`
	FollowersCount int            `json:"followersCount"`
	UpdatedAt      int64          `json:"updatedAt"`
}

// LikeCount 点赞统计（冗余表）
type LikeCount struct {
	LikeableType LikeableType `json:"likeableType"`
	LikeableID   string       `json:"likeableId"`
	LikesCount   int          `json:"likesCount"`
	UpdatedAt    int64        `json:"updatedAt"`
}

// MARK: - Bookmark (Save) Models - StoryCreationAppUI Alignment

// Bookmark 收藏/保存关系（多态关联）
type Bookmark struct {
	ID             string       `json:"id"`
	UserID         string       `json:"userId"`
	BookmarkType   BookmarkType `json:"bookmarkType"`   // story, fragment, storyboard
	BookmarkID     string       `json:"bookmarkId"`     // The ID of the bookmarked item
	CollectionName string       `json:"collectionName"` // Optional collection/folder name
	CreatedAt      int64        `json:"createdAt"`
}

// BookmarkType 可收藏对象类型
type BookmarkType string

const (
	BookmarkTypeStory      BookmarkType = "story"
	BookmarkTypeFragment   BookmarkType = "fragment"
	BookmarkTypeStoryboard BookmarkType = "storyboard"
)

// BookmarkCount 收藏统计（冗余表）
type BookmarkCount struct {
	BookmarkType   BookmarkType `json:"bookmarkType"`
	BookmarkID     string       `json:"bookmarkId"`
	SavesCount     int          `json:"savesCount"`
	BookmarksCount int          `json:"bookmarksCount"` // Alias for API compatibility
	UpdatedAt      int64        `json:"updatedAt"`
}
