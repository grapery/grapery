package service

import (
	"context"
	"errors"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

// CreateGroupRequest 创建群组请求
type CreateGroupRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Description string `json:"description" binding:"max=2000"`
	Avatar      string `json:"avatar" binding:"omitempty,url"`
	IsPublic    bool   `json:"isPublic"`
}

// UpdateGroupRequest 更新群组请求
type UpdateGroupRequest struct {
	Name        *string `json:"name" binding:"omitempty,min=1,max=100"`
	Description *string `json:"description" binding:"omitempty,max=2000"`
	Avatar      *string `json:"avatar" binding:"omitempty,url"`
	IsPublic    *bool   `json:"isPublic"`
}

// GroupListRequest 群组列表请求
type GroupListRequest struct {
	CreatorID string `form:"creatorId"`
	Search    string `form:"search"`
	IsPublic  *bool  `form:"isPublic"`
	MyGroups  *bool  `form:"myGroups"` // true: 只返回我加入的群组, false: 只返回公开且未加入的群组
	Limit     int    `form:"limit" binding:"omitempty,min=1,max=100"`
	Offset    int    `form:"offset" binding:"omitempty,min=0"`
}

// InviteMemberRequest 邀请成员请求
type InviteMemberRequest struct {
	UserID  string `json:"userId" binding:"required"`
	Message string `json:"message" binding:"max=500"`
}

// UpdateMemberRoleRequest 更新成员角色请求
type UpdateMemberRoleRequest struct {
	Role string `json:"role" binding:"required,oneof=owner admin moderator member"`
}

// CreateGroup 创建群组
func (s *Service) CreateGroup(ctx context.Context, userID string, req CreateGroupRequest) (*domain.Group, error) {
	s.logger.Info("creating group",
		zap.String("userID", userID),
		zap.String("name", req.Name),
	)

	// 获取创建者信息
	creator, err := s.repo.UserByID(ctx, userID)
	if err != nil {
		s.logger.Error("failed to get creator", zap.Error(err))
		return nil, errors.New("creator not found")
	}

	// 创建群组
	group := &domain.Group{
		Name:        req.Name,
		Description: req.Description,
		Avatar:      req.Avatar,
		Creator:     creator,
		Members:     0, // 成员数量由 AddGroupMember 自动增加
		Stories:     0,
		Public:      req.IsPublic,
		CreatedAt:   time.Now().Unix(),
		UpdatedAt:   time.Now().Unix(),
	}

	if err := s.repo.CreateGroup(ctx, group); err != nil {
		s.logger.Error("failed to create group", zap.Error(err))
		return nil, errors.New("failed to create group")
	}

	// 将创建者添加为群主
	if err := s.repo.AddGroupMember(ctx, group.ID, userID, domain.RoleOwner, ""); err != nil {
		s.logger.Error("failed to add creator as owner", zap.Error(err))
		// 回滚群组创建
		s.repo.DeleteGroup(ctx, group.ID)
		return nil, errors.New("failed to initialize group")
	}

	s.logger.Info("group created successfully", zap.String("groupID", group.ID))
	return group, nil
}

// GetGroup 获取群组详情
func (s *Service) GetGroup(ctx context.Context, groupID string) (*domain.Group, error) {
	s.logger.Info("getting group", zap.String("groupID", groupID))

	group, err := s.repo.GroupByID(ctx, groupID)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, errors.New("group not found")
		}
		s.logger.Error("failed to get group", zap.Error(err))
		return nil, errors.New("failed to get group")
	}

	return group, nil
}

