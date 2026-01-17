package http

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	authPkg "github.com/grapestree/fgrapery/grapery/internal/auth"
	"github.com/grapestree/fgrapery/grapery/internal/service"
)

// CreateGroup 创建群组
func (h *Handler) CreateGroup(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	var req service.CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, err.Error())
		return
	}

	group, err := h.svc.CreateGroup(c.Request.Context(), userID, req)
	if err != nil {
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, group)
}

// GetGroup 获取群组详情
func (h *Handler) GetGroup(c *gin.Context) {
	groupID := c.Param("id")
	if groupID == "" {
		InvalidParams(c, "group id is required")
		return
	}

	userID := authPkg.GetUserID(c)

	group, err := h.svc.GetGroup(c.Request.Context(), groupID, userID)
	if err != nil {
		if err.Error() == "group not found" {
			NotFound(c, "group not found")
			return
		}
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, group)
}

// ListGroups 获取群组列表
func (h *Handler) ListGroups(c *gin.Context) {
	// 获取当前用户ID（可能为空，表示未登录）
	userID := authPkg.GetUserID(c)

	var req service.GroupListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		InvalidParams(c, err.Error())
		return
	}

	groups, err := h.svc.ListGroups(c.Request.Context(), userID, req)
	if err != nil {
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, gin.H{
		"groups": groups,
		"count":  len(groups),
		"limit":  req.Limit,
		"offset": req.Offset,
	})
}

// UpdateGroup 更新群组
func (h *Handler) UpdateGroup(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	groupID := c.Param("id")
	if groupID == "" {
		InvalidParams(c, "group id is required")
		return
	}

	var req service.UpdateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, err.Error())
		return
	}

	group, err := h.svc.UpdateGroup(c.Request.Context(), userID, groupID, req)
	if err != nil {
		if err.Error() == "permission denied" {
			Forbidden(c, "you don't have permission to edit this group")
			return
		}
		if err.Error() == "group not found" {
			NotFound(c, "group not found")
			return
		}
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, group)
}

// UpdateGroupAvatar 更新群组头像
func (h *Handler) UpdateGroupAvatar(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	groupID := c.Param("id")
	if groupID == "" {
		InvalidParams(c, "group id is required")
		return
	}

	// 获取上传的文件
	file, err := c.FormFile("file")
	if err != nil {
		InvalidParams(c, "file is required")
		return
	}

	// 验证文件类型
	contentType := file.Header.Get("Content-Type")
	if contentType != "image/jpeg" && contentType != "image/png" && contentType != "image/gif" && contentType != "image/webp" {
		InvalidParams(c, "only JPEG, PNG, GIF and WebP images are allowed")
		return
	}

	// 限制文件大小 (5MB)
	if file.Size > 5*1024*1024 {
		InvalidParams(c, "file size must be less than 5MB")
		return
	}

	// 生成唯一文件名
	ext := ".jpg"
	switch contentType {
	case "image/png":
		ext = ".png"
	case "image/gif":
		ext = ".gif"
	case "image/webp":
		ext = ".webp"
	}
	newFilename := fmt.Sprintf("group_%s_%d%s", groupID, time.Now().Unix(), ext)

	// 创建上传目录
	uploadDir := "uploads/groups"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		InternalError(c, "failed to create upload directory")
		return
	}

	// 保存文件
	dst := filepath.Join(uploadDir, newFilename)
	if err := c.SaveUploadedFile(file, dst); err != nil {
		InternalError(c, "failed to save file")
		return
	}

	// 生成文件 URL
	avatarURL := fmt.Sprintf("/uploads/groups/%s", newFilename)

	// 更新群组头像
	err = h.svc.UpdateGroupAvatarURL(c.Request.Context(), userID, groupID, avatarURL)
	if err != nil {
		// 删除已上传的文件
		os.Remove(dst)
		if err.Error() == "permission denied" {
			Forbidden(c, "you don't have permission to edit this group")
			return
		}
		if err.Error() == "group not found" {
			NotFound(c, "group not found")
			return
		}
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, gin.H{"url": avatarURL})
}

