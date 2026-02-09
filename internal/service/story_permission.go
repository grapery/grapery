package service

import (
	"context"
	"errors"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

// CanViewStory 检查用户是否可以查看故事
// 可见性规则：
// 1. 已发布的故事 (status=published)：任何人都可以查看
// 2. 草稿状态的故事 (status=draft)：只有作者和小组成员可以查看
// 3. 作者永远可以查看自己的故事
func (s *Service) CanViewStory(ctx context.Context, storyID, userID string) (bool, error) {
	s.logger.Debug("checking story view permission",
		zap.String("storyID", storyID),
		zap.String("userID", userID))

	// Get the story
	story, err := s.repo.StoryByID(ctx, storyID)
	if err != nil {
		if err == domain.ErrNotFound {
			s.logger.Warn("story not found for view permission check",
				zap.String("storyID", storyID))
			return false, errors.New("story not found")
		}
		s.logger.Error("failed to get story for view permission check",
			zap.String("storyID", storyID),
			zap.Error(err))
		return false, errors.New("failed to get story")
	}

	// Case 1: Story author can always view
	if story.Author != nil && story.Author.ID == userID {
		return true, nil
	}

	// Case 2: Check status
	switch story.Status {
	case "published":
		// 已发布的故事：任何人可以查看
		s.logger.Debug("story is published, allowing view",
			zap.String("storyID", storyID))
		return true, nil

	case "draft", "rendering":
		// 草稿/渲染中的故事：需要检查权限
		// 如果故事属于小组，小组成员可以查看
		if story.GroupID != "" && userID != "" {
			isGroupMember, err := s.repo.IsGroupMember(ctx, story.GroupID, userID)
			if err != nil {
				s.logger.Error("failed to check group membership for view",
					zap.String("userID", userID),
					zap.String("groupID", story.GroupID),
					zap.Error(err))
				return false, errors.New("failed to verify permission")
			}
			if isGroupMember {
				s.logger.Debug("user is group member, allowing view of draft story",
					zap.String("userID", userID),
					zap.String("storyID", storyID))
				return true, nil
			}
		}
		s.logger.Warn("draft story view denied",
			zap.String("storyID", storyID),
			zap.String("userID", userID))
		return false, nil

	default:
		// 默认情况下，只允许作者查看
		s.logger.Warn("unknown status, denying view",
			zap.String("storyID", storyID),
			zap.String("status", story.Status))
		return false, nil
	}
}

// CanEditStory 检查用户是否可以编辑故事
// 编辑权限规则：
// 1. 作者永远可以编辑
// 2. 开放协作(IsCollaborationOpen=true)：任何人可以编辑
// 3. 受限协作(IsCollaborationOpen=false + GroupID!="")：只有小组成员可以编辑
// 4. 封闭协作(IsCollaborationOpen=false + GroupID=="")：只有作者可以编辑
func (s *Service) CanEditStory(ctx context.Context, userID, storyID string) (bool, error) {
	s.logger.Info("checking story edit permission",
		zap.String("userID", userID),
		zap.String("storyID", storyID),
	)

	// Get the story
	story, err := s.repo.StoryByID(ctx, storyID)
	if err != nil {
		if err == domain.ErrNotFound {
			s.logger.Warn("story not found for permission check",
				zap.String("storyID", storyID),
				zap.String("userID", userID))
			return false, errors.New("story not found")
		}
		s.logger.Error("failed to get story for permission check",
			zap.String("storyID", storyID),
			zap.String("userID", userID),
			zap.Error(err))
		return false, errors.New("failed to get story")
	}

	// Case 1: Story author can always edit
	if story.Author != nil && story.Author.ID == userID {
		s.logger.Debug("user is story author, can edit",
			zap.String("userID", userID),
			zap.String("storyID", storyID))
		return true, nil
	}

	// 使用 domain 层的权限检查逻辑
	status := story.GetCollaborationStatus()

	switch status {
	case domain.CollaborationStatusOpen:
		// 开放协作：任何人可以编辑
		s.logger.Debug("story has open collaboration, user can edit",
			zap.String("userID", userID),
			zap.String("storyID", storyID))
		return true, nil

	case domain.CollaborationStatusRestricted:
		// 受限协作：只有小组成员可以编辑
		if story.GroupID != "" {
			isGroupMember, err := s.repo.IsGroupMember(ctx, story.GroupID, userID)
			if err != nil {
				s.logger.Error("failed to check group membership",
					zap.String("userID", userID),
					zap.String("groupID", story.GroupID),
					zap.Error(err))
				return false, errors.New("failed to verify permission")
			}
			if isGroupMember {
				s.logger.Debug("user is group member, can edit",
					zap.String("userID", userID),
					zap.String("groupID", story.GroupID))
				return true, nil
			}
		}
		s.logger.Warn("user is not group member, cannot edit restricted story",
			zap.String("userID", userID),
			zap.String("storyID", storyID))
		return false, nil

	case domain.CollaborationStatusClosed:
		// 封闭创作：只有作者可以编辑
		s.logger.Warn("story is closed collaboration, only author can edit",
			zap.String("userID", userID),
			zap.String("storyID", storyID))
		return false, nil
	}

	// Default: No permission
	s.logger.Warn("user does not have permission to edit story",
		zap.String("userID", userID),
		zap.String("storyID", storyID))
	return false, nil
}

// CanCreateStoryboard 检查用户是否可以为故事创建故事板
// 使用 domain 层的权限检查逻辑
func (s *Service) CanCreateStoryboard(ctx context.Context, storyID, userID string) (bool, error) {
	s.logger.Info("checking storyboard creation permission",
		zap.String("storyID", storyID),
		zap.String("userID", userID))

	// Get the story
	story, err := s.repo.StoryByID(ctx, storyID)
	if err != nil {
		if err == domain.ErrNotFound {
			return false, errors.New("story not found")
		}
		return false, errors.New("failed to get story")
	}

	// Check if user is group member (for restricted collaboration)
	isGroupMember := false
	if story.GroupID != "" {
		isGroupMember, _ = s.repo.IsGroupMember(ctx, story.GroupID, userID)
	}

	// Use domain layer logic
	canCreate := story.CanCreateStoryboard(userID, isGroupMember)

	if canCreate {
		s.logger.Debug("user can create storyboard",
			zap.String("userID", userID),
			zap.String("storyID", storyID),
			zap.String("collaborationStatus", string(story.GetCollaborationStatus())))
	} else {
		s.logger.Warn("user cannot create storyboard",
			zap.String("userID", userID),
			zap.String("storyID", storyID),
			zap.String("collaborationStatus", string(story.GetCollaborationStatus())))
	}

	return canCreate, nil
}
