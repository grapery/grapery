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
	FragmentsCount      int    `json:"fragmentsCount"`  // Number of fragments created by this user
	GroupsCount         int    `json:"groupsCount"`     // Number of groups the user has joined
	GroupsCreated       int    `json:"groupsCreated"`   // Number of groups created by this user
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
	Language           string `json:"language"` // en, zh-CN, zh-TW, ja, ko
	Theme              string `json:"theme"`    // light, dark, system
	FontSize           string `json:"fontSize"` // small, medium, large
	DataSaver          bool   `json:"dataSaver"`

	// 隐私设置
	ProfileVisibility         string `json:"profileVisibility"`         // public, followers_only, private
	DefaultStoryVisibility    string `json:"defaultStoryVisibility"`    // public, unlisted, private
	DefaultFragmentVisibility string `json:"defaultFragmentVisibility"` // public, followers_only, private
	AllowFollowFrom           string `json:"allowFollowFrom"`           // everyone, followers_only, followers_of_followers, no_one
	AllowCommentsFrom         string `json:"allowCommentsFrom"`         // everyone, followers_only, followers_of_followers, no_one
	AllowMessagesFrom         string `json:"allowMessagesFrom"`         // everyone, followers_only, followers_of_followers, no_one
	ShowOnlineStatus          bool   `json:"showOnlineStatus"`
	ShowReadReceipts          bool   `json:"showReadReceipts"`

	// AI设置
	AIEnabled     bool `json:"aiEnabled"`
	AIDataSharing bool `json:"aiDataSharing"`

	// 通知设置 (JSON)
	NotificationSettings string `json:"notificationSettings"` // JSON string

	UpdatedAt int64 `json:"updatedAt"`

	// Relations
	User *User `json:"user,omitempty"`

	// 向后兼容字段（内部使用）
	EmailNotifications bool   `json:"emailNotifications"` // 兼容旧代码
	PushNotifications  bool   `json:"pushNotifications"`  // 兼容旧代码
	ShowAdultContent   bool   `json:"showAdultContent"`   // 兼容旧代码
	AllowComments      bool   `json:"allowComments"`      // 兼容旧代码
	AllowMessages      bool   `json:"allowMessages"`      // 兼容旧代码
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

// DevicePlatform 设备平台类型
type DevicePlatform string

const (
	PlatformIOS     DevicePlatform = "ios"
	PlatformAndroid DevicePlatform = "android"
	PlatformMacOS   DevicePlatform = "macos"
	PlatformIPadOS  DevicePlatform = "ipados"
	PlatformWatchOS DevicePlatform = "watchos"
	PlatformTVOS    DevicePlatform = "tvos"
	PlatformWeb     DevicePlatform = "web"
)

// UserDevice 用户设备信息（用于推送通知）
type UserDevice struct {
	ID           string         `json:"id"`
	UserID       string         `json:"userId"`
	DeviceToken  string         `json:"deviceToken"`           // APNs token 或 FCM token
	Platform     DevicePlatform `json:"platform"`              // ios, android, macos, etc.
	PushProvider string         `json:"pushProvider"`          // apns, fcm
	DeviceModel  string         `json:"deviceModel,omitempty"` // iPhone 14 Pro, Pixel 7, etc.
	OSVersion    string         `json:"osVersion,omitempty"`   // iOS 17.0, Android 14, etc.
	AppVersion   string         `json:"appVersion,omitempty"`  // 1.0.0
	AppBuild     string         `json:"appBuild,omitempty"`    // 100
	Locale       string         `json:"locale,omitempty"`      // en-US, zh-CN
	Timezone     string         `json:"timezone,omitempty"`    // Asia/Shanghai
	IsActive     bool           `json:"isActive"`              // 设备是否活跃
	LastActiveAt int64          `json:"lastActiveAt"`          // 最后活跃时间
	CreatedAt    int64          `json:"createdAt"`
	UpdatedAt    int64          `json:"updatedAt"`

	// Relations
	User *User `json:"user,omitempty"`
}

// PushNotificationPayload 推送通知载荷
type PushNotificationPayload struct {
	Title    string            `json:"title"`
	Body     string            `json:"body"`
	Sound    string            `json:"sound,omitempty"`
	Badge    int               `json:"badge,omitempty"`
	Category string            `json:"category,omitempty"`
	ThreadID string            `json:"threadId,omitempty"`
	Data     map[string]string `json:"data,omitempty"` // 自定义数据
}

// PushNotificationResult 推送结果
type PushNotificationResult struct {
	DeviceID    string `json:"deviceId"`
	DeviceToken string `json:"deviceToken"`
	Success     bool   `json:"success"`
	Error       string `json:"error,omitempty"`
	MessageID   string `json:"messageId,omitempty"`
}

// VisibilityType 可见性类型
type VisibilityType string

const (
	VisibilityPublic        VisibilityType = "public"
	VisibilityFollowersOnly VisibilityType = "followers_only"
	VisibilityPrivate       VisibilityType = "private"
	VisibilityUnlisted      VisibilityType = "unlisted"
)

// AllowFromType 允许来源类型
type AllowFromType string

const (
	AllowFromEveryone             AllowFromType = "everyone"
	AllowFromFollowersOnly        AllowFromType = "followers_only"
	AllowFromFollowersOfFollowers AllowFromType = "followers_of_followers"
	AllowFromNoOne                AllowFromType = "no_one"
)

// FontSizeType 字体大小类型
type FontSizeType string

const (
	FontSizeSmall  FontSizeType = "small"
	FontSizeMedium FontSizeType = "medium"
	FontSizeLarge  FontSizeType = "large"
)

// ThemeType 主题类型
type ThemeType string

const (
	ThemeLight  ThemeType = "light"
	ThemeDark   ThemeType = "dark"
	ThemeSystem ThemeType = "system"
)

// LanguageType 语言类型
type LanguageType string

const (
	LanguageEnglish   LanguageType = "en"
	LanguageChineseCN LanguageType = "zh-CN"
	LanguageChineseTW LanguageType = "zh-TW"
	LanguageJapanese  LanguageType = "ja"
	LanguageKorean    LanguageType = "ko"
)
