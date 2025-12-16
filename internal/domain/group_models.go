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
	Code        string `json:"code"`        // 角色代码：creator, admin, member, outsider
	Name        string `json:"name"`        // 角色名称：小组创建者、小组管理员、小组成员、小组外部人员
	Description string `json:"description"` // 角色描述
	IsSystem    bool   `json:"isSystem"`    // 是否为系统内置角色
	CreatedAt   int64  `json:"createdAt"`
	UpdatedAt   int64  `json:"updatedAt"`
}

// GroupRolePermission 群组角色权限关联（权限固定，存储在代码中）
type GroupRolePermission struct {
	RoleCode           string `json:"roleCode"`
	CanInviteMembers   bool   `json:"canInviteMembers"`
	CanRemoveMembers   bool   `json:"canRemoveMembers"`
	CanEditGroup       bool   `json:"canEditGroup"`
	CanDeleteGroup     bool   `json:"canDeleteGroup"`
	CanCreateStories   bool   `json:"canCreateStories"`
	CanEditStories      bool   `json:"canEditStories"`
	CanDeleteStories    bool   `json:"canDeleteStories"`
	CanManageRoles      bool   `json:"canManageRoles"`
	CanViewGroupContent bool   `json:"canViewGroupContent"` // 查看群组内容（外部人员可能没有此权限）
}

// Group represents a collaboration group
type Group struct {
	ID          string `json:"id"`
	CreatorID   string `json:"-"` // 内部使用，不序列化到JSON
	Name        string `json:"name"`
	Description string `json:"description"`
	Avatar      string `json:"avatar,omitempty"`
	Members     int    `json:"members"`
	Stories     int    `json:"stories"`
	Public      bool   `json:"is_public"`
	CreatedAt   int64  `json:"created_at"`
	UpdatedAt   int64  `json:"updated_at"`

	// Relations
	Creator *User            `json:"creator,omitempty"`
	MyRole  *GroupMemberRole `json:"my_role,omitempty"` // 当前用户在群组中的角色
}

// GroupMember 群组成员
type GroupMember struct {
	ID       string          `json:"id"`
	GroupID  string          `json:"group_id"`
	UserID   string          `json:"user_id"`
	Role     GroupMemberRole `json:"role"` // owner, admin, member
	JoinedAt int64           `json:"joined_at"`

	// Relations
	Group *Group `json:"group,omitempty"`
	User  *User  `json:"user,omitempty"`
}

// GroupMemberInfo 群组成员信息（扩展信息）
type GroupMemberInfo struct {
	ID        string          `json:"id"`
	GroupID   string          `json:"group_id"`
	UserID    string          `json:"user_id"`
	Role      GroupMemberRole `json:"role"`
	JoinedAt  int64           `json:"joined_at"`
	InvitedBy string          `json:"invited_by,omitempty"`

	// Relations
	User *User `json:"user,omitempty"`
}

// GroupInvitation 群组邀请
type GroupInvitation struct {
	ID        string `json:"id"`
	GroupID   string `json:"group_id"`
	InviterID string `json:"inviter_id"`
	InviteeID string `json:"invitee_id"`
	Status    string `json:"status"` // pending, accepted, rejected, expired
	Message   string `json:"message,omitempty"`
	CreatedAt int64  `json:"created_at"`
	ExpiresAt int64  `json:"expires_at"`

	// Relations
	Group   *Group `json:"group,omitempty"`
	Inviter *User  `json:"inviter,omitempty"`
	Invitee *User  `json:"invitee,omitempty"`
}

// GroupActivity represents an activity in a group feed
type GroupActivity struct {
	ID         string `json:"id"`
	GroupID    string `json:"-"`
	Type       string `json:"type"`
	UserID     string `json:"user_id"`
	UserName   string `json:"user_name"`
	UserAvatar string `json:"user_avatar"`
	StoryID    string `json:"story_id,omitempty"`
	StoryTitle string `json:"story_title,omitempty"`
	Message    string `json:"message"`
	Timestamp  int64  `json:"timestamp"`
	Date       string `json:"date,omitempty"` // YYYY-MM-DD format for heatmap grouping

	// Relations
	Group *Group `json:"group,omitempty"`
	User  *User  `json:"user,omitempty"`
	Story *Story `json:"story,omitempty"`
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
	TimeRange   ActivityTimeRange     `json:"time_range"`
	StartDate   string                `json:"start_date"`
	EndDate     string                `json:"end_date"`
	HeatmapData []ActivityHeatmapData `json:"heatmap_data"`
	TotalCount  int                   `json:"total_count"`
}

// ActivityListRequest represents a request for filtered activities
type ActivityListRequest struct {
	GroupID   string            `json:"group_id"`
	TimeRange ActivityTimeRange `json:"time_range,omitempty"`
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
			RoleCode:           RoleCodeCreator,
			CanInviteMembers:   true,
			CanRemoveMembers:   true,
			CanEditGroup:       true,
			CanDeleteGroup:     true,
			CanCreateStories:   true,
			CanEditStories:     true,
			CanDeleteStories:   true,
			CanManageRoles:     true,
			CanViewGroupContent: true,
		}
	case RoleCodeAdmin:
		return GroupRolePermission{
			RoleCode:           RoleCodeAdmin,
			CanInviteMembers:   true,
			CanRemoveMembers:   true,
			CanEditGroup:       true,
			CanDeleteGroup:     false,
			CanCreateStories:   true,
			CanEditStories:     true,
			CanDeleteStories:   true,
			CanManageRoles:     true,
			CanViewGroupContent: true,
		}
	case RoleCodeMember:
		return GroupRolePermission{
			RoleCode:           RoleCodeMember,
			CanInviteMembers:   false,
			CanRemoveMembers:   false,
			CanEditGroup:       false,
			CanDeleteGroup:     false,
			CanCreateStories:   true,
			CanEditStories:     false,
			CanDeleteStories:   false,
			CanManageRoles:     false,
			CanViewGroupContent: true,
		}
	case RoleCodeOutsider:
		return GroupRolePermission{
			RoleCode:           RoleCodeOutsider,
			CanInviteMembers:   false,
			CanRemoveMembers:   false,
			CanEditGroup:       false,
			CanDeleteGroup:     false,
			CanCreateStories:   false,
			CanEditStories:     false,
			CanDeleteStories:   false,
			CanManageRoles:     false,
			CanViewGroupContent: false,
		}
	default:
		// 默认返回外部人员权限（最严格）
		return GetRolePermissions(RoleCodeOutsider)
	}
}
