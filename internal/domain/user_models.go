package domain

import "github.com/grapestree/fgrapery/grapery/internal/common"

// User represents a storyteller profile
type User struct {
	// Base model fields (ID, CreatedAt, UpdatedAt)
	common.BaseModel

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

	// Social stats fields
	common.SocialStats

	StoryboardCount int    `json:"storyboardCount"` // Number of storyboards created by this user
	FragmentsCount  int    `json:"fragmentsCount"`  // Number of fragments created by this user
	Status          string `json:"status"`          // active, suspended, deleted
	EmailVerified   bool   `json:"emailVerified"`
	LastLoginAt     *int64 `json:"lastLoginAt,omitempty"`

	// StoryCreationAppUI Design - Points system
	Points       int    `json:"points"`       // 未择积分
	ReferralCode string `json:"referralCode"` // 用户专属邀请码

	// 非持久化字段，用于API响应
	IsFollowing *bool `json:"isFollowing,omitempty"` // 当前用户是否关注此用户
}

// UserFollow 用户关注关系
type UserFollow struct {
	// Base model fields
	common.BaseModel

	FollowerID string `json:"followerId"` // 关注者
	FolloweeID string `json:"followeeId"` // 被关注者

	// Relations
	Follower *User `json:"follower,omitempty"`
	Followee *User `json:"followee,omitempty"`
}

// UserSettings 用户设置
type UserSettings struct {
	// Base model fields
	common.BaseModel

	UserID    string `json:"userId"`
	Language  string `json:"language"` // en, zh-CN, zh-TW, ja, ko
	Region    string `json:"region"`   // CN, US, JP, KR, EU, OTHER
	Theme     string `json:"theme"`    // light, dark, system
	FontSize  string `json:"fontSize"` // small, medium, large
	DataSaver bool   `json:"dataSaver"`

	// 隐私设置
	ProfileVisibility         string `json:"profileVisibility"`         // public, followers_only, private
	DefaultStoryVisibility    string `json:"defaultStoryVisibility"`    // public, unlisted, private
	DefaultFragmentVisibility string `json:"defaultFragmentVisibility"` // public, followers_only, private
	AllowFollowFrom           string `json:"allowFollowFrom"`           // everyone, followers_only, followers_of_followers, no_one
	AllowCommentsFrom         string `json:"allowCommentsFrom"`         // everyone, followers_only, no_one
	AllowMessagesFrom         string `json:"allowMessagesFrom"`         // everyone, followers_only, no_one
	ShowOnlineStatus          bool   `json:"showOnlineStatus"`
	ShowReadReceipts          bool   `json:"showReadReceipts"`

	// AI设置
	AIEnabled     bool `json:"aiEnabled"`
	AIDataSharing bool `json:"aiDataSharing"`

	// 通知设置 (JSON)
	NotificationSettings string `json:"notificationSettings"` // JSON string

	// Relations
	User *User `json:"user,omitempty"`

	// 向后兼容字段（内部使用）
	EmailNotifications bool `json:"emailNotifications"` // 兼容旧代码
	PushNotifications  bool `json:"pushNotifications"`  // 兼容旧代码
	ShowAdultContent   bool `json:"showAdultContent"`   // 兼容旧代码
	AllowComments      bool `json:"allowComments"`      // 兼容旧代码
	AllowMessages      bool `json:"allowMessages"`      // 兼容旧代码
}

// REMOVED: UserActivity, ActivityHeatmapData - not in StoryCreationAppUI design

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
	// Base model fields
	common.BaseModel

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
	LanguageSystem    LanguageType = "system"
	LanguageEnglish   LanguageType = "en"
	LanguageChineseCN LanguageType = "zh-Hans"
	LanguageJapanese  LanguageType = "ja"
)

// MARK: - 登录历史管理

// LoginHistory 用户登录历史
type LoginHistory struct {
	// Base model fields (ID, CreatedAt, UpdatedAt)
	common.BaseModel

	UserID       string `json:"userId"`
	DeviceType   string `json:"deviceType"` // iPhone, iPad, Web
	DeviceName   string `json:"deviceName"` // "iPhone 15 Pro"
	DeviceID     string `json:"deviceId"`   // 设备唯一标识
	IPAddress    string `json:"ipAddress"`
	Location     string `json:"location"`
	LoginMethod  string `json:"loginMethod"` // password, apple, google, wechat, sms
	LoggedInAt   int64  `json:"loggedInAt"`
	LastActiveAt int64  `json:"lastActiveAt"`
	LoggedOutAt  int64  `json:"loggedOutAt,omitempty"`
	IsActive     bool   `json:"isActive"`
}

