package mysql

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/gorm"
)

// GroupByID retrieves a group by ID
func (r *Repository) GroupByID(ctx context.Context, id string) (*domain.Group, error) {
	var group Group
	if err := r.db.WithContext(ctx).Preload("Creator").First(&group, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	domainGroup := r.groupToDomain(group)
	return &domainGroup, nil
}

// CreateGroup creates a new group
func (r *Repository) CreateGroup(ctx context.Context, group *domain.Group) error {
	dbGroup := Group{
		ID:          uuid.New().String(),
		Name:        group.Name,
		Description: group.Description,
		Avatar:      group.Avatar,
		CreatorID:   group.Creator.ID,
		Members:     0, // 成员数量由 AddGroupMember 自动增加
		Stories:     0,
		Public:      group.Public,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := r.db.WithContext(ctx).Create(&dbGroup).Error; err != nil {
		return err
	}

	group.ID = dbGroup.ID
	group.CreatedAt = dbGroup.CreatedAt.Unix()
	group.UpdatedAt = dbGroup.UpdatedAt.Unix()
	return nil
}

// UpdateGroup updates an existing group
func (r *Repository) UpdateGroup(ctx context.Context, group *domain.Group) error {
	dbGroup := Group{
		ID:          group.ID,
		Name:        group.Name,
		Description: group.Description,
		Avatar:      group.Avatar,
		CoverImage:  group.CoverImage,
		CreatorID:   group.Creator.ID,
		Members:     group.Members,
		Stories:     group.Stories,
		Followers:   group.Followers,
		Public:      group.Public,
		UpdatedAt:   time.Now(),
	}

	if err := r.db.WithContext(ctx).Model(&Group{}).Where("id = ?", group.ID).Updates(&dbGroup).Error; err != nil {
		return err
	}

	var updated Group
	if err := r.db.WithContext(ctx).Preload("Creator").First(&updated, "id = ?", group.ID).Error; err != nil {
		return err
	}

	*group = r.groupToDomain(updated)
	return nil
}

// DeleteGroup deletes a group (soft delete)
func (r *Repository) DeleteGroup(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&Group{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// AddGroupMember adds a member to a group
func (r *Repository) AddGroupMember(ctx context.Context, groupID, userID string, role domain.GroupMemberRole, invitedBy string) error {
	member := GroupMember{
		ID:        uuid.New().String(),
		GroupID:   groupID,
		UserID:    userID,
		Role:      string(role),
		InvitedBy: invitedBy,
		JoinedAt:  time.Now(),
	}

	if err := r.db.WithContext(ctx).Create(&member).Error; err != nil {
		// Handle MySQL duplicate entry error (Error 1062)
		if strings.Contains(err.Error(), "Error 1062") ||
		   strings.Contains(err.Error(), "Duplicate entry") ||
		   strings.Contains(err.Error(), "23000") {
			return domain.ErrAlreadyExists
		}
		return err
	}

	// 更新群组成员数
	if err := r.db.WithContext(ctx).Model(&Group{}).
		Where("id = ?", groupID).
		UpdateColumn("members", gorm.Expr("members + ?", 1)).Error; err != nil {
		return err
	}

	return nil
}

// RemoveGroupMember removes a member from a group
func (r *Repository) RemoveGroupMember(ctx context.Context, groupID, userID string) error {
	result := r.db.WithContext(ctx).
		Where("group_id = ? AND user_id = ?", groupID, userID).
		Delete(&GroupMember{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("member not found")
	}

	// 更新群组成员数
	if err := r.db.WithContext(ctx).Model(&Group{}).
		Where("id = ?", groupID).
		UpdateColumn("members", gorm.Expr("GREATEST(members - ?, 0)", 1)).Error; err != nil {
		return err
	}

	return nil
}

// GetGroupMembers retrieves members of a group
func (r *Repository) GetGroupMembers(ctx context.Context, groupID string, limit, offset int) ([]*domain.GroupMemberInfo, error) {
	var members []GroupMember
	query := r.db.WithContext(ctx).Preload("User").Where("group_id = ?", groupID).Order("joined_at ASC")

	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	if err := query.Find(&members).Error; err != nil {
		return nil, err
	}

	result := make([]*domain.GroupMemberInfo, len(members))
	for i, m := range members {
		refUser := r.userToDomain(m.User)
		result[i] = &domain.GroupMemberInfo{
			ID:        m.ID,
			GroupID:   m.GroupID,
			UserID:    m.UserID,
			User:      &refUser,
			Role:      domain.GroupMemberRole(m.Role),
			JoinedAt:  m.JoinedAt.Unix(),
			InvitedBy: m.InvitedBy,
		}
	}
	return result, nil
}

// GetMemberRole retrieves a member's role in a group
func (r *Repository) GetMemberRole(ctx context.Context, groupID, userID string) (domain.GroupMemberRole, error) {
	var member GroupMember
	if err := r.db.WithContext(ctx).
		Where("group_id = ? AND user_id = ?", groupID, userID).
		First(&member).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", errors.New("not a member")
		}
		return "", err
	}
	return domain.GroupMemberRole(member.Role), nil
}

// UpdateMemberRole updates a member's role
func (r *Repository) UpdateMemberRole(ctx context.Context, groupID, userID string, role domain.GroupMemberRole) error {
	result := r.db.WithContext(ctx).Model(&GroupMember{}).
		Where("group_id = ? AND user_id = ?", groupID, userID).
		Update("role", string(role))

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("member not found")
	}

	return nil
}

// IsGroupMember checks if a user is a member of a group
func (r *Repository) IsGroupMember(ctx context.Context, groupID, userID string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&GroupMember{}).
		Where("group_id = ? AND user_id = ?", groupID, userID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// ========== Group Roles Management ==========

// CreateGroupRole 创建群组角色
func (r *Repository) CreateGroupRole(ctx context.Context, role *domain.GroupRole) error {
	model := &GroupRole{
		ID:          role.ID,
		RoleCode:    role.RoleCode,
		Name:        role.Name,
		Description: role.Description,
		IsSystem:    role.IsSystem,
		CreatedAt:   time.Unix(role.CreatedAt, 0),
		UpdatedAt:   time.Unix(role.UpdatedAt, 0),
	}
	return r.db.WithContext(ctx).Create(model).Error
}

// GetGroupRoleByCode 根据代码获取群组角色
func (r *Repository) GetGroupRoleByCode(ctx context.Context, code string) (*domain.GroupRole, error) {
	var model GroupRole
	if err := r.db.WithContext(ctx).Where("code = ?", code).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return r.groupRoleToDomain(&model), nil
}

// GetGroupRoleByID 根据ID获取群组角色
func (r *Repository) GetGroupRoleByID(ctx context.Context, id string) (*domain.GroupRole, error) {
	var model GroupRole
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return r.groupRoleToDomain(&model), nil
}

// ListGroupRoles 列出所有群组角色
func (r *Repository) ListGroupRoles(ctx context.Context) ([]*domain.GroupRole, error) {
	var models []GroupRole
	if err := r.db.WithContext(ctx).Order("created_at ASC").Find(&models).Error; err != nil {
		return nil, err
	}
	roles := make([]*domain.GroupRole, len(models))
	for i, m := range models {
		roles[i] = r.groupRoleToDomain(&m)
	}
	return roles, nil
}

// InitializeGroupRoles 初始化系统内置角色
func (r *Repository) InitializeGroupRoles(ctx context.Context) error {
	roles := []*domain.GroupRole{
		{
			ID:          uuid.New().String(),
			RoleCode:    domain.RoleCodeCreator,
			Name:        "小组创建者",
			Description: "小组的创建者，拥有所有权限",
			IsSystem:    true,
			CreatedAt:   time.Now().Unix(),
			UpdatedAt:   time.Now().Unix(),
		},
		{
			ID:          uuid.New().String(),
			RoleCode:    domain.RoleCodeAdmin,
			Name:        "小组管理员",
			Description: "小组的管理员，可以管理成员和内容",
			IsSystem:    true,
			CreatedAt:   time.Now().Unix(),
			UpdatedAt:   time.Now().Unix(),
		},
		{
			ID:          uuid.New().String(),
			RoleCode:    domain.RoleCodeMember,
			Name:        "小组成员",
			Description: "小组的普通成员，可以创建和查看内容",
			IsSystem:    true,
			CreatedAt:   time.Now().Unix(),
			UpdatedAt:   time.Now().Unix(),
		},
		{
			ID:          uuid.New().String(),
			RoleCode:    domain.RoleCodeOutsider,
			Name:        "小组外部人员",
			Description: "小组外部人员，无权限访问小组内容",
			IsSystem:    true,
			CreatedAt:   time.Now().Unix(),
			UpdatedAt:   time.Now().Unix(),
		},
	}

	for _, role := range roles {
		// 检查角色是否已存在
		existing, err := r.GetGroupRoleByCode(ctx, role.RoleCode)
		if err != nil && err != domain.ErrNotFound {
			return err
		}
		if existing != nil {
			continue // 角色已存在，跳过
		}
		// 创建角色
		if err := r.CreateGroupRole(ctx, role); err != nil {
			return err
		}
	}
	return nil
}

// UpdateMemberRoleID 更新成员的角色ID
func (r *Repository) UpdateMemberRoleID(ctx context.Context, groupID, userID, roleID string) error {
	result := r.db.WithContext(ctx).Model(&GroupMember{}).
		Where("group_id = ? AND user_id = ?", groupID, userID).
		Update("role_id", roleID)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("member not found")
	}

	return nil
}

// GetMemberRoleID 获取成员的角色ID
func (r *Repository) GetMemberRoleID(ctx context.Context, groupID, userID string) (string, error) {
	var member GroupMember
	if err := r.db.WithContext(ctx).
		Where("group_id = ? AND user_id = ?", groupID, userID).
		First(&member).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", errors.New("member not found")
		}
		return "", err
	}
	return member.RoleID, nil
}