// DeleteGroup 删除群组
func (h *Handler) DeleteGroup(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	groupID := c.Param("id")
	if groupID == "" {
		InvalidParams(c, "group id is required")
		return
	}

	err := h.svc.DeleteGroup(c.Request.Context(), userID, groupID)
	if err != nil {
		if err.Error() == "permission denied: only owner can delete group" {
			Forbidden(c, err.Error())
			return
		}
		if err.Error() == "group not found" {
			NotFound(c, "group not found")
			return
		}
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, gin.H{"message": "group deleted successfully"})
}

// GetGroupMembers 获取群组成员列表
func (h *Handler) GetGroupMembers(c *gin.Context) {
	groupID := c.Param("id")
	if groupID == "" {
		InvalidParams(c, "group id is required")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	members, err := h.svc.GetGroupMembers(c.Request.Context(), groupID, limit, offset)
	if err != nil {
		if err.Error() == "group not found" {
			NotFound(c, "group not found")
			return
		}
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, gin.H{
		"members": members,
		"count":   len(members),
		"limit":   limit,
		"offset":  offset,
	})
}

// InviteMember 邀请成员
func (h *Handler) InviteMember(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	groupID := c.Param("id")
	if groupID == "" {
		InvalidParams(c, "group id is required")
		return
	}

	var req service.InviteMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, err.Error())
		return
	}

	invitation, err := h.svc.InviteMember(c.Request.Context(), userID, groupID, req)
	if err != nil {
		if err.Error() == "permission denied: you cannot invite members" {
			Forbidden(c, err.Error())
			return
		}
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, invitation)
}

// GetPendingInvitations 获取用户待处理的邀请
func (h *Handler) GetPendingInvitations(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	invitations, err := h.svc.GetPendingInvitations(c.Request.Context(), userID, limit, offset)
	if err != nil {
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, gin.H{
		"invitations": invitations,
		"count":       len(invitations),
	})
}

// AcceptInvitation 接受邀请
func (h *Handler) AcceptInvitation(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	invitationID := c.Param("id")
	if invitationID == "" {
		InvalidParams(c, "invitation id is required")
		return
	}

	err := h.svc.AcceptInvitation(c.Request.Context(), userID, invitationID)
	if err != nil {
		if err.Error() == "this invitation is not for you" {
			Forbidden(c, err.Error())
			return
		}
		if err.Error() == "invitation not found" {
			NotFound(c, "invitation not found")
			return
		}
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, gin.H{"message": "invitation accepted successfully"})
}

// RejectInvitation 拒绝邀请
func (h *Handler) RejectInvitation(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	invitationID := c.Param("id")
	if invitationID == "" {
		InvalidParams(c, "invitation id is required")
		return
	}

	err := h.svc.RejectInvitation(c.Request.Context(), userID, invitationID)
	if err != nil {
		if err.Error() == "this invitation is not for you" {
			Forbidden(c, err.Error())
			return
		}
		if err.Error() == "invitation not found" {
			NotFound(c, "invitation not found")
			return
		}
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, gin.H{"message": "invitation rejected successfully"})
}

// RemoveMember 移除成员
func (h *Handler) RemoveMember(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	groupID := c.Param("id")
	memberID := c.Param("userId")
	if groupID == "" || memberID == "" {
		InvalidParams(c, "group id and user id are required")
		return
	}

	err := h.svc.RemoveMember(c.Request.Context(), userID, groupID, memberID)
	if err != nil {
		if err.Error() == "permission denied: you cannot remove members" {
			Forbidden(c, err.Error())
			return
		}
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, gin.H{"message": "member removed successfully"})
}

// UpdateMemberRole 更新成员角色
func (h *Handler) UpdateMemberRole(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	groupID := c.Param("id")
	memberID := c.Param("userId")
	if groupID == "" || memberID == "" {
		InvalidParams(c, "group id and user id are required")
		return
	}

	var req service.UpdateMemberRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, err.Error())
		return
	}

	err := h.svc.UpdateMemberRole(c.Request.Context(), userID, groupID, memberID, req)
	if err != nil {
		if err.Error() == "permission denied: you cannot manage roles" {
			Forbidden(c, err.Error())
			return
		}
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, gin.H{"message": "member role updated successfully"})
}

// LeaveGroup 离开群组
func (h *Handler) LeaveGroup(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	groupID := c.Param("id")
	if groupID == "" {
		InvalidParams(c, "group id is required")
		return
	}

	err := h.svc.LeaveGroup(c.Request.Context(), userID, groupID)
	if err != nil {
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, gin.H{"message": "left group successfully"})
}

