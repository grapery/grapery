package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/aliyun"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	genapi "github.com/grapestree/fgrapery/grapery/internal/genai"
	"go.uber.org/zap"
)

// Story asset creation sources.
const (
	AssetSourceManual = "manual"
	AssetSourceUpload = "upload"
	AssetSourceAI     = "ai"
)

type CreateStorySceneRequest struct {
	StoryID      string   `json:"storyId" binding:"omitempty"` // Not required - set from URL parameter
	Title        string   `json:"title" binding:"required,min=1,max=200"`
	Description  string   `json:"description" binding:"max=4000"`
	Image        string   `json:"image" binding:"omitempty,url"`
	Location     string   `json:"location" binding:"max=100"`
	TimeOfDay    string   `json:"timeOfDay" binding:"max=50"`
	SourceType   string   `json:"sourceType" binding:"required,oneof=manual upload ai"`
	SourcePrompt string   `json:"sourcePrompt"`
	SourceImage  string   `json:"sourceImage" binding:"omitempty,url"`
	IsPublic     bool     `json:"isPublic"`
	Tags         []string `json:"tags" binding:"dive"`
}

type UpdateStorySceneRequest struct {
	Title        *string  `json:"title" binding:"omitempty,min=1,max=200"`
	Description  *string  `json:"description" binding:"omitempty,max=4000"`
	Image        *string  `json:"image" binding:"omitempty,url"`
	Location     *string  `json:"location" binding:"omitempty,max=100"`
	TimeOfDay    *string  `json:"timeOfDay" binding:"omitempty,max=50"`
	SourceType   *string  `json:"sourceType" binding:"omitempty,oneof=manual upload ai"`
	SourcePrompt *string  `json:"sourcePrompt"`
	SourceImage  *string  `json:"sourceImage" binding:"omitempty,url"`
	IsPublic     *bool    `json:"isPublic"`
	Tags         []string `json:"tags" binding:"dive"`
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
		Tags:         req.Tags,
	}

	if err := s.repo.CreateStoryScene(ctx, scene); err != nil {
		s.logger.Error("create story scene failed", zap.Error(err), zap.String("storyId", req.StoryID))
		return nil, fmt.Errorf("failed to create story scene")
	}
	s.invalidateStoryCache(ctx, req.StoryID)
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
	s.invalidateStoryCache(ctx, storyID)
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
	if err := s.repo.DeleteStoryScene(ctx, storyID, sceneID); err != nil {
		return err
	}
	s.invalidateStoryCache(ctx, storyID)
	return nil
}

// ensureStoryOwnership validates that the user owns the story.
func (s *Service) ensureStoryOwnership(ctx context.Context, storyID, userID string) error {
	story, err := s.repo.StoryByID(ctx, storyID)
	if err != nil {
		s.logger.Error("failed to get story", zap.Error(err), zap.String("storyID", storyID))
		return fmt.Errorf("story not found")
	}

	// Check if user is the story author
	if story.UserID == userID {
		s.logger.Info("story ownership checked: author", zap.String("storyID", storyID), zap.String("userID", userID))
		return nil
	}

	s.logger.Warn("permission denied for story",
		zap.String("storyID", storyID),
		zap.String("userID", userID),
		zap.String("storyAuthorID", story.UserID))
	return fmt.Errorf("permission denied: insufficient rights")
}

// EnsureStoryOwnership validates that the user owns the story (public method for handlers).
func (s *Service) EnsureStoryOwnership(ctx context.Context, storyID, userID string) error {
	return s.ensureStoryOwnership(ctx, storyID, userID)
}

// UploadSceneImageRequest 请求上传场景图片
type UploadSceneImageRequest struct {
	StoryID   string `json:"storyId" binding:"required"`
	ImageData []byte `json:"-"` // Binary data, not from JSON
	Filename  string `json:"filename" binding:"required"`
}

// UploadSceneImageResponse 上传场景图片响应
type UploadSceneImageResponse struct {
	Success  bool   `json:"success"`
	URL      string `json:"url"`
	Filename string `json:"filename"`
}

// GenerateSceneImageRequest AI生成场景图片请求
type GenerateSceneImageRequest struct {
	StoryID      string  `json:"storyId" binding:"required"`
	SceneID      string  `json:"sceneId"`
	CustomPrompt *string `json:"prompt"`
}

// GenerateSceneImageResponse AI生成场景图片响应
type GenerateSceneImageResponse struct {
	Success bool   `json:"success"`
	URL     string `json:"url"`
	Prompt  string `json:"prompt"`
}

// UploadSceneImage 已废弃：图片上传已改为客户端直接上传到OSS
// 此方法保留仅为兼容性，实际不再使用
func (s *Service) UploadSceneImage(ctx context.Context, storyID, userID string, imageData []byte, filename string) (string, string, error) {
	return "", "", fmt.Errorf("image upload should be done directly from client to OSS")
}

