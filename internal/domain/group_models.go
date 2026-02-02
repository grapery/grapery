package domain

// GroupMemberRole 群组成员角色（保留用于向后兼容）
type GroupMemberRole string

const (
	RoleOwner     GroupMemberRole = "owner"     // 群主
	RoleAdmin     GroupMemberRole = "admin"     // 管理员
	RoleModerator GroupMemberRole = "moderator" // 协管
	RoleMember    GroupMemberRole = "member"    // 普通成员
)

// GroupRole 群组角色定义表
type GroupRole struct {
	ID          string `json:"id"`
	RoleCode    string `json:"code"`        // 角色代码：creator, admin, member, outsider
	Name        string `json:"name"`        // 角色名称：小组创建者、小组管理员、小组成员、小组外部人员
	Description string `json:"description"` // 角色描述
	IsSystem    bool   `json:"isSystem"`    // 是否为系统内置角色
	CreatedAt   int64  `json:"createdAt"`
	UpdatedAt   int64  `json:"updatedAt"`
}

// GroupRolePermission 群组角色权限关联（权限固定，存储在代码中）
type GroupRolePermission struct {
	RoleCode            string `json:"roleCode"`
	CanInviteMembers    bool   `json:"canInviteMembers"`
	CanRemoveMembers    bool   `json:"canRemoveMembers"`
	CanEditGroup        bool   `json:"canEditGroup"`
	CanDeleteGroup      bool   `json:"canDeleteGroup"`
	CanCreateStories    bool   `json:"canCreateStories"`
	CanEditStories      bool   `json:"canEditStories"`
	CanDeleteStories    bool   `json:"canDeleteStories"`
	CanManageRoles      bool   `json:"canManageRoles"`
	CanViewGroupContent bool   `json:"canViewGroupContent"` // 查看群组内容（外部人员可能没有此权限）
}

// Group represents a collaboration group
type Group struct {
	ID             string   `json:"id"`
	CreatorID      string   `json:"-"` // 内部使用，不序列化到JSON
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	Avatar         string   `json:"avatar,omitempty"`
	CoverImage     string   `json:"coverImage,omitempty"`
	Category       string   `json:"category,omitempty"`
	Tags           []string `json:"tags,omitempty"`
	Visibility     string   `json:"visibility"`   // 可见性: public, private, hidden
	JoinType       string   `json:"joinType"`     // 加入类型: open, apply, invite_only
	MaxMembers     int      `json:"maxMembers"`   // 最大成员数
	Status         string   `json:"status"`       // 状态: active, inactive, banned
	MembersCount   int      `json:"membersCount"`   // 成员数
	StoriesCount   int      `json:"storiesCount"`   // 故事数
	FollowersCount int      `json:"followersCount"` // 关注者数
	CreatedAt      int64    `json:"createdAt"`
	UpdatedAt      int64    `json:"updatedAt"`

	// 向后兼容的字段（内部使用）
	Members      int  `json:"members"`       // 兼容旧代码
	Stories      int  `json:"stories"`       // 兼容旧代码
	Followers    int  `json:"followers"`     // 兼容旧代码
	BlockedCount int  `json:"blockedCount"`  // 兼容旧代码
	Public       bool `json:"isPublic"`      // 兼容旧代码

	// Relations
	Creator     *User            `json:"creator,omitempty"`
	MyRole      *GroupMemberRole `json:"myRole,omitempty"`      // 当前用户在群组中的角色
	IsFollowing *bool            `json:"isFollowing,omitempty"` // 当前用户是否已关注此群组
}

// GroupMember 群组成员
type GroupMember struct {
	ID       string          `json:"id"`
	GroupID  string          `json:"groupId"`
	UserID   string          `json:"userId"`
	Role     GroupMemberRole `json:"role"` // owner, admin, member
	JoinedAt int64           `json:"joinedAt"`

	// Relations
	Group *Group `json:"group,omitempty"`
	User  *User  `json:"user,omitempty"`
}

// GroupFollow 群组关注记录
type GroupFollow struct {
	ID        string `json:"id"`
	GroupID   string `json:"groupId"`
	UserID    string `json:"userId"`
	CreatedAt int64  `json:"createdAt"`

	// Relations
	Group *Group `json:"group,omitempty"`
	User  *User  `json:"user,omitempty"`
}

// GroupMemberInfo 群组成员信息（扩展信息）
type GroupMemberInfo struct {
	ID        string          `json:"id"`
	GroupID   string          `json:"groupId"`
	UserID    string          `json:"userId"`
	Role      GroupMemberRole `json:"role"`
	JoinedAt  int64           `json:"joinedAt"`
	InvitedBy string          `json:"invitedBy,omitempty"`

	// Relations
	User *User `json:"user,omitempty"`
}