// groupRoleToDomain 将数据库模型转换为domain模型
func (r *Repository) groupRoleToDomain(m *GroupRole) *domain.GroupRole {
	return &domain.GroupRole{
		ID:          m.ID,
		RoleCode:    m.RoleCode,
		Name:        m.Name,
		Description: m.Description,
		IsSystem:    m.IsSystem,
		CreatedAt:   m.CreatedAt.Unix(),
		UpdatedAt:   m.UpdatedAt.Unix(),
	}
}

// CreateGroupInvitation creates a group invitation
func (r *Repository) CreateGroupInvitation(ctx context.Context, groupID, inviterID, inviteeID, message string) (*domain.GroupInvitation, error) {
	invitation := GroupInvitation{
		ID:        uuid.New().String(),
		GroupID:   groupID,
		InviterID: inviterID,
		InviteeID: inviteeID,
		Status:    "pending",
		Message:   message,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour), // 7 days
	}

	if err := r.db.WithContext(ctx).Create(&invitation).Error; err != nil {
		return nil, err
	}

	// 重新加载完整数据
	var loaded GroupInvitation
	if err := r.db.WithContext(ctx).
		Preload("Group").Preload("Group.Creator").
		Preload("Inviter").Preload("Invitee").
		First(&loaded, "id = ?", invitation.ID).Error; err != nil {
		return nil, err
	}

	refGroup := r.groupToDomain(loaded.Group)
	refInviter := r.userToDomain(loaded.Invitee)
	refInvitee := r.userToDomain(loaded.Invitee)
	return &domain.GroupInvitation{
		ID:        loaded.ID,
		GroupID:   loaded.GroupID,
		Group:     &refGroup,
		InviterID: loaded.InviterID,
		Inviter:   &refInviter,
		InviteeID: loaded.InviteeID,
		Invitee:   &refInvitee,
		Status:    loaded.Status,
		Message:   loaded.Message,
		CreatedAt: loaded.CreatedAt.Unix(),
		ExpiresAt: loaded.ExpiresAt.Unix(),
	}, nil
}

