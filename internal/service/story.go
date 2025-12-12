package service

// 故事渲染和发布相关功能扩展

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

// CreateStoryRequest 创建故事请求
type CreateStoryRequest struct {
	Title             string `json:"title" binding:"required,min=1,max=200"`
	Description       string `json:"description" binding:"max=2000"`
	CoverImage        string `json:"coverImage" binding:"omitempty,url"`
	Genre             string `json:"genre" binding:"required"`
	Status            string `json:"status" binding:"omitempty,oneof=draft published"`
	GroupID           string `json:"groupId" binding:"omitempty"`
	DefaultSceneCount int    `json:"defaultSceneCount"` // Default number of scenes for storyboards (2-8, default 3)

	// AI 丰富选项（可选）
	UseAIEnrich        bool   `json:"useAIEnrich"`                // 是否使用AI丰富故事描述
	GenerateCover      bool   `json:"generateCover"`              // 是否使用AI生成封面/海报
	GeneratePoster     bool   `json:"generatePoster"`             // 是否生成故事海报
	GenerateBackground bool   `json:"generateBackground"`         // 是否生成背景图片
	AIStyle            string `json:"aiStyle,omitempty"`          // AI 生成风格：realistic, anime, fantasy, etc.
	CoverAspectRatio   string `json:"coverAspectRatio,omitempty"` // 封面图比例：1:1, 16:9, 9:16, etc.
}

// StoryAIEnrichResponse AI丰富故事的响应
type StoryAIEnrichResponse struct {
	EnrichedDescription string   `json:"enrichedDescription"`     // 丰富后的描述
	SuggestedTags       []string `json:"suggestedTags,omitempty"` // 建议的标签
	TokensUsed          int      `json:"tokensUsed"`              // 消耗的token数
}

// StoryAICoverResponse AI生成封面的响应
type StoryAICoverResponse struct {
	CoverURL      string `json:"coverUrl,omitempty"`      // 生成的封面URL
	PosterURL     string `json:"posterUrl,omitempty"`     // 生成的海报URL
	BackgroundURL string `json:"backgroundUrl,omitempty"` // 生成的背景URL
	TokensUsed    int    `json:"tokensUsed"`              // 消耗的token数
}

// UpdateStoryRequest 更新故事请求
type UpdateStoryRequest struct {
	Title       *string `json:"title" binding:"omitempty,min=1,max=200"`
	Description *string `json:"description" binding:"omitempty,max=2000"`
	CoverImage  *string `json:"coverImage" binding:"omitempty,url"`
	Genre       *string `json:"genre" binding:"omitempty"`
	Status      *string `json:"status" binding:"omitempty,oneof=draft published rendering"`
}

// StoryListRequest 故事列表请求
type StoryListRequest struct {
	Status   string `form:"status" binding:"omitempty,oneof=draft published rendering"`
	Genre    string `form:"genre"`
	AuthorID string `form:"authorId"`
	GroupID  string `form:"groupId"`
	Search   string `form:"search"`
	Limit    int    `form:"limit" binding:"omitempty,min=1,max=100"`
	Offset   int    `form:"offset" binding:"omitempty,min=0"`
}

// CreateStory 创建故事
// 如果用户选择使用AI丰富，会自动丰富故事描述并生成封面/海报
func (s *Service) CreateStory(ctx context.Context, userID string, req CreateStoryRequest) (*domain.Story, error) {
	s.logger.Info("creating story",
		zap.String("userID", userID),
		zap.String("title", req.Title),
		zap.Bool("useAIEnrich", req.UseAIEnrich),
		zap.Bool("generateCover", req.GenerateCover),
	)

	// 获取作者信息
	author, err := s.repo.UserByID(ctx, userID)
	if err != nil {
		s.logger.Error("failed to get author", zap.Error(err))
		return nil, errors.New("author not found")
	}

	// 设置默认状态
	status := req.Status
	if status == "" {
		status = "draft"
	}

	now := time.Now().Unix()

	// Set default scene count (2-8, default 3)
	defaultSceneCount := req.DefaultSceneCount
	if defaultSceneCount < 2 || defaultSceneCount > 8 {
		defaultSceneCount = 3
	}

	// 创建故事基本信息
	story := &domain.Story{
		Title:               req.Title,
		Description:         req.Description,
		OriginalDescription: req.Description, // 保留原始描述
		CoverImage:          req.CoverImage,
		Author:              author,
		Genre:               req.Genre,
		Status:              status,
		GroupID:             req.GroupID,
		DefaultSceneCount:   defaultSceneCount,
		Likes:               0,
		Followers:           0,
		Panels:              0,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	// 保存故事到数据库（先创建，后续更新AI丰富的内容）
	if err := s.repo.CreateStory(ctx, story); err != nil {
		s.logger.Error("failed to create story", zap.Error(err))
		return nil, errors.New("failed to create story")
	}

	s.logger.Info("story created successfully", zap.String("storyID", story.ID))

	// 如果用户选择使用AI丰富描述
	if req.UseAIEnrich && req.Description != "" {
		enrichResp, err := s.EnrichStoryDescription(ctx, userID, story.ID, req.Description, req.Genre, req.AIStyle)
		if err != nil {
			s.logger.Warn("failed to enrich story description with AI, continuing with original",
				zap.String("storyID", story.ID),
				zap.Error(err))
		} else {
			story.EnrichedDescription = enrichResp.EnrichedDescription
			story.Description = enrichResp.EnrichedDescription // 使用丰富后的描述
			story.IsAIEnriched = true
			story.AIEnrichedAt = &now
			story.TextTokensUsed = enrichResp.TokensUsed
			story.TokensUsed += enrichResp.TokensUsed
		}
	}

	// 如果用户选择使用AI生成封面/海报/背景
	if req.GenerateCover || req.GeneratePoster || req.GenerateBackground {
		coverResp, err := s.GenerateStoryCover(ctx, userID, story.ID, GenerateStoryCoverRequest{
			Title:              req.Title,
			Description:        story.Description,
			Genre:              req.Genre,
			Style:              req.AIStyle,
			AspectRatio:        req.CoverAspectRatio,
			GenerateCover:      req.GenerateCover,
			GeneratePoster:     req.GeneratePoster,
			GenerateBackground: req.GenerateBackground,
		})
		if err != nil {
			s.logger.Warn("failed to generate story cover with AI, continuing without cover",
				zap.String("storyID", story.ID),
				zap.Error(err))
		} else {
			if coverResp.CoverURL != "" {
				story.CoverImage = coverResp.CoverURL
				story.CoverGeneratedByAI = true
			}
			if coverResp.PosterURL != "" {
				story.PosterImage = coverResp.PosterURL
			}
			if coverResp.BackgroundURL != "" {
				story.BackgroundImage = coverResp.BackgroundURL
			}
			story.ImageTokensUsed = coverResp.TokensUsed
			story.TokensUsed += coverResp.TokensUsed
		}
	}

	// 更新故事（如果有AI丰富的内容）
	if story.IsAIEnriched || story.CoverGeneratedByAI {
		story.UpdatedAt = time.Now().Unix()
		if err := s.repo.UpdateStory(ctx, story); err != nil {
			s.logger.Warn("failed to update story with AI enriched content",
				zap.String("storyID", story.ID),
				zap.Error(err))
		}

		// 更新用户的token使用量
		if story.TokensUsed > 0 {
			if err := s.updateUserTokenUsage(ctx, userID, story.TokensUsed); err != nil {
				s.logger.Warn("failed to update user token usage",
					zap.String("userID", userID),
					zap.Int("tokensUsed", story.TokensUsed),
					zap.Error(err))
			}
		}
	}

	// 如果故事属于群组，记录群组活动
	if req.GroupID != "" {
		go s.RecordGroupStoryCreated(context.Background(), req.GroupID, userID, story.ID, story.Title)
	}

	// 记录用户活动
	go s.RecordStoryCreated(context.Background(), userID, story.ID, story.Title)

	return story, nil
}

// GetStory 获取故事详情
func (s *Service) GetStory(ctx context.Context, storyID string) (*domain.Story, error) {
	s.logger.Info("getting story", zap.String("storyID", storyID))

	story, err := s.repo.StoryByID(ctx, storyID)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, errors.New("story not found")
		}
		s.logger.Error("failed to get story", zap.Error(err))
		return nil, errors.New("failed to get story")
	}

	// Fetch story characters
	characters, err := s.repo.CharactersByStory(ctx, storyID)
	if err != nil {
		s.logger.Warn("failed to fetch story characters", zap.Error(err), zap.String("storyID", storyID))
	} else {
		story.Characters = characters
	}

	// Fetch story scenes
	scenes, err := s.repo.StoryScenes(ctx, storyID, 100, 0)
	if err != nil {
		s.logger.Warn("failed to fetch story scenes", zap.Error(err), zap.String("storyID", storyID))
	} else {
		story.Scenes = scenes
	}

	// Fetch contributors
	contributors, err := s.repo.GetStoryContributors(ctx, storyID, 100, 0)
	if err != nil {
		s.logger.Warn("failed to fetch story contributors", zap.Error(err), zap.String("storyID", storyID))
	} else {
		// Populate flattened fields for client display
		for _, contributor := range contributors {
			if contributor.User != nil {
				if contributor.User.DisplayName != "" {
					contributor.Name = contributor.User.DisplayName
				} else {
					contributor.Name = contributor.User.Username
				}
				contributor.Avatar = contributor.User.Avatar
			}
			contributor.BadgeStyle = contributor.Role
		}
		story.Contributors = contributors
	}

	return story, nil
}