// GroupInvitation 群组邀请
type GroupInvitation struct {
	ID        string `json:"id"`
	GroupID   string `json:"groupId"`
	InviterID string `json:"inviterId"`
	InviteeID string `json:"inviteeId"`
	Status    string `json:"status"` // pending, accepted, rejected, expired
	Message   string `json:"message,omitempty"`
	CreatedAt int64  `json:"createdAt"`
	ExpiresAt int64  `json:"expiresAt"`

	// Relations
	Group   *Group `json:"group,omitempty"`
	Inviter *User  `json:"inviter,omitempty"`
	Invitee *User  `json:"invitee,omitempty"`
}

// GroupActivity represents an activity in a group feed
type GroupActivity struct {
	ID         string `json:"id"`
	GroupID    string `json:"groupId,omitempty"`
	Type       string `json:"type"`
	UserID     string `json:"userId"`
	UserName   string `json:"userName"`
	UserAvatar string `json:"userAvatar"`
	StoryID    string `json:"storyId,omitempty"`
	StoryTitle string `json:"storyTitle,omitempty"`
	Message    string `json:"message"`
	Timestamp  int64  `json:"timestamp"`
	Date       string `json:"date,omitempty"` // YYYY-MM-DD format for heatmap grouping

	// Relations
	Group *Group  `json:"group,omitempty"`
	User  *User   `json:"user,omitempty"`
	Story *Story  `json:"story,omitempty"`
}

// ActivityTimeRange represents the time range for activity queries
type ActivityTimeRange string

const (
	TimeRangeToday ActivityTimeRange = "today"
	TimeRangeWeek  ActivityTimeRange = "week"
	TimeRangeMonth ActivityTimeRange = "month"
)

// ActivityHeatmapData represents heatmap data for activity visualization
type ActivityHeatmapData struct {
	Date  string `json:"date"`  // YYYY-MM-DD format
	Count int    `json:"count"` // Number of activities on this date
}

// ActivityHeatmapResponse contains heatmap data and activity list
type ActivityHeatmapResponse struct {
	TimeRange   ActivityTimeRange     `json:"timeRange"`
	StartDate   string                `json:"startDate"`
	EndDate     string                `json:"endDate"`
	HeatmapData []ActivityHeatmapData `json:"heatmapData"`
	TotalCount  int                   `json:"totalCount"`
}

// ActivityListRequest represents a request for filtered activities
type ActivityListRequest struct {
	GroupID   string            `json:"groupId"`
	TimeRange ActivityTimeRange `json:"timeRange,omitempty"`
	Date      string            `json:"date,omitempty"` // Filter by specific date (YYYY-MM-DD)
	Limit     int               `json:"limit,omitempty"`
	Offset    int               `json:"offset,omitempty"`
}

// GroupPermission 群组权限
type GroupPermission struct {
	Role             GroupMemberRole
	CanInviteMembers bool
	CanRemoveMembers bool
	CanEditGroup     bool
	CanDeleteGroup   bool
	CanCreateStories bool
	CanEditStories   bool
	CanDeleteStories bool
	CanManageRoles   bool
}

// GetPermissions 获取角色权限（基于GroupMemberRole，保留用于向后兼容）
func GetPermissions(role GroupMemberRole) GroupPermission {
	switch role {
	case RoleOwner:
		return GroupPermission{
			Role:             RoleOwner,
			CanInviteMembers: true,
			CanRemoveMembers: true,
			CanEditGroup:     true,
			CanDeleteGroup:   true,
			CanCreateStories: true,
			CanEditStories:   true,
			CanDeleteStories: true,
			CanManageRoles:   true,
		}
	case RoleAdmin:
		return GroupPermission{
			Role:             RoleAdmin,
			CanInviteMembers: true,
			CanRemoveMembers: true,
			CanEditGroup:     true,
			CanDeleteGroup:   false,
			CanCreateStories: true,
			CanEditStories:   true,
			CanDeleteStories: true,
			CanManageRoles:   true,
		}
	case RoleModerator:
		return GroupPermission{
			Role:             RoleModerator,
			CanInviteMembers: true,
			CanRemoveMembers: false,
			CanEditGroup:     false,
			CanDeleteGroup:   false,
			CanCreateStories: true,
			CanEditStories:   true,
			CanDeleteStories: false,
			CanManageRoles:   false,
		}
	default: // RoleMember
		return GroupPermission{
			Role:             RoleMember,
			CanInviteMembers: false,
			CanRemoveMembers: false,
			CanEditGroup:     false,
			CanDeleteGroup:   false,
			CanCreateStories: true,
			CanEditStories:   false,
			CanDeleteStories: false,
			CanManageRoles:   false,
		}
	}
}