// GetInvitationByID retrieves an invitation by ID
func (r *Repository) GetInvitationByID(ctx context.Context, id string) (*domain.GroupInvitation, error) {
	var invitation GroupInvitation
	if err := r.db.WithContext(ctx).
		Preload("Group").Preload("Group.Creator").
		Preload("Inviter").Preload("Invitee").
		First(&invitation, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	refGroup := r.groupToDomain(invitation.Group)
	refInviter := r.userToDomain(invitation.Invitee)
	refInvitee := r.userToDomain(invitation.Invitee)
	return &domain.GroupInvitation{
		ID:        invitation.ID,
		GroupID:   invitation.GroupID,
		Group:     &refGroup,
		InviterID: invitation.InviterID,
		Inviter:   &refInviter,
		InviteeID: invitation.InviteeID,
		Invitee:   &refInvitee,
		Status:    invitation.Status,
		Message:   invitation.Message,
		CreatedAt: invitation.CreatedAt.Unix(),
		ExpiresAt: invitation.ExpiresAt.Unix(),
	}, nil
}

// GetPendingInvitation retrieves a pending invitation
func (r *Repository) GetPendingInvitation(ctx context.Context, groupID, inviteeID string) (*domain.GroupInvitation, error) {
	var invitation GroupInvitation
	if err := r.db.WithContext(ctx).
		Where("group_id = ? AND invitee_id = ? AND status = ?", groupID, inviteeID, "pending").
		First(&invitation).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &domain.GroupInvitation{
		ID:        invitation.ID,
		GroupID:   invitation.GroupID,
		InviterID: invitation.InviterID,
		InviteeID: invitation.InviteeID,
		Status:    invitation.Status,
		Message:   invitation.Message,
		CreatedAt: invitation.CreatedAt.Unix(),
		ExpiresAt: invitation.ExpiresAt.Unix(),
	}, nil
}

// GetPendingInvitationsForUser retrieves all pending invitations for a user
func (r *Repository) GetPendingInvitationsForUser(ctx context.Context, userID string, limit, offset int) ([]*domain.GroupInvitation, error) {
	var invitations []GroupInvitation

	if err := r.db.WithContext(ctx).
		Preload("Group").
		Preload("Inviter").
		Where("invitee_id = ? AND status = ?", userID, "pending").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&invitations).Error; err != nil {
		return nil, err
	}

	result := make([]*domain.GroupInvitation, len(invitations))
	for i, inv := range invitations {
		result[i] = &domain.GroupInvitation{
			ID:        inv.ID,
			GroupID:   inv.GroupID,
			InviterID: inv.InviterID,
			InviteeID: inv.InviteeID,
			Status:    inv.Status,
			Message:   inv.Message,
			CreatedAt: inv.CreatedAt.Unix(),
			ExpiresAt: inv.ExpiresAt.Unix(),
		}

		if inv.Group.ID != "" {
			result[i].Group = &domain.Group{
				ID:          inv.Group.ID,
				Name:        inv.Group.Name,
				Description: inv.Group.Description,
				Avatar:      inv.Group.Avatar,
				Members:     inv.Group.Members,
				Public:      inv.Group.Public,
			}
		}

		if inv.Inviter.ID != "" {
			result[i].Inviter = &domain.User{
				ID:          inv.Inviter.ID,
				Username:    inv.Inviter.Username,
				DisplayName: inv.Inviter.DisplayName,
				Avatar:      inv.Inviter.Avatar,
			}
		}
	}

	return result, nil
}

