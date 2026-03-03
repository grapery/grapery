package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/cache"
	"github.com/grapestree/fgrapery/grapery/internal/common"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

// CreateStoryboard 创建新的 storyboard
func (s *Service) CreateStoryboard(ctx context.Context, storyboard *domain.Storyboard) error {
	s.logger.Info("creating storyboard",
		zap.String("storyId", storyboard.StoryID),
		zap.String("creatorId", storyboard.UserID),
		zap.String("title", storyboard.Title),
		zap.String("parentId", storyboard.ParentID),
		zap.Bool("isStandalone", storyboard.IsStandalone),
		zap.Int("sceneCount", storyboard.SceneCount))

	// 验证故事存在
	story, err := s.repo.StoryByID(ctx, storyboard.StoryID)
	if err != nil {
		s.logger.Error("story not found for storyboard creation",
			zap.String("storyId", storyboard.StoryID),
			zap.Error(err))
		return fmt.Errorf("story not found: %w", err)
	}

	// Validate parent ID and isStandalone consistency
	if !storyboard.IsStandalone && storyboard.ParentID == "" {
		s.logger.Error("isStandalone is false but parentId is empty",
			zap.String("storyboardId", storyboard.ID),
			zap.String("storyId", storyboard.StoryID),
			zap.Bool("isStandalone", storyboard.IsStandalone))
		return fmt.Errorf("parent storyboard ID is required when isStandalone is false")
	}

	// 如果没有父节点或父节点为空，设置为 root marker
	if storyboard.ParentID == "" {
		storyboard.ParentID = domain.StoryboardRootMarker
		s.logger.Info("creating root storyboard (no parent)",
			zap.String("storyboardId", storyboard.ID),
			zap.String("storyId", storyboard.StoryID))
	} else {
		s.logger.Info("creating child storyboard with parent",
			zap.String("storyboardId", storyboard.ID),
			zap.String("storyId", storyboard.StoryID),
			zap.String("parentId", storyboard.ParentID))
	}

	// 如果有父节点（不是 root），验证父节点存在
	if storyboard.ParentID != domain.StoryboardRootMarker {
		parent, err := s.repo.StoryboardByID(ctx, storyboard.ParentID)
		if err != nil {
			s.logger.Error("parent storyboard not found",
				zap.String("parentId", storyboard.ParentID),
				zap.String("storyId", storyboard.StoryID),
				zap.Error(err))
			return fmt.Errorf("parent storyboard not found: %w", err)
		}
		// 确保父节点属于同一个故事
		if parent.StoryID != storyboard.StoryID {
			s.logger.Error("parent storyboard belongs to different story",
				zap.String("parentId", storyboard.ParentID),
				zap.String("parentStoryId", parent.StoryID),
				zap.String("storyboardStoryId", storyboard.StoryID))
			return fmt.Errorf("parent storyboard belongs to different story")
		}
		s.logger.Debug("parent storyboard validated",
			zap.String("parentId", storyboard.ParentID),
			zap.String("parentTitle", parent.Title))
	}

	// 使用 AI 生成内容（如果提供了 geminiClient）
	if s.geminiClient != nil && storyboard.RawInput != "" {
		s.logger.Info("starting AI generation for storyboard",
			zap.String("storyboardId", storyboard.ID),
			zap.String("rawInput", truncateForLog(storyboard.RawInput, 200)))
		if err := s.GenerateStoryboardWithAI(ctx, storyboard); err != nil {
			s.logger.Warn("AI generation failed, creating storyboard without AI content",
				zap.String("storyboardId", storyboard.ID),
				zap.Error(err))
			// AI 生成失败不影响创建流程，继续使用原始输入
		} else {
			s.logger.Info("AI generation completed for storyboard",
				zap.String("storyboardId", storyboard.ID),
				zap.Int("scenesGenerated", len(storyboard.StoryboardScenes)))
		}
	}

	// Validate linked assets
	s.logger.Debug("validating storyboard assets",
		zap.String("storyboardId", storyboard.ID),
		zap.Int("characterRefs", len(storyboard.CharacterRefs)),
		zap.Int("sceneRefs", len(storyboard.SceneRefs)))
	if err := s.validateStoryboardAssets(ctx, storyboard); err != nil {
		s.logger.Error("storyboard asset validation failed",
			zap.String("storyboardId", storyboard.ID),
			zap.Error(err))
		return err
	}

	// 创建 storyboard（使用事务确保原子性）
	s.logger.Debug("saving storyboard to database",
		zap.String("storyboardId", storyboard.ID),
		zap.String("storyId", storyboard.StoryID))

	err = s.repo.WithTransaction(ctx, func(tx domain.Repository) error {
		// 创建 storyboard
		if err := tx.CreateStoryboard(ctx, storyboard); err != nil {
			s.logger.Error("failed to create storyboard in database",
				zap.String("storyboardId", storyboard.ID),
				zap.String("storyId", storyboard.StoryID),
				zap.Error(err))
			return fmt.Errorf("failed to create storyboard: %w", err)
		}

		// Persist AI-generated storyboard scenes (if any)
		if len(storyboard.StoryboardScenes) > 0 {
			s.logger.Debug("persisting AI-generated storyboard scenes",
				zap.String("storyboardId", storyboard.ID),
				zap.Int("sceneCount", len(storyboard.StoryboardScenes)))

			// Convert to pointer slice for repository
			scenes := make([]*domain.StoryboardScene, len(storyboard.StoryboardScenes))
			for i := range storyboard.StoryboardScenes {
				scenes[i] = &storyboard.StoryboardScenes[i]
			}

			if err := tx.CreateStoryboardScenes(ctx, storyboard.ID, scenes); err != nil {
				s.logger.Error("failed to persist storyboard scenes to database",
					zap.String("storyboardId", storyboard.ID),
					zap.Int("sceneCount", len(scenes)),
					zap.Error(err))
				return fmt.Errorf("failed to persist storyboard scenes: %w", err)
			}
			s.logger.Info("storyboard scenes persisted successfully",
				zap.String("storyboardId", storyboard.ID),
				zap.Int("sceneCount", len(scenes)))
		}

		return nil
	})

	if err != nil {
		return err
	}

	// 缓存新创建的 storyboard（在事务外执行）
	c := s.getCache()
	if c != nil {
		key := cache.StoryboardKey(storyboard.ID)
		// 不缓存 scenes（动态数据）
		scenes := storyboard.StoryboardScenes
		storyboard.StoryboardScenes = nil
		if err := c.Set(ctx, key, storyboard, entityCacheTTL); err != nil {
			s.logger.Warn("failed to cache new storyboard",
				zap.String("storyboardId", storyboard.ID),
				zap.Error(err))
		}
		storyboard.StoryboardScenes = scenes
		// 清除相关列表缓存
		for limit := 20; limit <= 100; limit += 20 {
			for offset := 0; offset < 200; offset += limit {
				_ = c.Delete(ctx, cache.StoryboardsListKey(storyboard.StoryID, limit, offset))
			}
		}
	}

	s.logger.Info("storyboard created successfully",
		zap.String("id", storyboard.ID),
		zap.String("storyId", storyboard.StoryID),
		zap.String("creatorId", storyboard.UserID),
		zap.String("title", storyboard.Title),
		zap.String("parentId", storyboard.ParentID),
		zap.String("workflowStatus", storyboard.WorkflowStatus),
		zap.Int("sceneCount", len(storyboard.StoryboardScenes)),
		zap.Bool("isAIGenerated", storyboard.IsAIGenerated))

	// Record metrics
	if s.metrics != nil {
		// Record storyboard scene count
		s.metrics.RecordStoryboardSceneCount(storyboard.ID, float64(len(storyboard.StoryboardScenes)))

		// Record storyboard token consumption
		if storyboard.TokenConsumption > 0 {
			s.metrics.RecordStoryboardTokenConsumed(storyboard.ID, float64(storyboard.TokenConsumption))
		}

		// Record child count if this is a child storyboard
		if storyboard.ParentID != "" && storyboard.ParentID != domain.StoryboardRootMarker {
			// Query actual child count for the parent
			childStoryboards, err := s.repo.StoryboardsByStory(ctx, storyboard.StoryID, 1000, 0)
			if err == nil {
				childCount := 0
				for _, sb := range childStoryboards {
					if sb.ParentID == storyboard.ParentID {
						childCount++
					}
				}
				s.metrics.RecordStoryboardChildCount(storyboard.ParentID, float64(childCount))
			}
		}
	}

	// 创建通知
	// 通知故事作者（如果创建者不是故事作者本人）
	if story.UserID != storyboard.UserID {
		// 获取创建者信息
		creator, err := s.repo.UserByID(ctx, storyboard.UserID)
		if err == nil {
			if err := s.NotifyStoryboardCreated(ctx,
				story.UserID,
				storyboard.UserID,
				creator.DisplayName,
				creator.Avatar,
				storyboard.StoryID,
				storyboard.ID); err != nil {
				s.logger.Warn("failed to send storyboard created notification",
					zap.Error(err),
					zap.String("storyboardId", storyboard.ID))
			} else {
				s.logger.Info("storyboard created notification sent",
					zap.String("recipientId", story.UserID),
					zap.String("storyboardId", storyboard.ID))
			}
		}
	}

	// 如果是 fork（不是 root），通知父节点作者
	if storyboard.ParentID != "" && storyboard.ParentID != domain.StoryboardRootMarker {
		parent, err := s.repo.StoryboardByID(ctx, storyboard.ParentID)
		if err == nil && parent.UserID != storyboard.UserID {
			// 获取创建者信息
			creator, err := s.repo.UserByID(ctx, storyboard.UserID)
			if err == nil {
				if err := s.NotifyStoryboardForked(ctx,
					parent.UserID,
					storyboard.UserID,
					creator.DisplayName,
					creator.Avatar,
					storyboard.StoryID,
					storyboard.ParentID,
					storyboard.ID); err != nil {
					s.logger.Warn("failed to send storyboard forked notification",
						zap.Error(err),
						zap.String("storyboardId", storyboard.ID))
				} else {
					s.logger.Info("storyboard forked notification sent",
						zap.String("recipientId", parent.UserID),
						zap.String("storyboardId", storyboard.ID))
				}
			}
		}
	}

	// REMOVED: RecordStoryboardCreated - not in StoryCreationAppUI design

	// 更新故事的故事板数量
	if err := s.repo.IncrementStoryStoryboardCount(ctx, storyboard.StoryID); err != nil {
		s.logger.Warn("failed to increment story storyboard count",
			zap.String("storyId", storyboard.StoryID),
			zap.Error(err))
		// 不返回错误，因为故事板已经创建成功
	} else {
		s.logger.Debug("story storyboard count incremented",
			zap.String("storyId", storyboard.StoryID))
	}

	// Update storyboard count metric (increment by 1)
	if s.metrics != nil {
		s.metrics.StoryboardCount.Inc()
	}

	return nil
}