// 角色代码常量
const (
	RoleCodeCreator  = "creator"  // 小组创建者
	RoleCodeAdmin    = "admin"    // 小组管理员
	RoleCodeMember   = "member"   // 小组成员
	RoleCodeOutsider = "outsider" // 小组外部人员
)

// GetRolePermissions 根据角色代码获取权限（新的权限系统）
func GetRolePermissions(roleCode string) GroupRolePermission {
	switch roleCode {
	case RoleCodeCreator:
		return GroupRolePermission{
			RoleCode:            RoleCodeCreator,
			CanInviteMembers:    true,
			CanRemoveMembers:    true,
			CanEditGroup:        true,
			CanDeleteGroup:      true,
			CanCreateStories:    true,
			CanEditStories:      true,
			CanDeleteStories:    true,
			CanManageRoles:      true,
			CanViewGroupContent: true,
		}
	case RoleCodeAdmin:
		return GroupRolePermission{
			RoleCode:            RoleCodeAdmin,
			CanInviteMembers:    true,
			CanRemoveMembers:    true,
			CanEditGroup:        true,
			CanDeleteGroup:      false,
			CanCreateStories:    true,
			CanEditStories:      true,
			CanDeleteStories:    true,
			CanManageRoles:      true,
			CanViewGroupContent: true,
		}
	case RoleCodeMember:
		return GroupRolePermission{
			RoleCode:            RoleCodeMember,
			CanInviteMembers:    false,
			CanRemoveMembers:    false,
			CanEditGroup:        false,
			CanDeleteGroup:      false,
			CanCreateStories:    true,
			CanEditStories:      false,
			CanDeleteStories:    false,
			CanManageRoles:      false,
			CanViewGroupContent: true,
		}
	case RoleCodeOutsider:
		return GroupRolePermission{
			RoleCode:            RoleCodeOutsider,
			CanInviteMembers:    false,
			CanRemoveMembers:    false,
			CanEditGroup:        false,
			CanDeleteGroup:      false,
			CanCreateStories:    false,
			CanEditStories:      false,
			CanDeleteStories:    false,
			CanManageRoles:      false,
			CanViewGroupContent: false,
		}
	default:
		// 默认返回外部人员权限（最严格）
		return GetRolePermissions(RoleCodeOutsider)
	}
}

// ========== Group Blacklist Models ==========

// GroupBlacklist 小组黑名单
type GroupBlacklist struct {
	ID        string `json:"id"`
	GroupID   string `json:"groupId"`   // 小组ID
	UserID    string `json:"userId"`    // 被拉黑的用户ID
	BlockedBy string `json:"blockedBy"` // 操作者ID（谁拉黑的）
	Reason    string `json:"reason"`    // 拉黑原因（可选）
	CreatedAt int64  `json:"createdAt"`

	// Relations
	Group *Group `json:"group,omitempty"`
	User  *User  `json:"user,omitempty"`
	Admin *User  `json:"admin,omitempty"` // 执行拉黑操作的管理员
}

// GroupBlacklistInfo 黑名单信息（扩展）
type GroupBlacklistInfo struct {
	ID        string `json:"id"`
	GroupID   string `json:"groupId"`
	UserID    string `json:"userId"`
	BlockedBy string `json:"blockedBy"`
	Reason    string `json:"reason,omitempty"`
	CreatedAt int64  `json:"createdAt"`

	// Relations
	User  *User `json:"user,omitempty"`
	Admin *User `json:"admin,omitempty"`
}

// GroupVisibility 群组可见性
type GroupVisibility string

const (
	GroupVisibilityPublic  GroupVisibility = "public"
	GroupVisibilityPrivate GroupVisibility = "private"
	GroupVisibilityHidden  GroupVisibility = "hidden"
)

// GroupJoinType 群组加入类型
type GroupJoinType string

const (
	GroupJoinTypeOpen       GroupJoinType = "open"
	GroupJoinTypeApply      GroupJoinType = "apply"
	GroupJoinTypeInviteOnly GroupJoinType = "invite_only"
)

// GroupStatus 群组状态
type GroupStatus string

const (
	GroupStatusActive   GroupStatus = "active"
	GroupStatusInactive GroupStatus = "inactive"
	GroupStatusBanned   GroupStatus = "banned"
)