// ListGroups 获取群组列表
func (s *Service) ListGroups(ctx context.Context, userID string, req GroupListRequest) ([]*domain.Group, error) {
	s.logger.Info("listing groups",
		zap.String("userID", userID),
		zap.String("creatorId", req.CreatorID),
		zap.Int("limit", req.Limit),
		zap.Any("myGroups", req.MyGroups),
	)

	// 设置默认分页参数
	if req.Limit == 0 {
		req.Limit = 20
	}

	var groups []*domain.Group
	var err error

	// 根据 myGroups 参数选择不同的查询方式
	if req.MyGroups != nil && *req.MyGroups {
		// 只返回用户加入的群组
		if userID == "" {
			return []*domain.Group{}, nil // 未登录用户没有群组
		}
		groups, err = s.repo.ListMyGroups(ctx, userID, req.Limit, req.Offset)
		if err != nil {
			s.logger.Error("failed to list my groups", zap.Error(err))
			return nil, errors.New("failed to list my groups")
		}
		// 为每个群组查询当前用户的角色
		for _, group := range groups {
			role, err := s.repo.GetMemberRole(ctx, group.ID, userID)
			if err == nil {
				group.MyRole = &role
			}
		}
	} else if req.MyGroups != nil && !*req.MyGroups {
		// 只返回公开且用户未加入的群组
		groups, err = s.repo.ListPublicGroups(ctx, userID, req.Limit, req.Offset)
		if err != nil {
			s.logger.Error("failed to list public groups", zap.Error(err))
			return nil, errors.New("failed to list public groups")
		}
	} else {
		// 返回所有群组（兼容旧接口）
		groups, err = s.repo.ListGroups(ctx, req.Limit, req.Offset)
		if err != nil {
			s.logger.Error("failed to list groups", zap.Error(err))
			return nil, errors.New("failed to list groups")
		}
		// 为每个群组查询当前用户的角色
		if userID != "" {
			for _, group := range groups {
				role, err := s.repo.GetMemberRole(ctx, group.ID, userID)
				if err == nil {
					group.MyRole = &role
				}
			}
		}
	}

	s.logger.Info("groups listed successfully", zap.Int("count", len(groups)))
	return groups, nil
}

// UpdateGroup 更新群组
func (s *Service) UpdateGroup(ctx context.Context, userID, groupID string, req UpdateGroupRequest) (*domain.Group, error) {
	s.logger.Info("updating group",
		zap.String("userID", userID),
		zap.String("groupID", groupID),
	)

	// 获取群组
	group, err := s.repo.GroupByID(ctx, groupID)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, errors.New("group not found")
		}
		return nil, errors.New("failed to get group")
	}

	// 检查权限
	if !s.checkGroupPermission(ctx, userID, groupID, "edit") {
		return nil, errors.New("permission denied")
	}

	// 更新字段
	if req.Name != nil {
		group.Name = *req.Name
	}
	if req.Description != nil {
		group.Description = *req.Description
	}
	if req.Avatar != nil {
		group.Avatar = *req.Avatar
	}
	if req.IsPublic != nil {
		group.Public = *req.IsPublic
	}

	group.UpdatedAt = time.Now().Unix()

	if err := s.repo.UpdateGroup(ctx, group); err != nil {
		s.logger.Error("failed to update group", zap.Error(err))
		return nil, errors.New("failed to update group")
	}

	s.logger.Info("group updated successfully", zap.String("groupID", groupID))
	return group, nil
}

// UpdateGroupAvatarURL 更新群组头像URL
func (s *Service) UpdateGroupAvatarURL(ctx context.Context, userID, groupID, avatarURL string) error {
	s.logger.Info("updating group avatar",
		zap.String("userID", userID),
		zap.String("groupID", groupID),
	)

	// 获取群组
	group, err := s.repo.GroupByID(ctx, groupID)
	if err != nil {
		if err == domain.ErrNotFound {
			return errors.New("group not found")
		}
		return errors.New("failed to get group")
	}

	// 检查权限
	if !s.checkGroupPermission(ctx, userID, groupID, "edit") {
		return errors.New("permission denied")
	}

	// 更新群组头像
	group.Avatar = avatarURL
	group.UpdatedAt = time.Now().Unix()

	if err := s.repo.UpdateGroup(ctx, group); err != nil {
		s.logger.Error("failed to update group avatar", zap.Error(err))
		return errors.New("failed to update group avatar")
	}

	s.logger.Info("group avatar updated successfully", zap.String("groupID", groupID))
	return nil
}

// DeleteGroup 删除群组
func (s *Service) DeleteGroup(ctx context.Context, userID, groupID string) error {
	s.logger.Info("deleting group",
		zap.String("userID", userID),
		zap.String("groupID", groupID),
	)

	// 获取群组
	_, err := s.repo.GroupByID(ctx, groupID)
	if err != nil {
		if err == domain.ErrNotFound {
			return errors.New("group not found")
		}
		return errors.New("failed to get group")
	}

	// 检查权限（仅群主可删除）
	if !s.checkGroupPermission(ctx, userID, groupID, "delete") {
		return errors.New("permission denied: only owner can delete group")
	}

	if err := s.repo.DeleteGroup(ctx, groupID); err != nil {
		s.logger.Error("failed to delete group", zap.Error(err))
		return errors.New("failed to delete group")
	}

	s.logger.Info("group deleted successfully", zap.String("groupID", groupID))
	return nil
}