// UpdateInvitationStatus updates an invitation status
func (r *Repository) UpdateInvitationStatus(ctx context.Context, id, status string) error {
	result := r.db.WithContext(ctx).Model(&GroupInvitation{}).
		Where("id = ?", id).
		Update("status", status)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("invitation not found")
	}

	return nil
}

// ========== Group Follow Management ==========

// FollowGroup 关注群组
func (r *Repository) FollowGroup(ctx context.Context, userID, groupID string) error {
	// 检查是否已经关注
	var count int64
	if err := r.db.WithContext(ctx).Model(&GroupFollow{}).
		Where("group_id = ? AND user_id = ?", groupID, userID).
		Count(&count).Error; err != nil {
		return err
	}

	if count > 0 {
		return domain.ErrAlreadyExists
	}

	follow := GroupFollow{
		ID:      uuid.New().String(),
		GroupID: groupID,
		UserID:  userID,
	}

	if err := r.db.WithContext(ctx).Create(&follow).Error; err != nil {
		// Handle MySQL duplicate entry error (Error 1062)
		if strings.Contains(err.Error(), "Error 1062") ||
		   strings.Contains(err.Error(), "Duplicate entry") ||
		   strings.Contains(err.Error(), "23000") {
			return domain.ErrAlreadyExists
		}
		return err
	}

	// 更新群组关注者数量
	if err := r.db.WithContext(ctx).Model(&Group{}).
		Where("id = ?", groupID).
		UpdateColumn("followers", gorm.Expr("followers + ?", 1)).Error; err != nil {
		return err
	}

	return nil
}

// UnfollowGroup 取消关注群组
func (r *Repository) UnfollowGroup(ctx context.Context, userID, groupID string) error {
	result := r.db.WithContext(ctx).
		Where("group_id = ? AND user_id = ?", groupID, userID).
		Delete(&GroupFollow{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}

	// 更新群组关注者数量
	if err := r.db.WithContext(ctx).Model(&Group{}).
		Where("id = ?", groupID).
		UpdateColumn("followers", gorm.Expr("GREATEST(followers - ?, 0)", 1)).Error; err != nil {
		return err
	}

	return nil
}

// IsFollowingGroup 检查用户是否已关注群组
func (r *Repository) IsFollowingGroup(ctx context.Context, userID, groupID string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&GroupFollow{}).
		Where("group_id = ? AND user_id = ?", groupID, userID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// ListFollowedGroups 获取用户关注的群组列表
func (r *Repository) ListFollowedGroups(ctx context.Context, userID string, limit, offset int) ([]*domain.Group, error) {
	var follows []GroupFollow
	query := r.db.WithContext(ctx).Preload("Group").Preload("Group.Creator").
		Where("user_id = ?", userID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	if err := query.Find(&follows).Error; err != nil {
		return nil, err
	}

	result := make([]*domain.Group, len(follows))
	for i, f := range follows {
		refGroup := r.groupToDomain(f.Group)
		result[i] = &refGroup
	}
	return result, nil
}