// ListStories 获取故事列表
func (s *Service) ListStories(ctx context.Context, req StoryListRequest) ([]*domain.Story, int64, error) {
	s.logger.Info("listing stories",
		zap.String("status", req.Status),
		zap.String("genre", req.Genre),
		zap.Int("limit", req.Limit),
	)

	// 设置默认分页参数
	if req.Limit == 0 {
		req.Limit = 20
	}

	filter := domain.StoryFilter{
		Status:   req.Status,
		Genre:    req.Genre,
		AuthorID: req.AuthorID,
		GroupID:  req.GroupID,
		Search:   req.Search,
		Limit:    req.Limit,
		Offset:   req.Offset,
	}

	stories, total, err := s.repo.ListStories(ctx, filter)
	if err != nil {
		s.logger.Error("failed to list stories", zap.Error(err))
		return nil, 0, errors.New("failed to list stories")
	}

	s.logger.Info("stories listed successfully", zap.Int64("total", total))
	return stories, total, nil
}

// UpdateStory 更新故事
func (s *Service) UpdateStory(ctx context.Context, userID, storyID string, req UpdateStoryRequest) (*domain.Story, error) {
	s.logger.Info("updating story",
		zap.String("userID", userID),
		zap.String("storyID", storyID),
	)

	// 获取故事
	story, err := s.repo.StoryByID(ctx, storyID)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, errors.New("story not found")
		}
		return nil, errors.New("failed to get story")
	}

	// 验证权限
	if story.Author.ID != userID {
		s.logger.Warn("unauthorized story update attempt",
			zap.String("userID", userID),
			zap.String("authorID", story.Author.ID),
		)
		return nil, errors.New("unauthorized")
	}

	// 更新字段
	if req.Title != nil {
		story.Title = *req.Title
	}
	if req.Description != nil {
		story.Description = *req.Description
	}
	if req.CoverImage != nil {
		story.CoverImage = *req.CoverImage
	}
	if req.Genre != nil {
		story.Genre = *req.Genre
	}
	if req.Status != nil {
		story.Status = *req.Status
	}

	story.UpdatedAt = time.Now().Unix()

	if err := s.repo.UpdateStory(ctx, story); err != nil {
		s.logger.Error("failed to update story", zap.Error(err))
		return nil, errors.New("failed to update story")
	}

	s.logger.Info("story updated successfully", zap.String("storyID", storyID))
	return story, nil
}

// DeleteStory 删除故事
func (s *Service) DeleteStory(ctx context.Context, userID, storyID string) error {
	s.logger.Info("deleting story",
		zap.String("userID", userID),
		zap.String("storyID", storyID),
	)

	// 获取故事
	story, err := s.repo.StoryByID(ctx, storyID)
	if err != nil {
		if err == domain.ErrNotFound {
			return errors.New("story not found")
		}
		return errors.New("failed to get story")
	}

	// 验证权限
	if story.Author.ID != userID {
		s.logger.Warn("unauthorized story delete attempt",
			zap.String("userID", userID),
			zap.String("authorID", story.Author.ID),
		)
		return errors.New("unauthorized")
	}

	if err := s.repo.DeleteStory(ctx, storyID); err != nil {
		s.logger.Error("failed to delete story", zap.Error(err))
		return errors.New("failed to delete story")
	}

	s.logger.Info("story deleted successfully", zap.String("storyID", storyID))
	return nil
}

// LikeStory 点赞故事
func (s *Service) LikeStory(ctx context.Context, userID, storyID string) error {
	s.logger.Info("liking story",
		zap.String("userID", userID),
		zap.String("storyID", storyID),
	)

	// 验证故事存在
	_, err := s.repo.StoryByID(ctx, storyID)
	if err != nil {
		if err == domain.ErrNotFound {
			return errors.New("story not found")
		}
		return errors.New("failed to get story")
	}

	if err := s.repo.LikeStory(ctx, userID, storyID); err != nil {
		s.logger.Error("failed to like story", zap.Error(err))
		return errors.New("failed to like story")
	}

	s.logger.Info("story liked successfully", zap.String("storyID", storyID))
	return nil
}

// UnlikeStory 取消点赞故事
func (s *Service) UnlikeStory(ctx context.Context, userID, storyID string) error {
	s.logger.Info("unliking story",
		zap.String("userID", userID),
		zap.String("storyID", storyID),
	)

	if err := s.repo.UnlikeStory(ctx, userID, storyID); err != nil {
		s.logger.Error("failed to unlike story", zap.Error(err))
		return errors.New("failed to unlike story")
	}

	s.logger.Info("story unliked successfully", zap.String("storyID", storyID))
	return nil
}

// FollowStory 关注故事
func (s *Service) FollowStory(ctx context.Context, userID, storyID string) error {
	s.logger.Info("following story",
		zap.String("userID", userID),
		zap.String("storyID", storyID),
	)

	// 验证故事存在
	_, err := s.repo.StoryByID(ctx, storyID)
	if err != nil {
		if err == domain.ErrNotFound {
			return errors.New("story not found")
		}
		return errors.New("failed to get story")
	}

	if err := s.repo.FollowStory(ctx, userID, storyID); err != nil {
		s.logger.Error("failed to follow story", zap.Error(err))
		return errors.New("failed to follow story")
	}

	s.logger.Info("story followed successfully", zap.String("storyID", storyID))
	return nil
}

// UnfollowStory 取消关注故事
func (s *Service) UnfollowStory(ctx context.Context, userID, storyID string) error {
	s.logger.Info("unfollowing story",
		zap.String("userID", userID),
		zap.String("storyID", storyID),
	)

	if err := s.repo.UnfollowStory(ctx, userID, storyID); err != nil {
		s.logger.Error("failed to unfollow story", zap.Error(err))
		return errors.New("failed to unfollow story")
	}

	s.logger.Info("story unfollowed successfully", zap.String("storyID", storyID))
	return nil
}

// ========== 故事 AI 渲染功能（同步） ==========