// GetStoryboard 获取 storyboard 详情（带缓存）
func (s *Service) GetStoryboard(ctx context.Context, id string) (*domain.Storyboard, error) {
	s.logger.Debug("GetStoryboard called",
		zap.String("storyboardId", id))

	// 尝试从缓存获取（注意：storyboard 包含动态数据如 scenes，可能需要特殊处理）
	c := s.getCache()
	if c != nil {
		key := cache.StoryboardKey(id)
		var cachedStoryboard domain.Storyboard
		if err := c.Get(ctx, key, &cachedStoryboard); err == nil {
			s.logger.Debug("storyboard cache hit",
				zap.String("storyboardId", id))
			// 从缓存获取后，仍需要加载 scenes（因为 scenes 可能更新）
			storyboardScenes, err := s.repo.StoryboardScenes(ctx, id)
			if err == nil && len(storyboardScenes) > 0 {
				cachedStoryboard.StoryboardScenes = make([]domain.StoryboardScene, len(storyboardScenes))
				for i, scene := range storyboardScenes {
					cachedStoryboard.StoryboardScenes[i] = *scene
				}
			}
			// Populate missing scene images/videos
			s.populateMissingSceneImages(ctx, &cachedStoryboard)
			s.populateMissingSceneVideos(ctx, &cachedStoryboard)
			// 增加浏览量（异步）
			go func() {
				if err := s.repo.IncrementStoryboardViews(context.Background(), id); err != nil {
					s.logger.Warn("failed to increment storyboard views", zap.Error(err))
				}
			}()
			return &cachedStoryboard, nil
		} else {
			s.logger.Debug("storyboard cache miss",
				zap.String("storyboardId", id),
				zap.Error(err))
		}
	}

	storyboard, err := s.repo.StoryboardByID(ctx, id)
	if err != nil {
		s.logger.Error("GetStoryboard: StoryboardByID failed",
			zap.String("storyboardId", id),
			zap.Error(err))
		return nil, err
	}

	// Load StoryboardScenes (AI-generated plot scenes)
	storyboardScenes, err := s.repo.StoryboardScenes(ctx, id)
	if err != nil {
		s.logger.Warn("failed to load storyboard scenes", zap.Error(err), zap.String("storyboardId", id))
	} else {
		storyboard.StoryboardScenes = make([]domain.StoryboardScene, len(storyboardScenes))
		for i, scene := range storyboardScenes {
			storyboard.StoryboardScenes[i] = *scene
			s.logger.Debug("GetStoryboard: loaded scene from DB",
				zap.String("sceneId", scene.ID),
				zap.String("sceneTitle", scene.Title),
				zap.String("image", scene.Image),
				zap.String("videoUrl", scene.VideoUrl))
		}
		s.logger.Debug("GetStoryboard: loaded scenes from DB",
			zap.String("storyboardId", id),
			zap.Int("sceneCount", len(storyboardScenes)))
	}

	// Populate missing scene images from image generation records
	s.populateMissingSceneImages(ctx, storyboard)

	// Populate missing scene videos from video generation records
	s.populateMissingSceneVideos(ctx, storyboard)

	// Log final scene state
	for i, scene := range storyboard.StoryboardScenes {
		s.logger.Debug("GetStoryboard: final scene state",
			zap.Int("index", i),
			zap.String("sceneId", scene.ID),
			zap.String("title", scene.Title),
			zap.String("image", scene.Image),
			zap.String("videoUrl", scene.VideoUrl))
	}

	// 增加浏览量
	go func() {
		if err := s.repo.IncrementStoryboardViews(context.Background(), id); err != nil {
			s.logger.Warn("failed to increment storyboard views", zap.Error(err))
		}
	}()

	return storyboard, nil
}

// populateMissingSceneImages fetches completed image generations and fills in empty scene images
func (s *Service) populateMissingSceneImages(ctx context.Context, storyboard *domain.Storyboard) {
	if len(storyboard.StoryboardScenes) == 0 {
		return
	}

	// Check if any scene is missing an image
	hasMissingImages := false
	for _, scene := range storyboard.StoryboardScenes {
		if scene.Image == "" {
			hasMissingImages = true
			break
		}
	}

	if !hasMissingImages {
		return
	}

	// Fetch completed image generations for this storyboard
	imageGens, err := s.repo.ListImageGenerations(ctx, storyboard.ID)
	if err != nil {
		s.logger.Warn("failed to fetch image generations", zap.Error(err), zap.String("storyboardId", storyboard.ID))
		return
	}

	// Build a map of sceneID -> generated image URL
	sceneImageMap := make(map[string]string)
	for _, gen := range imageGens {
		if gen.Status == domain.GenerationStatusCompleted && gen.GeneratedImageURL != "" {
			sceneImageMap[gen.SceneID] = gen.GeneratedImageURL
		}
	}

	// Fill in missing scene images
	for i := range storyboard.StoryboardScenes {
		scene := &storyboard.StoryboardScenes[i]
		if scene.Image == "" {
			if imageURL, ok := sceneImageMap[scene.ID]; ok {
				scene.Image = imageURL
				s.logger.Debug("populated missing scene image from generation record",
					zap.String("sceneId", scene.ID),
					zap.String("imageURL", imageURL))
			}
		}
	}
}

// populateMissingSceneVideos fetches completed video generations and fills in empty scene videos
func (s *Service) populateMissingSceneVideos(ctx context.Context, storyboard *domain.Storyboard) {
	if len(storyboard.StoryboardScenes) == 0 {
		s.logger.Debug("populateMissingSceneVideos: no scenes to process", zap.String("storyboardId", storyboard.ID))
		return
	}

	// Check if any scene is missing a video URL
	hasMissingVideos := false
	for _, scene := range storyboard.StoryboardScenes {
		if scene.VideoUrl == "" {
			hasMissingVideos = true
			break
		}
	}

	if !hasMissingVideos {
		s.logger.Debug("populateMissingSceneVideos: all scenes have video URLs", zap.String("storyboardId", storyboard.ID))
		return
	}

	// Fetch completed video generations for this storyboard
	videoGens, err := s.repo.ListVideoGenerations(ctx, storyboard.ID)
	if err != nil {
		s.logger.Warn("failed to fetch video generations", zap.Error(err), zap.String("storyboardId", storyboard.ID))
		return
	}

	s.logger.Info("populateMissingSceneVideos: fetched video generations",
		zap.String("storyboardId", storyboard.ID),
		zap.Int("videoGenCount", len(videoGens)))

	// Build maps for sceneID -> video info including subdivision data
	type sceneVideoInfo struct {
		VideoURL            string
		IsSubdivided        bool
		VideoSegments       []domain.VideoSegmentInfo
		MiddleFrameURLs     []string
		VideoSegmentsJSON   string
		MiddleFrameURLsJSON string
	}
	sceneVideoMap := make(map[string]sceneVideoInfo)
	for _, gen := range videoGens {
		// Log each video generation record for debugging
		s.logger.Info("populateMissingSceneVideos: checking video generation",
			zap.String("sceneId", gen.SceneID),
			zap.String("status", string(gen.Status)),
			zap.String("videoURL", gen.GeneratedVideoURL),
			zap.String("errorMessage", gen.ErrorMessage),
			zap.Bool("isSubdivided", gen.IsSubdivided),
			zap.String("providerTaskID", gen.ProviderTaskID))

		if gen.Status == domain.GenerationStatusCompleted && gen.GeneratedVideoURL != "" {
			sceneVideoMap[gen.SceneID] = sceneVideoInfo{
				VideoURL:            gen.GeneratedVideoURL,
				IsSubdivided:        gen.IsSubdivided,
				VideoSegments:       gen.VideoSegments,
				MiddleFrameURLs:     gen.MiddleFrameURLs,
				VideoSegmentsJSON:   gen.VideoSegmentsJSON,
				MiddleFrameURLsJSON: gen.MiddleFrameURLsJSON,
			}
		} else {
			// Log why this generation was skipped
			s.logger.Warn("populateMissingSceneVideos: skipping video generation",
				zap.String("sceneId", gen.SceneID),
				zap.String("reason", fmt.Sprintf("status=%s, hasVideoURL=%v", gen.Status, gen.GeneratedVideoURL != "")))
		}
	}

	s.logger.Info("populateMissingSceneVideos: built video URL map",
		zap.String("storyboardId", storyboard.ID),
		zap.Int("scenesWithVideo", len(sceneVideoMap)))

	// Fill in missing scene videos with subdivision info
	for i := range storyboard.StoryboardScenes {
		scene := &storyboard.StoryboardScenes[i]
		if scene.VideoUrl == "" {
			if info, ok := sceneVideoMap[scene.ID]; ok {
				scene.VideoUrl = info.VideoURL
				scene.IsSubdivided = info.IsSubdivided
				scene.VideoSegments = info.VideoSegments
				scene.MiddleFrameURLs = info.MiddleFrameURLs
				scene.VideoSegmentsJSON = info.VideoSegmentsJSON
				s.logger.Info("populated missing scene video from generation record",
					zap.String("sceneId", scene.ID),
					zap.String("videoURL", info.VideoURL),
					zap.Bool("isSubdivided", info.IsSubdivided))
			}
		}
	}
}

// UpdateStoryboard 更新 storyboard
func (s *Service) UpdateStoryboard(ctx context.Context, storyboard *domain.Storyboard, userID string) error {
	s.logger.Info("updating storyboard",
		zap.String("storyboardId", storyboard.ID),
		zap.String("userId", userID),
		zap.String("title", storyboard.Title))

	// 验证权限
	existing, err := s.repo.StoryboardByID(ctx, storyboard.ID)
	if err != nil {
		s.logger.Error("failed to get existing storyboard for update",
			zap.String("storyboardId", storyboard.ID),
			zap.Error(err))
		return err
	}

	if existing.UserID != userID {
		s.logger.Warn("permission denied: user is not the creator",
			zap.String("storyboardId", storyboard.ID),
			zap.String("userId", userID),
			zap.String("creatorId", existing.UserID))
		return fmt.Errorf("permission denied: not the creator")
	}

	// Validate linked assets (SceneRefs are references to StoryScenes, not StoryboardScenes)
	if storyboard.CharacterRefs != nil || storyboard.SceneRefs != nil {
		s.logger.Debug("validating storyboard assets for update",
			zap.String("storyboardId", storyboard.ID),
			zap.Int("characterRefs", len(storyboard.CharacterRefs)),
			zap.Int("sceneRefs", len(storyboard.SceneRefs)))
		if err := s.validateStoryboardAssets(ctx, storyboard); err != nil {
			s.logger.Error("storyboard asset validation failed during update",
				zap.String("storyboardId", storyboard.ID),
				zap.Error(err))
			return err
		}
	}

	if err := s.repo.UpdateStoryboard(ctx, storyboard); err != nil {
		s.logger.Error("failed to update storyboard in database",
			zap.String("storyboardId", storyboard.ID),
			zap.Error(err))
		return fmt.Errorf("failed to update storyboard: %w", err)
	}

	// 使缓存失效并重新缓存
	c := s.getCache()
	if c != nil {
		key := cache.StoryboardKey(storyboard.ID)
		if err := c.Delete(ctx, key); err != nil {
			s.logger.Warn("failed to invalidate storyboard cache",
				zap.String("storyboardId", storyboard.ID),
				zap.Error(err))
		}
		// 重新缓存（不包含 scenes）
		scenes := storyboard.StoryboardScenes
		storyboard.StoryboardScenes = nil
		if err := c.Set(ctx, key, storyboard, entityCacheTTL); err != nil {
			s.logger.Warn("failed to cache updated storyboard",
				zap.String("storyboardId", storyboard.ID),
				zap.Error(err))
		}
		storyboard.StoryboardScenes = scenes
		// 清除相关列表缓存
		for limit := 20; limit <= 100; limit += 20 {
			for offset := 0; offset < 200; offset += limit {
				_ = c.Delete(ctx, cache.StoryboardsListKey(storyboard.StoryID, limit, offset))
			}
		}
	}

	s.logger.Info("storyboard updated successfully",
		zap.String("id", storyboard.ID),
		zap.String("userId", userID),
		zap.String("title", storyboard.Title))

	return nil
}

