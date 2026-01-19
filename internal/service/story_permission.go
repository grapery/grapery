package service

import (
	"context"
	"errors"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

// CanEditStory checks if a user can edit a story
// Returns:
//   - canEdit: true if the user has permission to edit the story
//   - error: nil on success, or an error if verification fails
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

	// Case 2: Open collaboration allows all users to edit
	if story.IsCollaborationOpen {
		s.logger.Debug("story has open collaboration, user can edit",
			zap.String("userID", userID),
			zap.String("storyID", storyID),
			zap.Bool("isCollaborationOpen", true))
		return true, nil
	}

	// Case 3: Restricted collaboration - only author and group members can edit
	if !story.IsCollaborationOpen {
		s.logger.Debug("story has restricted collaboration, checking group membership",
			zap.String("userID", userID),
			zap.String("storyID", storyID),
			zap.Bool("isCollaborationOpen", false))

		// Check if user is a group member
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
	}

	// Default: No permission
	s.logger.Warn("user does not have permission to edit story",
		zap.String("userID", userID),
		zap.String("storyID", storyID))
	return false, nil
}