// RenderStoryRequest AI渲染故事请求（同步模式）
// 用于丰富故事描述和生成背景图片
type RenderStoryRequest struct {
	// AI 丰富选项
	EnrichDescription  bool   `json:"enrichDescription"`     // 是否丰富故事描述
	GenerateBackground bool   `json:"generateBackground"`    // 是否生成背景图片
	GenerateCover      bool   `json:"generateCover"`         // 是否生成封面图片
	Style              string `json:"style,omitempty"`       // AI 生成风格：realistic, anime, fantasy, etc.
	AspectRatio        string `json:"aspectRatio,omitempty"` // 图片比例：1:1, 16:9, 9:16, etc.
}

// RenderStoryResponse AI渲染故事响应
type RenderStoryResponse struct {
	Story               *domain.Story `json:"story"`                         // 更新后的故事
	EnrichedDescription string        `json:"enrichedDescription,omitempty"` // 丰富后的描述
	BackgroundURL       string        `json:"backgroundUrl,omitempty"`       // 生成的背景图片URL
	CoverURL            string        `json:"coverUrl,omitempty"`            // 生成的封面图片URL
	TokensUsed          int           `json:"tokensUsed"`                    // 消耗的token数
}

// RenderStory AI渲染故事（同步模式）
// 根据用户输入的标题和描述，同步丰富故事文本并生成背景图片
func (s *Service) RenderStory(ctx context.Context, userID, storyID string, req RenderStoryRequest) (*RenderStoryResponse, error) {
	s.logger.Info("rendering story with AI (sync mode)",
		zap.String("userID", userID),
		zap.String("storyID", storyID),
		zap.Bool("enrichDescription", req.EnrichDescription),
		zap.Bool("generateBackground", req.GenerateBackground),
		zap.Bool("generateCover", req.GenerateCover),
	)

	// 验证故事存在且属于用户
	story, err := s.repo.StoryByID(ctx, storyID)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, errors.New("story not found")
		}
		s.logger.Error("failed to get story", zap.Error(err))
		return nil, errors.New("failed to get story")
	}

	if story.Author.ID != userID {
		return nil, errors.New("unauthorized: not story owner")
	}

	// 检查是否有需要处理的任务
	if !req.EnrichDescription && !req.GenerateBackground && !req.GenerateCover {
		return nil, errors.New("no render options selected")
	}

	response := &RenderStoryResponse{
		Story: story,
	}
	totalTokens := 0
	now := time.Now().Unix()
	storyUpdated := false

	// 1. 同步丰富故事描述
	if req.EnrichDescription && story.Description != "" {
		s.logger.Info("enriching story description",
			zap.String("storyID", storyID))

		enrichResp, err := s.EnrichStoryDescription(ctx, userID, storyID, story.Description, story.Genre, req.Style)
		if err != nil {
			s.logger.Warn("failed to enrich story description with AI",
				zap.String("storyID", storyID),
				zap.Error(err))
			// 不中断流程，继续处理其他任务
		} else {
			response.EnrichedDescription = enrichResp.EnrichedDescription
			story.EnrichedDescription = enrichResp.EnrichedDescription
			story.Description = enrichResp.EnrichedDescription // 使用丰富后的描述
			story.IsAIEnriched = true
			story.AIEnrichedAt = &now
			story.TextTokensUsed = enrichResp.TokensUsed
			totalTokens += enrichResp.TokensUsed
			storyUpdated = true

			s.logger.Info("story description enriched successfully",
				zap.String("storyID", storyID),
				zap.Int("tokensUsed", enrichResp.TokensUsed))
		}
	}

	// 2. 同步生成背景图片和封面
	if req.GenerateBackground || req.GenerateCover {
		s.logger.Info("generating story images",
			zap.String("storyID", storyID),
			zap.Bool("background", req.GenerateBackground),
			zap.Bool("cover", req.GenerateCover))

		coverResp, err := s.GenerateStoryCover(ctx, userID, storyID, GenerateStoryCoverRequest{
			Title:              story.Title,
			Description:        story.Description,
			Genre:              story.Genre,
			Style:              req.Style,
			AspectRatio:        req.AspectRatio,
			GenerateCover:      req.GenerateCover,
			GeneratePoster:     false,
			GenerateBackground: req.GenerateBackground,
		})
		if err != nil {
			s.logger.Warn("failed to generate story images with AI",
				zap.String("storyID", storyID),
				zap.Error(err))
			// 不中断流程
		} else {
			if coverResp.BackgroundURL != "" {
				response.BackgroundURL = coverResp.BackgroundURL
				story.BackgroundImage = coverResp.BackgroundURL
				storyUpdated = true
			}
			if coverResp.CoverURL != "" {
				response.CoverURL = coverResp.CoverURL
				story.CoverImage = coverResp.CoverURL
				story.CoverGeneratedByAI = true
				storyUpdated = true
			}
			story.ImageTokensUsed += coverResp.TokensUsed
			totalTokens += coverResp.TokensUsed

			s.logger.Info("story images generated successfully",
				zap.String("storyID", storyID),
				zap.String("backgroundURL", coverResp.BackgroundURL),
				zap.String("coverURL", coverResp.CoverURL),
				zap.Int("tokensUsed", coverResp.TokensUsed))
		}
	}

	// 3. 更新故事
	if storyUpdated {
		story.TokensUsed += totalTokens
		story.UpdatedAt = now
		if err := s.repo.UpdateStory(ctx, story); err != nil {
			s.logger.Error("failed to update story after AI rendering",
				zap.String("storyID", storyID),
				zap.Error(err))
			return nil, errors.New("failed to save rendered story")
		}

		// 更新用户的token使用量
		if totalTokens > 0 {
			if err := s.updateUserTokenUsage(ctx, userID, totalTokens); err != nil {
				s.logger.Warn("failed to update user token usage",
					zap.String("userID", userID),
					zap.Int("tokensUsed", totalTokens),
					zap.Error(err))
			}
		}
	}

	response.Story = story
	response.TokensUsed = totalTokens

	s.logger.Info("story AI rendering completed",
		zap.String("storyID", storyID),
		zap.Int("totalTokensUsed", totalTokens),
		zap.Bool("descriptionEnriched", response.EnrichedDescription != ""),
		zap.Bool("backgroundGenerated", response.BackgroundURL != ""),
		zap.Bool("coverGenerated", response.CoverURL != ""))

	return response, nil
}

// ========== 故事媒体渲染功能（异步） ==========

// MediaRenderRequest 媒体渲染请求（视频/图片集/动画）
type MediaRenderRequest struct {
	Type        domain.RenderTaskType `json:"type" binding:"required,oneof=video image_set animation"`
	Resolution  string                `json:"resolution,omitempty"`
	FrameRate   int                   `json:"frameRate,omitempty"`
	Quality     string                `json:"quality,omitempty"`
	Format      string                `json:"format,omitempty"`
	Duration    int                   `json:"duration,omitempty"`
	Transitions string                `json:"transitions,omitempty"`
	BGM         string                `json:"bgm,omitempty"`
	Narration   bool                  `json:"narration,omitempty"`
	Subtitles   bool                  `json:"subtitles,omitempty"`
}

