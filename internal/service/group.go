package service

import (
	"context"
	"errors"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/cache"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

// CreateGroupRequest 创建群组请求
type CreateGroupRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Description string `json:"description" binding:"max=2000"`
	Avatar      string `json:"avatar" binding:"omitempty,url"`
	// Keep consistent with API responses and mobile clients (snake_case).
	IsPublic bool `json:"is_public"`
}

// UpdateGroupRequest 更新群组请求
type UpdateGroupRequest struct {
	Name        *string `json:"name" binding:"omitempty,min=1,max=100"`
	Description *string `json:"description" binding:"omitempty,max=2000"`
	Avatar      *string `json:"avatar" binding:"omitempty,url"`
	// Keep consistent with API responses and mobile clients (snake_case).
	IsPublic *bool `json:"is_public"`
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
		s.logger.Error("failed to create group",
			zap.String("groupID", group.ID),
			zap.Error(err))
		return nil, errors.New("failed to create group")
	}

	// 将创建者添加为群主
	if err := s.repo.AddGroupMember(ctx, group.ID, userID, domain.RoleOwner, ""); err != nil {
		s.logger.Error("failed to add creator as owner",
			zap.String("groupID", group.ID),
			zap.String("userID", userID),
			zap.Error(err))
		// 回滚群组创建
		s.repo.DeleteGroup(ctx, group.ID)
		return nil, errors.New("failed to initialize group")
	}

	// Record metrics: group member count (creator is first member)
	if s.metrics != nil {
		s.metrics.RecordGroupMemberCount(group.ID, 1.0)
	}

	// 缓存新创建的群组
	c := s.getCache()
	if c != nil {
		key := cache.GroupKey(group.ID)
		if err := c.Set(ctx, key, group, entityCacheTTL); err != nil {
			s.logger.Warn("failed to cache new group",
				zap.String("groupID", group.ID),
				zap.Error(err))
		}
	}

	s.logger.Info("group created successfully",
		zap.String("groupID", group.ID))
	return group, nil
}