// MARK: - 连接账号管理

// ConnectedAccount 用户已连接的第三方账号
type ConnectedAccount struct {
	// Base model fields
	common.BaseModel

	UserID         string `json:"userId"`
	Provider       string `json:"provider"`       // apple, google, wechat, weibo
	ProviderUserID string `json:"providerUserId"` // 第三方账号ID
	Email          string `json:"email,omitempty"`
	DisplayName    string `json:"displayName,omitempty"`
	AvatarURL      string `json:"avatarUrl,omitempty"`
	ConnectedAt    int64  `json:"connectedAt"`
	LastUsedAt     int64  `json:"lastUsedAt"`
}

// AuthProvider 认证提供商类型
type AuthProvider string

const (
	AuthProviderApple  AuthProvider = "apple"
	AuthProviderGoogle AuthProvider = "google"
	AuthProviderWechat AuthProvider = "wechat"
	AuthProviderWeibo  AuthProvider = "weibo"
)

// MARK: - 账号删除管理

// AccountDeletionRequest 账号删除申请
type AccountDeletionRequest struct {
	// Base model fields
	common.BaseModel

	UserID              string `json:"userId"`
	Reason              string `json:"reason,omitempty"`
	Feedback            string `json:"feedback,omitempty"`
	RequestedAt         int64  `json:"requestedAt"`
	ScheduledDeletionAt int64  `json:"scheduledDeletionAt"`
	Status              string `json:"status"` // pending, processing, completed, cancelled
	CancelledAt         int64  `json:"cancelledAt,omitempty"`
	CancelledReason     string `json:"cancelledReason,omitempty"`
	ProcessedAt         int64  `json:"processedAt,omitempty"`
	ProcessedBy         string `json:"processedBy,omitempty"`
}

// DeletionStatus 删除状态类型 - 使用 common.DeletionStatus 作为类型别名以保持向后兼容
type DeletionStatus = common.DeletionStatus

const (
	DeletionStatusPending    DeletionStatus = common.DeletionStatusPending
	DeletionStatusProcessing DeletionStatus = common.DeletionStatusProcessing
	DeletionStatusCompleted  DeletionStatus = common.DeletionStatusCompleted
	DeletionStatusCancelled  DeletionStatus = common.DeletionStatusCancelled
)

// UserStatus 用户状态类型 - 使用 common.BaseStatus 作为类型别名以保持向后兼容
type UserStatus = common.BaseStatus

const (
	UserStatusActive    UserStatus = common.StatusActive
	UserStatusSuspended UserStatus = common.StatusSuspended
	UserStatusDeleted   UserStatus = common.StatusDeleted
)

// DeletionReason 删除原因类型
type DeletionReason string

const (
	DeletionReasonNotUsing           DeletionReason = "not_using"
	DeletionReasonTooComplicated     DeletionReason = "too_complicated"
	DeletionReasonPrivacyConcern     DeletionReason = "privacy_concern"
	DeletionReasonCreatingNewAccount DeletionReason = "creating_new_account"
	DeletionReasonOther              DeletionReason = "other"
)

// MARK: - 通知设置结构

// NotificationSettings 通知设置结构
type NotificationSettings struct {
	Push  PushNotificationSettings  `json:"push"`
	Email EmailNotificationSettings `json:"email"`
	InApp InAppNotificationSettings `json:"inApp"`
}

// PushNotificationSettings 推送通知设置
type PushNotificationSettings struct {
	Enabled            bool `json:"enabled"`
	NewFollower        bool `json:"newFollower"`
	NewLike            bool `json:"newLike"`
	NewComment         bool `json:"newComment"`
	StoryUpdate        bool `json:"storyUpdate"`
	DirectMessage      bool `json:"directMessage"`
	SystemAnnouncement bool `json:"systemAnnouncement"`
	Marketing          bool `json:"marketing"`
}

// EmailNotificationSettings 邮件通知设置
type EmailNotificationSettings struct {
	Enabled        bool `json:"enabled"`
	WeeklyDigest   bool `json:"weeklyDigest"`
	SecurityAlert  bool `json:"securityAlert"`
	Marketing      bool `json:"marketing"`
	ProductUpdates bool `json:"productUpdates"`
}

// InAppNotificationSettings 站内通知设置
type InAppNotificationSettings struct {
	Enabled          bool `json:"enabled"`
	ShowPreview      bool `json:"showPreview"`
	SoundEnabled     bool `json:"soundEnabled"`
	VibrationEnabled bool `json:"vibrationEnabled"`
}

// REMOVED: ActivityTimeRange, ActivityHeatmapResponse - not in StoryCreationAppUI design