// RenderStoryMedia 渲染故事媒体（异步模式）
// 用于生成视频、图片集、动画等复杂媒体内容
func (s *Service) RenderStoryMedia(ctx context.Context, userID, storyID string, req MediaRenderRequest) (*domain.RenderTask, error) {
	s.logger.Info("rendering story media (async mode)",
		zap.String("userID", userID),
		zap.String("storyID", storyID),
		zap.String("type", string(req.Type)),
	)

	// 验证故事存在且属于用户
	story, err := s.repo.StoryByID(ctx, storyID)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, errors.New("story not found")
		}
		return nil, errors.New("failed to get story")
	}

	if story.Author.ID != userID {
		return nil, errors.New("unauthorized: not story owner")
	}

	// 检查故事是否有内容可以渲染
	if story.Panels == 0 {
		return nil, errors.New("story has no panels to render")
	}

	// 创建渲染配置
	config := domain.RenderConfig{
		Type:        req.Type,
		Resolution:  req.Resolution,
		FrameRate:   req.FrameRate,
		Quality:     req.Quality,
		Format:      req.Format,
		Duration:    req.Duration,
		Transitions: req.Transitions,
		BGM:         req.BGM,
		Narration:   req.Narration,
		Subtitles:   req.Subtitles,
	}

	// 设置默认值
	if config.Resolution == "" {
		config.Resolution = "1080p"
	}
	if config.Quality == "" {
		config.Quality = "high"
	}
	if config.Format == "" {
		if req.Type == domain.RenderTaskTypeVideo {
			config.Format = "mp4"
		} else {
			config.Format = "jpg"
		}
	}

	// 序列化配置
	configJSON, err := jsonMarshal(config)
	if err != nil {
		return nil, errors.New("failed to serialize config")
	}

	// 创建渲染任务
	task := &domain.RenderTask{
		ID:        generateID(),
		UserID:    userID,
		StoryID:   storyID,
		Type:      req.Type,
		Status:    domain.RenderTaskStatusPending,
		Config:    string(configJSON),
		Progress:  0,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}

	// 保存渲染任务到数据库
	if err := s.repo.CreateRenderTask(ctx, task); err != nil {
		s.logger.Error("failed to create render task", zap.Error(err))
		return nil, errors.New("failed to create render task")
	}

	// 更新故事状态为渲染中
	_, err = s.UpdateStory(ctx, userID, storyID, UpdateStoryRequest{
		Status: stringPtr("rendering"),
	})
	if err != nil {
		s.logger.Error("failed to update story status", zap.Error(err))
	}

	// 启动异步渲染
	go s.processMediaRenderTask(context.Background(), task)

	s.logger.Info("media render task created",
		zap.String("taskID", task.ID),
		zap.String("storyID", storyID),
	)

	return task, nil
}

// processMediaRenderTask 处理媒体渲染任务（异步）
func (s *Service) processMediaRenderTask(ctx context.Context, task *domain.RenderTask) {
	startTime := time.Now().Unix()

	// panic 恢复，确保任务不会永远停留在 processing 状态
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("panic in media render task processing",
				zap.String("taskID", task.ID),
				zap.Any("panic", r),
			)
			s.markRenderTaskFailed(ctx, task, fmt.Sprintf("internal error: %v", r))
		}
	}()

	s.logger.Info("processing media render task",
		zap.String("taskID", task.ID),
		zap.String("storyID", task.StoryID),
		zap.String("type", string(task.Type)),
	)

	// 创建带超时的 context（默认 30 分钟）
	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	// 更新状态为处理中
	task.Status = domain.RenderTaskStatusProcessing
	task.Progress = 10
	task.StartedAt = &startTime
	task.UpdatedAt = time.Now().Unix()
	if err := s.repo.UpdateRenderTask(ctx, task); err != nil {
		s.logger.Error("failed to update render task status", zap.String("taskID", task.ID), zap.Error(err))
		return
	}

	// 获取故事的分镜信息
	storyboards, err := s.repo.StoryboardsByStory(ctx, task.StoryID, 100, 0)
	if err != nil {
		s.logger.Error("failed to get storyboards",
			zap.String("taskID", task.ID),
			zap.String("storyID", task.StoryID),
			zap.Error(err))
		s.markRenderTaskFailed(ctx, task, "failed to get storyboards: "+err.Error())
		return
	}

	if len(storyboards) == 0 {
		s.logger.Warn("no storyboards found for story",
			zap.String("taskID", task.ID),
			zap.String("storyID", task.StoryID))
		s.markRenderTaskFailed(ctx, task, "no storyboards found for this story")
		return
	}

	s.logger.Info("found storyboards for rendering",
		zap.String("taskID", task.ID),
		zap.Int("count", len(storyboards)))

	// 解析渲染配置
	var config domain.RenderConfig
	if task.Config != "" {
		if err := json.Unmarshal([]byte(task.Config), &config); err != nil {
			s.logger.Warn("failed to parse render config, using defaults",
				zap.String("taskID", task.ID),
				zap.Error(err))
		}
	}

	// 设置默认配置
	if config.Resolution == "" {
		config.Resolution = "1080p"
	}
	if config.Quality == "" {
		config.Quality = "high"
	}

	// 根据任务类型执行不同的渲染流程
	var renderErr error
	switch task.Type {
	case domain.RenderTaskTypeVideo:
		renderErr = s.renderVideoContent(ctx, task, storyboards, &config)
	case domain.RenderTaskTypeImageSet:
		renderErr = s.renderImageSetContent(ctx, task, storyboards, &config)
	case domain.RenderTaskTypeAnimation:
		renderErr = s.renderAnimationContent(ctx, task, storyboards, &config)
	default:
		renderErr = fmt.Errorf("unknown render type: %s", task.Type)
	}

	if renderErr != nil {
		s.logger.Error("media render failed",
			zap.String("taskID", task.ID),
			zap.String("type", string(task.Type)),
			zap.Error(renderErr))
		s.markRenderTaskFailed(ctx, task, renderErr.Error())
		return
	}

	// 生成输出URL
	task.OutputURL = fmt.Sprintf("/uploads/renders/%s.%s", task.ID, getFileExtension(task.Type))
	task.ThumbnailURL = fmt.Sprintf("/uploads/renders/%s_thumb.jpg", task.ID)
	task.FileSize = 10485760 // 10MB（示例）

	if task.Type == domain.RenderTaskTypeVideo {
		task.Duration = 120 // 2分钟（示例）
		task.Resolution = "1080p"
	}

	// 完成
	task.Status = domain.RenderTaskStatusCompleted
	task.Progress = 100
	completedTime := time.Now().Unix()
	task.CompletedAt = &completedTime
	task.UpdatedAt = time.Now().Unix()
	if err := s.repo.UpdateRenderTask(ctx, task); err != nil {
		s.logger.Error("failed to update render task completion", zap.String("taskID", task.ID), zap.Error(err))
		return
	}

	// 更新故事状态为 rendered（渲染完成，等待用户发布）
	_, err = s.UpdateStory(ctx, task.UserID, task.StoryID, UpdateStoryRequest{
		Status: stringPtr("rendered"),
	})
	if err != nil {
		s.logger.Error("failed to update story status after render", zap.Error(err))
	}

	s.logger.Info("media render task completed",
		zap.String("taskID", task.ID),
		zap.Duration("duration", time.Duration(time.Now().Unix()-startTime)*time.Second),
	)
}

// renderVideoContent 渲染视频内容
func (s *Service) renderVideoContent(ctx context.Context, task *domain.RenderTask, storyboards []*domain.Storyboard, config *domain.RenderConfig) error {
	totalSteps := len(storyboards)
	baseProgress := 20

	for i := range storyboards {
		// 检查取消
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if cancelled := s.isRenderTaskCancelled(ctx, task.ID); cancelled {
			return errors.New("task cancelled by user")
		}

		progress := baseProgress + ((i + 1) * 60 / totalSteps)
		if err := s.repo.UpdateRenderTaskProgress(ctx, task.ID, progress); err != nil {
			s.logger.Warn("failed to update progress", zap.Error(err))
		}

		// TODO: 实际的视频生成逻辑
		time.Sleep(500 * time.Millisecond)
	}

	if err := s.repo.UpdateRenderTaskProgress(ctx, task.ID, 90); err != nil {
		s.logger.Warn("failed to update progress", zap.Error(err))
	}

	return nil
}