// GetGroup 获取群组详情（带缓存）
func (s *Service) GetGroup(ctx context.Context, groupID, userID string) (*domain.Group, error) {
	s.logger.Info("getting group",
		zap.String("groupID", groupID),
		zap.String("userID", userID))

	// 尝试从缓存获取
	c := s.getCache()
	if c != nil {
		key := cache.GroupKey(groupID)
		var cachedGroup domain.Group
		if err := c.Get(ctx, key, &cachedGroup); err == nil {
			s.logger.Debug("group cache hit",
				zap.String("groupID", groupID))
			// 缓存命中后，仍然需要查询实时的角色和关注状态
			if userID != "" {
				role, err := s.repo.GetMemberRole(ctx, groupID, userID)
				if err == nil {
					cachedGroup.MyRole = &role
				}
				isFollowing, err := s.repo.IsFollowingGroup(ctx, userID, groupID)
				if err == nil {
					cachedGroup.IsFollowing = &isFollowing
				}
			}
			return &cachedGroup, nil
		} else {
			s.logger.Debug("group cache miss",
				zap.String("groupID", groupID),
				zap.Error(err))
		}
	}

	// 从数据库获取
	group, err := s.repo.GroupByID(ctx, groupID)
	if err != nil {
		if err == domain.ErrNotFound {
			s.logger.Warn("group not found",
				zap.String("groupID", groupID))
			return nil, errors.New("group not found")
		}
		s.logger.Error("failed to get group",
			zap.String("groupID", groupID),
			zap.Error(err))
		return nil, errors.New("failed to get group")
	}

	// 查询用户角色和关注状态
	if userID != "" {
		role, err := s.repo.GetMemberRole(ctx, groupID, userID)
		if err == nil {
			group.MyRole = &role
		}
		isFollowing, err := s.repo.IsFollowingGroup(ctx, userID, groupID)
		if err == nil {
			group.IsFollowing = &isFollowing
		}
	}

	// 写入缓存
	if c != nil {
		key := cache.GroupKey(groupID)
		if err := c.Set(ctx, key, group, entityCacheTTL); err != nil {
			s.logger.Warn("failed to cache group",
				zap.String("groupID", groupID),
				zap.Error(err))
		} else {
			s.logger.Debug("group cached",
				zap.String("groupID", groupID))
		}
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
			// 为每个群组查询当前用户的角色和关注状态
		for _, group := range groups {
			role, err := s.repo.GetMemberRole(ctx, group.ID, userID)
			if err == nil {
				group.MyRole = &role
			}

			// 查询关注状态
			isFollowing, err := s.repo.IsFollowingGroup(ctx, userID, group.ID)
			if err == nil {
				group.IsFollowing = &isFollowing
			}
		}
	} else if req.MyGroups != nil && !*req.MyGroups {
		// 只返回公开且用户未加入的群组
		groups, err = s.repo.ListPublicGroups(ctx, userID, req.Limit, req.Offset)
		if err != nil {
			s.logger.Error("failed to list public groups", zap.Error(err))
			return nil, errors.New("failed to list public groups")
		}
		// 为每个群组查询关注状态
		for _, group := range groups {
			if userID != "" {
				isFollowing, err := s.repo.IsFollowingGroup(ctx, userID, group.ID)
				if err == nil {
					group.IsFollowing = &isFollowing
				}
			}
		}
	} else {
		// 返回所有群组(兼容旧接口)
		groups, err = s.repo.ListGroups(ctx, req.Limit, req.Offset)
		if err != nil {
			s.logger.Error("failed to list groups", zap.Error(err))
			return nil, errors.New("failed to list groups")
		}
		// 为每个群组查询当前用户的角色和关注状态
		if userID != "" {
			for _, group := range groups {
				role, err := s.repo.GetMemberRole(ctx, group.ID, userID)
				if err == nil {
					group.MyRole = &role
				}

				// 查询关注状态
				isFollowing, err := s.repo.IsFollowingGroup(ctx, userID, group.ID)
				if err == nil {
					group.IsFollowing = &isFollowing
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
		s.logger.Error("failed to update group",
			zap.String("groupID", groupID),
			zap.Error(err))
		return nil, errors.New("failed to update group")
	}

	// 使缓存失效并重新缓存
	c := s.getCache()
	if c != nil {
		key := cache.GroupKey(groupID)
		if err := c.Delete(ctx, key); err != nil {
			s.logger.Warn("failed to invalidate group cache",
				zap.String("groupID", groupID),
				zap.Error(err))
		}
		// 重新缓存
		if err := c.Set(ctx, key, group, entityCacheTTL); err != nil {
			s.logger.Warn("failed to cache updated group",
				zap.String("groupID", groupID),
				zap.Error(err))
		}
	}

	s.logger.Info("group updated successfully",
		zap.String("groupID", groupID))
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
		s.logger.Error("failed to delete group",
			zap.String("groupID", groupID),
			zap.Error(err))
		return errors.New("failed to delete group")
	}

	// 使缓存失效
	c := s.getCache()
	if c != nil {
		key := cache.GroupKey(groupID)
		if err := c.Delete(ctx, key); err != nil {
			s.logger.Warn("failed to invalidate group cache",
				zap.String("groupID", groupID),
				zap.Error(err))
		}
		// 清除相关列表缓存
		for limit := 20; limit <= 100; limit += 20 {
			for offset := 0; offset < 200; offset += limit {
				_ = c.Delete(ctx, cache.GroupMembersListKey(groupID, limit, offset))
				_ = c.Delete(ctx, cache.GroupActivitiesKey(groupID, limit))
			}
		}
	}

	s.logger.Info("group deleted successfully",
		zap.String("groupID", groupID))
	return nil
}

// GetGroupMembers 获取群组成员列表（带缓存）
func (s *Service) GetGroupMembers(ctx context.Context, groupID string, limit, offset int) ([]*domain.GroupMemberInfo, error) {
	s.logger.Info("getting group members",
		zap.String("groupID", groupID),
		zap.Int("limit", limit),
		zap.Int("offset", offset))

	// 验证群组存在
	_, err := s.repo.GroupByID(ctx, groupID)
	if err != nil {
		if err == domain.ErrNotFound {
			s.logger.Warn("group not found",
				zap.String("groupID", groupID))
			return nil, errors.New("group not found")
		}
		s.logger.Error("failed to get group",
			zap.String("groupID", groupID),
			zap.Error(err))
		return nil, errors.New("failed to get group")
	}

	// 尝试从缓存获取
	c := s.getCache()
	if c != nil {
		cacheKey := cache.GroupMembersListKey(groupID, limit, offset)
		var cachedMembers []*domain.GroupMemberInfo
		if err := c.Get(ctx, cacheKey, &cachedMembers); err == nil {
			s.logger.Debug("group members cache hit",
				zap.String("groupID", groupID),
				zap.Int("count", len(cachedMembers)))
			return cachedMembers, nil
		} else {
			s.logger.Debug("group members cache miss",
				zap.String("groupID", groupID),
				zap.Error(err))
		}
	}

	// 从数据库获取
	members, err := s.repo.GetGroupMembers(ctx, groupID, limit, offset)
	if err != nil {
		s.logger.Error("failed to get group members",
			zap.String("groupID", groupID),
			zap.Error(err))
		return nil, errors.New("failed to get group members")
	}

	// Record metrics - get total member count from group
	if s.metrics != nil {
		group, err := s.repo.GroupByID(ctx, groupID)
		if err == nil {
			s.metrics.RecordGroupMemberCount(groupID, float64(group.Members))
		}
	}

	// 写入缓存
	if c != nil && len(members) > 0 {
		cacheKey := cache.GroupMembersListKey(groupID, limit, offset)
		if err := c.Set(ctx, cacheKey, members, groupMemberCacheTTL); err != nil {
			s.logger.Warn("failed to cache group members",
				zap.String("groupID", groupID),
				zap.Error(err))
		} else {
			s.logger.Debug("group members cached",
				zap.String("groupID", groupID),
				zap.Int("count", len(members)))
		}
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
		s.logger.Error("failed to add member",
			zap.String("groupID", invitation.GroupID),
			zap.String("userID", userID),
			zap.Error(err))
		return errors.New("failed to join group")
	}

	// Record metrics: update group member count
	if s.metrics != nil {
		members, err := s.repo.GetGroupMembers(ctx, invitation.GroupID, 1000, 0)
		if err == nil {
			s.metrics.RecordGroupMemberCount(invitation.GroupID, float64(len(members)))
		}
	}

	// 更新邀请状态
	if err := s.repo.UpdateInvitationStatus(ctx, invitationID, "accepted"); err != nil {
		s.logger.Error("failed to update invitation status",
			zap.String("invitationID", invitationID),
			zap.Error(err))
	}

	// 使相关缓存失效
	c := s.getCache()
	if c != nil {
		// 清除群组成员列表缓存
		for limit := 20; limit <= 100; limit += 20 {
			for offset := 0; offset < 200; offset += limit {
				_ = c.Delete(ctx, cache.GroupMembersListKey(invitation.GroupID, limit, offset))
			}
		}
		// 清除用户群组列表缓存
		for limit := 20; limit <= 100; limit += 20 {
			for offset := 0; offset < 200; offset += limit {
				_ = c.Delete(ctx, cache.UserGroupsListKey(userID, limit, offset))
			}
		}
		s.logger.Debug("group member cache invalidated",
			zap.String("groupID", invitation.GroupID),
			zap.String("userID", userID))
	}

	// 记录新成员加入的群组活动
	go s.RecordGroupMemberJoined(context.Background(), invitation.GroupID, userID)

	s.logger.Info("invitation accepted successfully",
		zap.String("groupID", invitation.GroupID),
		zap.String("userID", userID))
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
		s.logger.Error("failed to remove member",
			zap.String("groupID", groupID),
			zap.String("memberID", memberID),
			zap.Error(err))
		return errors.New("failed to remove member")
	}

	// Record metrics: update group member count
	if s.metrics != nil {
		members, err := s.repo.GetGroupMembers(ctx, groupID, 1000, 0)
		if err == nil {
			s.metrics.RecordGroupMemberCount(groupID, float64(len(members)))
		}
	}

	// 使相关缓存失效
	c := s.getCache()
	if c != nil {
		// 清除群组成员列表缓存
		for limit := 20; limit <= 100; limit += 20 {
			for offset := 0; offset < 200; offset += limit {
				_ = c.Delete(ctx, cache.GroupMembersListKey(groupID, limit, offset))
			}
		}
		// 清除用户群组列表缓存
		for limit := 20; limit <= 100; limit += 20 {
			for offset := 0; offset < 200; offset += limit {
				_ = c.Delete(ctx, cache.UserGroupsListKey(memberID, limit, offset))
			}
		}
		s.logger.Debug("group member cache invalidated",
			zap.String("groupID", groupID),
			zap.String("memberID", memberID))
	}

	s.logger.Info("member removed successfully",
		zap.String("groupID", groupID),
		zap.String("memberID", memberID))
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

	// Record metrics: update group member count
	if s.metrics != nil {
		members, err := s.repo.GetGroupMembers(ctx, groupID, 1000, 0)
		if err == nil {
			s.metrics.RecordGroupMemberCount(groupID, float64(len(members)))
		}
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

// ========== Group Roles Management ==========

// InitializeGroupRoles 初始化系统内置角色
func (s *Service) InitializeGroupRoles(ctx context.Context) error {
	s.logger.Info("initializing group roles")
	return s.repo.InitializeGroupRoles(ctx)
}

// GetGroupRoleByCode 根据代码获取群组角色
func (s *Service) GetGroupRoleByCode(ctx context.Context, code string) (*domain.GroupRole, error) {
	role, err := s.repo.GetGroupRoleByCode(ctx, code)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, errors.New("role not found")
		}
		return nil, errors.New("failed to get role")
	}
	return role, nil
}

// GetGroupRoleByID 根据ID获取群组角色
func (s *Service) GetGroupRoleByID(ctx context.Context, id string) (*domain.GroupRole, error) {
	role, err := s.repo.GetGroupRoleByID(ctx, id)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, errors.New("role not found")
		}
		return nil, errors.New("failed to get role")
	}
	return role, nil
}