// persistStoryboardScenes saves AI-generated storyboard scenes to the database
func (s *Service) persistStoryboardScenes(ctx context.Context, storyboard *domain.Storyboard) error {
	if len(storyboard.StoryboardScenes) == 0 {
		s.logger.Debug("no scenes to persist",
			zap.String("storyboardId", storyboard.ID))
		return nil
	}

	s.logger.Debug("persisting storyboard scenes",
		zap.String("storyboardId", storyboard.ID),
		zap.Int("sceneCount", len(storyboard.StoryboardScenes)))

	// Convert to pointer slice for repository
	scenes := make([]*domain.StoryboardScene, len(storyboard.StoryboardScenes))
	for i := range storyboard.StoryboardScenes {
		scenes[i] = &storyboard.StoryboardScenes[i]
		s.logger.Debug("preparing scene for persistence",
			zap.String("storyboardId", storyboard.ID),
			zap.Int("sceneIndex", i),
			zap.String("sceneTitle", scenes[i].Title),
			zap.String("sceneId", scenes[i].ID))
	}

	if err := s.repo.CreateStoryboardScenes(ctx, storyboard.ID, scenes); err != nil {
		s.logger.Error("failed to persist storyboard scenes to database",
			zap.String("storyboardId", storyboard.ID),
			zap.Int("sceneCount", len(scenes)),
			zap.Error(err))
		return fmt.Errorf("failed to persist storyboard scenes: %w", err)
	}

	s.logger.Info("storyboard scenes persisted successfully",
		zap.String("storyboardId", storyboard.ID),
		zap.Int("count", len(scenes)))

	return nil
}

func (s *Service) validateStoryboardAssets(ctx context.Context, storyboard *domain.Storyboard) error {
	// Validate character refs
	if storyboard.CharacterRefs != nil {
		s.logger.Debug("validating character references",
			zap.String("storyboardId", storyboard.ID),
			zap.Int("characterRefCount", len(storyboard.CharacterRefs)))

		var charactersWithoutAvatar []string

		for i := range storyboard.CharacterRefs {
			ref := &storyboard.CharacterRefs[i]
			if ref.CharacterID == "" {
				s.logger.Error("character reference missing characterId",
					zap.String("storyboardId", storyboard.ID),
					zap.Int("refIndex", i))
				return fmt.Errorf("character reference requires characterId")
			}

			// 查询角色（GORM 会自动过滤软删除的记录）
			character, err := s.repo.CharacterByID(ctx, ref.CharacterID)
			if err != nil {
				s.logger.Error("character not found or deleted",
					zap.String("storyboardId", storyboard.ID),
					zap.String("characterId", ref.CharacterID),
					zap.Int("refIndex", i),
					zap.Error(err))
				return fmt.Errorf("character %s not found or has been deleted", ref.CharacterID)
			}

			// 验证角色是否属于该故事
			if character.StoryID != storyboard.StoryID {
				s.logger.Error("character does not belong to this story",
					zap.String("storyboardId", storyboard.ID),
					zap.String("characterId", ref.CharacterID),
					zap.String("characterStoryId", character.StoryID),
					zap.String("storyboardStoryId", storyboard.StoryID))
				return fmt.Errorf("character %s does not belong to story %s", ref.CharacterID, storyboard.StoryID)
			}

			// 检查角色是否有 avatar（渲染故事板图片需要）
			if character.Avatar == "" {
				charactersWithoutAvatar = append(charactersWithoutAvatar, character.Name)
				s.logger.Warn("character missing avatar, image generation may be affected",
					zap.String("storyboardId", storyboard.ID),
					zap.String("characterId", ref.CharacterID),
					zap.String("characterName", character.Name))
			}

			// 填充角色信息到引用中，供后续 AI 生成使用
			ref.Character = character

			if ref.Order == 0 {
				ref.Order = i
			}

			s.logger.Debug("character reference validated",
				zap.String("storyboardId", storyboard.ID),
				zap.String("characterId", ref.CharacterID),
				zap.String("characterName", character.Name),
				zap.Bool("hasAvatar", character.Avatar != ""),
				zap.Int("order", ref.Order))
		}

		// 如果有角色缺少 avatar，记录汇总日志
		if len(charactersWithoutAvatar) > 0 {
			s.logger.Warn("some characters are missing avatars",
				zap.String("storyboardId", storyboard.ID),
				zap.Strings("charactersWithoutAvatar", charactersWithoutAvatar),
				zap.Int("count", len(charactersWithoutAvatar)))
		}
	}

	// Validate scene refs
	if storyboard.SceneRefs != nil {
		s.logger.Debug("validating scene references",
			zap.String("storyboardId", storyboard.ID),
			zap.String("storyId", storyboard.StoryID),
			zap.Int("sceneRefCount", len(storyboard.SceneRefs)))

		var scenesWithoutImage []string

		for i := range storyboard.SceneRefs {
			ref := &storyboard.SceneRefs[i]
			if ref.StorySceneID == "" {
				s.logger.Error("scene reference missing storySceneId",
					zap.String("storyboardId", storyboard.ID),
					zap.Int("refIndex", i))
				return fmt.Errorf("scene reference requires storySceneId")
			}

			scene, err := s.repo.StorySceneByID(ctx, storyboard.StoryID, ref.StorySceneID)
			if err != nil {
				s.logger.Error("story scene not found or deleted",
					zap.String("storyboardId", storyboard.ID),
					zap.String("storyId", storyboard.StoryID),
					zap.String("storySceneId", ref.StorySceneID),
					zap.Int("refIndex", i),
					zap.Error(err))
				return fmt.Errorf("story scene %s not found or has been deleted", ref.StorySceneID)
			}

			// 检查场景是否有图片（可选警告）
			if scene.Image == "" {
				scenesWithoutImage = append(scenesWithoutImage, scene.Title)
				s.logger.Debug("scene missing image",
					zap.String("storyboardId", storyboard.ID),
					zap.String("storySceneId", ref.StorySceneID),
					zap.String("sceneTitle", scene.Title))
			}

			// 填充场景信息到引用中，供后续 AI 生成使用
			ref.StoryScene = scene

			if ref.Sequence == 0 {
				ref.Sequence = i
			}

			s.logger.Debug("scene reference validated",
				zap.String("storyboardId", storyboard.ID),
				zap.String("storySceneId", ref.StorySceneID),
				zap.String("sceneTitle", scene.Title),
				zap.Bool("hasImage", scene.Image != ""),
				zap.Int("sequence", ref.Sequence))
		}

		// 如果有场景缺少图片，记录汇总日志
		if len(scenesWithoutImage) > 0 {
			s.logger.Debug("some scenes are missing images",
				zap.String("storyboardId", storyboard.ID),
				zap.Strings("scenesWithoutImage", scenesWithoutImage),
				zap.Int("count", len(scenesWithoutImage)))
		}
	}

	s.logger.Debug("storyboard assets validation completed",
		zap.String("storyboardId", storyboard.ID),
		zap.Int("characterRefs", len(storyboard.CharacterRefs)),
		zap.Int("sceneRefs", len(storyboard.SceneRefs)))

	return nil
}

// DeleteStoryboard 删除 storyboard
func (s *Service) DeleteStoryboard(ctx context.Context, id, userID string) error {
	s.logger.Info("deleting storyboard",
		zap.String("storyboardId", id),
		zap.String("userId", userID))

	// 验证权限
	storyboard, err := s.repo.StoryboardByID(ctx, id)
	if err != nil {
		s.logger.Error("failed to get storyboard for deletion",
			zap.String("storyboardId", id),
			zap.Error(err))
		return err
	}

	if storyboard.UserID != userID {
		s.logger.Warn("permission denied: user is not the creator",
			zap.String("storyboardId", id),
			zap.String("userId", userID),
			zap.String("creatorId", storyboard.UserID))
		return fmt.Errorf("permission denied: not the creator")
	}

	s.logger.Debug("storyboard deletion authorized",
		zap.String("storyboardId", id),
		zap.String("storyId", storyboard.StoryID),
		zap.String("title", storyboard.Title),
		zap.String("workflowStatus", storyboard.WorkflowStatus))

	// 检查是否为已发布的故事板，已发布的故事板有有效的统计数据
	isPublished := storyboard.WorkflowStatus == string(common.ContentStatusPublished)

	// 记录删除前的统计数据（用于日志和调试）
	if isPublished {
		s.logger.Info("deleting published storyboard with stats",
			zap.String("storyboardId", id),
			zap.Int("likes", storyboard.Likes),
			zap.Int("comments", storyboard.Comments),
			zap.Int("shares", storyboard.Shares),
			zap.Int("views", storyboard.Views),
			zap.Int("forkCount", storyboard.ForkCount),
			zap.Int("tokenConsumption", storyboard.TokenConsumption))
	} else {
		s.logger.Debug("deleting unpublished storyboard (stats are 0)",
			zap.String("storyboardId", id),
			zap.String("workflowStatus", storyboard.WorkflowStatus))
	}

	// 使用事务确保删除和计数更新的原子性
	err = s.repo.WithTransaction(ctx, func(tx domain.Repository) error {
		// 删除
		if err := tx.DeleteStoryboard(ctx, id); err != nil {
			s.logger.Error("failed to delete storyboard from database",
				zap.String("storyboardId", id),
				zap.Error(err))
			return fmt.Errorf("failed to delete storyboard: %w", err)
		}

		// 更新故事的故事板数量（所有故事板都需要更新）
		if err := tx.DecrementStoryStoryboardCount(ctx, storyboard.StoryID); err != nil {
			s.logger.Warn("failed to decrement story storyboard count",
				zap.String("storyId", storyboard.StoryID),
				zap.Error(err))
			return fmt.Errorf("failed to decrement story storyboard count: %w", err)
		}
		s.logger.Debug("story storyboard count decremented",
			zap.String("storyId", storyboard.StoryID))

		return nil
	})

	if err != nil {
		return err
	}

	// 更新 metrics（只有已发布的故事板才需要更新统计指标）
	if s.metrics != nil {
		// 减少故事板总数
		s.metrics.StoryboardCount.Dec()

		// 只有已发布的故事板才有有效的统计数据需要记录
		if isPublished {
			s.logger.Debug("updating metrics for deleted published storyboard",
				zap.String("storyboardId", id),
				zap.Int("likes", storyboard.Likes),
				zap.Int("views", storyboard.Views))
		}
	}

	s.logger.Info("storyboard deleted successfully",
		zap.String("id", id),
		zap.String("userId", userID),
		zap.String("storyId", storyboard.StoryID),
		zap.Bool("wasPublished", isPublished))

	return nil
}

// ListStoryboards 获取故事的 storyboards 列表（带缓存）
func (s *Service) ListStoryboards(ctx context.Context, storyID string, limit, offset int) ([]*domain.Storyboard, error) {
	s.logger.Info("listing storyboards",
		zap.String("storyId", storyID),
		zap.Int("limit", limit),
		zap.Int("offset", offset))

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	// 尝试从缓存获取
	c := s.getCache()
	if c != nil {
		cacheKey := cache.StoryboardsListKey(storyID, limit, offset)
		var cachedStoryboards []*domain.Storyboard
		if err := c.Get(ctx, cacheKey, &cachedStoryboards); err == nil {
			s.logger.Debug("storyboards list cache hit",
				zap.String("storyId", storyID),
				zap.Int("count", len(cachedStoryboards)))
			return cachedStoryboards, nil
		} else {
			s.logger.Debug("storyboards list cache miss",
				zap.String("storyId", storyID),
				zap.Error(err))
		}
	}

	storyboards, err := s.repo.StoryboardsByStory(ctx, storyID, limit, offset)
	if err != nil {
		s.logger.Error("failed to list storyboards",
			zap.String("storyId", storyID),
			zap.Int("limit", limit),
			zap.Int("offset", offset),
			zap.Error(err))
		return nil, fmt.Errorf("failed to list storyboards: %w", err)
	}

	// 写入缓存
	if c != nil && len(storyboards) > 0 {
		cacheKey := cache.StoryboardsListKey(storyID, limit, offset)
		if err := c.Set(ctx, cacheKey, storyboards, listCacheTTL); err != nil {
			s.logger.Warn("failed to cache storyboards list",
				zap.String("storyId", storyID),
				zap.Error(err))
		} else {
			s.logger.Debug("storyboards list cached",
				zap.String("storyId", storyID),
				zap.Int("count", len(storyboards)))
		}
	}

	s.logger.Info("storyboards listed successfully",
		zap.String("storyId", storyID),
		zap.Int("count", len(storyboards)),
		zap.Int("limit", limit),
		zap.Int("offset", offset))

	return storyboards, nil
}

