package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

// CreateStoryboard 创建新的 storyboard
func (s *Service) CreateStoryboard(ctx context.Context, storyboard *domain.Storyboard) error {
	// 验证故事存在
	story, err := s.repo.StoryByID(ctx, storyboard.StoryID)
	if err != nil {
		return fmt.Errorf("story not found: %w", err)
	}

	// 如果没有父节点或父节点为空，设置为 root marker
	if storyboard.ParentID == "" {
		storyboard.ParentID = domain.StoryboardRootMarker
	}

	// 如果有父节点（不是 root），验证父节点存在
	if storyboard.ParentID != domain.StoryboardRootMarker {
		parent, err := s.repo.StoryboardByID(ctx, storyboard.ParentID)
		if err != nil {
			return fmt.Errorf("parent storyboard not found: %w", err)
		}
		// 确保父节点属于同一个故事
		if parent.StoryID != storyboard.StoryID {
			return fmt.Errorf("parent storyboard belongs to different story")
		}
	}

	// 使用 AI 生成内容（如果提供了 geminiClient）
	if s.geminiClient != nil && storyboard.RawInput != "" {
		if err := s.GenerateStoryboardWithAI(ctx, storyboard); err != nil {
			s.logger.Warn("AI generation failed, creating storyboard without AI content",
				zap.Error(err))
			// AI 生成失败不影响创建流程，继续使用原始输入
		}
	}

	// Validate linked assets
	if err := s.validateStoryboardAssets(ctx, storyboard); err != nil {
		return err
	}

	// 创建 storyboard
	if err := s.repo.CreateStoryboard(ctx, storyboard); err != nil {
		return fmt.Errorf("failed to create storyboard: %w", err)
	}

	// Persist AI-generated storyboard scenes (if any)
	if err := s.persistStoryboardScenes(ctx, storyboard); err != nil {
		s.logger.Warn("failed to persist storyboard scenes",
			zap.String("storyboardId", storyboard.ID),
			zap.Error(err))
		// Don't fail the entire creation for this
	}

	s.logger.Info("storyboard created",
		zap.String("id", storyboard.ID),
		zap.String("storyId", storyboard.StoryID),
		zap.String("creatorId", storyboard.CreatorID),
	)

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
		go s.RecordGroupStoryboardCreated(context.Background(), story.GroupID, storyboard.CreatorID, story.ID, story.Title)
	}

	// 记录用户活动
	go s.RecordStoryboardCreated(context.Background(), storyboard.CreatorID, storyboard.ID, storyboard.Title)

	// 更新故事的 panels 计数
	_ = story // 暂时不更新

	return nil
}