// GetGroupMembers 获取群组成员列表
func (s *Service) GetGroupMembers(ctx context.Context, groupID string, limit, offset int) ([]*domain.GroupMemberInfo, error) {
	s.logger.Info("getting group members", zap.String("groupID", groupID))

	// 验证群组存在
	_, err := s.repo.GroupByID(ctx, groupID)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, errors.New("group not found")
		}
		return nil, errors.New("failed to get group")
	}

	members, err := s.repo.GetGroupMembers(ctx, groupID, limit, offset)
	if err != nil {
		s.logger.Error("failed to get group members", zap.Error(err))
		return nil, errors.New("failed to get group members")
	}

	return members, nil
}

// InviteMember 邀请成员加入群组
func (s *Service) InviteMember(ctx context.Context, inviterID, groupID string, req InviteMemberRequest) (*domain.GroupInvitation, error) {
	s.logger.Info("inviting member",
		zap.String("inviterID", inviterID),
		zap.String("groupID", groupID),
		zap.String("inviteeID", req.UserID),
	)

	// 检查邀请权限
	if !s.checkGroupPermission(ctx, inviterID, groupID, "invite") {
		return nil, errors.New("permission denied: you cannot invite members")
	}

	// 检查被邀请者是否已是成员
	isMember, err := s.repo.IsGroupMember(ctx, groupID, req.UserID)
	if err != nil {
		return nil, errors.New("failed to check membership")
	}
	if isMember {
		return nil, errors.New("user is already a member")
	}

	// 检查是否已有待处理的邀请
	existingInvitation, _ := s.repo.GetPendingInvitation(ctx, groupID, req.UserID)
	if existingInvitation != nil {
		return nil, errors.New("invitation already sent")
	}

	// 创建邀请
	invitation, err := s.repo.CreateGroupInvitation(ctx, groupID, inviterID, req.UserID, req.Message)
	if err != nil {
		s.logger.Error("failed to create invitation", zap.Error(err))
		return nil, errors.New("failed to create invitation")
	}

	s.logger.Info("invitation created successfully", zap.String("invitationID", invitation.ID))
	return invitation, nil
}

// AcceptInvitation 接受群组邀请
func (s *Service) AcceptInvitation(ctx context.Context, userID, invitationID string) error {
	s.logger.Info("accepting invitation",
		zap.String("userID", userID),
		zap.String("invitationID", invitationID),
	)

	// 获取邀请
	invitation, err := s.repo.GetInvitationByID(ctx, invitationID)
	if err != nil {
		if err == domain.ErrNotFound {
			return errors.New("invitation not found")
		}
		return errors.New("failed to get invitation")
	}

	// 验证邀请接收者
	if invitation.InviteeID != userID {
		return errors.New("this invitation is not for you")
	}

	// 验证邀请状态
	if invitation.Status != "pending" {
		return errors.New("invitation is no longer valid")
	}

	// 验证邀请是否过期
	if time.Now().Unix() > invitation.ExpiresAt {
		s.repo.UpdateInvitationStatus(ctx, invitationID, "expired")
		return errors.New("invitation has expired")
	}

	// 添加成员
	if err := s.repo.AddGroupMember(ctx, invitation.GroupID, userID, domain.RoleMember, invitation.InviterID); err != nil {
		s.logger.Error("failed to add member", zap.Error(err))
		return errors.New("failed to join group")
	}

	// 更新邀请状态
	if err := s.repo.UpdateInvitationStatus(ctx, invitationID, "accepted"); err != nil {
		s.logger.Error("failed to update invitation status", zap.Error(err))
	}

	// 记录新成员加入的群组活动
	go s.RecordGroupMemberJoined(context.Background(), invitation.GroupID, userID)

	s.logger.Info("invitation accepted successfully", zap.String("groupID", invitation.GroupID))
	return nil
}

// RejectInvitation 拒绝群组邀请
func (s *Service) RejectInvitation(ctx context.Context, userID, invitationID string) error {
	s.logger.Info("rejecting invitation",
		zap.String("userID", userID),
		zap.String("invitationID", invitationID),
	)

	// 获取邀请
	invitation, err := s.repo.GetInvitationByID(ctx, invitationID)
	if err != nil {
		if err == domain.ErrNotFound {
			return errors.New("invitation not found")
		}
		return errors.New("failed to get invitation")
	}

	// 验证邀请接收者
	if invitation.InviteeID != userID {
		return errors.New("this invitation is not for you")
	}

	// 更新邀请状态
	if err := s.repo.UpdateInvitationStatus(ctx, invitationID, "rejected"); err != nil {
		s.logger.Error("failed to update invitation status", zap.Error(err))
		return errors.New("failed to reject invitation")
	}

	s.logger.Info("invitation rejected successfully")
	return nil
}