// renderImageSetContent 渲染图片集内容
func (s *Service) renderImageSetContent(ctx context.Context, task *domain.RenderTask, storyboards []*domain.Storyboard, config *domain.RenderConfig) error {
	totalSteps := len(storyboards)
	baseProgress := 20

	for i := range storyboards {
		// 检查取消
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if cancelled := s.isRenderTaskCancelled(ctx, task.ID); cancelled {
			return errors.New("task cancelled by user")
		}

		progress := baseProgress + ((i + 1) * 70 / totalSteps)
		if err := s.repo.UpdateRenderTaskProgress(ctx, task.ID, progress); err != nil {
			s.logger.Warn("failed to update progress", zap.Error(err))
		}

		// TODO: 实际的图片集生成逻辑
		time.Sleep(300 * time.Millisecond)
	}

	if err := s.repo.UpdateRenderTaskProgress(ctx, task.ID, 90); err != nil {
		s.logger.Warn("failed to update progress", zap.Error(err))
	}

	return nil
}

// renderAnimationContent 渲染动画内容
func (s *Service) renderAnimationContent(ctx context.Context, task *domain.RenderTask, storyboards []*domain.Storyboard, config *domain.RenderConfig) error {
	totalSteps := len(storyboards)
	baseProgress := 20

	for i := range storyboards {
		// 检查取消
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if cancelled := s.isRenderTaskCancelled(ctx, task.ID); cancelled {
			return errors.New("task cancelled by user")
		}

		progress := baseProgress + ((i + 1) * 50 / totalSteps)
		if err := s.repo.UpdateRenderTaskProgress(ctx, task.ID, progress); err != nil {
			s.logger.Warn("failed to update progress", zap.Error(err))
		}

		// TODO: 实际的动画生成逻辑
		time.Sleep(400 * time.Millisecond)
	}

	if err := s.repo.UpdateRenderTaskProgress(ctx, task.ID, 90); err != nil {
		s.logger.Warn("failed to update progress", zap.Error(err))
	}

	return nil
}

// isRenderTaskCancelled 检查渲染任务是否被用户取消
func (s *Service) isRenderTaskCancelled(ctx context.Context, taskID string) bool {
	currentTask, err := s.repo.GetRenderTask(ctx, taskID)
	if err != nil {
		return false
	}
	return currentTask != nil && currentTask.Status == domain.RenderTaskStatusCancelled
}

// markRenderTaskFailed 将渲染任务标记为失败并恢复故事状态
func (s *Service) markRenderTaskFailed(ctx context.Context, task *domain.RenderTask, errMsg string) {
	task.Status = domain.RenderTaskStatusFailed
	task.ErrorMessage = errMsg
	task.UpdatedAt = time.Now().Unix()

	// 使用新的 context 防止原 context 已取消
	updateCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.repo.UpdateRenderTask(updateCtx, task); err != nil {
		s.logger.Error("failed to mark render task as failed",
			zap.String("taskID", task.ID),
			zap.Error(err),
		)
	}

	s.restoreStoryStatus(updateCtx, task)
}

// markRenderTaskCancelled 将渲染任务标记为取消（用于超时等情况）
func (s *Service) markRenderTaskCancelled(ctx context.Context, task *domain.RenderTask, reason string) {
	task.Status = domain.RenderTaskStatusCancelled
	task.ErrorMessage = reason
	task.UpdatedAt = time.Now().Unix()

	// 使用新的 context 防止原 context 已取消
	updateCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.repo.UpdateRenderTask(updateCtx, task); err != nil {
		s.logger.Error("failed to mark render task as cancelled",
			zap.String("taskID", task.ID),
			zap.Error(err),
		)
	}

	s.restoreStoryStatus(updateCtx, task)
}

// restoreStoryStatus 恢复故事状态（从 rendering 恢复为 draft）
func (s *Service) restoreStoryStatus(ctx context.Context, task *domain.RenderTask) {
	_, err := s.UpdateStory(ctx, task.UserID, task.StoryID, UpdateStoryRequest{
		Status: stringPtr("draft"),
	})
	if err != nil {
		s.logger.Error("failed to restore story status after render failure/cancellation",
			zap.String("storyID", task.StoryID),
			zap.Error(err),
		)
	}
}

