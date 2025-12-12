package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

// Story asset creation sources.
const (
	AssetSourceManual = "manual"
	AssetSourceUpload = "upload"
	AssetSourceAI     = "ai"
)

type CreateStorySceneRequest struct {
	StoryID      string `json:"storyId" binding:"required"`
	Title        string `json:"title" binding:"required,min=1,max=200"`
	Description  string `json:"description" binding:"max=4000"`
	Image        string `json:"image" binding:"omitempty,url"`
	Location     string `json:"location" binding:"max=100"`
	TimeOfDay    string `json:"timeOfDay" binding:"max=50"`
	SourceType   string `json:"sourceType" binding:"required,oneof=manual upload ai"`
	SourcePrompt string `json:"sourcePrompt"`
	SourceImage  string `json:"sourceImage" binding:"omitempty,url"`
	IsPublic     bool   `json:"isPublic"`
}

type UpdateStorySceneRequest struct {
	Title        *string `json:"title" binding:"omitempty,min=1,max=200"`
	Description  *string `json:"description" binding:"omitempty,max=4000"`
	Image        *string `json:"image" binding:"omitempty,url"`
	Location     *string `json:"location" binding:"omitempty,max=100"`
	TimeOfDay    *string `json:"timeOfDay" binding:"omitempty,max=50"`
	SourceType   *string `json:"sourceType" binding:"omitempty,oneof=manual upload ai"`
	SourcePrompt *string `json:"sourcePrompt"`
	SourceImage  *string `json:"sourceImage" binding:"omitempty,url"`
	IsPublic     *bool   `json:"isPublic"`
}

// CreateStoryScene creates a story scoped scene asset.
func (s *Service) CreateStoryScene(ctx context.Context, userID string, req CreateStorySceneRequest) (*domain.StoryScene, error) {
	if err := s.ensureStoryOwnership(ctx, req.StoryID, userID); err != nil {
		return nil, err
	}

	sourceType := strings.ToLower(req.SourceType)
	if sourceType == "" {
		sourceType = AssetSourceManual
	}

	scene := &domain.StoryScene{
		StoryID:      req.StoryID,
		Title:        req.Title,
		Description:  req.Description,
		Image:        req.Image,
		Location:     req.Location,
		TimeOfDay:    req.TimeOfDay,
		SourceType:   sourceType,
		SourcePrompt: req.SourcePrompt,
		SourceImage:  req.SourceImage,
		CreatedBy:    userID,
		LastEditedBy: userID,
		IsPublic:     req.IsPublic,
		CreatedAt:    time.Now().Unix(),
		UpdatedAt:    time.Now().Unix(),
	}

	if err := s.repo.CreateStoryScene(ctx, scene); err != nil {
		s.logger.Error("create story scene failed", zap.Error(err), zap.String("storyId", req.StoryID))
		return nil, fmt.Errorf("failed to create story scene")
	}
	return scene, nil
}

// UpdateStoryScene updates a story scene asset.
func (s *Service) UpdateStoryScene(ctx context.Context, userID, storyID, sceneID string, req UpdateStorySceneRequest) (*domain.StoryScene, error) {
	scene, err := s.repo.StorySceneByID(ctx, storyID, sceneID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureStoryOwnership(ctx, storyID, userID); err != nil {
		return nil, err
	}

	if req.Title != nil {
		scene.Title = *req.Title
	}
	if req.Description != nil {
		scene.Description = *req.Description
	}
	if req.Image != nil {
		scene.Image = *req.Image
	}
	if req.Location != nil {
		scene.Location = *req.Location
	}
	if req.TimeOfDay != nil {
		scene.TimeOfDay = *req.TimeOfDay
	}
	if req.SourceType != nil {
		scene.SourceType = strings.ToLower(*req.SourceType)
	}
	if req.SourcePrompt != nil {
		scene.SourcePrompt = *req.SourcePrompt
	}
	if req.SourceImage != nil {
		scene.SourceImage = *req.SourceImage
	}
	if req.IsPublic != nil {
		scene.IsPublic = *req.IsPublic
	}

	scene.LastEditedBy = userID
	scene.UpdatedAt = time.Now().Unix()

	if err := s.repo.UpdateStoryScene(ctx, scene); err != nil {
		s.logger.Error("update story scene failed",
			zap.Error(err),
			zap.String("sceneId", sceneID),
			zap.String("storyId", storyID))
		return nil, fmt.Errorf("failed to update story scene")
	}
	return scene, nil
}

// ListStoryScenes returns paginated story scenes.
func (s *Service) ListStoryScenes(ctx context.Context, storyID string, limit, offset int) ([]*domain.StoryScene, error) {
	return s.repo.StoryScenes(ctx, storyID, limit, offset)
}

// DeleteStoryScene removes a story scene asset.
func (s *Service) DeleteStoryScene(ctx context.Context, userID, storyID, sceneID string) error {
	if err := s.ensureStoryOwnership(ctx, storyID, userID); err != nil {
		return err
	}
	return s.repo.DeleteStoryScene(ctx, storyID, sceneID)
}

// ensureStoryOwnership validates that the user owns the story.
func (s *Service) ensureStoryOwnership(ctx context.Context, storyID, userID string) error {
	story, err := s.repo.StoryByID(ctx, storyID)
	if err != nil {
		s.logger.Error("failed to get story", zap.Error(err), zap.String("storyID", storyID))
		return fmt.Errorf("story not found")
	}

	// Check if user is the story author
	if story.AuthorID == userID {
		s.logger.Info("story ownership checked: author", zap.String("storyID", storyID), zap.String("userID", userID))
		return nil
	}

	// Check if user is a group member (for group stories)
	if story.GroupID != "" {
		isMember, err := s.repo.IsGroupMember(ctx, story.GroupID, userID)
		if err == nil && isMember {
			s.logger.Info("story ownership checked: group member", zap.String("storyID", storyID), zap.String("userID", userID))
			return nil
		}
	}

	s.logger.Warn("permission denied for story",
		zap.String("storyID", storyID),
		zap.String("userID", userID),
		zap.String("storyAuthorID", story.AuthorID),
		zap.String("groupID", story.GroupID))
	return fmt.Errorf("permission denied: insufficient rights")
}
