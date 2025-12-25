package pay

import (
	"time"

	"gorm.io/gorm"
)

// BadgeCategory 徽章类别
type BadgeCategory string

const (
	BadgeCategoryStory      BadgeCategory = "story"      // 故事相关
	BadgeCategoryStoryboard BadgeCategory = "storyboard" // 故事版相关
	BadgeCategoryLike       BadgeCategory = "like"       // 点赞相关
	BadgeCategoryFollower   BadgeCategory = "follower"   // 粉丝相关
	BadgeCategorySocial     BadgeCategory = "social"     // 社交相关
	BadgeCategoryCreator    BadgeCategory = "creator"    // 创作者相关
	BadgeCategorySpecial    BadgeCategory = "special"    // 特殊徽章
)

// BadgeTier 徽章等级
type BadgeTier string

const (
	BadgeTierBronze   BadgeTier = "bronze"   // 青铜
	BadgeTierSilver   BadgeTier = "silver"   // 白银
	BadgeTierGold     BadgeTier = "gold"     // 黄金
	BadgeTierPlatinum BadgeTier = "platinum" // 铂金
	BadgeTierDiamond  BadgeTier = "diamond"  // 钻石
)

// Badge 徽章定义
type Badge struct {
	ID          uint           `gorm:"primaryKey;column:id;autoIncrement" json:"id"`
	Code        string         `gorm:"size:50;uniqueIndex;not null" json:"code"`       // 徽章代码 (如: story_creator_bronze)
	Name        string         `gorm:"size:100;not null" json:"name"`                  // 徽章名称
	NameZh      string         `gorm:"size:100" json:"name_zh"`                        // 中文名称
	Description string         `gorm:"size:500" json:"description"`                    // 描述
	DescZh      string         `gorm:"size:500" json:"desc_zh"`                        // 中文描述
	Category    BadgeCategory  `gorm:"size:20;not null;index" json:"category"`         // 类别
	Tier        BadgeTier      `gorm:"size:20;not null" json:"tier"`                   // 等级
	IconURL     string         `gorm:"size:500" json:"icon_url"`                       // 图标URL
	IconEmoji   string         `gorm:"size:10" json:"icon_emoji"`                      // 图标Emoji
	ColorHex    string         `gorm:"size:10" json:"color_hex"`                       // 颜色
	Threshold   int            `gorm:"default:0" json:"threshold"`                     // 获得条件阈值
	Points      int            `gorm:"default:0" json:"points"`                        // 徽章积分价值
	IsActive    bool           `gorm:"default:true;index" json:"is_active"`            // 是否激活
	DisplayOrder int           `gorm:"default:0" json:"display_order"`                 // 显示顺序
	CreatedAt   time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at;index" json:"deleted_at,omitempty"`
}

func (Badge) TableName() string {
	return "badges"
}

// UserBadge 用户已获得的徽章
type UserBadge struct {
	ID        uint           `gorm:"primaryKey;column:id;autoIncrement" json:"id"`
	UserID    string         `gorm:"size:36;not null;index:idx_user_badge" json:"user_id"`
	BadgeID   uint           `gorm:"not null;index:idx_user_badge" json:"badge_id"`
	EarnedAt  int64          `gorm:"type:bigint;not null" json:"earned_at"`            // 获得时间戳
	IsNew     bool           `gorm:"default:true" json:"is_new"`                       // 是否新徽章（用户未查看）
	IsPinned  bool           `gorm:"default:false" json:"is_pinned"`                   // 是否置顶展示
	CreatedAt time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index" json:"deleted_at,omitempty"`

	// Relations
	Badge *Badge `gorm:"foreignKey:BadgeID" json:"badge,omitempty"`
}

func (UserBadge) TableName() string {
	return "user_badges"
}