// GetRenderTaskStatus 获取渲染任务状态
func (s *Service) GetRenderTaskStatus(ctx context.Context, taskID string) (*domain.RenderTask, error) {
	s.logger.Info("getting render task status",
		zap.String("taskID", taskID))

	// 参数验证
	if taskID == "" {
		s.logger.Error("task ID is empty")
		return nil, errors.New("task ID is required")
	}

	// 从数据库获取任务
	task, err := s.repo.GetRenderTask(ctx, taskID)
	if err != nil {
		if err == domain.ErrNotFound {
			s.logger.Warn("render task not found",
				zap.String("taskID", taskID))
			return nil, errors.New("task not found")
		}
		s.logger.Error("failed to get render task",
			zap.String("taskID", taskID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get render task: %w", err)
	}

	s.logger.Info("render task status retrieved",
		zap.String("taskID", taskID),
		zap.String("status", string(task.Status)),
		zap.Int("progress", task.Progress))

	return task, nil
}

// GetLatestRenderTaskByStoryID 获取故事的最新渲染任务
func (s *Service) GetLatestRenderTaskByStoryID(ctx context.Context, storyID string) (*domain.RenderTask, error) {
	s.logger.Info("getting latest render task by story ID",
		zap.String("storyID", storyID))

	// 参数验证
	if storyID == "" {
		s.logger.Error("story ID is empty")
		return nil, errors.New("story ID is required")
	}

	// 验证故事是否存在
	_, err := s.repo.StoryByID(ctx, storyID)
	if err != nil {
		if err == domain.ErrNotFound {
			s.logger.Warn("story not found",
				zap.String("storyID", storyID))
			return nil, errors.New("story not found")
		}
		s.logger.Error("failed to get story",
			zap.String("storyID", storyID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get story: %w", err)
	}

	// 从数据库获取最新的渲染任务
	task, err := s.repo.GetRenderTaskByStoryID(ctx, storyID)
	if err != nil {
		if err == domain.ErrNotFound {
			s.logger.Info("no render task found for story",
				zap.String("storyID", storyID))
			return nil, errors.New("task not found")
		}
		s.logger.Error("failed to get render task",
			zap.String("storyID", storyID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get render task: %w", err)
	}

	s.logger.Info("latest render task retrieved",
		zap.String("storyID", storyID),
		zap.String("taskID", task.ID),
		zap.String("status", string(task.Status)),
		zap.Int("progress", task.Progress))

	return task, nil
}

// CancelRenderTask 取消渲染任务
func (s *Service) CancelRenderTask(ctx context.Context, userID, taskID string) error {
	s.logger.Info("cancelling render task",
		zap.String("userID", userID),
		zap.String("taskID", taskID),
	)

	// 1. 获取任务
	task, err := s.repo.GetRenderTask(ctx, taskID)
	if err != nil {
		s.logger.Error("failed to get render task", zap.Error(err))
		return errors.New("failed to get render task")
	}
	if task == nil {
		return errors.New("render task not found")
	}

	// 2. 验证任务属于用户
	if task.UserID != userID {
		return errors.New("unauthorized: task belongs to another user")
	}

	// 3. 检查任务状态是否可以取消
	if task.Status != domain.RenderTaskStatusPending && task.Status != domain.RenderTaskStatusProcessing {
		return fmt.Errorf("cannot cancel task with status: %s", task.Status)
	}

	// 4. 更新任务状态为 cancelled
	task.Status = domain.RenderTaskStatusCancelled
	task.UpdatedAt = time.Now().Unix()
	completedTime := time.Now().Unix()
	task.CompletedAt = &completedTime

	if err := s.repo.UpdateRenderTask(ctx, task); err != nil {
		s.logger.Error("failed to update render task status", zap.Error(err))
		return errors.New("failed to cancel render task")
	}

	// 5. 恢复故事状态为 draft
	_, err = s.UpdateStory(ctx, userID, task.StoryID, UpdateStoryRequest{
		Status: stringPtr("draft"),
	})
	if err != nil {
		s.logger.Error("failed to restore story status", zap.Error(err))
	}

	s.logger.Info("render task cancelled",
		zap.String("taskID", taskID),
		zap.String("userID", userID),
	)

	return nil
}

// ========== 故事发布功能 ==========

// PublishStory 发布故事
func (s *Service) PublishStory(ctx context.Context, userID, storyID string) (*domain.StoryPublication, error) {
	s.logger.Info("publishing story",
		zap.String("userID", userID),
		zap.String("storyID", storyID),
	)

	// 验证故事存在且属于用户
	story, err := s.repo.StoryByID(ctx, storyID)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, errors.New("story not found")
		}
		return nil, errors.New("failed to get story")
	}

	if story.Author.ID != userID {
		return nil, errors.New("unauthorized: not story owner")
	}

	// 检查故事状态
	if story.Status == "published" {
		return nil, errors.New("story is already published")
	}

	// 检查故事是否有内容
	if story.Panels == 0 {
		return nil, errors.New("cannot publish empty story")
	}

	// 获取当前最新版本号
	nextVersion := 1
	latestPublication, err := s.repo.GetLatestStoryPublication(ctx, storyID)
	if err != nil {
		s.logger.Warn("failed to get latest publication, using version 1", zap.Error(err))
	} else if latestPublication != nil {
		nextVersion = latestPublication.Version + 1
	}

	// 创建发布记录
	now := time.Now().Unix()
	publication := &domain.StoryPublication{
		ID:          generateID(),
		StoryID:     storyID,
		Version:     nextVersion,
		Status:      "published",
		PublishedAt: now,
		UpdatedAt:   now,
	}

	// 保存发布记录
	if err := s.repo.CreateStoryPublication(ctx, publication); err != nil {
		s.logger.Error("failed to create publication record", zap.Error(err))
		return nil, errors.New("failed to create publication record")
	}

	// 更新故事状态为已发布
	_, err = s.UpdateStory(ctx, userID, storyID, UpdateStoryRequest{
		Status: stringPtr("published"),
	})
	if err != nil {
		s.logger.Error("failed to update story status", zap.Error(err))
		return nil, errors.New("failed to update story status")
	}

	s.logger.Info("story published successfully",
		zap.String("storyID", storyID),
		zap.String("publicationID", publication.ID),
		zap.Int("version", nextVersion),
	)

	return publication, nil
}

// UnpublishStory 取消发布故事
func (s *Service) UnpublishStory(ctx context.Context, userID, storyID string) error {
	s.logger.Info("unpublishing story",
		zap.String("userID", userID),
		zap.String("storyID", storyID),
	)

	// 验证故事存在且属于用户
	story, err := s.repo.StoryByID(ctx, storyID)
	if err != nil {
		if err == domain.ErrNotFound {
			return errors.New("story not found")
		}
		return errors.New("failed to get story")
	}

	if story.Author.ID != userID {
		return errors.New("unauthorized: not story owner")
	}

	if story.Status != "published" {
		return errors.New("story is not published")
	}

	// 获取最新的发布记录
	publication, err := s.repo.GetLatestStoryPublication(ctx, storyID)
	if err != nil {
		s.logger.Warn("failed to get latest publication, continuing anyway",
			zap.String("storyID", storyID),
			zap.Error(err))
	}

	// 更新发布记录状态
	if publication != nil && publication.Status == "published" {
		now := time.Now().Unix()
		publication.Status = "unpublished"
		publication.UnpublishedAt = &now
		publication.UpdatedAt = now

		if err := s.repo.UpdateStoryPublication(ctx, publication); err != nil {
			s.logger.Error("failed to update publication record",
				zap.String("storyID", storyID),
				zap.String("publicationID", publication.ID),
				zap.Error(err))
			// 继续执行，不阻塞取消发布流程
		} else {
			s.logger.Info("publication record updated",
				zap.String("publicationID", publication.ID),
				zap.Int("version", publication.Version))
		}
	}

	// 更新故事状态为草稿
	_, err = s.UpdateStory(ctx, userID, storyID, UpdateStoryRequest{
		Status: stringPtr("draft"),
	})
	if err != nil {
		s.logger.Error("failed to update story status", zap.Error(err))
		return errors.New("failed to update story status")
	}

	s.logger.Info("story unpublished successfully", zap.String("storyID", storyID))
	return nil
}

// ========== 辅助函数 ==========

// getFileExtension 根据渲染类型获取文件扩展名
func getFileExtension(renderType domain.RenderTaskType) string {
	switch renderType {
	case domain.RenderTaskTypeVideo:
		return "mp4"
	case domain.RenderTaskTypeImageSet:
		return "zip"
	case domain.RenderTaskTypeAnimation:
		return "gif"
	default:
		return "mp4"
	}
}

// stringPtr 字符串指针辅助函数
func stringPtr(s string) *string {
	return &s
}

// jsonMarshal JSON序列化辅助函数
func jsonMarshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// generateID 生成ID辅助函数
func generateID() string {
	return fmt.Sprintf("id_%d", time.Now().UnixNano())
}

// ========== AI 故事丰富功能 ==========

// EnrichStoryDescription 使用AI丰富故事描述
// 根据用户输入的原始描述，生成更加丰富、生动的故事背景描述
func (s *Service) EnrichStoryDescription(ctx context.Context, userID, storyID, originalDesc, genre, style string) (*StoryAIEnrichResponse, error) {
	s.logger.Info("enriching story description with AI",
		zap.String("userID", userID),
		zap.String("storyID", storyID),
		zap.String("genre", genre),
	)

	// 构建丰富描述的提示词
	prompt := s.buildEnrichDescriptionPrompt(originalDesc, genre, style)

	// 调用AI生成服务
	if s.aiGenService == nil {
		return nil, errors.New("AI generation service not available")
	}

	result, err := s.aiGenService.GenerateText(ctx, &GenerateTextRequest{
		UserID:            userID,
		OriginalPrompt:    prompt,
		SystemPrompt:      "你是一位专业的故事创作顾问，擅长将简单的故事概念扩展成引人入胜的故事背景描述。请保持原意的同时，添加更多细节、氛围描写和世界观设定。",
		Model:             "", // 使用默认模型
		Temperature:       0.7,
		MaxTokens:         1000,
		RelatedEntityID:   storyID,
		RelatedEntityType: "story",
		Metadata: map[string]interface{}{
			"operation": "enrich_description",
			"genre":     genre,
			"style":     style,
		},
	})
	if err != nil {
		s.logger.Error("failed to enrich story description",
			zap.String("storyID", storyID),
			zap.Error(err))
		return nil, fmt.Errorf("AI enrichment failed: %w", err)
	}

	s.logger.Info("story description enriched successfully",
		zap.String("storyID", storyID),
		zap.Int("tokensUsed", result.TokensUsed))

	return &StoryAIEnrichResponse{
		EnrichedDescription: result.Text,
		TokensUsed:          result.TokensUsed,
	}, nil
}

// buildEnrichDescriptionPrompt 构建丰富描述的提示词
func (s *Service) buildEnrichDescriptionPrompt(originalDesc, genre, style string) string {
	prompt := fmt.Sprintf(`请根据以下故事概念，创作一段更加丰富、生动的故事背景描述（300-500字）：

原始描述：
%s

`, originalDesc)

	if genre != "" {
		prompt += fmt.Sprintf("故事类型：%s\n", genre)
	}

	if style != "" {
		prompt += fmt.Sprintf("风格偏好：%s\n", style)
	}

	prompt += `
要求：
1. 保持原有故事的核心概念和主题
2. 添加更多环境描写和氛围营造
3. 可以适当扩展世界观设定
4. 暗示可能的冲突或悬念
5. 语言生动，具有画面感
6. 直接输出丰富后的描述，不要包含任何解释或前缀

请直接输出丰富后的故事描述：`

	return prompt
}

// GenerateStoryCoverRequest 生成故事封面请求
type GenerateStoryCoverRequest struct {
	Title              string `json:"title"`
	Description        string `json:"description"`
	Genre              string `json:"genre"`
	Style              string `json:"style,omitempty"`
	AspectRatio        string `json:"aspectRatio,omitempty"`
	GenerateCover      bool   `json:"generateCover"`
	GeneratePoster     bool   `json:"generatePoster"`
	GenerateBackground bool   `json:"generateBackground"`
}

// GenerateStoryCover 使用AI生成故事封面/海报/背景图片
func (s *Service) GenerateStoryCover(ctx context.Context, userID, storyID string, req GenerateStoryCoverRequest) (*StoryAICoverResponse, error) {
	s.logger.Info("generating story cover with AI",
		zap.String("userID", userID),
		zap.String("storyID", storyID),
		zap.Bool("cover", req.GenerateCover),
		zap.Bool("poster", req.GeneratePoster),
		zap.Bool("background", req.GenerateBackground),
	)

	if s.aiGenService == nil {
		return nil, errors.New("AI generation service not available")
	}

	response := &StoryAICoverResponse{}
	totalTokens := 0

	// 设置默认比例
	if req.AspectRatio == "" {
		req.AspectRatio = "16:9"
	}

	// 构建图片生成提示词
	imagePrompt := s.buildCoverImagePrompt(req.Title, req.Description, req.Genre, req.Style)

	// 生成封面图
	if req.GenerateCover {
		coverResult, err := s.aiGenService.GenerateImage(ctx, &GenerateImageRequest{
			UserID:            userID,
			Prompt:            imagePrompt + " 作为书籍封面，突出标题文字空间，专业设计感",
			Provider:          "gemini", // 使用默认provider
			Model:             "",
			AspectRatio:       req.AspectRatio,
			Quality:           "high",
			OutputCount:       1,
			RelatedEntityID:   storyID,
			RelatedEntityType: "story_cover",
			Metadata: map[string]interface{}{
				"operation": "generate_cover",
				"storyId":   storyID,
			},
		})
		if err != nil {
			s.logger.Warn("failed to generate cover image",
				zap.String("storyID", storyID),
				zap.Error(err))
		} else if len(coverResult.ImageURLs) > 0 {
			response.CoverURL = coverResult.ImageURLs[0]
			totalTokens += coverResult.TokensUsed
		}
	}

	// 生成海报图
	if req.GeneratePoster {
		posterResult, err := s.aiGenService.GenerateImage(ctx, &GenerateImageRequest{
			UserID:            userID,
			Prompt:            imagePrompt + " 电影海报风格，戏剧性光影，史诗感构图",
			Provider:          "gemini",
			Model:             "",
			AspectRatio:       "2:3", // 海报通常是竖版
			Quality:           "high",
			OutputCount:       1,
			RelatedEntityID:   storyID,
			RelatedEntityType: "story_poster",
			Metadata: map[string]interface{}{
				"operation": "generate_poster",
				"storyId":   storyID,
			},
		})
		if err != nil {
			s.logger.Warn("failed to generate poster image",
				zap.String("storyID", storyID),
				zap.Error(err))
		} else if len(posterResult.ImageURLs) > 0 {
			response.PosterURL = posterResult.ImageURLs[0]
			totalTokens += posterResult.TokensUsed
		}
	}

	// 生成背景图
	if req.GenerateBackground {
		bgResult, err := s.aiGenService.GenerateImage(ctx, &GenerateImageRequest{
			UserID:            userID,
			Prompt:            imagePrompt + " 宽幅背景图，适合作为页面背景，柔和色调，不抢眼",
			Provider:          "gemini",
			Model:             "",
			AspectRatio:       "21:9", // 宽幅背景
			Quality:           "standard",
			OutputCount:       1,
			RelatedEntityID:   storyID,
			RelatedEntityType: "story_background",
			Metadata: map[string]interface{}{
				"operation": "generate_background",
				"storyId":   storyID,
			},
		})
		if err != nil {
			s.logger.Warn("failed to generate background image",
				zap.String("storyID", storyID),
				zap.Error(err))
		} else if len(bgResult.ImageURLs) > 0 {
			response.BackgroundURL = bgResult.ImageURLs[0]
			totalTokens += bgResult.TokensUsed
		}
	}

	response.TokensUsed = totalTokens

	s.logger.Info("story cover generation completed",
		zap.String("storyID", storyID),
		zap.Int("totalTokensUsed", totalTokens),
		zap.Bool("hasCover", response.CoverURL != ""),
		zap.Bool("hasPoster", response.PosterURL != ""),
		zap.Bool("hasBackground", response.BackgroundURL != ""))

	return response, nil
}

// buildCoverImagePrompt 构建封面图片生成提示词
func (s *Service) buildCoverImagePrompt(title, description, genre, style string) string {
	prompt := fmt.Sprintf(`为故事《%s》生成一张精美的插画。

故事描述：%s
`, title, description)

	// 根据类型添加风格指导
	genreStyles := map[string]string{
		"fantasy":   "奇幻风格，魔法元素，神秘氛围",
		"scifi":     "科幻风格，未来科技，太空元素",
		"romance":   "浪漫唯美，柔和光线，温馨氛围",
		"horror":    "黑暗神秘，紧张氛围，阴影对比",
		"adventure": "冒险风格，宏大场景，动感构图",
		"mystery":   "悬疑风格，神秘元素，暗色调",
		"comedy":    "轻松明快，鲜艳色彩，有趣构图",
		"drama":     "戏剧性，情感张力，人物特写",
	}

	if genreStyle, ok := genreStyles[genre]; ok {
		prompt += fmt.Sprintf("\n风格要求：%s", genreStyle)
	}

	if style != "" {
		prompt += fmt.Sprintf("\n艺术风格：%s", style)
	}

	prompt += "\n\n要求：高质量，专业插画，适合作为故事封面"

	return prompt
}

// updateUserTokenUsage 更新用户的token使用量
func (s *Service) updateUserTokenUsage(ctx context.Context, userID string, tokensUsed int) error {
	s.logger.Info("updating user token usage",
		zap.String("userID", userID),
		zap.Int("tokensUsed", tokensUsed))

	// 这里可以调用用户服务更新token使用记录
	// 实际实现可能需要：
	// 1. 检查用户的token余额
	// 2. 扣除使用的token
	// 3. 记录使用日志

	// 简单实现：记录到AI生成记录表
	// 实际项目中可能需要更复杂的计费逻辑
	return nil
}

// RequestAIEnrichment 请求AI丰富故事（用户确认后调用）
// 这个方法用于在故事创建后，用户单独选择是否使用AI丰富
type RequestAIEnrichmentRequest struct {
	EnrichDescription  bool   `json:"enrichDescription"`
	GenerateCover      bool   `json:"generateCover"`
	GeneratePoster     bool   `json:"generatePoster"`
	GenerateBackground bool   `json:"generateBackground"`
	Style              string `json:"style,omitempty"`
	AspectRatio        string `json:"aspectRatio,omitempty"`
}

// RequestAIEnrichment 请求对已存在的故事进行AI丰富
func (s *Service) RequestAIEnrichment(ctx context.Context, userID, storyID string, req RequestAIEnrichmentRequest) (*domain.Story, error) {
	s.logger.Info("requesting AI enrichment for existing story",
		zap.String("userID", userID),
		zap.String("storyID", storyID),
		zap.Bool("enrichDescription", req.EnrichDescription),
		zap.Bool("generateCover", req.GenerateCover),
	)

	// 获取故事
	story, err := s.repo.StoryByID(ctx, storyID)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, errors.New("story not found")
		}
		return nil, errors.New("failed to get story")
	}

	// 验证权限
	if story.Author == nil || story.Author.ID != userID {
		return nil, errors.New("unauthorized: not story owner")
	}

	totalTokens := 0
	now := time.Now().Unix()

	// 丰富描述
	if req.EnrichDescription && story.Description != "" {
		// 使用原始描述进行丰富（如果有的话）
		descToEnrich := story.OriginalDescription
		if descToEnrich == "" {
			descToEnrich = story.Description
		}

		enrichResp, err := s.EnrichStoryDescription(ctx, userID, storyID, descToEnrich, story.Genre, req.Style)
		if err != nil {
			s.logger.Warn("failed to enrich story description",
				zap.String("storyID", storyID),
				zap.Error(err))
		} else {
			// 保留原始描述
			if story.OriginalDescription == "" {
				story.OriginalDescription = story.Description
			}
			story.EnrichedDescription = enrichResp.EnrichedDescription
			story.Description = enrichResp.EnrichedDescription
			story.IsAIEnriched = true
			story.AIEnrichedAt = &now
			story.TextTokensUsed += enrichResp.TokensUsed
			totalTokens += enrichResp.TokensUsed
		}
	}

	// 生成封面/海报/背景
	if req.GenerateCover || req.GeneratePoster || req.GenerateBackground {
		coverResp, err := s.GenerateStoryCover(ctx, userID, storyID, GenerateStoryCoverRequest{
			Title:              story.Title,
			Description:        story.Description,
			Genre:              story.Genre,
			Style:              req.Style,
			AspectRatio:        req.AspectRatio,
			GenerateCover:      req.GenerateCover,
			GeneratePoster:     req.GeneratePoster,
			GenerateBackground: req.GenerateBackground,
		})
		if err != nil {
			s.logger.Warn("failed to generate story cover",
				zap.String("storyID", storyID),
				zap.Error(err))
		} else {
			if coverResp.CoverURL != "" {
				story.CoverImage = coverResp.CoverURL
				story.CoverGeneratedByAI = true
			}
			if coverResp.PosterURL != "" {
				story.PosterImage = coverResp.PosterURL
			}
			if coverResp.BackgroundURL != "" {
				story.BackgroundImage = coverResp.BackgroundURL
			}
			story.ImageTokensUsed += coverResp.TokensUsed
			totalTokens += coverResp.TokensUsed
		}
	}

	// 更新token统计
	story.TokensUsed += totalTokens
	story.UpdatedAt = now

	// 保存更新
	if err := s.repo.UpdateStory(ctx, story); err != nil {
		s.logger.Error("failed to update story with AI enrichment",
			zap.String("storyID", storyID),
			zap.Error(err))
		return nil, errors.New("failed to save AI enrichment")
	}

	// 更新用户token使用量
	if totalTokens > 0 {
		if err := s.updateUserTokenUsage(ctx, userID, totalTokens); err != nil {
			s.logger.Warn("failed to update user token usage",
				zap.String("userID", userID),
				zap.Int("tokensUsed", totalTokens),
				zap.Error(err))
		}
	}

	s.logger.Info("AI enrichment completed",
		zap.String("storyID", storyID),
		zap.Int("totalTokensUsed", totalTokens))

	return story, nil
}

// GetStoryAIUsage 获取故事的AI使用统计
func (s *Service) GetStoryAIUsage(ctx context.Context, storyID string) (map[string]interface{}, error) {
	story, err := s.repo.StoryByID(ctx, storyID)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"storyId":            storyID,
		"isAIEnriched":       story.IsAIEnriched,
		"coverGeneratedByAI": story.CoverGeneratedByAI,
		"tokensUsed":         story.TokensUsed,
		"textTokensUsed":     story.TextTokensUsed,
		"imageTokensUsed":    story.ImageTokensUsed,
		"aiGenerationCost":   story.AIGenerationCost,
		"aiEnrichedAt":       story.AIEnrichedAt,
	}, nil
}

// ========== Story Contributor operations ==========

// InviteStoryContributorRequest 邀请故事贡献者请求
type InviteStoryContributorRequest struct {
	UserID string `json:"userId" binding:"required"`
	Role   string `json:"role" binding:"required,oneof=collaborator contributor"`
}

// InviteStoryContributor 邀请用户成为故事贡献者
func (s *Service) InviteStoryContributor(ctx context.Context, inviterID, storyID string, req InviteStoryContributorRequest) (*domain.StoryContributor, error) {
	s.logger.Info("inviting story contributor",
		zap.String("inviterID", inviterID),
		zap.String("storyID", storyID),
		zap.String("userID", req.UserID),
		zap.String("role", req.Role),
	)

	// 验证故事存在
	story, err := s.repo.StoryByID(ctx, storyID)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, errors.New("story not found")
		}
		return nil, errors.New("failed to get story")
	}

	// 验证邀请者权限（必须是作者或贡献者）
	isAuthor := story.Author != nil && story.Author.ID == inviterID
	if !isAuthor {
		isContributor, err := s.repo.IsStoryContributor(ctx, storyID, inviterID)
		if err != nil || !isContributor {
			return nil, errors.New("permission denied: not a contributor")
		}
	}

	// 验证被邀请用户存在
	invitee, err := s.repo.UserByID(ctx, req.UserID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// 检查用户是否已经是贡献者
	isContributor, _ := s.repo.IsStoryContributor(ctx, storyID, req.UserID)
	if isContributor {
		return nil, errors.New("user is already a contributor")
	}

	// 检查是否是作者本人
	if story.Author != nil && story.Author.ID == req.UserID {
		return nil, errors.New("cannot invite the story author")
	}

	// 创建贡献者
	contributor := &domain.StoryContributor{
		ID:        generateID(),
		StoryID:   storyID,
		UserID:    req.UserID,
		Role:      domain.StoryContributorRole(req.Role),
		InvitedBy: inviterID,
	}

	if err := s.repo.AddStoryContributor(ctx, contributor); err != nil {
		s.logger.Error("failed to add story contributor", zap.Error(err))
		return nil, errors.New("failed to invite contributor")
	}

	contributor.User = invitee

	s.logger.Info("story contributor invited successfully",
		zap.String("storyID", storyID),
		zap.String("contributorID", contributor.ID),
	)

	return contributor, nil
}