// ListRootStoryboards 获取故事的根 storyboards（ParentID 为空或 "__root__"，带缓存）
func (s *Service) ListRootStoryboards(ctx context.Context, storyID string, limit, offset int) ([]*domain.Storyboard, error) {
	s.logger.Info("listing root storyboards",
		zap.String("storyId", storyID),
		zap.Int("limit", limit),
		zap.Int("offset", offset))

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	// 尝试从缓存获取
	c := s.getCache()
	if c != nil {
		cacheKey := cache.StoryboardsListKey(storyID+"_root", limit, offset)
		var cachedStoryboards []*domain.Storyboard
		if err := c.Get(ctx, cacheKey, &cachedStoryboards); err == nil {
			s.logger.Debug("root storyboards cache hit",
				zap.String("storyId", storyID),
				zap.Int("count", len(cachedStoryboards)))
			return cachedStoryboards, nil
		} else {
			s.logger.Debug("root storyboards cache miss",
				zap.String("storyId", storyID),
				zap.Error(err))
		}
	}

	storyboards, err := s.repo.RootStoryboardsByStory(ctx, storyID, limit, offset)
	if err != nil {
		s.logger.Error("failed to list root storyboards",
			zap.String("storyId", storyID),
			zap.Int("limit", limit),
			zap.Int("offset", offset),
			zap.Error(err))
		return nil, fmt.Errorf("failed to list root storyboards: %w", err)
	}

	// 写入缓存
	if c != nil && len(storyboards) > 0 {
		cacheKey := cache.StoryboardsListKey(storyID+"_root", limit, offset)
		if err := c.Set(ctx, cacheKey, storyboards, listCacheTTL); err != nil {
			s.logger.Warn("failed to cache root storyboards",
				zap.String("storyId", storyID),
				zap.Error(err))
		} else {
			s.logger.Debug("root storyboards cached",
				zap.String("storyId", storyID),
				zap.Int("count", len(storyboards)))
		}
	}

	s.logger.Info("root storyboards listed successfully",
		zap.String("storyId", storyID),
		zap.Int("count", len(storyboards)),
		zap.Int("limit", limit),
		zap.Int("offset", offset))

	return storyboards, nil
}

// ListStoryboardsByParent 获取指定父级的 storyboards（带缓存）
func (s *Service) ListStoryboardsByParent(ctx context.Context, storyID, parentID string, limit, offset int) ([]*domain.Storyboard, error) {
	s.logger.Info("listing storyboards by parent",
		zap.String("storyId", storyID),
		zap.String("parentId", parentID),
		zap.Int("limit", limit),
		zap.Int("offset", offset))

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	// 尝试从缓存获取
	c := s.getCache()
	if c != nil {
		cacheKey := cache.StoryboardsListKey(storyID+"_parent_"+parentID, limit, offset)
		var cachedStoryboards []*domain.Storyboard
		if err := c.Get(ctx, cacheKey, &cachedStoryboards); err == nil {
			s.logger.Debug("storyboards by parent cache hit",
				zap.String("storyId", storyID),
				zap.String("parentId", parentID),
				zap.Int("count", len(cachedStoryboards)))
			return cachedStoryboards, nil
		} else {
			s.logger.Debug("storyboards by parent cache miss",
				zap.String("storyId", storyID),
				zap.String("parentId", parentID),
				zap.Error(err))
		}
	}

	storyboards, err := s.repo.StoryboardsByParent(ctx, storyID, parentID, limit, offset)
	if err != nil {
		s.logger.Error("failed to list storyboards by parent",
			zap.String("storyId", storyID),
			zap.String("parentId", parentID),
			zap.Int("limit", limit),
			zap.Int("offset", offset),
			zap.Error(err))
		return nil, fmt.Errorf("failed to list storyboards by parent: %w", err)
	}

	// 写入缓存
	if c != nil && len(storyboards) > 0 {
		cacheKey := cache.StoryboardsListKey(storyID+"_parent_"+parentID, limit, offset)
		if err := c.Set(ctx, cacheKey, storyboards, listCacheTTL); err != nil {
			s.logger.Warn("failed to cache storyboards by parent",
				zap.String("storyId", storyID),
				zap.String("parentId", parentID),
				zap.Error(err))
		} else {
			s.logger.Debug("storyboards by parent cached",
				zap.String("storyId", storyID),
				zap.String("parentId", parentID),
				zap.Int("count", len(storyboards)))
		}
	}

	s.logger.Info("storyboards by parent listed successfully",
		zap.String("storyId", storyID),
		zap.String("parentId", parentID),
		zap.Int("count", len(storyboards)),
		zap.Int("limit", limit),
		zap.Int("offset", offset))

	return storyboards, nil
}

