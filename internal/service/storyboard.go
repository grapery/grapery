package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/cache"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

// CreateStoryboard 创建新的 storyboard
func (s *Service) CreateStoryboard(ctx context.Context, storyboard *domain.Storyboard) error {
	s.logger.Info("creating storyboard",
		zap.String("storyId", storyboard.StoryID),
		zap.String("creatorId", storyboard.CreatorID),
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

	// 如果没有父节点或父节点为空，设置为 root marker
	if storyboard.ParentID == "" {
		storyboard.ParentID = domain.StoryboardRootMarker
		s.logger.Debug("setting parentId to root marker",
			zap.String("storyboardId", storyboard.ID))
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

	// 创建 storyboard
	s.logger.Debug("saving storyboard to database",
		zap.String("storyboardId", storyboard.ID),
		zap.String("storyId", storyboard.StoryID))
	if err := s.repo.CreateStoryboard(ctx, storyboard); err != nil {
		s.logger.Error("failed to create storyboard in database",
			zap.String("storyboardId", storyboard.ID),
			zap.String("storyId", storyboard.StoryID),
			zap.Error(err))
		return fmt.Errorf("failed to create storyboard: %w", err)
	}

	// 缓存新创建的 storyboard
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

	// Persist AI-generated storyboard scenes (if any)
	if len(storyboard.StoryboardScenes) > 0 {
		s.logger.Debug("persisting AI-generated storyboard scenes",
			zap.String("storyboardId", storyboard.ID),
			zap.Int("sceneCount", len(storyboard.StoryboardScenes)))
		if err := s.persistStoryboardScenes(ctx, storyboard); err != nil {
			s.logger.Warn("failed to persist storyboard scenes",
				zap.String("storyboardId", storyboard.ID),
				zap.Error(err))
			// Don't fail the entire creation for this
		}
	}

	s.logger.Info("storyboard created successfully",
		zap.String("id", storyboard.ID),
		zap.String("storyId", storyboard.StoryID),
		zap.String("creatorId", storyboard.CreatorID),
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
	if story.AuthorID != storyboard.CreatorID {
		// 获取创建者信息
		creator, err := s.repo.UserByID(ctx, storyboard.CreatorID)
		if err == nil {
			if err := s.NotifyStoryboardCreated(ctx,
				story.AuthorID,
				storyboard.CreatorID,
				creator.DisplayName,
				creator.Avatar,
				storyboard.StoryID,
				storyboard.ID); err != nil {
				s.logger.Warn("failed to send storyboard created notification",
					zap.Error(err),
					zap.String("storyboardId", storyboard.ID))
			} else {
				s.logger.Info("storyboard created notification sent",
					zap.String("recipientId", story.AuthorID),
					zap.String("storyboardId", storyboard.ID))
			}
		}
	}

	// 如果是 fork（不是 root），通知父节点作者
	if storyboard.ParentID != "" && storyboard.ParentID != domain.StoryboardRootMarker {
		parent, err := s.repo.StoryboardByID(ctx, storyboard.ParentID)
		if err == nil && parent.CreatorID != storyboard.CreatorID {
			// 获取创建者信息
			creator, err := s.repo.UserByID(ctx, storyboard.CreatorID)
			if err == nil {
				if err := s.NotifyStoryboardForked(ctx,
					parent.CreatorID,
					storyboard.CreatorID,
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
						zap.String("recipientId", parent.CreatorID),
						zap.String("storyboardId", storyboard.ID))
				}
			}
		}
	}

	// 如果故事属于群组，记录群组活动
	if story.GroupID != "" {
		s.logger.Debug("recording group activity for storyboard creation",
			zap.String("groupId", story.GroupID),
			zap.String("storyboardId", storyboard.ID))
		go s.RecordGroupStoryboardCreated(context.Background(), story.GroupID, storyboard.CreatorID, story.ID, story.Title)
	}

	// 记录用户活动
	s.logger.Debug("recording user activity for storyboard creation",
		zap.String("userId", storyboard.CreatorID),
		zap.String("storyboardId", storyboard.ID))
	go s.RecordStoryboardCreated(context.Background(), storyboard.CreatorID, storyboard.ID, storyboard.Title)

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
	s.logger.Info("GetStoryboard called",
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
		s.logger.Info("GetStoryboard: loaded scenes from DB",
			zap.String("storyboardId", id),
			zap.Int("sceneCount", len(storyboardScenes)))
	}

	// Populate missing scene images from image generation records
	s.populateMissingSceneImages(ctx, storyboard)

	// Populate missing scene videos from video generation records
	s.populateMissingSceneVideos(ctx, storyboard)

	// Log final scene state
	for i, scene := range storyboard.StoryboardScenes {
		s.logger.Info("GetStoryboard: final scene state",
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

	// Build a map of sceneID -> generated video URL
	sceneVideoMap := make(map[string]string)
	for _, gen := range videoGens {
		s.logger.Debug("populateMissingSceneVideos: checking video generation",
			zap.String("sceneId", gen.SceneID),
			zap.String("status", string(gen.Status)),
			zap.String("videoURL", gen.GeneratedVideoURL))
		if gen.Status == domain.GenerationStatusCompleted && gen.GeneratedVideoURL != "" {
			sceneVideoMap[gen.SceneID] = gen.GeneratedVideoURL
		}
	}

	s.logger.Info("populateMissingSceneVideos: built video URL map",
		zap.String("storyboardId", storyboard.ID),
		zap.Int("scenesWithVideo", len(sceneVideoMap)))

	// Fill in missing scene videos
	for i := range storyboard.StoryboardScenes {
		scene := &storyboard.StoryboardScenes[i]
		if scene.VideoUrl == "" {
			if videoURL, ok := sceneVideoMap[scene.ID]; ok {
				scene.VideoUrl = videoURL
				s.logger.Info("populated missing scene video from generation record",
					zap.String("sceneId", scene.ID),
					zap.String("videoURL", videoURL))
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

	if existing.CreatorID != userID {
		s.logger.Warn("permission denied: user is not the creator",
			zap.String("storyboardId", storyboard.ID),
			zap.String("userId", userID),
			zap.String("creatorId", existing.CreatorID))
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
		for i := range storyboard.CharacterRefs {
			ref := &storyboard.CharacterRefs[i]
			if ref.CharacterID == "" {
				s.logger.Error("character reference missing characterId",
					zap.String("storyboardId", storyboard.ID),
					zap.Int("refIndex", i))
				return fmt.Errorf("character reference requires characterId")
			}
			character, err := s.repo.CharacterByID(ctx, ref.CharacterID)
			if err != nil {
				s.logger.Error("character not found for reference",
					zap.String("storyboardId", storyboard.ID),
					zap.String("characterId", ref.CharacterID),
					zap.Int("refIndex", i),
					zap.Error(err))
				return fmt.Errorf("character %s not found", ref.CharacterID)
			}
			if ref.Order == 0 {
				ref.Order = i
			}
			s.logger.Debug("character reference validated",
				zap.String("storyboardId", storyboard.ID),
				zap.String("characterId", ref.CharacterID),
				zap.String("characterName", character.Name),
				zap.Int("order", ref.Order))
		}
	}

	// Validate scene refs
	if storyboard.SceneRefs != nil {
		s.logger.Debug("validating scene references",
			zap.String("storyboardId", storyboard.ID),
			zap.String("storyId", storyboard.StoryID),
			zap.Int("sceneRefCount", len(storyboard.SceneRefs)))
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
				s.logger.Error("story scene not found for reference",
					zap.String("storyboardId", storyboard.ID),
					zap.String("storyId", storyboard.StoryID),
					zap.String("storySceneId", ref.StorySceneID),
					zap.Int("refIndex", i),
					zap.Error(err))
				return fmt.Errorf("story scene %s not found or not part of story", ref.StorySceneID)
			}
			if ref.Sequence == 0 {
				ref.Sequence = i
			}
			s.logger.Debug("scene reference validated",
				zap.String("storyboardId", storyboard.ID),
				zap.String("storySceneId", ref.StorySceneID),
				zap.String("sceneTitle", scene.Title),
				zap.Int("sequence", ref.Sequence))
		}
	}
	s.logger.Debug("storyboard assets validation completed",
		zap.String("storyboardId", storyboard.ID))
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

	if storyboard.CreatorID != userID {
		s.logger.Warn("permission denied: user is not the creator",
			zap.String("storyboardId", id),
			zap.String("userId", userID),
			zap.String("creatorId", storyboard.CreatorID))
		return fmt.Errorf("permission denied: not the creator")
	}

	s.logger.Debug("storyboard deletion authorized",
		zap.String("storyboardId", id),
		zap.String("storyId", storyboard.StoryID),
		zap.String("title", storyboard.Title))

	// 删除
	if err := s.repo.DeleteStoryboard(ctx, id); err != nil {
		s.logger.Error("failed to delete storyboard from database",
			zap.String("storyboardId", id),
			zap.Error(err))
		return fmt.Errorf("failed to delete storyboard: %w", err)
	}

	// 更新故事的故事板数量
	if err := s.repo.DecrementStoryStoryboardCount(ctx, storyboard.StoryID); err != nil {
		s.logger.Warn("failed to decrement story storyboard count",
			zap.String("storyId", storyboard.StoryID),
			zap.Error(err))
		// 不返回错误，因为故事板已经删除成功
	} else {
		s.logger.Debug("story storyboard count decremented",
			zap.String("storyId", storyboard.StoryID))
	}

	s.logger.Info("storyboard deleted successfully",
		zap.String("id", id),
		zap.String("userId", userID),
		zap.String("storyId", storyboard.StoryID))

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
	newStoryboard.CreatorID = userID

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
	if parent.CreatorID != userID {
		// 获取 fork 者信息
		forker, err := s.repo.UserByID(ctx, userID)
		if err == nil {
			if err := s.NotifyStoryboardForked(ctx,
				parent.CreatorID,
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
					zap.String("recipientId", parent.CreatorID),
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
		zap.String("creatorId", storyboard.CreatorID))

	// 创建通知给 storyboard 创建者（如果点赞者不是创建者本人）
	if storyboard.CreatorID != userID {
		// 获取点赞者信息
		liker, err := s.repo.UserByID(ctx, userID)
		if err == nil {
			if err := s.NotifyLike(ctx,
				storyboard.CreatorID,
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
					zap.String("recipientId", storyboard.CreatorID),
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
		zap.String("creatorId", storyboard.CreatorID),
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
	systemPrompt := `你是专业的故事创作助手，擅长创建生动的故事分镜。
重要要求：
1. 直接返回纯JSON，不要使用markdown代码块（不要用` + "```json" + `或` + "```" + `包裹）
2. 确保JSON完整闭合，所有括号和引号配对正确
3. 内容简洁精炼，每个场景描述控制在200字以内`

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
		UserID:            storyboard.CreatorID,
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

	// 5. 更新业务数据：storyboard 内容
	storyboard.Content = storyboardResult.Content
	storyboard.IsAIGenerated = true
	s.logger.Debug("storyboard content updated from AI generation",
		zap.String("storyboardId", storyboard.ID),
		zap.Int("contentLength", len(storyboard.Content)),
		zap.Int("scenesCount", len(storyboardResult.Scenes)))

	// 将 map 转换为 StoryboardScene 对象 (AI-generated plot scenes)
	storyboardScenes := make([]domain.StoryboardScene, 0, len(storyboardResult.Scenes))
	for i, sceneMap := range storyboardResult.Scenes {
		scene := domain.StoryboardScene{
			Sequence:      i,
			IsAIGenerated: true,
		}
		if title, ok := sceneMap["title"].(string); ok {
			scene.Title = title
		}
		if desc, ok := sceneMap["description"].(string); ok {
			scene.Description = desc
		}
		if location, ok := sceneMap["location"].(string); ok {
			scene.Location = location
		}
		if timeOfDay, ok := sceneMap["timeOfDay"].(string); ok {
			scene.TimeOfDay = timeOfDay
		}
		if mood, ok := sceneMap["mood"].(string); ok {
			scene.Mood = mood
		}
		if chars, ok := sceneMap["characters"].([]interface{}); ok {
			for _, c := range chars {
				if charName, ok := c.(string); ok {
					scene.Characters = append(scene.Characters, charName)
				}
			}
		}
		// Image 将在后续生成
		scene.Image = ""
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

	// 添加选定的角色信息
	if len(storyboard.CharacterRefs) > 0 {
		s.logger.Debug("adding character references to context",
			zap.String("storyboardId", storyboard.ID),
			zap.Int("characterCount", len(storyboard.CharacterRefs)))
		context += "参与角色：\n"
		for _, ref := range storyboard.CharacterRefs {
			if ref.Character != nil {
				context += fmt.Sprintf("- %s", ref.Character.Name)
				if ref.Character.Description != "" {
					context += fmt.Sprintf(": %s", truncateForLog(ref.Character.Description, 100))
				}
				context += "\n"
			}
		}
		context += "\n"
	}

	// 添加选定的故事场景（静态地点）作为可用场景
	if len(storyboard.SceneRefs) > 0 {
		s.logger.Debug("adding scene references to context",
			zap.String("storyboardId", storyboard.ID),
			zap.Int("sceneCount", len(storyboard.SceneRefs)))
		context += "可用场景地点（剧情应发生在这些场景中）：\n"
		for _, ref := range storyboard.SceneRefs {
			if ref.StoryScene != nil {
				context += fmt.Sprintf("- %s", ref.StoryScene.Title)
				if ref.StoryScene.Description != "" {
					context += fmt.Sprintf(": %s", truncateForLog(ref.StoryScene.Description, 100))
				}
				if ref.StoryScene.Location != "" {
					context += fmt.Sprintf(" (地点: %s)", ref.StoryScene.Location)
				}
				context += "\n"
			}
		}
		context += "\n"
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

	prompt := "作为专业的故事创作者，请根据以下信息生成精彩的故事分镜内容：\n\n"

	if contextInfo != "" {
		prompt += contextInfo
	}

	prompt += fmt.Sprintf("用户输入: %s\n\n", storyboard.RawInput)

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
	prompt += "  \"content\": \"润色后的故事内容（控制在500字以内）\",\n"
	prompt += "  \"scenes\": [\n"
	prompt += "    {\n"
	prompt += "      \"title\": \"场景标题（10字以内）\",\n"
	prompt += "      \"description\": \"场景描述（100-200字）\",\n"
	prompt += "      \"location\": \"地点\",\n"
	prompt += "      \"timeOfDay\": \"时间\",\n"
	prompt += "      \"characters\": [\"角色名\"],\n"
	prompt += "      \"mood\": \"氛围\"\n"
	prompt += "    }\n"
	prompt += "  ],\n"
	prompt += "  \"generateImages\": false\n"
	prompt += "}\n"
	prompt += fmt.Sprintf("\n重要：请生成恰好 %d 个场景，确保JSON格式完整闭合。", sceneCount)

	s.logger.Debug("storyboard prompt built",
		zap.String("storyboardId", storyboard.ID),
		zap.Int("promptLength", len(prompt)),
		zap.Int("sceneCount", sceneCount))

	return prompt
}

// StoryboardResult AI 生成的 storyboard 结果
type StoryboardResult struct {
	Content        string                   `json:"content"`
	Scenes         []map[string]interface{} `json:"scenes"`
	GenerateImages bool                     `json:"generateImages"`
}

// parseStoryboardResult 解析 AI 生成的结果
func (s *Service) parseStoryboardResult(text string) *StoryboardResult {
	var result StoryboardResult

	// Log raw AI response for debugging
	s.logger.Debug("AI raw response for storyboard",
		zap.Int("rawLength", len(text)),
		zap.String("rawText", truncateForLog(text, 2000)),
	)

	// Strip markdown code blocks if present (```json ... ```)
	cleanedText := strings.TrimSpace(text)
	hadMarkdown := false
	if strings.HasPrefix(cleanedText, "```") {
		hadMarkdown = true
		// Find the end of the first line (after ```json or ```)
		if idx := strings.Index(cleanedText, "\n"); idx != -1 {
			cleanedText = cleanedText[idx+1:]
		}
		// Remove trailing ```
		if idx := strings.LastIndex(cleanedText, "```"); idx != -1 {
			cleanedText = strings.TrimSpace(cleanedText[:idx])
		}
	}

	s.logger.Debug("AI response after cleaning",
		zap.Bool("hadMarkdownBlock", hadMarkdown),
		zap.Int("cleanedLength", len(cleanedText)),
		zap.String("cleanedText", truncateForLog(cleanedText, 2000)),
	)

	// 尝试解析 JSON
	if err := json.Unmarshal([]byte(cleanedText), &result); err != nil {
		// 如果不是 JSON，使用原始文本
		s.logger.Warn("failed to parse JSON, using raw text",
			zap.Error(err),
			zap.Int("rawLength", len(text)),
			zap.Int("cleanedLength", len(cleanedText)),
			zap.String("rawTextPreview", truncateForLog(text, 500)),
			zap.String("cleanedTextPreview", truncateForLog(cleanedText, 500)),
		)
		result.Content = text
		result.Scenes = []map[string]interface{}{}
	} else {
		s.logger.Info("AI response JSON parsed successfully",
			zap.Int("contentLength", len(result.Content)),
			zap.Int("scenesCount", len(result.Scenes)),
		)
	}

	return &result
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

	s.logger.Info("generating images for storyboard scenes",
		zap.String("storyboardId", storyboard.ID),
		zap.Int("totalScenes", len(storyboard.StoryboardScenes)))

	// 为每个故事板场景生成图片
	for i := range storyboard.StoryboardScenes {
		scene := &storyboard.StoryboardScenes[i]

		s.logger.Debug("generating image for scene",
			zap.String("storyboardId", storyboard.ID),
			zap.Int("sceneIndex", i),
			zap.String("sceneId", scene.ID),
			zap.String("sceneTitle", scene.Title))

		// 构建图片提示词
		prompt := s.buildStoryboardSceneImagePrompt(scene)
		s.logger.Debug("scene image prompt built",
			zap.String("storyboardId", storyboard.ID),
			zap.Int("sceneIndex", i),
			zap.Int("promptLength", len(prompt)))

		// 使用AI生成服务生成图片（自动记录AI使用数据）
		imageReq := &GenerateImageRequest{
			UserID:            storyboard.CreatorID,
			Prompt:            prompt,
			Provider:          "gemini",
			Model:             "imagen-3.0-generate-001",
			AspectRatio:       "16:9",
			Quality:           "high",
			OutputCount:       1,
			RelatedEntityID:   storyboard.ID,
			RelatedEntityType: "storyboard_scene",
			Metadata: map[string]interface{}{
				"storyId":    storyboard.StoryID,
				"sceneIndex": i,
				"sceneTitle": scene.Title,
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