// UserBadgeStats 用户徽章统计数据
type UserBadgeStats struct {
	ID              uint           `gorm:"primaryKey;column:id;autoIncrement" json:"id"`
	UserID          string         `gorm:"size:36;uniqueIndex;not null" json:"user_id"`
	StoryCount      int            `gorm:"default:0" json:"story_count"`       // 创建的故事数量
	StoryboardCount int            `gorm:"default:0" json:"storyboard_count"`  // 创建的故事版数量
	TotalLikes      int            `gorm:"default:0" json:"total_likes"`       // 获得的总点赞数
	StoryLikes      int            `gorm:"default:0" json:"story_likes"`       // 故事获得的点赞数
	StoryboardLikes int            `gorm:"default:0" json:"storyboard_likes"`  // 故事版获得的点赞数
	FollowerCount   int            `gorm:"default:0" json:"follower_count"`    // 粉丝数量
	FollowingCount  int            `gorm:"default:0" json:"following_count"`   // 关注数量
	TotalBadges     int            `gorm:"default:0" json:"total_badges"`      // 已获得徽章总数
	TotalPoints     int            `gorm:"default:0" json:"total_points"`      // 徽章总积分
	LastUpdated     int64          `gorm:"type:bigint" json:"last_updated"`    // 最后更新时间
	CreatedAt       time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"column:deleted_at;index" json:"deleted_at,omitempty"`
}

func (UserBadgeStats) TableName() string {
	return "user_badge_stats"
}

// BadgeProgress 徽章进度（用于展示未获得徽章的进度）
type BadgeProgress struct {
	Badge       *Badge  `json:"badge"`
	Current     int     `json:"current"`      // 当前进度
	Target      int     `json:"target"`       // 目标值
	Progress    float64 `json:"progress"`     // 进度百分比 (0-100)
	IsCompleted bool    `json:"is_completed"` // 是否已完成
}

// UserBadgeProfile 用户徽章档案（用于API响应）
type UserBadgeProfile struct {
	UserID         string           `json:"user_id"`
	Stats          *UserBadgeStats  `json:"stats"`
	EarnedBadges   []*UserBadge     `json:"earned_badges"`
	PinnedBadges   []*UserBadge     `json:"pinned_badges"`
	NewBadges      []*UserBadge     `json:"new_badges"`
	BadgeProgress  []*BadgeProgress `json:"badge_progress"`
	TotalBadges    int              `json:"total_badges"`
	TotalPoints    int              `json:"total_points"`
	CompletionRate float64          `json:"completion_rate"` // 徽章完成率
}

// 预定义徽章列表
var PredefinedBadges = []Badge{
	// 故事创作者徽章
	{Code: "story_creator_bronze", Name: "Story Starter", NameZh: "故事新手", Description: "Created your first story", DescZh: "创建了第一个故事", Category: BadgeCategoryStory, Tier: BadgeTierBronze, IconEmoji: "📝", ColorHex: "#CD7F32", Threshold: 1, Points: 10, DisplayOrder: 1},
	{Code: "story_creator_silver", Name: "Story Teller", NameZh: "故事讲述者", Description: "Created 5 stories", DescZh: "创建了5个故事", Category: BadgeCategoryStory, Tier: BadgeTierSilver, IconEmoji: "📚", ColorHex: "#C0C0C0", Threshold: 5, Points: 25, DisplayOrder: 2},
	{Code: "story_creator_gold", Name: "Story Master", NameZh: "故事大师", Description: "Created 20 stories", DescZh: "创建了20个故事", Category: BadgeCategoryStory, Tier: BadgeTierGold, IconEmoji: "📖", ColorHex: "#FFD700", Threshold: 20, Points: 50, DisplayOrder: 3},
	{Code: "story_creator_platinum", Name: "Story Legend", NameZh: "故事传奇", Description: "Created 50 stories", DescZh: "创建了50个故事", Category: BadgeCategoryStory, Tier: BadgeTierPlatinum, IconEmoji: "🏆", ColorHex: "#E5E4E2", Threshold: 50, Points: 100, DisplayOrder: 4},
	{Code: "story_creator_diamond", Name: "Story Virtuoso", NameZh: "故事大神", Description: "Created 100 stories", DescZh: "创建了100个故事", Category: BadgeCategoryStory, Tier: BadgeTierDiamond, IconEmoji: "💎", ColorHex: "#B9F2FF", Threshold: 100, Points: 200, DisplayOrder: 5},

	// 故事版创作者徽章
	{Code: "storyboard_creator_bronze", Name: "Board Beginner", NameZh: "故事版新手", Description: "Created your first storyboard", DescZh: "创建了第一个故事版", Category: BadgeCategoryStoryboard, Tier: BadgeTierBronze, IconEmoji: "🎬", ColorHex: "#CD7F32", Threshold: 1, Points: 10, DisplayOrder: 10},
	{Code: "storyboard_creator_silver", Name: "Board Artist", NameZh: "故事版艺术家", Description: "Created 10 storyboards", DescZh: "创建了10个故事版", Category: BadgeCategoryStoryboard, Tier: BadgeTierSilver, IconEmoji: "🎨", ColorHex: "#C0C0C0", Threshold: 10, Points: 25, DisplayOrder: 11},
	{Code: "storyboard_creator_gold", Name: "Board Master", NameZh: "故事版大师", Description: "Created 50 storyboards", DescZh: "创建了50个故事版", Category: BadgeCategoryStoryboard, Tier: BadgeTierGold, IconEmoji: "🎭", ColorHex: "#FFD700", Threshold: 50, Points: 50, DisplayOrder: 12},
	{Code: "storyboard_creator_platinum", Name: "Board Legend", NameZh: "故事版传奇", Description: "Created 100 storyboards", DescZh: "创建了100个故事版", Category: BadgeCategoryStoryboard, Tier: BadgeTierPlatinum, IconEmoji: "🌟", ColorHex: "#E5E4E2", Threshold: 100, Points: 100, DisplayOrder: 13},
	{Code: "storyboard_creator_diamond", Name: "Board Virtuoso", NameZh: "故事版大神", Description: "Created 500 storyboards", DescZh: "创建了500个故事版", Category: BadgeCategoryStoryboard, Tier: BadgeTierDiamond, IconEmoji: "✨", ColorHex: "#B9F2FF", Threshold: 500, Points: 200, DisplayOrder: 14},

	// 点赞相关徽章（被点赞）
	{Code: "liked_bronze", Name: "Appreciated", NameZh: "受欢迎", Description: "Received 10 likes", DescZh: "获得了10个点赞", Category: BadgeCategoryLike, Tier: BadgeTierBronze, IconEmoji: "❤️", ColorHex: "#CD7F32", Threshold: 10, Points: 10, DisplayOrder: 20},
	{Code: "liked_silver", Name: "Popular", NameZh: "人气新星", Description: "Received 50 likes", DescZh: "获得了50个点赞", Category: BadgeCategoryLike, Tier: BadgeTierSilver, IconEmoji: "💕", ColorHex: "#C0C0C0", Threshold: 50, Points: 25, DisplayOrder: 21},
	{Code: "liked_gold", Name: "Beloved", NameZh: "万人迷", Description: "Received 200 likes", DescZh: "获得了200个点赞", Category: BadgeCategoryLike, Tier: BadgeTierGold, IconEmoji: "💖", ColorHex: "#FFD700", Threshold: 200, Points: 50, DisplayOrder: 22},
	{Code: "liked_platinum", Name: "Adored", NameZh: "超级人气", Description: "Received 500 likes", DescZh: "获得了500个点赞", Category: BadgeCategoryLike, Tier: BadgeTierPlatinum, IconEmoji: "💝", ColorHex: "#E5E4E2", Threshold: 500, Points: 100, DisplayOrder: 23},
	{Code: "liked_diamond", Name: "Legendary", NameZh: "传说级人气", Description: "Received 1000 likes", DescZh: "获得了1000个点赞", Category: BadgeCategoryLike, Tier: BadgeTierDiamond, IconEmoji: "💗", ColorHex: "#B9F2FF", Threshold: 1000, Points: 200, DisplayOrder: 24},

	// 粉丝相关徽章
	{Code: "follower_bronze", Name: "Gathering Crowd", NameZh: "小有名气", Description: "Have 5 followers", DescZh: "拥有5个粉丝", Category: BadgeCategoryFollower, Tier: BadgeTierBronze, IconEmoji: "👥", ColorHex: "#CD7F32", Threshold: 5, Points: 10, DisplayOrder: 30},
	{Code: "follower_silver", Name: "Rising Star", NameZh: "冉冉新星", Description: "Have 20 followers", DescZh: "拥有20个粉丝", Category: BadgeCategoryFollower, Tier: BadgeTierSilver, IconEmoji: "🌟", ColorHex: "#C0C0C0", Threshold: 20, Points: 25, DisplayOrder: 31},
	{Code: "follower_gold", Name: "Influencer", NameZh: "影响力达人", Description: "Have 100 followers", DescZh: "拥有100个粉丝", Category: BadgeCategoryFollower, Tier: BadgeTierGold, IconEmoji: "⭐", ColorHex: "#FFD700", Threshold: 100, Points: 50, DisplayOrder: 32},
	{Code: "follower_platinum", Name: "Celebrity", NameZh: "名人", Description: "Have 500 followers", DescZh: "拥有500个粉丝", Category: BadgeCategoryFollower, Tier: BadgeTierPlatinum, IconEmoji: "🌠", ColorHex: "#E5E4E2", Threshold: 500, Points: 100, DisplayOrder: 33},
	{Code: "follower_diamond", Name: "Super Star", NameZh: "超级明星", Description: "Have 1000 followers", DescZh: "拥有1000个粉丝", Category: BadgeCategoryFollower, Tier: BadgeTierDiamond, IconEmoji: "🌌", ColorHex: "#B9F2FF", Threshold: 1000, Points: 200, DisplayOrder: 34},

	// 特殊徽章
	{Code: "early_adopter", Name: "Early Adopter", NameZh: "早期用户", Description: "Joined during early access", DescZh: "早期加入的用户", Category: BadgeCategorySpecial, Tier: BadgeTierGold, IconEmoji: "🚀", ColorHex: "#FFD700", Threshold: 0, Points: 50, DisplayOrder: 50},
	{Code: "vip_member", Name: "VIP Member", NameZh: "VIP会员", Description: "Subscribed to VIP", DescZh: "订阅了VIP会员", Category: BadgeCategorySpecial, Tier: BadgeTierPlatinum, IconEmoji: "👑", ColorHex: "#E5E4E2", Threshold: 0, Points: 100, DisplayOrder: 51},
	{Code: "first_story_published", Name: "First Publish", NameZh: "首次发布", Description: "Published your first story", DescZh: "发布了第一个故事", Category: BadgeCategoryCreator, Tier: BadgeTierBronze, IconEmoji: "📢", ColorHex: "#CD7F32", Threshold: 1, Points: 15, DisplayOrder: 60},
}