// GetPendingInvitations 获取用户待处理的群组邀请
func (s *Service) GetPendingInvitations(ctx context.Context, userID string, limit, offset int) ([]*domain.GroupInvitation, error) {
	s.logger.Info("getting pending invitations",
		zap.String("userID", userID),
	)

	if limit == 0 {
		limit = 20
	}

	invitations, err := s.repo.GetPendingInvitationsForUser(ctx, userID, limit, offset)
	if err != nil {
		s.logger.Error("failed to get pending invitations", zap.Error(err))
		return nil, errors.New("failed to get pending invitations")
	}

	return invitations, nil
}

// RemoveMember 移除群组成员
func (s *Service) RemoveMember(ctx context.Context, operatorID, groupID, memberID string) error {
	s.logger.Info("removing member",
		zap.String("operatorID", operatorID),
		zap.String("groupID", groupID),
		zap.String("memberID", memberID),
	)

	// 检查权限
	if !s.checkGroupPermission(ctx, operatorID, groupID, "remove") {
		return errors.New("permission denied: you cannot remove members")
	}

	// 不能移除群主
	memberRole, err := s.repo.GetMemberRole(ctx, groupID, memberID)
	if err != nil {
		return errors.New("failed to get member role")
	}
	if memberRole == domain.RoleOwner {
		return errors.New("cannot remove group owner")
	}

	// 移除成员
	if err := s.repo.RemoveGroupMember(ctx, groupID, memberID); err != nil {
		s.logger.Error("failed to remove member", zap.Error(err))
		return errors.New("failed to remove member")
	}

	s.logger.Info("member removed successfully")
	return nil
}

// UpdateMemberRole 更新成员角色
func (s *Service) UpdateMemberRole(ctx context.Context, operatorID, groupID, memberID string, req UpdateMemberRoleRequest) error {
	s.logger.Info("updating member role",
		zap.String("operatorID", operatorID),
		zap.String("groupID", groupID),
		zap.String("memberID", memberID),
		zap.String("newRole", req.Role),
	)

	// 检查权限
	if !s.checkGroupPermission(ctx, operatorID, groupID, "manageRoles") {
		return errors.New("permission denied: you cannot manage roles")
	}

	// 不能修改群主角色
	memberRole, err := s.repo.GetMemberRole(ctx, groupID, memberID)
	if err != nil {
		return errors.New("failed to get member role")
	}
	if memberRole == domain.RoleOwner {
		return errors.New("cannot change owner role")
	}

	// 更新角色
	newRole := domain.GroupMemberRole(req.Role)
	if err := s.repo.UpdateMemberRole(ctx, groupID, memberID, newRole); err != nil {
		s.logger.Error("failed to update member role", zap.Error(err))
		return errors.New("failed to update member role")
	}

	s.logger.Info("member role updated successfully")
	return nil
}

// LeaveGroup 离开群组
func (s *Service) LeaveGroup(ctx context.Context, userID, groupID string) error {
	s.logger.Info("leaving group",
		zap.String("userID", userID),
		zap.String("groupID", groupID),
	)

	// 检查是否是群主
	role, err := s.repo.GetMemberRole(ctx, groupID, userID)
	if err != nil {
		return errors.New("you are not a member of this group")
	}

	if role == domain.RoleOwner {
		return errors.New("owner cannot leave group, please delete the group or transfer ownership first")
	}

	// 离开群组
	if err := s.repo.RemoveGroupMember(ctx, groupID, userID); err != nil {
		s.logger.Error("failed to leave group", zap.Error(err))
		return errors.New("failed to leave group")
	}

	s.logger.Info("left group successfully")
	return nil
}

// checkGroupPermission 检查群组权限
func (s *Service) checkGroupPermission(ctx context.Context, userID, groupID, action string) bool {
	role, err := s.repo.GetMemberRole(ctx, groupID, userID)
	if err != nil {
		return false
	}

	perm := domain.GetPermissions(role)

	switch action {
	case "edit":
		return perm.CanEditGroup
	case "delete":
		return perm.CanDeleteGroup
	case "invite":
		return perm.CanInviteMembers
	case "remove":
		return perm.CanRemoveMembers
	case "manageRoles":
		return perm.CanManageRoles
	default:
		return false
	}
}