// ========== Group Roles Management ==========

// InitializeGroupRoles 初始化系统内置角色
// POST /api/groups/roles/initialize
func (h *Handler) InitializeGroupRoles(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	// TODO: 可以添加管理员权限检查
	err := h.svc.InitializeGroupRoles(c.Request.Context())
	if err != nil {
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, gin.H{"message": "group roles initialized successfully"})
}

// ListGroupRoles 获取所有群组角色列表
// GET /api/groups/roles
func (h *Handler) ListGroupRoles(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	roles, err := h.svc.ListGroupRoles(c.Request.Context())
	if err != nil {
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, gin.H{
		"roles": roles,
		"count": len(roles),
	})
}

// GetGroupRoleByCode 根据代码获取群组角色
// GET /api/groups/roles/:code
func (h *Handler) GetGroupRoleByCode(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	code := c.Param("code")
	if code == "" {
		InvalidParams(c, "role code is required")
		return
	}

	role, err := h.svc.GetGroupRoleByCode(c.Request.Context(), code)
	if err != nil {
		if err.Error() == "role not found" {
			NotFound(c, "role not found")
			return
		}
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, role)
}

// GetRolePermissions 获取角色权限
// GET /api/groups/roles/:code/permissions
func (h *Handler) GetRolePermissions(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	code := c.Param("code")
	if code == "" {
		InvalidParams(c, "role code is required")
		return
	}

	permissions, err := h.svc.GetGroupRolePermissions(c.Request.Context(), code)
	if err != nil {
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, permissions)
}

// UpdateMemberRoleByCode 根据角色代码更新成员角色
// POST /api/groups/:id/members/:userId/role-by-code
func (h *Handler) UpdateMemberRoleByCode(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	groupID := c.Param("id")
	memberID := c.Param("userId")
	if groupID == "" || memberID == "" {
		InvalidParams(c, "group id and user id are required")
		return
	}

	var req struct {
		RoleCode string `json:"roleCode" binding:"required,oneof=creator admin member outsider"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, err.Error())
		return
	}

	err := h.svc.UpdateMemberRoleByCode(c.Request.Context(), userID, groupID, memberID, req.RoleCode)
	if err != nil {
		if err.Error() == "you are not a member of this group" {
			Forbidden(c, err.Error())
			return
		}
		if err.Error() == "you do not have permission to manage roles" {
			Forbidden(c, err.Error())
			return
		}
		if err.Error() == "member not found" {
			NotFound(c, err.Error())
			return
		}
		if err.Error() == "cannot change owner role" {
			Forbidden(c, err.Error())
			return
		}
		if err.Error() == "invalid role code" {
			InvalidParams(c, err.Error())
			return
		}
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, gin.H{"message": "member role updated successfully"})
}

// FollowGroup 关注群组
// POST /api/groups/:id/follow
func (h *Handler) FollowGroup(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	groupID := c.Param("id")
	if groupID == "" {
		InvalidParams(c, "group id is required")
		return
	}

	group, err := h.svc.FollowGroup(c.Request.Context(), userID, groupID)
	if err != nil {
		if err.Error() == "group not found" {
			NotFound(c, "group not found")
			return
		}
		if err.Error() == "already following this group" {
			Error(c, CodeError, err.Error())
			return
		}
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, group)
}

// UnfollowGroup 取消关注群组
// DELETE /api/groups/:id/follow
func (h *Handler) UnfollowGroup(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	groupID := c.Param("id")
	if groupID == "" {
		InvalidParams(c, "group id is required")
		return
	}

	err := h.svc.UnfollowGroup(c.Request.Context(), userID, groupID)
	if err != nil {
		if err.Error() == "group not found" {
			NotFound(c, "group not found")
			return
		}
		if err.Error() == "not following this group" {
			Error(c, CodeError, err.Error())
			return
		}
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, gin.H{"message": "unfollowed group successfully"})
}

// ListFollowedGroups 获取用户关注的群组列表
// GET /api/groups/followed
func (h *Handler) ListFollowedGroups(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	groups, err := h.svc.ListFollowedGroups(c.Request.Context(), userID, limit, offset)
	if err != nil {
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, groups)
}
