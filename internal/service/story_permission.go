package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

const (
	ForkPermissionAllowed             = "allowed"
	ForkPermissionCollaborationClosed = "collaboration_closed"
	ForkPermissionSourceNotPublished  = "source_not_published"
	ForkPermissionSourceNotVisible    = "source_not_visible"
)

// StoryboardForkPermission is the server-authoritative result used by both
// preflight UI and the actual fork/continue write paths.
type StoryboardForkPermission struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason,omitempty"`
}

// CanViewStory 检查用户是否可以查看故事
// 可见性规则：
// 1. 已发布的故事 (status=published)：任何人都可以查看
// 2. 草稿状态的故事 (status=draft)：只有作者和受邀贡献者可以查看
// 3. 作者和受邀贡献者始终可以查看协作内容
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
	if story.UserID == userID {
		return true, nil
	}
	if userID != "" {
		isContributor, contributorErr := s.repo.IsStoryContributor(ctx, story.ID, userID)
		if contributorErr == nil && isContributor {
			return true, nil
		}
	}

	// Case 2: Check status
	switch story.Status {
	case "published":
		// 已发布的故事：再按可见性（public/followers/private）裁决。
		if s.CanViewerSeeStory(ctx, userID, story) {
			s.logger.Debug("story is published and visible to viewer",
				zap.String("storyID", storyID))
			return true, nil
		}
		s.logger.Debug("story is published but not visible to viewer",
			zap.String("storyID", storyID),
			zap.String("userID", userID),
			zap.String("visibility", story.Visibility))
		return false, nil

	case "draft", "rendering":
		// 草稿/渲染中的故事：只有作者可以查看（V1/V2 MVP）
		s.logger.Warn("draft story view denied - author only",
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

// CanEditStory checks Story-level management permission. Open collaboration
// grants content contribution, never permission to change Story metadata.
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

	if story.UserID == userID || (story.Author != nil && story.Author.ID == userID) {
		s.logger.Debug("user is story author, can edit",
			zap.String("userID", userID),
			zap.String("storyID", storyID))
		return true, nil
	}

	s.logger.Warn("story metadata is author-managed",
		zap.String("userID", userID),
		zap.String("storyID", storyID))
	return false, nil
}

// CanCreateStoryboard checks permission to create a root storyboard. In the
// current product contract only the Story owner may start a new root; other
// users contribute by continuing an existing node.
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

	canCreate := isStoryOwner(story, userID)

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

// CanViewUnpublishedStoryboards reports whether the viewer belongs to the
// Story's authoring boundary. Only the Story author and explicitly invited
// contributors may discover draft/unpublished nodes in hierarchy responses.
func (s *Service) CanViewUnpublishedStoryboards(ctx context.Context, story *domain.Story, userID string) bool {
	if story == nil || userID == "" {
		return false
	}
	if isStoryOwner(story, userID) {
		return true
	}
	isContributor, err := s.repo.IsStoryContributor(ctx, story.ID, userID)
	return err == nil && isContributor
}

// CanForkStoryboard checks permission against the parent node and its Story.
// Fork remains inside the original Story; it never creates an independent Story.
func (s *Service) CanForkStoryboard(ctx context.Context, parentID, userID string) (*StoryboardForkPermission, error) {
	parent, err := s.repo.StoryboardByID(ctx, parentID)
	if err != nil || parent == nil {
		return nil, domain.ErrNotFound
	}

	story, err := s.repo.StoryByID(ctx, parent.StoryID)
	if err != nil || story == nil {
		return nil, domain.ErrNotFound
	}

	if isStoryOwner(story, userID) {
		return &StoryboardForkPermission{Allowed: true, Reason: ForkPermissionAllowed}, nil
	}

	// Explicit collaborators are part of the authoring boundary, not the public
	// audience boundary. They may continue private/draft Stories and draft nodes.
	isContributor, err := s.repo.IsStoryContributor(ctx, story.ID, userID)
	if err != nil {
		s.logger.Warn("failed to check explicit story contributor",
			zap.String("storyId", story.ID),
			zap.String("userId", userID),
			zap.Error(err))
		return &StoryboardForkPermission{Allowed: false, Reason: ForkPermissionCollaborationClosed}, nil
	}
	if isContributor {
		return &StoryboardForkPermission{Allowed: true, Reason: ForkPermissionAllowed}, nil
	}

	if story.Status != "published" {
		return &StoryboardForkPermission{Allowed: false, Reason: ForkPermissionSourceNotPublished}, nil
	}
	if !s.CanViewerSeeStory(ctx, userID, story) {
		return &StoryboardForkPermission{Allowed: false, Reason: ForkPermissionSourceNotVisible}, nil
	}
	if !story.IsCollaborationOpen {
		return &StoryboardForkPermission{Allowed: false, Reason: ForkPermissionCollaborationClosed}, nil
	}

	if parent.WorkflowStatus != domain.WorkflowStatusPublished {
		return &StoryboardForkPermission{Allowed: false, Reason: ForkPermissionSourceNotPublished}, nil
	}

	return &StoryboardForkPermission{Allowed: true, Reason: ForkPermissionAllowed}, nil
}

func isStoryOwner(story *domain.Story, userID string) bool {
	return story != nil && userID != "" &&
		(story.UserID == userID || (story.Author != nil && story.Author.ID == userID))
}

func (s *Service) enforceCanForkStoryboard(ctx context.Context, parentID, userID string) error {
	permission, err := s.CanForkStoryboard(ctx, parentID, userID)
	if err != nil {
		return err
	}
	if !permission.Allowed {
		return fmt.Errorf("%w: %s", domain.ErrForbidden, permission.Reason)
	}
	return nil
}