// GetStoryboard 获取 storyboard 详情
func (s *Service) GetStoryboard(ctx context.Context, id string) (*domain.Storyboard, error) {
	s.logger.Info("GetStoryboard called", zap.String("storyboardId", id))
	
	storyboard, err := s.repo.StoryboardByID(ctx, id)
	if err != nil {
		s.logger.Error("GetStoryboard: StoryboardByID failed", zap.String("storyboardId", id), zap.Error(err))
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
	// 验证权限
	existing, err := s.repo.StoryboardByID(ctx, storyboard.ID)
	if err != nil {
		return err
	}

	if existing.CreatorID != userID {
		return fmt.Errorf("permission denied: not the creator")
	}

	// Validate linked assets (SceneRefs are references to StoryScenes, not StoryboardScenes)
	if storyboard.CharacterRefs != nil || storyboard.SceneRefs != nil {
		if err := s.validateStoryboardAssets(ctx, storyboard); err != nil {
			return err
		}
	}

	if err := s.repo.UpdateStoryboard(ctx, storyboard); err != nil {
		return fmt.Errorf("failed to update storyboard: %w", err)
	}

	s.logger.Info("storyboard updated",
		zap.String("id", storyboard.ID),
		zap.String("userId", userID),
	)

	return nil
}

// persistStoryboardScenes saves AI-generated storyboard scenes to the database
func (s *Service) persistStoryboardScenes(ctx context.Context, storyboard *domain.Storyboard) error {
	if len(storyboard.StoryboardScenes) == 0 {
		return nil
	}

	// Convert to pointer slice for repository
	scenes := make([]*domain.StoryboardScene, len(storyboard.StoryboardScenes))
	for i := range storyboard.StoryboardScenes {
		scenes[i] = &storyboard.StoryboardScenes[i]
	}

	if err := s.repo.CreateStoryboardScenes(ctx, storyboard.ID, scenes); err != nil {
		return fmt.Errorf("failed to persist storyboard scenes: %w", err)
	}

	s.logger.Info("storyboard scenes persisted",
		zap.String("storyboardId", storyboard.ID),
		zap.Int("count", len(scenes)))

	return nil
}

func (s *Service) validateStoryboardAssets(ctx context.Context, storyboard *domain.Storyboard) error {
	// Validate character refs
	if storyboard.CharacterRefs != nil {
		for i := range storyboard.CharacterRefs {
			ref := &storyboard.CharacterRefs[i]
			if ref.CharacterID == "" {
				return fmt.Errorf("character reference requires characterId")
			}
			_, err := s.repo.CharacterByID(ctx, ref.CharacterID)
			if err != nil {
				return fmt.Errorf("character %s not found", ref.CharacterID)
			}
			if ref.Order == 0 {
				ref.Order = i
			}
		}
	}

	// Validate scene refs
	if storyboard.SceneRefs != nil {
		for i := range storyboard.SceneRefs {
			ref := &storyboard.SceneRefs[i]
			if ref.StorySceneID == "" {
				return fmt.Errorf("scene reference requires storySceneId")
			}
			if _, err := s.repo.StorySceneByID(ctx, storyboard.StoryID, ref.StorySceneID); err != nil {
				return fmt.Errorf("story scene %s not found or not part of story", ref.StorySceneID)
			}
			if ref.Sequence == 0 {
				ref.Sequence = i
			}
		}
	}
	return nil
}

// DeleteStoryboard 删除 storyboard
func (s *Service) DeleteStoryboard(ctx context.Context, id, userID string) error {
	// 验证权限
	storyboard, err := s.repo.StoryboardByID(ctx, id)
	if err != nil {
		return err
	}

	if storyboard.CreatorID != userID {
		return fmt.Errorf("permission denied: not the creator")
	}

	// 删除
	if err := s.repo.DeleteStoryboard(ctx, id); err != nil {
		return fmt.Errorf("failed to delete storyboard: %w", err)
	}

	s.logger.Info("storyboard deleted",
		zap.String("id", id),
		zap.String("userId", userID),
	)

	return nil
}

// ListStoryboards 获取故事的 storyboards 列表
func (s *Service) ListStoryboards(ctx context.Context, storyID string, limit, offset int) ([]*domain.Storyboard, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	storyboards, err := s.repo.StoryboardsByStory(ctx, storyID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list storyboards: %w", err)
	}

	return storyboards, nil
}

// ListRootStoryboards 获取故事的根 storyboards（ParentID 为空或 "__root__"）
func (s *Service) ListRootStoryboards(ctx context.Context, storyID string, limit, offset int) ([]*domain.Storyboard, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	storyboards, err := s.repo.RootStoryboardsByStory(ctx, storyID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list root storyboards: %w", err)
	}

	return storyboards, nil
}

// ListStoryboardsByParent 获取指定父级的 storyboards
func (s *Service) ListStoryboardsByParent(ctx context.Context, storyID, parentID string, limit, offset int) ([]*domain.Storyboard, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	storyboards, err := s.repo.StoryboardsByParent(ctx, storyID, parentID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list storyboards by parent: %w", err)
	}

	return storyboards, nil
}

// GetStoryboardChildren 获取子 storyboards (forks/continuations)
func (s *Service) GetStoryboardChildren(ctx context.Context, parentID string) ([]*domain.Storyboard, error) {
	children, err := s.repo.StoryboardChildren(ctx, parentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get children: %w", err)
	}

	return children, nil
}

// GetStoryboardTree 获取完整的 storyboard 树
func (s *Service) GetStoryboardTree(ctx context.Context, rootID string) ([]*domain.Storyboard, error) {
	tree, err := s.repo.StoryboardTree(ctx, rootID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tree: %w", err)
	}

	return tree, nil
}

// GetStoryboardFeed 获取社区故事板 feed 流（按时间倒序）
func (s *Service) GetStoryboardFeed(ctx context.Context, limit, offset int) ([]*domain.Storyboard, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	storyboards, total, err := s.repo.StoryboardFeed(ctx, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get storyboard feed: %w", err)
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
	// 验证父节点存在
	parent, err := s.repo.StoryboardByID(ctx, parentID)
	if err != nil {
		return fmt.Errorf("parent storyboard not found: %w", err)
	}

	// 设置基本信息
	newStoryboard.StoryID = parent.StoryID
	newStoryboard.ParentID = parentID
	newStoryboard.CreatorID = userID

	// 如果用户提供了新的 rawInput，调用 AI 生成新内容
	if s.geminiClient != nil && newStoryboard.RawInput != "" {
		if err := s.GenerateStoryboardWithAI(ctx, newStoryboard); err != nil {
			s.logger.Warn("AI generation failed for fork",
				zap.Error(err))
		}
	} else {
		// 否则，复制父节点的内容
		newStoryboard.Content = parent.Content
	}

	if len(newStoryboard.CharacterRefs) == 0 && len(parent.CharacterRefs) > 0 {
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
	if err := s.repo.ForkStoryboard(ctx, parentID, userID, newStoryboard); err != nil {
		return fmt.Errorf("failed to fork storyboard: %w", err)
	}

	s.logger.Info("storyboard forked",
		zap.String("parentId", parentID),
		zap.String("newId", newStoryboard.ID),
		zap.String("userId", userID),
	)

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
	if err := s.repo.UnlikeStoryboard(ctx, userID, storyboardID); err != nil {
		return fmt.Errorf("failed to unlike storyboard: %w", err)
	}

	s.logger.Info("storyboard unliked",
		zap.String("userId", userID),
		zap.String("storyboardId", storyboardID),
	)

	return nil
}

// GenerateStoryboardWithAI 使用 AI 生成 storyboard 内容
// 业务数据：storyboard的内容和scenes
// AI任务数据：通过AIGenerationService记录
func (s *Service) GenerateStoryboardWithAI(ctx context.Context, storyboard *domain.Storyboard) error {
	// 1. 获取故事背景信息
	story, err := s.repo.StoryByID(ctx, storyboard.StoryID)
	if err != nil {
		return fmt.Errorf("failed to get story: %w", err)
	}

	// 2. 构建上下文和提示词
	contextInfo := s.buildStoryboardContext(ctx, storyboard, story)
	prompt := s.buildStoryboardPrompt(storyboard, story, contextInfo)
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
	storyboardResult := s.parseStoryboardResult(result.Text)

	// 5. 更新业务数据：storyboard 内容
	storyboard.Content = storyboardResult.Content
	storyboard.IsAIGenerated = true

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
		if err := s.generateSceneImages(ctx, storyboard); err != nil {
			s.logger.Warn("failed to generate scene images", zap.Error(err))
			// 图片生成失败不影响整体流程
		}
	}

	return nil
}

// getAncestorStoryboards fetches up to maxLevels ancestor storyboards in chronological order (oldest first)
func (s *Service) getAncestorStoryboards(ctx context.Context, storyboard *domain.Storyboard, maxLevels int) []*domain.Storyboard {
	var ancestors []*domain.Storyboard
	currentParentID := storyboard.ParentID

	for i := 0; i < maxLevels; i++ {
		if currentParentID == "" || currentParentID == domain.StoryboardRootMarker {
			break
		}

		parent, err := s.repo.StoryboardByID(ctx, currentParentID)
		if err != nil {
			s.logger.Warn("failed to fetch ancestor storyboard",
				zap.String("parentId", currentParentID),
				zap.Error(err))
			break
		}

		// Prepend to get chronological order (oldest first)
		ancestors = append([]*domain.Storyboard{parent}, ancestors...)
		currentParentID = parent.ParentID
	}

	return ancestors
}

// buildStoryboardContext 构建 storyboard 生成上下文
func (s *Service) buildStoryboardContext(ctx context.Context, storyboard *domain.Storyboard, story *domain.Story) string {
	var context string

	// 添加故事信息
	context += fmt.Sprintf("故事标题: %s\n", story.Title)
	context += fmt.Sprintf("故事简介: %s\n\n", story.Description)

	// 如果是独立故事板，不添加父节点内容
	if storyboard.IsStandalone {
		context += "（独立故事线，不参考前情）\n\n"
	} else if storyboard.ParentID != "" && storyboard.ParentID != domain.StoryboardRootMarker {
		// 获取最多5级祖先故事板作为上下文
		ancestors := s.getAncestorStoryboards(ctx, storyboard, 5)
		if len(ancestors) > 0 {
			context += "前情提要（按时间顺序）：\n"
			for i, ancestor := range ancestors {
				// 限制每个祖先内容长度，避免上下文过长
				ancestorContent := truncateForLog(ancestor.Content, 300)
				context += fmt.Sprintf("\n【第%d章 - %s】\n%s\n", i+1, ancestor.Title, ancestorContent)
			}
			context += "\n"
		}
	}

	// 添加选定的角色信息
	if len(storyboard.CharacterRefs) > 0 {
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

	return context
}

// buildStoryboardPrompt 构建 storyboard 生成提示词
func (s *Service) buildStoryboardPrompt(storyboard *domain.Storyboard, story *domain.Story, contextInfo string) string {
	prompt := "作为专业的故事创作者，请根据以下信息生成精彩的故事分镜内容：\n\n"

	if contextInfo != "" {
		prompt += contextInfo
	}

	prompt += fmt.Sprintf("用户输入: %s\n\n", storyboard.RawInput)

	// Determine scene count (default to 3 if not set or out of range)
	sceneCount := storyboard.SceneCount
	if sceneCount < 2 || sceneCount > 5 {
		sceneCount = 3
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
		return nil
	}

	// 为每个故事板场景生成图片
	for i := range storyboard.StoryboardScenes {
		scene := &storyboard.StoryboardScenes[i]

		// 构建图片提示词
		prompt := s.buildStoryboardSceneImagePrompt(scene)

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
				zap.Int("sceneIndex", i),
				zap.String("aiRecordId", ""),
				zap.Error(err))
			continue
		}

		if len(result.ImageURLs) > 0 {
			scene.Image = result.ImageURLs[0]
			s.logger.Info("scene image generated",
				zap.Int("sceneIndex", i),
				zap.String("aiRecordId", result.RecordID),
				zap.String("imageURL", scene.Image))
		}
	}

	return nil
}

// buildStoryboardSceneImagePrompt 构建故事板场景图片提示词
func (s *Service) buildStoryboardSceneImagePrompt(scene *domain.StoryboardScene) string {
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

	return prompt
}