// GenerateStorySceneImage AI生成场景图片
func (s *Service) GenerateStorySceneImage(ctx context.Context, storyID, sceneID, userID string, customPrompt string) (string, string, error) {
	s.logger.Info("generating scene image with AI",
		zap.String("storyId", storyID),
		zap.String("sceneId", sceneID),
		zap.String("userId", userID),
		zap.String("customPrompt", customPrompt))

	// Verify story ownership
	if err := s.ensureStoryOwnership(ctx, storyID, userID); err != nil {
		return "", "", err
	}

	// Get scene details if sceneID is provided
	var scene *domain.StoryScene
	if sceneID != "" {
		sceneReq, err := s.repo.StorySceneByID(ctx, storyID, sceneID)
		if err != nil {
			s.logger.Error("failed to get scene", zap.Error(err))
			return "", "", fmt.Errorf("scene not found")
		}
		scene = sceneReq
	}

	// Get story details
	story, err := s.repo.StoryByID(ctx, storyID)
	if err != nil {
		s.logger.Error("failed to get story", zap.Error(err))
		return "", "", fmt.Errorf("story not found")
	}

	// Build AI prompt
	var promptParts []string
	if scene != nil {
		promptParts = []string{"Scene: " + scene.Title}
		if scene.Description != "" {
			promptParts = append(promptParts, "Description: "+scene.Description)
		}
		if scene.Location != "" {
			promptParts = append(promptParts, "Location: "+scene.Location)
		}
		if scene.TimeOfDay != "" {
			promptParts = append(promptParts, "Time: "+scene.TimeOfDay)
		}
	} else {
		// If no scene, use story title as base
		promptParts = []string{"Scene: " + story.Title}
	}

	// Add story context
	if story.Description != "" {
		promptParts = append(promptParts, "Story Context: "+story.Description)
	}

	finalPrompt := ""
	if customPrompt != "" {
		finalPrompt = customPrompt
	} else {
		finalPrompt = strings.Join(promptParts, ". ")
	}

	s.logger.Info("AI prompt generated", zap.String("prompt", finalPrompt))

	// Check if AI service is available
	if s.genAPI == nil {
		return "", "", fmt.Errorf("AI image generation service not available")
	}

	// Generate image using genAPI
	imageProvider := s.imageProvider
	if imageProvider == "" {
		imageProvider = "huoshan"
	}

	genReq := &genapi.GenerateRequest{
		Prompt:      finalPrompt,
		AspectRatio: "16:9",
		Quality:     "high",
		OutputCount: 1,
		Operation:   genapi.OperationTextToImage,
	}

	s.logger.Info("calling AI image generation service",
		zap.String("provider", imageProvider),
		zap.String("prompt", finalPrompt))

	if strings.EqualFold(imageProvider, "huoshan") {
		PrepareHuoshanGenAPIImageRequest(genReq)
	}

	resp, err := s.genAPI.GenerateImage(ctx, imageProvider, genReq)
	if err != nil {
		s.logger.Error("AI image generation failed",
			zap.String("storyId", storyID),
			zap.String("sceneId", sceneID),
			zap.String("provider", imageProvider),
			zap.Error(err))
		return "", "", fmt.Errorf("failed to generate image: %w", err)
	}

	if resp == nil || len(resp.ImageURLs) == 0 {
		return "", "", fmt.Errorf("AI image generation returned no images")
	}

	// Upload generated image to OSS
	originalImageURL := resp.ImageURLs[0]
	ossClient := aliyun.GetGlobalClient()
	if ossClient == nil {
		s.logger.Warn("OSS client not available, returning original URL",
			zap.String("originalURL", originalImageURL))
		// Return original URL if OSS is not available
		generatedFilename := fmt.Sprintf("scene-ai-generated-%s.jpg", uuid.New().String())
		return originalImageURL, generatedFilename, nil
	}

	// Generate OSS object key
	objectKey := fmt.Sprintf("stories/%s/scenes/ai-generated-%s.jpg", storyID, uuid.New().String())

	// Upload to OSS
	ossURL, err := ossClient.UploadFileFromURL(objectKey, originalImageURL)
	if err != nil {
		s.logger.Warn("failed to upload image to OSS, using original URL",
			zap.String("sceneId", sceneID),
			zap.String("originalURL", originalImageURL),
			zap.Error(err))
		// Upload failed, return original URL
		generatedFilename := fmt.Sprintf("scene-ai-generated-%s.jpg", uuid.New().String())
		return originalImageURL, generatedFilename, nil
	}

	// Clean URL: remove query parameters, ensure HTTPS
	ossURL = strings.Split(ossURL, "?")[0]
	ossURL = strings.ReplaceAll(ossURL, "http://", "https://")

	generatedFilename := fmt.Sprintf("scene-ai-generated-%s.jpg", uuid.New().String())

	s.logger.Info("scene image generated and uploaded successfully",
		zap.String("storyId", storyID),
		zap.String("sceneId", sceneID),
		zap.String("url", ossURL))

	return ossURL, generatedFilename, nil
}