// ListGroupRoles 列出所有群组角色
func (s *Service) ListGroupRoles(ctx context.Context) ([]*domain.GroupRole, error) {
	roles, err := s.repo.ListGroupRoles(ctx)
	if err != nil {
		s.logger.Error("failed to list group roles", zap.Error(err))
		return nil, errors.New("failed to list roles")
	}
	return roles, nil
}

// GetGroupRolePermissions 获取角色的权限
func (s *Service) GetGroupRolePermissions(ctx context.Context, roleCode string) (*domain.GroupRolePermission, error) {
	permissions := domain.GetRolePermissions(roleCode)
	return &permissions, nil
}

// UpdateMemberRoleByCode 根据角色代码更新成员角色
func (s *Service) UpdateMemberRoleByCode(ctx context.Context, operatorID, groupID, memberID, roleCode string) error {
	s.logger.Info("updating member role by code",
		zap.String("operatorID", operatorID),
		zap.String("groupID", groupID),
		zap.String("memberID", memberID),
		zap.String("roleCode", roleCode),
	)

	// 验证操作者权限
	operatorRole, err := s.repo.GetMemberRole(ctx, groupID, operatorID)
	if err != nil {
		return errors.New("you are not a member of this group")
	}

	operatorPerm := domain.GetPermissions(operatorRole)
	if !operatorPerm.CanManageRoles {
		return errors.New("you do not have permission to manage roles")
	}

	// 获取目标成员当前角色
	memberRole, err := s.repo.GetMemberRole(ctx, groupID, memberID)
	if err != nil {
		return errors.New("member not found")
	}

	// 不能修改群主角色
	if memberRole == domain.RoleOwner {
		return errors.New("cannot change owner role")
	}

	// 获取角色ID
	role, err := s.repo.GetGroupRoleByCode(ctx, roleCode)
	if err != nil {
		return errors.New("invalid role code")
	}

	// 更新角色ID
	if err := s.repo.UpdateMemberRoleID(ctx, groupID, memberID, role.ID); err != nil {
		s.logger.Error("failed to update member role ID", zap.Error(err))
		return errors.New("failed to update member role")
	}

	// 同时更新旧的Role字段以保持兼容性
	var newRole domain.GroupMemberRole
	switch roleCode {
	case domain.RoleCodeCreator:
		newRole = domain.RoleOwner
	case domain.RoleCodeAdmin:
		newRole = domain.RoleAdmin
	case domain.RoleCodeMember:
		newRole = domain.RoleMember
	default:
		newRole = domain.RoleMember
	}
	if err := s.repo.UpdateMemberRole(ctx, groupID, memberID, newRole); err != nil {
		s.logger.Warn("failed to update legacy role field", zap.Error(err))
	}

	s.logger.Info("member role updated successfully")
	return nil
}

