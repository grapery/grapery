package domain

// User represents a storyteller profile
type User struct {
	ID                  string `json:"id"`
	Username            string `json:"username"`
	Email               string `json:"email"`
	PasswordHash        string `json:"-"`
	DisplayName         string `json:"displayName"`
	Avatar              string `json:"avatar,omitempty"`
	Background          string `json:"background,omitempty"`
	Bio                 string `json:"bio,omitempty"`
	Location            string `json:"location,omitempty"`
	Website             string `json:"website,omitempty"`
	AIPromptPreferences string `json:"aiPromptPreferences,omitempty"`
	DateOfBirth         *int64 `json:"dateOfBirth,omitempty"`
	Followers           int    `json:"followers"`
	Following           int    `json:"following"`
	StoryboardCount     int    `json:"storyboardCount"` // Number of storyboards created by this user
	Status              string `json:"status"`          // active, suspended, deleted
	EmailVerified       bool   `json:"emailVerified"`
	LastLoginAt         *int64 `json:"lastLoginAt,omitempty"`
	CreatedAt           int64  `json:"createdAt"`
	UpdatedAt           int64  `json:"updatedAt"`

	// 非持久化字段，用于API响应
	IsFollowing *bool `json:"isFollowing,omitempty"` // 当前用户是否关注此用户
}

// UserFollow 用户关注关系
type UserFollow struct {
	ID         string `json:"id"`
	FollowerID string `json:"followerId"` // 关注者
	FolloweeID string `json:"followeeId"` // 被关注者
	CreatedAt  int64  `json:"createdAt"`

	// Relations
	Follower *User `json:"follower,omitempty"`
	Followee *User `json:"followee,omitempty"`
}

// UserSettings 用户设置
type UserSettings struct {
	ID                 string `json:"id"`
	UserID             string `json:"userId"`
	Language           string `json:"language"` // en, zh, ja
	Theme              string `json:"theme"`    // light, dark, auto
	EmailNotifications bool   `json:"emailNotifications"`
	PushNotifications  bool   `json:"pushNotifications"`
	ShowAdultContent   bool   `json:"showAdultContent"`
	ProfileVisibility  string `json:"profileVisibility"` // public, friends, private
	AllowComments      bool   `json:"allowComments"`
	AllowMessages      bool   `json:"allowMessages"`
	ShowOnlineStatus   bool   `json:"showOnlineStatus"`
	UpdatedAt          int64  `json:"updatedAt"`

	// Relations
	User *User `json:"user,omitempty"`
}

// UserActivity 用户活动记录
type UserActivity struct {
	ID          string `json:"id"`
	UserID      string `json:"userId"`
	Type        string `json:"type"` // story_created, story_updated, story_published, story_liked, character_created, character_updated, user_followed, storyboard_created, panel_added
	TargetID    string `json:"targetId,omitempty"`
	TargetType  string `json:"targetType,omitempty"` // story, character, user, storyboard
	TargetTitle string `json:"targetTitle,omitempty"`
	Message     string `json:"message,omitempty"`
	CreatedAt   int64  `json:"createdAt"`

	// Relations
	User *User `json:"user,omitempty"`
}