// RemoveStoryContributor 移除故事贡献者
func (s *Service) RemoveStoryContributor(ctx context.Context, operatorID, storyID, contributorID string) error {
	s.logger.Info("removing story contributor",
		zap.String("operatorID", operatorID),
		zap.String("storyID", storyID),
		zap.String("contributorID", contributorID),
	)

	// 验证故事存在
	story, err := s.repo.StoryByID(ctx, storyID)
	if err != nil {
		if err == domain.ErrNotFound {
			return errors.New("story not found")
		}
		return errors.New("failed to get story")
	}

	// 验证操作者权限（必须是作者）
	if story.Author == nil || story.Author.ID != operatorID {
		return errors.New("permission denied: only author can remove contributors")
	}

	// 移除贡献者
	if err := s.repo.RemoveStoryContributor(ctx, storyID, contributorID); err != nil {
		if err == domain.ErrNotFound {
			return errors.New("contributor not found")
		}
		s.logger.Error("failed to remove story contributor", zap.Error(err))
		return errors.New("failed to remove contributor")
	}

	s.logger.Info("story contributor removed successfully",
		zap.String("storyID", storyID),
		zap.String("contributorID", contributorID),
	)

	return nil
}

// GetStoryContributors 获取故事贡献者列表
func (s *Service) GetStoryContributors(ctx context.Context, storyID string, limit, offset int) ([]*domain.StoryContributor, error) {
	s.logger.Info("getting story contributors",
		zap.String("storyID", storyID),
	)

	// 设置默认分页
	if limit == 0 {
		limit = 20
	}

	contributors, err := s.repo.GetStoryContributors(ctx, storyID, limit, offset)
	if err != nil {
		s.logger.Error("failed to get story contributors", zap.Error(err))
		return nil, errors.New("failed to get contributors")
	}

	return contributors, nil
}