// ========== Group Follow Management ==========

// FollowGroup 关注群组
func (s *Service) FollowGroup(ctx context.Context, userID, groupID string) (*domain.Group, error) {
	s.logger.Info("following group",
		zap.String("userID", userID),
		zap.String("groupID", groupID),
	)

	// 检查群组是否存在
	group, err := s.repo.GroupByID(ctx, groupID)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, errors.New("group not found")
		}
		return nil, errors.New("failed to get group")
	}

	// 检查是否已关注
	isFollowing, err := s.repo.IsFollowingGroup(ctx, userID, groupID)
	if err != nil {
		s.logger.Error("failed to check follow status", zap.Error(err))
		return nil, errors.New("failed to check follow status")
	}
	if isFollowing {
		return nil, errors.New("already following this group")
	}

	// 创建关注记录
	if err := s.repo.FollowGroup(ctx, userID, groupID); err != nil {
		// Treat "already following" as success (idempotent operation)
		if errors.Is(err, domain.ErrAlreadyExists) {
			s.logger.Info("group already followed",
				zap.String("userID", userID),
				zap.String("groupID", groupID))
			// Return the current group without incrementing followers
			isFollowing := true
			group.IsFollowing = &isFollowing
			return group, nil
		} else {
			s.logger.Error("failed to follow group", zap.Error(err))
			return nil, errors.New("failed to follow group")
		}
	}

	// 更新群组关注者数量
	group.Followers++
	group.UpdatedAt = time.Now().Unix()
	if err := s.repo.UpdateGroup(ctx, group); err != nil {
		s.logger.Error("failed to update follower count", zap.Error(err))
	}

	// 使缓存失效
	c := s.getCache()
	if c != nil {
		key := cache.GroupKey(groupID)
		_ = c.Delete(ctx, key)
	}

	s.logger.Info("group followed successfully",
		zap.String("groupID", groupID),
		zap.Int("followers", group.Followers))

	return group, nil
}