// GetStoryboardChildren 获取子 storyboards (forks/continuations，带缓存)
func (s *Service) GetStoryboardChildren(ctx context.Context, parentID string) ([]*domain.Storyboard, error) {
	s.logger.Info("getting storyboard children",
		zap.String("parentId", parentID))

	// 尝试从缓存获取
	c := s.getCache()
	if c != nil {
		cacheKey := cache.StoryboardKey(parentID) + ":children"
		var cachedChildren []*domain.Storyboard
		if err := c.Get(ctx, cacheKey, &cachedChildren); err == nil {
			s.logger.Debug("storyboard children cache hit",
				zap.String("parentId", parentID),
				zap.Int("count", len(cachedChildren)))
			return cachedChildren, nil
		} else {
			s.logger.Debug("storyboard children cache miss",
				zap.String("parentId", parentID),
				zap.Error(err))
		}
	}

	children, err := s.repo.StoryboardChildren(ctx, parentID)
	if err != nil {
		s.logger.Error("failed to get storyboard children",
			zap.String("parentId", parentID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get children: %w", err)
	}

	// 写入缓存
	if c != nil && len(children) > 0 {
		cacheKey := cache.StoryboardKey(parentID) + ":children"
		if err := c.Set(ctx, cacheKey, children, listCacheTTL); err != nil {
			s.logger.Warn("failed to cache storyboard children",
				zap.String("parentId", parentID),
				zap.Error(err))
		} else {
			s.logger.Debug("storyboard children cached",
				zap.String("parentId", parentID),
				zap.Int("count", len(children)))
		}
	}

	s.logger.Info("storyboard children retrieved successfully",
		zap.String("parentId", parentID),
		zap.Int("childCount", len(children)))

	return children, nil
}

// GetStoryboardTree 获取完整的 storyboard 树（带缓存）
func (s *Service) GetStoryboardTree(ctx context.Context, rootID string) ([]*domain.Storyboard, error) {
	s.logger.Info("getting storyboard tree",
		zap.String("rootId", rootID))

	// 尝试从缓存获取
	c := s.getCache()
	if c != nil {
		cacheKey := cache.StoryboardKey(rootID) + ":tree"
		var cachedTree []*domain.Storyboard
		if err := c.Get(ctx, cacheKey, &cachedTree); err == nil {
			s.logger.Debug("storyboard tree cache hit",
				zap.String("rootId", rootID),
				zap.Int("nodeCount", len(cachedTree)))
			return cachedTree, nil
		} else {
			s.logger.Debug("storyboard tree cache miss",
				zap.String("rootId", rootID),
				zap.Error(err))
		}
	}

	tree, err := s.repo.StoryboardTree(ctx, rootID)
	if err != nil {
		s.logger.Error("failed to get storyboard tree",
			zap.String("rootId", rootID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get tree: %w", err)
	}

	// 写入缓存
	if c != nil && len(tree) > 0 {
		cacheKey := cache.StoryboardKey(rootID) + ":tree"
		if err := c.Set(ctx, cacheKey, tree, listCacheTTL); err != nil {
			s.logger.Warn("failed to cache storyboard tree",
				zap.String("rootId", rootID),
				zap.Error(err))
		} else {
			s.logger.Debug("storyboard tree cached",
				zap.String("rootId", rootID),
				zap.Int("nodeCount", len(tree)))
		}
	}

	s.logger.Info("storyboard tree retrieved successfully",
		zap.String("rootId", rootID),
		zap.Int("nodeCount", len(tree)))

	return tree, nil
}

// GetStoryboardFeed 获取社区故事板 feed 流（按时间倒序，带缓存）
func (s *Service) GetStoryboardFeed(ctx context.Context, limit, offset int) ([]*domain.Storyboard, int64, error) {
	s.logger.Debug("getting storyboard feed",
		zap.Int("limit", limit),
		zap.Int("offset", offset))

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	// 尝试从缓存获取（feed 流缓存时间较短，因为内容变化频繁）
	c := s.getCache()
	if c != nil {
		cacheKey := "storyboard_feed:" + fmt.Sprintf("%d:%d", limit, offset)
		var cachedFeed []*domain.Storyboard
		var cachedTotal int64
		if err := c.Get(ctx, cacheKey, &cachedFeed); err == nil {
			totalKey := cacheKey + ":total"
			_ = c.Get(ctx, totalKey, &cachedTotal)
			s.logger.Debug("storyboard feed cache hit",
				zap.Int("count", len(cachedFeed)))
			return cachedFeed, cachedTotal, nil
		} else {
			s.logger.Debug("storyboard feed cache miss",
				zap.Error(err))
		}
	}

	storyboards, total, err := s.repo.StoryboardFeed(ctx, limit, offset)
	if err != nil {
		s.logger.Error("failed to get storyboard feed",
			zap.Error(err))
		return nil, 0, fmt.Errorf("failed to get storyboard feed: %w", err)
	}

	// 写入缓存（较短的过期时间，因为 feed 流变化频繁）
	if c != nil && len(storyboards) > 0 {
		cacheKey := "storyboard_feed:" + fmt.Sprintf("%d:%d", limit, offset)
		if err := c.Set(ctx, cacheKey, storyboards, 5*time.Minute); err != nil {
			s.logger.Warn("failed to cache storyboard feed",
				zap.Error(err))
		} else {
			totalKey := cacheKey + ":total"
			_ = c.Set(ctx, totalKey, total, 5*time.Minute)
			s.logger.Debug("storyboard feed cached",
				zap.Int("count", len(storyboards)))
		}
	}

	s.logger.Info("storyboard feed fetched",
		zap.Int("count", len(storyboards)),
		zap.Int64("total", total),
		zap.Int("limit", limit),
		zap.Int("offset", offset))

	return storyboards, total, nil
}

// ForkStoryboard Fork 一个 storyboard
func (s *Service) ForkStoryboard(ctx context.Context, parentID, userID string, newStoryboard *domain.Storyboard) error {
	s.logger.Info("forking storyboard",
		zap.String("parentId", parentID),
		zap.String("userId", userID),
		zap.String("newTitle", newStoryboard.Title))

	// 验证父节点存在
	parent, err := s.repo.StoryboardByID(ctx, parentID)
	if err != nil {
		s.logger.Error("parent storyboard not found for fork",
			zap.String("parentId", parentID),
			zap.Error(err))
		return fmt.Errorf("parent storyboard not found: %w", err)
	}

	s.logger.Debug("parent storyboard found for fork",
		zap.String("parentId", parentID),
		zap.String("parentTitle", parent.Title),
		zap.String("parentStoryId", parent.StoryID))

	// 设置基本信息
	newStoryboard.StoryID = parent.StoryID
	newStoryboard.ParentID = parentID
	newStoryboard.UserID = userID

	// 如果用户提供了新的 rawInput，调用 AI 生成新内容
	if s.geminiClient != nil && newStoryboard.RawInput != "" {
		s.logger.Info("starting AI generation for forked storyboard",
			zap.String("parentId", parentID),
			zap.String("newStoryboardId", newStoryboard.ID),
			zap.String("rawInput", truncateForLog(newStoryboard.RawInput, 200)))
		if err := s.GenerateStoryboardWithAI(ctx, newStoryboard); err != nil {
			s.logger.Warn("AI generation failed for fork",
				zap.String("parentId", parentID),
				zap.String("newStoryboardId", newStoryboard.ID),
				zap.Error(err))
		} else {
			s.logger.Info("AI generation completed for forked storyboard",
				zap.String("newStoryboardId", newStoryboard.ID),
				zap.Int("scenesGenerated", len(newStoryboard.StoryboardScenes)))
		}
	} else {
		// 否则，复制父节点的内容
		s.logger.Debug("copying parent content for fork",
			zap.String("parentId", parentID),
			zap.String("newStoryboardId", newStoryboard.ID))
		newStoryboard.Content = parent.Content
	}

	if len(newStoryboard.CharacterRefs) == 0 && len(parent.CharacterRefs) > 0 {
		s.logger.Debug("copying character refs from parent",
			zap.String("parentId", parentID),
			zap.Int("characterRefCount", len(parent.CharacterRefs)))
		newStoryboard.CharacterRefs = make([]domain.StoryboardCharacterRef, len(parent.CharacterRefs))
		for i, ref := range parent.CharacterRefs {
			newStoryboard.CharacterRefs[i] = domain.StoryboardCharacterRef{
				CharacterID: ref.CharacterID,
				Role:        ref.Role,
				Order:       ref.Order,
				Notes:       ref.Notes,
			}
		}
	}

	if len(newStoryboard.SceneRefs) == 0 && len(parent.SceneRefs) > 0 {
		s.logger.Debug("copying scene refs from parent",
			zap.String("parentId", parentID),
			zap.Int("sceneRefCount", len(parent.SceneRefs)))
		newStoryboard.SceneRefs = make([]domain.StoryboardSceneRef, len(parent.SceneRefs))
		for i, ref := range parent.SceneRefs {
			newStoryboard.SceneRefs[i] = domain.StoryboardSceneRef{
				StorySceneID:   ref.StorySceneID,
				Sequence:       ref.Sequence,
				IsPrimaryScene: ref.IsPrimaryScene,
			}
		}
	}

	// 创建 fork
	s.logger.Debug("saving forked storyboard to database",
		zap.String("parentId", parentID),
		zap.String("newStoryboardId", newStoryboard.ID))
	if err := s.repo.ForkStoryboard(ctx, parentID, userID, newStoryboard); err != nil {
		s.logger.Error("failed to fork storyboard in database",
			zap.String("parentId", parentID),
			zap.String("newStoryboardId", newStoryboard.ID),
			zap.Error(err))
		return fmt.Errorf("failed to fork storyboard: %w", err)
	}

	// 缓存新创建的 fork storyboard
	c := s.getCache()
	if c != nil {
		key := cache.StoryboardKey(newStoryboard.ID)
		scenes := newStoryboard.StoryboardScenes
		newStoryboard.StoryboardScenes = nil
		if err := c.Set(ctx, key, newStoryboard, entityCacheTTL); err != nil {
			s.logger.Warn("failed to cache forked storyboard",
				zap.String("storyboardId", newStoryboard.ID),
				zap.Error(err))
		}
		newStoryboard.StoryboardScenes = scenes
		// 清除相关列表缓存
		for limit := 20; limit <= 100; limit += 20 {
			for offset := 0; offset < 200; offset += limit {
				_ = c.Delete(ctx, cache.StoryboardsListKey(newStoryboard.StoryID, limit, offset))
			}
		}
	}

	s.logger.Info("storyboard forked successfully",
		zap.String("parentId", parentID),
		zap.String("newId", newStoryboard.ID),
		zap.String("userId", userID),
		zap.String("newTitle", newStoryboard.Title),
		zap.String("storyId", newStoryboard.StoryID),
		zap.Int("sceneCount", len(newStoryboard.StoryboardScenes)))

	// 创建通知给父节点作者
	if parent.UserID != userID {
		// 获取 fork 者信息
		forker, err := s.repo.UserByID(ctx, userID)
		if err == nil {
			if err := s.NotifyStoryboardForked(ctx,
				parent.UserID,
				userID,
				forker.DisplayName,
				forker.Avatar,
				newStoryboard.StoryID,
				parentID,
				newStoryboard.ID); err != nil {
				s.logger.Warn("failed to send storyboard forked notification",
					zap.Error(err),
					zap.String("parentId", parentID),
					zap.String("newId", newStoryboard.ID))
			} else {
				s.logger.Info("storyboard forked notification sent",
					zap.String("recipientId", parent.UserID),
					zap.String("parentId", parentID),
					zap.String("newId", newStoryboard.ID))
			}
		} else {
			s.logger.Warn("failed to get forker info for notification",
				zap.Error(err),
				zap.String("forkerId", userID))
		}
	}

	return nil
}

// LikeStoryboard 点赞 storyboard
func (s *Service) LikeStoryboard(ctx context.Context, userID, storyboardID string) error {
	s.logger.Info("liking storyboard",
		zap.String("userId", userID),
		zap.String("storyboardId", storyboardID))

	// 获取 storyboard 信息
	storyboard, err := s.repo.StoryboardByID(ctx, storyboardID)
	if err != nil {
		s.logger.Error("failed to get storyboard",
			zap.String("storyboardId", storyboardID),
			zap.Error(err))
		return fmt.Errorf("failed to get storyboard: %w", err)
	}

	// 执行点赞操作
	if err := s.repo.LikeStoryboard(ctx, userID, storyboardID); err != nil {
		s.logger.Error("failed to like storyboard",
			zap.String("userId", userID),
			zap.String("storyboardId", storyboardID),
			zap.Error(err))
		return fmt.Errorf("failed to like storyboard: %w", err)
	}

	// 使 storyboard 缓存失效（因为点赞数可能变化）
	c := s.getCache()
	if c != nil {
		key := cache.StoryboardKey(storyboardID)
		if err := c.Delete(ctx, key); err != nil {
			s.logger.Warn("failed to invalidate storyboard cache after like",
				zap.String("storyboardId", storyboardID),
				zap.Error(err))
		}
	}

	s.logger.Info("storyboard liked",
		zap.String("userId", userID),
		zap.String("storyboardId", storyboardID),
		zap.String("creatorId", storyboard.UserID))

	// 创建通知给 storyboard 创建者（如果点赞者不是创建者本人）
	if storyboard.UserID != userID {
		// 获取点赞者信息
		liker, err := s.repo.UserByID(ctx, userID)
		if err == nil {
			if err := s.NotifyLike(ctx,
				storyboard.UserID,
				userID,
				liker.DisplayName,
				liker.Avatar,
				"storyboard",
				storyboardID); err != nil {
				s.logger.Warn("failed to send like notification",
					zap.Error(err),
					zap.String("storyboardId", storyboardID))
			} else {
				s.logger.Info("like notification sent",
					zap.String("recipientId", storyboard.UserID),
					zap.String("likerId", userID),
					zap.String("storyboardId", storyboardID))
			}
		} else {
			s.logger.Warn("failed to get liker info for notification",
				zap.Error(err),
				zap.String("likerId", userID))
		}
	}

	return nil
}

// UnlikeStoryboard 取消点赞 storyboard
func (s *Service) UnlikeStoryboard(ctx context.Context, userID, storyboardID string) error {
	s.logger.Info("unliking storyboard",
		zap.String("userId", userID),
		zap.String("storyboardId", storyboardID))

	if err := s.repo.UnlikeStoryboard(ctx, userID, storyboardID); err != nil {
		s.logger.Error("failed to unlike storyboard",
			zap.String("userId", userID),
			zap.String("storyboardId", storyboardID),
			zap.Error(err))
		return fmt.Errorf("failed to unlike storyboard: %w", err)
	}

	// 使 storyboard 缓存失效（因为点赞数可能变化）
	c := s.getCache()
	if c != nil {
		key := cache.StoryboardKey(storyboardID)
		if err := c.Delete(ctx, key); err != nil {
			s.logger.Warn("failed to invalidate storyboard cache after unlike",
				zap.String("storyboardId", storyboardID),
				zap.Error(err))
		}
	}

	s.logger.Info("storyboard unliked successfully",
		zap.String("userId", userID),
		zap.String("storyboardId", storyboardID))

	return nil
}

// GenerateStoryboardWithAI 使用 AI 生成 storyboard 内容
// 业务数据：storyboard的内容和scenes
// AI任务数据：通过AIGenerationService记录
func (s *Service) GenerateStoryboardWithAI(ctx context.Context, storyboard *domain.Storyboard) error {
	s.logger.Info("starting AI storyboard generation",
		zap.String("storyboardId", storyboard.ID),
		zap.String("storyId", storyboard.StoryID),
		zap.String("creatorId", storyboard.UserID),
		zap.String("rawInput", truncateForLog(storyboard.RawInput, 200)),
		zap.Int("sceneCount", storyboard.SceneCount),
		zap.Bool("isStandalone", storyboard.IsStandalone))

	// 1. 获取故事背景信息
	story, err := s.repo.StoryByID(ctx, storyboard.StoryID)
	if err != nil {
		s.logger.Error("failed to get story for AI generation",
			zap.String("storyboardId", storyboard.ID),
			zap.String("storyId", storyboard.StoryID),
			zap.Error(err))
		return fmt.Errorf("failed to get story: %w", err)
	}

	s.logger.Debug("story retrieved for AI generation",
		zap.String("storyId", story.ID),
		zap.String("storyTitle", story.Title),
		zap.String("storyGenre", story.Genre))

	// 2. 构建上下文和提示词
	s.logger.Debug("building storyboard context and prompt",
		zap.String("storyboardId", storyboard.ID),
		zap.Int("characterRefs", len(storyboard.CharacterRefs)),
		zap.Int("sceneRefs", len(storyboard.SceneRefs)))
	contextInfo := s.buildStoryboardContext(ctx, storyboard, story)
	prompt := s.buildStoryboardPrompt(storyboard, story, contextInfo)
	s.logger.Debug("storyboard context and prompt built",
		zap.String("storyboardId", storyboard.ID),
		zap.Int("contextLength", len(contextInfo)),
		zap.Int("promptLength", len(prompt)))
	systemPrompt := s.buildStoryboardSystemPrompt(story)

	// 3. 使用AI生成服务生成文本（自动记录AI使用数据）
	if s.aiGenService == nil {
		return fmt.Errorf("AI generation service not configured")
	}

	// Log the prompt being sent for debugging
	s.logger.Debug("AI storyboard generation request",
		zap.String("storyId", storyboard.StoryID),
		zap.String("systemPrompt", systemPrompt),
		zap.String("prompt", truncateForLog(prompt, 2000)),
		zap.Int("promptLength", len(prompt)),
	)

	genReq := &GenerateTextRequest{
		UserID:            storyboard.UserID,
		OriginalPrompt:    prompt,
		SystemPrompt:      systemPrompt,
		Model:             "gemini-2.5-flash",
		Temperature:       0.7,
		MaxTokens:         4000, // Increased for Chinese content which requires more tokens
		RelatedEntityID:   storyboard.ID,
		RelatedEntityType: "storyboard",
		Metadata: map[string]interface{}{
			"step":         "storyboard_content_generation",
			"storyboardId": storyboard.ID,
			"storyId":      storyboard.StoryID,
			"operation":    "generate_storyboard_content",
		},
	}

	result, err := s.aiGenService.GenerateText(ctx, genReq)
	if err != nil {
		s.logger.Error("AI generation failed", zap.Error(err))
		return fmt.Errorf("AI generation failed: %w", err)
	}

	// 4. 解析生成结果
	s.logger.Debug("parsing AI generation result",
		zap.String("storyboardId", storyboard.ID),
		zap.Int("responseLength", len(result.Text)))
	storyboardResult := s.parseStoryboardResult(result.Text)

	// 5. 截断保护：确保content字段不超过1024字符
	const maxContentLength = 1024
	if len(storyboardResult.Content) > maxContentLength {
		originalLength := len(storyboardResult.Content)
		// 截断到1024字符，尽量在完整的句子处截断
		storyboardResult.Content = truncateAtSentence(storyboardResult.Content, maxContentLength)
		s.logger.Warn("storyboard content truncated due to length limit",
			zap.String("storyboardId", storyboard.ID),
			zap.Int("originalLength", originalLength),
			zap.Int("truncatedLength", len(storyboardResult.Content)),
			zap.Int("maxLength", maxContentLength))
	}

	// 6. 更新业务数据：storyboard 内容
	storyboard.Content = storyboardResult.Content
	storyboard.IsAIGenerated = true
	s.logger.Debug("storyboard content updated from AI generation",
		zap.String("storyboardId", storyboard.ID),
		zap.Int("contentLength", len(storyboard.Content)),
		zap.Int("scenesCount", len(storyboardResult.Scenes)))

	// 将 StoryboardSceneResult 转换为 domain.StoryboardScene (AI-generated plot scenes)
	storyboardScenes := make([]domain.StoryboardScene, 0, len(storyboardResult.Scenes))
	for i, sceneResult := range storyboardResult.Scenes {
		// 从角色对象数组中提取角色名称
		characterNames := make([]string, 0, len(sceneResult.Characters))
		characterIDs := make([]string, 0, len(sceneResult.Characters))
		for _, charRef := range sceneResult.Characters {
			if charRef.Name != "" {
				characterNames = append(characterNames, charRef.Name)
			}
			if charRef.ID != "" {
				characterIDs = append(characterIDs, charRef.ID)
			}
		}

		scene := domain.StoryboardScene{
			Sequence:      i,
			Title:         sceneResult.Title,
			Description:   sceneResult.Description,
			Location:      sceneResult.Location,
			TimeOfDay:     sceneResult.TimeOfDay,
			Mood:          sceneResult.Mood,
			Characters:    characterNames,           // 角色名称数组
			StorySceneID:  sceneResult.StorySceneID, // 保存关联的场景ID
			Image:         "",                       // 图片将在后续生成
			IsAIGenerated: true,
		}
		// 确保 Characters 不为 nil
		if scene.Characters == nil {
			scene.Characters = []string{}
		}
		// 记录角色ID信息（用于日志和后续关联）
		if len(characterIDs) > 0 {
			s.logger.Debug("scene has character IDs from AI",
				zap.String("storyboardId", storyboard.ID),
				zap.Int("sceneIndex", i),
				zap.Strings("characterIds", characterIDs),
				zap.Strings("characterNames", characterNames))
		}
		// 记录场景ID信息
		if sceneResult.StorySceneID != "" {
			s.logger.Debug("scene has story scene ID from AI",
				zap.String("storyboardId", storyboard.ID),
				zap.Int("sceneIndex", i),
				zap.String("storySceneId", sceneResult.StorySceneID))
		}
		storyboardScenes = append(storyboardScenes, scene)
	}
	storyboard.StoryboardScenes = storyboardScenes

	s.logger.Info("storyboard AI generation completed",
		zap.String("storyboardId", storyboard.ID),
		zap.String("aiRecordId", result.RecordID),
		zap.Int("tokensUsed", result.TokensUsed),
		zap.Int64("durationMs", result.DurationMs))

	// 6. 可选：生成场景图片
	if storyboardResult.GenerateImages && s.genAPI != nil {
		s.logger.Info("starting scene image generation",
			zap.String("storyboardId", storyboard.ID),
			zap.Int("sceneCount", len(storyboard.StoryboardScenes)))
		if err := s.generateSceneImages(ctx, storyboard); err != nil {
			s.logger.Warn("failed to generate scene images",
				zap.String("storyboardId", storyboard.ID),
				zap.Error(err))
			// 图片生成失败不影响整体流程
		} else {
			s.logger.Info("scene images generation completed",
				zap.String("storyboardId", storyboard.ID),
				zap.Int("sceneCount", len(storyboard.StoryboardScenes)))
		}
	}

	return nil
}

// getAncestorStoryboards fetches up to maxLevels ancestor storyboards in chronological order (oldest first)
func (s *Service) getAncestorStoryboards(ctx context.Context, storyboard *domain.Storyboard, maxLevels int) []*domain.Storyboard {
	s.logger.Debug("fetching ancestor storyboards",
		zap.String("storyboardId", storyboard.ID),
		zap.String("parentId", storyboard.ParentID),
		zap.Int("maxLevels", maxLevels))

	var ancestors []*domain.Storyboard
	currentParentID := storyboard.ParentID

	for i := 0; i < maxLevels; i++ {
		if currentParentID == "" || currentParentID == domain.StoryboardRootMarker {
			s.logger.Debug("reached root marker, stopping ancestor fetch",
				zap.String("storyboardId", storyboard.ID),
				zap.Int("ancestorsFound", len(ancestors)))
			break
		}

		parent, err := s.repo.StoryboardByID(ctx, currentParentID)
		if err != nil {
			s.logger.Warn("failed to fetch ancestor storyboard",
				zap.String("storyboardId", storyboard.ID),
				zap.String("parentId", currentParentID),
				zap.Int("level", i+1),
				zap.Error(err))
			break
		}

		// Prepend to get chronological order (oldest first)
		ancestors = append([]*domain.Storyboard{parent}, ancestors...)
		s.logger.Debug("ancestor storyboard fetched",
			zap.String("storyboardId", storyboard.ID),
			zap.String("ancestorId", parent.ID),
			zap.String("ancestorTitle", parent.Title),
			zap.Int("level", i+1),
			zap.Int("totalAncestors", len(ancestors)))
		currentParentID = parent.ParentID
	}

	s.logger.Debug("ancestor storyboards fetch completed",
		zap.String("storyboardId", storyboard.ID),
		zap.Int("ancestorsFound", len(ancestors)))

	return ancestors
}

// buildStoryboardContext 构建 storyboard 生成上下文
// AI 自动从故事的完整角色和场景列表中选择合适的元素
func (s *Service) buildStoryboardContext(ctx context.Context, storyboard *domain.Storyboard, story *domain.Story) string {
	s.logger.Debug("building storyboard context",
		zap.String("storyboardId", storyboard.ID),
		zap.String("storyId", story.ID),
		zap.Bool("isStandalone", storyboard.IsStandalone))

	var context string

	// 添加故事信息
	context += fmt.Sprintf("故事标题: %s\n", story.Title)
	context += fmt.Sprintf("故事简介: %s\n\n", story.Description)
	s.logger.Debug("added story information to context",
		zap.String("storyboardId", storyboard.ID),
		zap.String("storyTitle", story.Title))

	// 如果是独立故事板，不添加父节点内容
	if storyboard.IsStandalone {
		context += "（独立故事线，不参考前情）\n\n"
		s.logger.Debug("storyboard is standalone, skipping ancestor context",
			zap.String("storyboardId", storyboard.ID))
	} else if storyboard.ParentID != "" && storyboard.ParentID != domain.StoryboardRootMarker {
		// 获取最多5级祖先故事板作为上下文
		ancestors := s.getAncestorStoryboards(ctx, storyboard, 5)
		if len(ancestors) > 0 {
			s.logger.Debug("adding ancestor context",
				zap.String("storyboardId", storyboard.ID),
				zap.Int("ancestorCount", len(ancestors)))
			context += "前情提要（按时间顺序）：\n"
			for i, ancestor := range ancestors {
				// 限制每个祖先内容长度，避免上下文过长
				ancestorContent := truncateForLog(ancestor.Content, 300)
				context += fmt.Sprintf("\n【第%d章 - %s】\n%s\n", i+1, ancestor.Title, ancestorContent)
			}
			context += "\n"
		} else {
			s.logger.Debug("no ancestors found for context",
				zap.String("storyboardId", storyboard.ID))
		}
	}

	// 获取故事的所有角色供 AI 选择
	characters, err := s.repo.CharactersByStory(ctx, story.ID)
	if err == nil && len(characters) > 0 {
		s.logger.Debug("adding all story characters to context",
			zap.String("storyboardId", storyboard.ID),
			zap.Int("characterCount", len(characters)))
		context += "故事中的可用角色（AI 请根据剧情需要智能选择）：\n"
		for _, char := range characters {
			context += fmt.Sprintf("- %s [角色ID: %s]", char.Name, char.ID)
			if char.Description != "" {
				context += fmt.Sprintf(": %s", truncateForLog(char.Description, 100))
			}
			context += "\n"
		}
		context += "\n"
		context += "重要提示：AI 应根据用户描述的故事情节，智能选择合适的角色。只有确实参与场景的角色才需要包含在characters数组中，每个角色对象必须包含name（角色完整名称）和id（对应的角色ID）。\n\n"
	}

	// 获取故事的所有场景供 AI 选择
	scenes, err := s.repo.StoryScenes(ctx, story.ID, 100, 0)
	if err == nil && len(scenes) > 0 {
		s.logger.Debug("adding all story scenes to context",
			zap.String("storyboardId", storyboard.ID),
			zap.Int("sceneCount", len(scenes)))
		context += "故事中的可用场景地点（AI 请根据剧情需要智能选择）：\n"
		for _, scene := range scenes {
			context += fmt.Sprintf("- %s [场景ID: %s]", scene.Title, scene.ID)
			if scene.Description != "" {
				context += fmt.Sprintf(": %s", truncateForLog(scene.Description, 100))
			}
			if scene.Location != "" {
				context += fmt.Sprintf(" (地点: %s)", scene.Location)
			}
			context += "\n"
		}
		context += "\n"
		context += "重要提示：AI 应根据用户描述的故事情节，智能选择合适的场景。如果场景地点与剧情相关，应在storySceneId字段中提供对应的场景ID。\n\n"
	}

	s.logger.Debug("storyboard context built",
		zap.String("storyboardId", storyboard.ID),
		zap.Int("contextLength", len(context)))

	return context
}

// buildStoryboardPrompt 构建 storyboard 生成提示词
func (s *Service) buildStoryboardPrompt(storyboard *domain.Storyboard, story *domain.Story, contextInfo string) string {
	s.logger.Debug("building storyboard prompt",
		zap.String("storyboardId", storyboard.ID),
		zap.Int("requestedSceneCount", storyboard.SceneCount))

	prompt := "作为专业的故事创作者，请根据用户的自然语言描述，创作精彩的故事分镜内容。\n\n"
	prompt += "**创作指南**：\n"
	prompt += "1. 深入理解用户的描述意图，把握故事的核心情感和情节走向\n"
	prompt += "2. 从提供的角色列表中选择最适合情节的角色，不必使用所有角色\n"
	prompt += "3. 从提供的场景列表中选择最适合氛围的地点，或根据描述创造合适的场景\n"
	prompt += "4. 让情节自然展开，角色互动应符合其性格设定\n\n"

	if contextInfo != "" {
		prompt += contextInfo
	}

	prompt += fmt.Sprintf("**用户的故事描述**: %s\n\n", storyboard.RawInput)

	// Determine scene count (default to 3 if not set or out of range)
	sceneCount := storyboard.SceneCount
	if sceneCount < 2 || sceneCount > 5 {
		sceneCount = 3
		s.logger.Debug("scene count adjusted to default",
			zap.String("storyboardId", storyboard.ID),
			zap.Int("requested", storyboard.SceneCount),
			zap.Int("adjusted", sceneCount))
	}

	prompt += "请直接返回纯JSON（不要使用```包裹），格式如下：\n"
	prompt += "{\n"
	prompt += "  \"content\": \"润色后的故事内容（必须控制在420字以内，建议300-400字）\",\n"
	prompt += "  \"scenes\": [\n"
	prompt += "    {\n"
	prompt += "      \"title\": \"场景标题（10字以内）\",\n"
	prompt += "      \"description\": \"场景描述（100-200字）\",\n"
	prompt += "      \"location\": \"地点\",\n"
	prompt += "      \"timeOfDay\": \"时间\",\n"
	prompt += "      \"storySceneId\": \"场景ID（仅当使用提供的场景时填写）\",\n"
	prompt += "      \"characters\": [\n"
	prompt += "        {\n"
	prompt += "          \"name\": \"角色名\",\n"
	prompt += "          \"id\": \"角色ID\"\n"
	prompt += "        }\n"
	prompt += "      ],\n"
	prompt += "      \"mood\": \"氛围\"\n"
	prompt += "    }\n"
	prompt += "  ],\n"
	prompt += "  \"generateImages\": false\n"
	prompt += "}\n"
	prompt += fmt.Sprintf("\n重要：请生成恰好 %d 个场景，确保JSON格式完整闭合。", sceneCount)
	prompt += "\n重要：content字段必须严格控制在420字以内。"
	prompt += "\n重要：只在characters数组中包含确实参与该场景的角色，并为每个角色提供正确的ID。"
	prompt += "\n重要：如果使用了提供的场景地点，请填写对应的storySceneId。"

	s.logger.Debug("storyboard prompt built",
		zap.String("storyboardId", storyboard.ID),
		zap.Int("promptLength", len(prompt)),
		zap.Int("sceneCount", sceneCount))

	return prompt
}

// buildStoryboardSystemPrompt 构建 storyboard 生成的系统提示词
func (s *Service) buildStoryboardSystemPrompt(story *domain.Story) string {
	// 根据故事类型获取风格指导
	genreGuidance := s.getGenreStyleGuidance(story.Genre)

	systemPrompt := `# 角色定义
你是一位专业的故事分镜编剧，擅长将用户的创意构思转化为生动、有画面感的故事分镜脚本。你具备以下专业能力：
- 故事结构设计：精通三幕结构、起承转合等叙事技巧
- 场景视觉化：能够将抽象概念转化为具体的视觉场景描述
- 角色刻画：善于通过场景互动展现角色性格和关系
- 氛围营造：精准把握不同场景的情感基调和环境氛围
- 智能选择：能够根据故事情节需要，从可用资源中智能选择合适的角色和场景

# 创作原则
1. **用户意图优先**：以用户的自然语言描述为核心，理解用户的创作意图
2. **自由发挥**：根据故事主旨和情节需要，智能选择最合适的角色和场景，不必拘泥于使用所有可用资源
3. **连贯性**：确保场景之间有自然的过渡和逻辑联系
4. **画面感**：每个场景描述应具有强烈的视觉冲击力，便于后续图像/视频生成
5. **角色一致性**：使用角色时严格遵循其设定，保持人物行为和性格的连贯
6. **情感递进**：场景间应有情感的起伏变化，避免单调

` + genreGuidance + `

# 角色和场景选择指南
- **智能选择**：根据用户描述的故事情节，选择最相关的角色参与场景
- **不必求全**：不需要使用所有角色，只选择对情节发展有作用的角色
- **场景匹配**：选择与故事氛围和情节相符的场景地点
- **灵活创作**：如果用户描述需要，可以创造新的场景感，不必严格限制在提供的场景列表中

# 输出格式要求
**重要**：直接返回纯JSON，不要使用markdown代码块包裹（不要用` + "`" + "`" + "`" + `json或` + "`" + "`" + "`" + `）

确保输出的JSON满足以下要求：
- 所有括号、引号正确配对和闭合
- 字符串内的特殊字符正确转义
- 不包含注释或多余的逗号

# 内容质量标准
- content字段：润色后的完整故事概述，**必须严格控制在420字以内**（建议300-400字），语言流畅优美，避免冗长
- 场景标题：简洁有力，10字以内，体现场景核心
- 场景描述：100-200字，包含环境、动作、情感三个维度
- 地点和时间：具体明确，与场景内容呼应
- 氛围关键词：精准概括场景情感基调（如：紧张、温馨、神秘、悲伤）
- 角色选择：只在characters数组中包含确实参与该场景的角色

# 长度限制说明
**极其重要**：content字段长度限制为420字是硬性要求，这是为了避免后续处理时的内容截断问题。系统内部最多可处理1024字符，但为确保质量和完整性，生成内容应控制在420字以内。`

	return systemPrompt
}

// getGenreStyleGuidance 根据故事类型返回创作风格指导
func (s *Service) getGenreStyleGuidance(genre string) string {
	genreGuides := map[string]string{
		"fantasy": `# 奇幻类型创作指南
- 注重魔法元素和奇幻世界观的展现
- 场景描述应突出神秘感和想象力
- 可适当加入奇幻生物、魔法特效等元素`,
		"romance": `# 浪漫类型创作指南
- 注重人物情感互动和内心描写
- 场景应营造浪漫、温馨的氛围
- 关注眼神交流、肢体语言等细节`,
		"thriller": `# 悬疑/惊悚类型创作指南
- 注重悬念铺设和紧张氛围的营造
- 场景描述应突出阴影、光影对比
- 节奏把控：张弛有度，层层递进`,
		"scifi": `# 科幻类型创作指南
- 注重未来科技感和视觉震撼
- 场景应体现科技元素与人文关怀的平衡
- 关注高科技设备、太空场景等元素的描述`,
		"adventure": `# 冒险类型创作指南
- 注重动作场面和探险元素
- 场景应体现挑战性和刺激感
- 展现角色的勇气和成长`,
		"comedy": `# 喜剧类型创作指南
- 注重幽默元素和轻松氛围
- 场景可适当夸张，突出喜剧效果
- 关注人物表情和滑稽动作的描写`,
		"horror": `# 恐怖类型创作指南
- 注重恐怖氛围的层层铺垫
- 场景描述应突出阴暗、压抑的环境
- 善用心理恐惧，而非单纯的视觉冲击`,
		"drama": `# 剧情类型创作指南
- 注重人物内心冲突和人际关系
- 场景应服务于情感表达
- 关注细腻的情感变化和对话张力`,
	}

	if guidance, ok := genreGuides[genre]; ok {
		return guidance
	}

	// 默认通用指导
	return `# 创作风格指南
- 根据故事内容自然选择合适的叙事风格
- 保持场景描述的生动性和画面感
- 注重情节发展的逻辑性和趣味性`
}

// CharacterRef AI生成的角色引用
type CharacterRef struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

// StoryboardSceneResult AI 生成的场景结果（强类型解析）
type StoryboardSceneResult struct {
	Title        string         `json:"title"`
	Description  string         `json:"description"`
	Location     string         `json:"location"`
	TimeOfDay    string         `json:"timeOfDay"`
	Characters   []CharacterRef `json:"characters"`             // 角色对象数组，包含name和id
	StorySceneID string         `json:"storySceneId,omitempty"` // 关联的场景ID
	Mood         string         `json:"mood"`
}

// StoryboardResult AI 生成的 storyboard 结果
type StoryboardResult struct {
	Content        string                  `json:"content"`
	Scenes         []StoryboardSceneResult `json:"scenes"`
	GenerateImages bool                    `json:"generateImages"`
}

// parseStoryboardResult 解析 AI 生成的结果
func (s *Service) parseStoryboardResult(text string) *StoryboardResult {
	var result StoryboardResult

	// Log raw AI response for debugging
	s.logger.Debug("AI raw response for storyboard",
		zap.Int("rawLength", len(text)),
		zap.String("rawText", truncateForLog(text, 2000)),
	)

	// 清理文本
	cleanedText := s.cleanAIResponseText(text)

	s.logger.Debug("AI response after cleaning",
		zap.Int("cleanedLength", len(cleanedText)),
		zap.String("cleanedText", truncateForLog(cleanedText, 2000)),
	)

	// 尝试解析 JSON
	if err := json.Unmarshal([]byte(cleanedText), &result); err != nil {
		s.logger.Warn("initial JSON parse failed, attempting recovery",
			zap.Error(err))

		// 尝试修复常见的 JSON 问题后重新解析
		fixedText := s.fixCommonJSONIssues(cleanedText)
		if fixedText != cleanedText {
			if err2 := json.Unmarshal([]byte(fixedText), &result); err2 == nil {
				s.logger.Info("JSON parsed successfully after fixing",
					zap.Int("contentLength", len(result.Content)),
					zap.Int("scenesCount", len(result.Scenes)))
				s.validateStoryboardResult(&result)
				return &result
			}
		}

		// 如果仍然失败，使用原始文本
		s.logger.Warn("failed to parse JSON, using raw text as content",
			zap.Error(err),
			zap.Int("rawLength", len(text)),
			zap.Int("cleanedLength", len(cleanedText)),
			zap.String("rawTextPreview", truncateForLog(text, 500)),
		)
		result.Content = text
		result.Scenes = []StoryboardSceneResult{}
	} else {
		s.logger.Info("AI response JSON parsed successfully",
			zap.Int("contentLength", len(result.Content)),
			zap.Int("scenesCount", len(result.Scenes)),
		)
		s.validateStoryboardResult(&result)
	}

	return &result
}

// cleanAIResponseText 清理 AI 响应文本
func (s *Service) cleanAIResponseText(text string) string {
	cleanedText := strings.TrimSpace(text)

	// 移除 markdown 代码块 (```json ... ``` 或 ``` ... ```)
	if strings.HasPrefix(cleanedText, "```") {
		// 找到第一行的结束位置（跳过 ```json 或 ```）
		if idx := strings.Index(cleanedText, "\n"); idx != -1 {
			cleanedText = cleanedText[idx+1:]
		}
		// 移除尾部的 ```
		if idx := strings.LastIndex(cleanedText, "```"); idx != -1 {
			cleanedText = strings.TrimSpace(cleanedText[:idx])
		}
	}

	// 移除可能的 BOM 标记
	cleanedText = strings.TrimPrefix(cleanedText, "\ufeff")

	// 移除开头可能的非 JSON 文本（找到第一个 { 的位置）
	if idx := strings.Index(cleanedText, "{"); idx > 0 {
		// 检查前面是否只有空白字符
		prefix := strings.TrimSpace(cleanedText[:idx])
		if prefix != "" {
			s.logger.Debug("removing non-JSON prefix",
				zap.String("prefix", truncateForLog(prefix, 100)))
		}
		cleanedText = cleanedText[idx:]
	}

	// 确保以 } 结尾，移除尾部可能的非 JSON 文本
	if idx := strings.LastIndex(cleanedText, "}"); idx >= 0 && idx < len(cleanedText)-1 {
		suffix := strings.TrimSpace(cleanedText[idx+1:])
		if suffix != "" {
			s.logger.Debug("removing non-JSON suffix",
				zap.String("suffix", truncateForLog(suffix, 100)))
		}
		cleanedText = cleanedText[:idx+1]
	}

	return cleanedText
}

// fixCommonJSONIssues 修复常见的 JSON 格式问题
func (s *Service) fixCommonJSONIssues(text string) string {
	fixed := text

	// 移除尾部多余的逗号（如 ",}" 或 ",]"）
	// 使用简单的字符串替换处理常见情况
	fixed = strings.ReplaceAll(fixed, ",\n}", "\n}")
	fixed = strings.ReplaceAll(fixed, ",\n]", "\n]")
	fixed = strings.ReplaceAll(fixed, ", }", " }")
	fixed = strings.ReplaceAll(fixed, ", ]", " ]")
	fixed = strings.ReplaceAll(fixed, ",}", "}")
	fixed = strings.ReplaceAll(fixed, ",]", "]")

	// 修复可能的中文标点问题
	fixed = strings.ReplaceAll(fixed, "，", ",")
	fixed = strings.ReplaceAll(fixed, "：", ":")
	fixed = strings.ReplaceAll(fixed, "\u201c", `"`) // 中文左双引号
	fixed = strings.ReplaceAll(fixed, "\u201d", `"`) // 中文右双引号
	fixed = strings.ReplaceAll(fixed, "【", "[")
	fixed = strings.ReplaceAll(fixed, "】", "]")

	return fixed
}

// validateStoryboardResult 验证并修正解析结果
func (s *Service) validateStoryboardResult(result *StoryboardResult) {
	// 验证 content
	if result.Content == "" {
		s.logger.Warn("storyboard result has empty content")
	} else if len(result.Content) > 2000 {
		s.logger.Debug("storyboard content exceeds recommended length",
			zap.Int("contentLength", len(result.Content)))
	}

	// 验证 scenes
	if len(result.Scenes) == 0 {
		s.logger.Warn("storyboard result has no scenes")
	} else {
		for i, scene := range result.Scenes {
			if scene.Title == "" {
				s.logger.Debug("scene missing title",
					zap.Int("sceneIndex", i))
			}
			if scene.Description == "" {
				s.logger.Debug("scene missing description",
					zap.Int("sceneIndex", i))
			}
			// 确保 Characters 不为 nil
			if scene.Characters == nil {
				result.Scenes[i].Characters = []CharacterRef{}
			}
		}
	}
}

// truncateAtSentence 在指定长度处截断文本，尽量在句子边界处截断
func truncateAtSentence(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}

	// 截取到最大长度
	truncated := text[:maxLength]

	// 尝试在句子结束符处截断（。！？）
	sentenceEnds := []string{"。", "！", "？", ".", "!", "?"}
	lastSentencePos := -1
	for _, end := range sentenceEnds {
		if pos := strings.LastIndex(truncated, end); pos > lastSentencePos && pos > maxLength/2 {
			lastSentencePos = pos
		}
	}

	// 如果找到了句子结束符，在那里截断
	if lastSentencePos > 0 {
		return truncated[:lastSentencePos+len("。")]
	}

	// 否则尝试在标点符号处截断
	punctuations := []string{"，", "、", ",", ";", "；"}
	lastPuncPos := -1
	for _, punct := range punctuations {
		if pos := strings.LastIndex(truncated, punct); pos > lastPuncPos && pos > maxLength/2 {
			lastPuncPos = pos
		}
	}

	if lastPuncPos > 0 {
		return truncated[:lastPuncPos+len("，")]
	}

	// 最后在空格处截断
	if pos := strings.LastIndex(truncated, " "); pos > maxLength/2 {
		return truncated[:pos]
	}

	// 如果都找不到，直接截断并添加省略号
	return truncated + "..."
}

// truncateForLog truncates a string for logging purposes
func truncateForLog(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "...(truncated)"
}

// generateSceneImages 为故事板场景生成图片（使用AI生成服务记录）
func (s *Service) generateSceneImages(ctx context.Context, storyboard *domain.Storyboard) error {
	if len(storyboard.StoryboardScenes) == 0 || s.aiGenService == nil {
		s.logger.Debug("skipping scene image generation",
			zap.String("storyboardId", storyboard.ID),
			zap.Int("sceneCount", len(storyboard.StoryboardScenes)),
			zap.Bool("aiGenServiceAvailable", s.aiGenService != nil))
		return nil
	}

	// 获取故事的风格配置
	var storyStyle *domain.StyleConfig
	if storyboard.StoryID != "" {
		story, err := s.repo.StoryByID(ctx, storyboard.StoryID)
		if err == nil && story.Style != nil {
			storyStyle = story.Style
			s.logger.Debug("fetched story style for scene image generation",
				zap.String("storyId", storyboard.StoryID),
				zap.String("style", storyStyle.Style))
		}
	}

	// 获取故事的所有角色（用于根据场景角色名获取角色图片）
	var storyCharacters []*domain.Character
	if storyboard.StoryID != "" {
		chars, err := s.repo.CharactersByStory(ctx, storyboard.StoryID)
		if err == nil {
			storyCharacters = chars
			s.logger.Debug("fetched story characters for reference images",
				zap.String("storyId", storyboard.StoryID),
				zap.Int("characterCount", len(chars)))
		}
	}

	// 创建角色名称到角色的映射
	charMap := make(map[string]*domain.Character)
	for _, char := range storyCharacters {
		charMap[char.Name] = char
	}

	s.logger.Info("generating images for storyboard scenes",
		zap.String("storyboardId", storyboard.ID),
		zap.Int("totalScenes", len(storyboard.StoryboardScenes)),
		zap.Bool("hasStoryStyle", storyStyle != nil))

	// 为每个故事板场景生成图片
	for i := range storyboard.StoryboardScenes {
		scene := &storyboard.StoryboardScenes[i]

		// 判断是否为过渡场景（没有角色出现）
		isTransitionScene := len(scene.Characters) == 0

		// 收集场景关联角色的图片（限制最多5个主要角色）
		var referenceImages []string
		if !isTransitionScene {
			maxMainCharacters := 5
			mainCharacterNames := scene.Characters
			if len(mainCharacterNames) > maxMainCharacters {
				mainCharacterNames = mainCharacterNames[:maxMainCharacters]
				s.logger.Debug("limiting main characters to 5 for scene image generation",
					zap.String("storyboardId", storyboard.ID),
					zap.Int("sceneIndex", i),
					zap.Int("originalCount", len(scene.Characters)),
					zap.Int("limitedCount", len(mainCharacterNames)))
			}

			for _, charName := range mainCharacterNames {
				if char, ok := charMap[charName]; ok {
					// 只使用Portrait（完整角色形象图），如果没有Portrait则跳过该角色（该角色只会在文本中描述，不参与图片生成）
					if char.Portrait != "" {
						referenceImages = append(referenceImages, char.Portrait)
					} else {
						s.logger.Debug("character has no portrait, skipping from scene image generation",
							zap.String("storyboardId", storyboard.ID),
							zap.Int("sceneIndex", i),
							zap.String("characterName", charName),
							zap.String("characterId", char.ID))
					}
				}
			}
		}

		s.logger.Debug("generating image for scene",
			zap.String("storyboardId", storyboard.ID),
			zap.Int("sceneIndex", i),
			zap.String("sceneId", scene.ID),
			zap.String("sceneTitle", scene.Title),
			zap.Bool("isTransitionScene", isTransitionScene),
			zap.Int("characterReferenceCount", len(referenceImages)))

		// 构建图片提示词（包含故事风格配置）
		prompt := s.buildStoryboardSceneImagePromptWithStyle(scene, storyStyle, isTransitionScene)
		s.logger.Debug("scene image prompt built",
			zap.String("storyboardId", storyboard.ID),
			zap.Int("sceneIndex", i),
			zap.Int("promptLength", len(prompt)))

		// 使用AI生成服务生成图片（自动记录AI使用数据）
		imageReq := &GenerateImageRequest{
			UserID:            storyboard.UserID,
			Prompt:            prompt,
			Provider:          "gemini",
			Model:             "imagen-3.0-generate-001",
			AspectRatio:       "16:9",
			Quality:           "high",
			OutputCount:       1,
			ReferenceImages:   referenceImages, // 使用角色参考图片
			RelatedEntityID:   storyboard.ID,
			RelatedEntityType: "storyboard_scene",
			Metadata: map[string]interface{}{
				"storyId":           storyboard.StoryID,
				"sceneIndex":        i,
				"sceneTitle":        scene.Title,
				"isTransitionScene": isTransitionScene,
				"sceneCharacters":   scene.Characters,
			},
		}

		result, err := s.aiGenService.GenerateImage(ctx, imageReq)
		if err != nil {
			s.logger.Warn("failed to generate scene image",
				zap.String("storyboardId", storyboard.ID),
				zap.Int("sceneIndex", i),
				zap.String("sceneId", scene.ID),
				zap.String("sceneTitle", scene.Title),
				zap.Error(err))
			continue
		}

		if len(result.ImageURLs) > 0 {
			scene.Image = result.ImageURLs[0]
			s.logger.Info("scene image generated successfully",
				zap.String("storyboardId", storyboard.ID),
				zap.Int("sceneIndex", i),
				zap.String("sceneId", scene.ID),
				zap.String("sceneTitle", scene.Title),
				zap.String("aiRecordId", result.RecordID),
				zap.String("imageURL", scene.Image),
				zap.Int("tokensUsed", result.TokensUsed))
		} else {
			s.logger.Warn("scene image generation returned no URLs",
				zap.String("storyboardId", storyboard.ID),
				zap.Int("sceneIndex", i),
				zap.String("sceneId", scene.ID))
		}
	}

	s.logger.Info("scene images generation completed",
		zap.String("storyboardId", storyboard.ID),
		zap.Int("totalScenes", len(storyboard.StoryboardScenes)))

	return nil
}

// buildStoryboardSceneImagePrompt 构建故事板场景图片提示词
func (s *Service) buildStoryboardSceneImagePrompt(scene *domain.StoryboardScene) string {
	s.logger.Debug("building scene image prompt",
		zap.String("sceneId", scene.ID),
		zap.String("sceneTitle", scene.Title),
		zap.String("location", scene.Location),
		zap.String("timeOfDay", scene.TimeOfDay),
		zap.String("mood", scene.Mood))

	prompt := scene.Description

	if scene.Location != "" {
		prompt += fmt.Sprintf(", 地点: %s", scene.Location)
	}

	if scene.TimeOfDay != "" {
		prompt += fmt.Sprintf(", 时间: %s", scene.TimeOfDay)
	}

	if scene.Mood != "" {
		prompt += fmt.Sprintf(", 氛围: %s", scene.Mood)
	}

	s.logger.Debug("scene image prompt built",
		zap.String("sceneId", scene.ID),
		zap.Int("promptLength", len(prompt)))

	return prompt
}

// buildStoryboardSceneImagePromptWithStyle 构建故事板场景图片提示词（包含故事风格配置）
func (s *Service) buildStoryboardSceneImagePromptWithStyle(scene *domain.StoryboardScene, storyStyle *domain.StyleConfig, isTransitionScene bool) string {
	s.logger.Debug("building scene image prompt with style",
		zap.String("sceneId", scene.ID),
		zap.String("sceneTitle", scene.Title),
		zap.Bool("hasStoryStyle", storyStyle != nil),
		zap.Bool("isTransitionScene", isTransitionScene))

	var promptBuilder strings.Builder

	// 添加故事风格配置
	if storyStyle != nil {
		promptBuilder.WriteString(fmt.Sprintf("[Art Style: %s] ", storyStyle.Style))
		if storyStyle.Description != "" {
			promptBuilder.WriteString(fmt.Sprintf("[Style Guide: %s] ", storyStyle.Description))
		}
	}

	// 添加场景类型信息
	if isTransitionScene {
		promptBuilder.WriteString("[Scene Type: Transition/Environment Scene - No characters] ")
	} else if len(scene.Characters) > 0 {
		maxMainCharacters := 5
		mainCharacters := scene.Characters
		if len(mainCharacters) > maxMainCharacters {
			mainCharacters = mainCharacters[:maxMainCharacters]
		}

		promptBuilder.WriteString(fmt.Sprintf("[Main Characters (limit to maximum 5, must be accurately depicted): %s] ", strings.Join(mainCharacters, ", ")))

		// 如果有超过5个角色，说明还有其他角色（群众、路人等）
		if len(scene.Characters) > maxMainCharacters {
			promptBuilder.WriteString(fmt.Sprintf("[Note: There are %d total characters mentioned. The above are the MAIN CHARACTERS (maximum 5). You may include additional background characters, crowds, bystanders, or passersby as needed. These additional characters do not need to match specific reference images.] ", len(scene.Characters)))
		}
	}

	// 添加场景描述
	promptBuilder.WriteString(scene.Description)

	if scene.Location != "" {
		promptBuilder.WriteString(fmt.Sprintf(", 地点: %s", scene.Location))
	}

	if scene.TimeOfDay != "" {
		promptBuilder.WriteString(fmt.Sprintf(", 时间: %s", scene.TimeOfDay))
	}

	if scene.Mood != "" {
		promptBuilder.WriteString(fmt.Sprintf(", 氛围: %s", scene.Mood))
	}

	prompt := promptBuilder.String()
	s.logger.Debug("scene image prompt with style built",
		zap.String("sceneId", scene.ID),
		zap.Int("promptLength", len(prompt)))

	return prompt
}