// UnfollowGroup 取消关注群组
func (s *Service) UnfollowGroup(ctx context.Context, userID, groupID string) error {
	s.logger.Info("unfollowing group",
		zap.String("userID", userID),
		zap.String("groupID", groupID),
	)

	// 检查群组是否存在
	_, err := s.repo.GroupByID(ctx, groupID)
	if err != nil {
		if err == domain.ErrNotFound {
			return errors.New("group not found")
		}
		return errors.New("failed to get group")
	}

	// 检查是否已关注
	isFollowing, err := s.repo.IsFollowingGroup(ctx, userID, groupID)
	if err != nil {
		s.logger.Error("failed to check follow status", zap.Error(err))
		return errors.New("failed to check follow status")
	}
	if !isFollowing {
		return errors.New("not following this group")
	}

	// 删除关注记录
	if err := s.repo.UnfollowGroup(ctx, userID, groupID); err != nil {
		s.logger.Error("failed to unfollow group", zap.Error(err))
		return errors.New("failed to unfollow group")
	}

	// 更新群组关注者数量
	group, err := s.repo.GroupByID(ctx, groupID)
	if err == nil {
		if group.Followers > 0 {
			group.Followers--
		}
		group.UpdatedAt = time.Now().Unix()
		if err := s.repo.UpdateGroup(ctx, group); err != nil {
			s.logger.Error("failed to update follower count", zap.Error(err))
		}
	}

	// 使缓存失效
	c := s.getCache()
	if c != nil {
		key := cache.GroupKey(groupID)
		_ = c.Delete(ctx, key)
	}

	s.logger.Info("group unfollowed successfully",
		zap.String("groupID", groupID))
	return nil
}

// ListFollowedGroups 获取用户关注的群组列表
func (s *Service) ListFollowedGroups(ctx context.Context, userID string, limit, offset int) ([]*domain.Group, error) {
	s.logger.Info("listing followed groups",
		zap.String("userID", userID),
		zap.Int("limit", limit),
		zap.Int("offset", offset),
	)

	if limit == 0 {
		limit = 20
	}

	groups, err := s.repo.ListFollowedGroups(ctx, userID, limit, offset)
	if err != nil {
		s.logger.Error("failed to list followed groups", zap.Error(err))
		return nil, errors.New("failed to list followed groups")
	}

	// 为每个群组设置关注状态
	for _, group := range groups {
		isFollowing := true
		group.IsFollowing = &isFollowing
	}

	s.logger.Info("followed groups listed successfully", zap.Int("count", len(groups)))
	return groups, nil
}

// ToggleGroupFollow 切换群组关注状态
func (s *Service) ToggleGroupFollow(ctx context.Context, userID, groupID string) (*domain.Group, error) {
	s.logger.Info("toggling group follow",
		zap.String("userID", userID),
		zap.String("groupID", groupID),
	)

	// 检查当前关注状态
	isFollowing, err := s.repo.IsFollowingGroup(ctx, userID, groupID)
	if err != nil {
		s.logger.Error("failed to check follow status", zap.Error(err))
		return nil, errors.New("failed to check follow status")
	}

	if isFollowing {
		// 取消关注
		if err := s.UnfollowGroup(ctx, userID, groupID); err != nil {
			return nil, err
		}
		group, _ := s.repo.GroupByID(ctx, groupID)
		if group != nil {
			following := false
			group.IsFollowing = &following
			return group, nil
		}
		return nil, nil
	} else {
		// 关注
		return s.FollowGroup(ctx, userID, groupID)
	}
}
