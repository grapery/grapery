package service

// 故事渲染和发布相关功能扩展

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/cache"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/utils"
	"go.uber.org/zap"
)

// GetTrendingStories24h returns up to `limit` trending published stories.
// Trending is determined by: followers > likes > updated_at.
// No time range restriction - includes all published stories.
// Intended for the guest-accessible dashboard trending endpoint (backend caps limit to 20).
func (s *Service) GetTrendingStories24h(ctx context.Context, limit int) ([]*domain.Story, error) {
	return s.repo.TrendingStories(ctx, limit)
}

// CreateStoryRequest 创建故事请求
type CreateStoryRequest struct {
	Title             string   `json:"title" binding:"required,min=1,max=200"`
	Description       string   `json:"description" binding:"max=2000"`
	CoverImage        string   `json:"coverImage" binding:"omitempty,url"`
	Genre             string   `json:"genre" binding:"required"`
	Status            string   `json:"status" binding:"omitempty,oneof=draft published"`
	GroupID           string   `json:"groupId" binding:"omitempty"`
	DefaultSceneCount int      `json:"defaultSceneCount"`                                // Default number of scenes for storyboards (2-8, default 3)
	Tags              []string `json:"tags" binding:"omitempty,max=3,dive,min=1,max=50"` // 最多3个标签，每个标签1-50字符

	// Collaboration settings
	IsCollaborationOpen bool `json:"isCollaborationOpen"` // Whether collaboration is open: true=anyone can edit, false=only author and group members can edit

	// AI 策略设置（新增）
	AIEnabled *bool `json:"aiEnabled"` // 是否允许AI辅助，默认 true

	// AI 丰富选项（可选）
	UseAIEnrich        bool                `json:"useAIEnrich"`                // 是否使用AI丰富故事描述
	GenerateCover      bool                `json:"generateCover"`              // 是否使用AI生成封面/海报
	GeneratePoster     bool                `json:"generatePoster"`             // 是否生成故事海报
	GenerateBackground bool                `json:"generateBackground"`         // 是否生成背景图片
	Style              string              `json:"style,omitempty"`            // AI 生成风格名称（字符串，用于AI辅助创建）
	AIStyle            *domain.StyleConfig `json:"aiStyle,omitempty"`          // AI 生成风格配置（完整信息，可为空）
	CoverAspectRatio   string              `json:"coverAspectRatio,omitempty"` // 封面图比例：1:1, 16:9, 9:16, etc.
}

// StoryAIEnrichResponse AI丰富故事的响应
type StoryAIEnrichResponse struct {
	EnrichedDescription string   `json:"enrichedDescription"`     // 丰富后的描述
	SuggestedTags       []string `json:"suggestedTags,omitempty"` // 建议的标签
	TokensUsed          int      `json:"tokensUsed"`              // 消耗的token数
	RecordID            string   `json:"recordId,omitempty"`      // AI生成记录ID（用于追踪/计费）
	DurationMs          int64    `json:"durationMs,omitempty"`    // 生成耗时（毫秒）
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

	// Collaboration settings
	IsCollaborationOpen *bool `json:"isCollaborationOpen"` // Whether collaboration is open: true=anyone can edit, false=only author and group members can edit
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
	s.logger.Debug("fetching author information",
		zap.String("userID", userID))
	author, err := s.repo.UserByID(ctx, userID)
	if err != nil {
		s.logger.Error("failed to get author",
			zap.String("userID", userID),
			zap.Error(err))
		return nil, errors.New("author not found")
	}

	s.logger.Debug("author retrieved",
		zap.String("userID", userID),
		zap.String("authorName", author.DisplayName))

	// 设置默认状态
	status := req.Status
	if status == "" {
		status = "draft"
		s.logger.Debug("using default status",
			zap.String("status", status))
	} else {
		s.logger.Debug("using provided status",
			zap.String("status", status))
	}

	now := time.Now().Unix()

	// Set default scene count (2-8, default 3)
	defaultSceneCount := req.DefaultSceneCount
	if defaultSceneCount < 2 || defaultSceneCount > 16 {
		defaultSceneCount = 6
		s.logger.Debug("using default scene count",
			zap.Int("requested", req.DefaultSceneCount),
			zap.Int("adjusted", defaultSceneCount))
	} else {
		s.logger.Debug("using provided scene count",
			zap.Int("sceneCount", defaultSceneCount))
	}

	// 确定 AIEnabled 值
	aiEnabled := true // 默认值
	if req.AIEnabled != nil {
		aiEnabled = *req.AIEnabled
	}

	// 如果用户选择了 AI 生成选项，自动启用 AI
	if req.UseAIEnrich || req.GenerateCover || req.GeneratePoster || req.GenerateBackground {
		aiEnabled = true
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
		Style:               req.AIStyle,
		IsCollaborationOpen: req.IsCollaborationOpen, // New field: default false (restricted)
		AIEnabled:           aiEnabled,               // 新增
	}

	// 保存故事到数据库（先创建，后续更新AI丰富的内容）
	s.logger.Debug("saving story to database",
		zap.String("storyID", story.ID),
		zap.String("title", story.Title),
		zap.String("genre", story.Genre),
		zap.String("status", story.Status))
	if err := s.repo.CreateStory(ctx, story); err != nil {
		s.logger.Error("failed to create story in database",
			zap.String("storyID", story.ID),
			zap.String("title", story.Title),
			zap.Error(err))
		return nil, errors.New("failed to create story")
	}

	s.logger.Info("story created successfully",
		zap.String("storyID", story.ID),
		zap.String("title", story.Title),
		zap.String("genre", story.Genre),
		zap.String("status", story.Status),
		zap.Int("defaultSceneCount", defaultSceneCount))

	// Record metrics
	if s.metrics != nil {
		s.metrics.RecordStoryCreation(story.Genre)
		// Record story participant count (will be updated when contributors are added)
		s.metrics.RecordStoryParticipantCount(story.ID, 1.0) // At least the author

		// Update story count metric (increment by 1 since we just created one)
		s.metrics.StoryCount.Inc()
	}

	// 如果提供了 Style 字符串但没有 AIStyle，尝试获取 StyleConfig
	if req.Style != "" && req.AIStyle == nil {
		s.logger.Debug("style string provided, attempting to load StyleConfig",
			zap.String("storyID", story.ID),
			zap.String("style", req.Style))
		// 尝试通过风格名称获取 StyleConfig
		styleConfig, err := s.GetStyleConfigByStyle(ctx, req.Style)
		if err == nil && styleConfig != nil {
			req.AIStyle = styleConfig
			s.logger.Debug("StyleConfig loaded from style name",
				zap.String("storyID", story.ID),
				zap.String("style", req.Style))
		} else {
			s.logger.Debug("could not load StyleConfig from style name, will use style string",
				zap.String("storyID", story.ID),
				zap.String("style", req.Style),
				zap.Error(err))
		}
	}

	// 如果用户选择使用AI丰富描述
	if req.UseAIEnrich && req.Description != "" {
		s.logger.Debug("AI enrichment requested",
			zap.String("storyID", story.ID),
			zap.Int("descriptionLength", len(req.Description)))
		enrichResp, err := s.EnrichStoryDescription(ctx, userID, story.ID, req.Description, req.Genre, req.AIStyle)
		if err != nil {
			s.logger.Warn("failed to enrich story description with AI, continuing with original",
				zap.String("storyID", story.ID),
				zap.String("userID", userID),
				zap.Error(err))
		} else {
			story.EnrichedDescription = enrichResp.EnrichedDescription
			story.Description = enrichResp.EnrichedDescription // 使用丰富后的描述
			story.IsAIEnriched = true
			story.AIEnrichedAt = &now
			story.TextTokensUsed = enrichResp.TokensUsed
			story.TokensUsed += enrichResp.TokensUsed
			s.logger.Info("story description enriched successfully",
				zap.String("storyID", story.ID),
				zap.Int("originalLength", len(req.Description)),
				zap.Int("enrichedLength", len(enrichResp.EnrichedDescription)),
				zap.Int("tokensUsed", enrichResp.TokensUsed))
		}
	} else {
		s.logger.Debug("AI enrichment not requested or description is empty",
			zap.String("storyID", story.ID),
			zap.Bool("useAIEnrich", req.UseAIEnrich),
			zap.Bool("hasDescription", req.Description != ""))
	}

	// 如果用户选择使用AI生成封面/海报/背景
	if req.GenerateCover || req.GeneratePoster || req.GenerateBackground {
		s.logger.Debug("AI cover generation requested",
			zap.String("storyID", story.ID),
			zap.Bool("generateCover", req.GenerateCover),
			zap.Bool("generatePoster", req.GeneratePoster),
			zap.Bool("generateBackground", req.GenerateBackground),
			zap.Any("style", req.AIStyle),
			zap.String("aspectRatio", req.CoverAspectRatio))
		// 如果 Style 字符串有值但 AIStyle 为空，创建一个简单的 StyleConfig
		styleToUse := req.AIStyle
		if styleToUse == nil && req.Style != "" {
			styleToUse = &domain.StyleConfig{
				Style: req.Style,
			}
		}

		coverResp, err := s.GenerateStoryCover(ctx, userID, story.ID, GenerateStoryCoverRequest{
			Title:              req.Title,
			Description:        story.Description,
			Genre:              req.Genre,
			Style:              styleToUse,
			AspectRatio:        req.CoverAspectRatio,
			GenerateCover:      req.GenerateCover,
			GeneratePoster:     req.GeneratePoster,
			GenerateBackground: req.GenerateBackground,
		})
		if err != nil {
			s.logger.Warn("failed to generate story cover with AI, continuing without cover",
				zap.String("storyID", story.ID),
				zap.String("userID", userID),
				zap.Error(err))
		} else {
			imagesGenerated := 0
			if coverResp.CoverURL != "" {
				story.CoverImage = coverResp.CoverURL
				story.CoverGeneratedByAI = true
				imagesGenerated++
				s.logger.Debug("cover image assigned",
					zap.String("storyID", story.ID),
					zap.String("coverURL", coverResp.CoverURL))
			}
			if coverResp.PosterURL != "" {
				story.PosterImage = coverResp.PosterURL
				imagesGenerated++
				s.logger.Debug("poster image assigned",
					zap.String("storyID", story.ID),
					zap.String("posterURL", coverResp.PosterURL))
			}
			if coverResp.BackgroundURL != "" {
				story.BackgroundImage = coverResp.BackgroundURL
				imagesGenerated++
				s.logger.Debug("background image assigned",
					zap.String("storyID", story.ID),
					zap.String("backgroundURL", coverResp.BackgroundURL))
			}
			story.ImageTokensUsed = coverResp.TokensUsed
			story.TokensUsed += coverResp.TokensUsed
			s.logger.Info("story cover generation completed",
				zap.String("storyID", story.ID),
				zap.Int("imagesGenerated", imagesGenerated),
				zap.Int("tokensUsed", coverResp.TokensUsed))
		}
	} else {
		s.logger.Debug("AI cover generation not requested",
			zap.String("storyID", story.ID))
	}

	// 更新故事（如果有AI丰富的内容）
	if story.IsAIEnriched || story.CoverGeneratedByAI {
		s.logger.Debug("updating story with AI enriched content",
			zap.String("storyID", story.ID),
			zap.Bool("isAIEnriched", story.IsAIEnriched),
			zap.Bool("coverGeneratedByAI", story.CoverGeneratedByAI),
			zap.Int("totalTokensUsed", story.TokensUsed))
		story.UpdatedAt = time.Now().Unix()
		if err := s.repo.UpdateStory(ctx, story); err != nil {
			s.logger.Warn("failed to update story with AI enriched content",
				zap.String("storyID", story.ID),
				zap.Error(err))
		} else {
			s.logger.Debug("story updated with AI enriched content",
				zap.String("storyID", story.ID))
		}

		// 更新用户的token使用量
		if story.TokensUsed > 0 {
			s.logger.Debug("updating user token usage",
				zap.String("userID", userID),
				zap.Int("tokensUsed", story.TokensUsed))
			if err := s.updateUserTokenUsage(ctx, userID, story.TokensUsed); err != nil {
				s.logger.Warn("failed to update user token usage",
					zap.String("userID", userID),
					zap.String("storyID", story.ID),
					zap.Int("tokensUsed", story.TokensUsed),
					zap.Error(err))
			} else {
				s.logger.Debug("user token usage updated",
					zap.String("userID", userID),
					zap.Int("tokensUsed", story.TokensUsed))
			}
		} else {
			s.logger.Debug("no tokens used, skipping token usage update",
				zap.String("userID", userID),
				zap.String("storyID", story.ID))
		}
	} else {
		s.logger.Debug("no AI enriched content to update",
			zap.String("storyID", story.ID))
	}

	// 如果故事属于群组，记录群组活动
	if req.GroupID != "" {
		s.logger.Debug("recording group activity for story creation",
			zap.String("storyID", story.ID),
			zap.String("groupId", req.GroupID))
		go s.RecordGroupStoryCreated(context.Background(), req.GroupID, userID, story.ID, story.Title)

		// 增加群组的故事计数
		if err := s.repo.IncrementGroupStoryCount(ctx, req.GroupID); err != nil {
			s.logger.Warn("failed to increment group story count",
				zap.String("groupId", req.GroupID),
				zap.String("storyId", story.ID),
				zap.Error(err))
		}
	} else {
		s.logger.Debug("story not in group, skipping group activity",
			zap.String("storyID", story.ID))
	}

	// 添加标签（如果有）
	if len(req.Tags) > 0 {
		s.logger.Debug("adding tags to story",
			zap.String("storyID", story.ID),
			zap.Strings("tags", req.Tags))
		// 去重并规范化标签（转小写，去除前后空格）
		uniqueTags := make(map[string]bool)
		normalizedTags := make([]string, 0, len(req.Tags))
		for _, tag := range req.Tags {
			normalized := strings.TrimSpace(strings.ToLower(tag))
			if normalized != "" && !uniqueTags[normalized] {
				uniqueTags[normalized] = true
				normalizedTags = append(normalizedTags, normalized)
			}
		}
		if len(normalizedTags) > 0 {
			if err := s.AddStoryTags(ctx, story.ID, normalizedTags); err != nil {
				s.logger.Warn("failed to add tags to story",
					zap.String("storyID", story.ID),
					zap.Strings("tags", normalizedTags),
					zap.Error(err))
				// 不返回错误，标签添加失败不影响故事创建
			} else {
				s.logger.Info("tags added to story successfully",
					zap.String("storyID", story.ID),
					zap.Strings("tags", normalizedTags))
			}
		}
	}

	// 记录用户活动
	s.logger.Debug("recording user activity for story creation",
		zap.String("userID", userID),
		zap.String("storyID", story.ID))
	go s.RecordStoryCreated(context.Background(), userID, story.ID, story.Title)

	return story, nil
}

// GetStory 获取故事详情
func (s *Service) GetStory(ctx context.Context, storyID string) (*domain.Story, error) {
	s.logger.Info("getting story",
		zap.String("storyID", storyID))

	story, err := s.repo.StoryByID(ctx, storyID)
	if err != nil {
		if err == domain.ErrNotFound {
			s.logger.Warn("story not found",
				zap.String("storyID", storyID))
			return nil, errors.New("story not found")
		}
		s.logger.Error("failed to get story",
			zap.String("storyID", storyID),
			zap.Error(err))
		return nil, errors.New("failed to get story")
	}

	s.logger.Debug("story retrieved",
		zap.String("storyID", storyID),
		zap.String("title", story.Title),
		zap.String("status", story.Status),
		zap.String("genre", story.Genre))

	// Fetch story characters
	s.logger.Debug("fetching story characters",
		zap.String("storyID", storyID))
	characters, err := s.repo.CharactersByStory(ctx, storyID)
	if err != nil {
		s.logger.Warn("failed to fetch story characters",
			zap.String("storyID", storyID),
			zap.Error(err))
	} else {
		story.Characters = characters
		s.logger.Debug("story characters fetched",
			zap.String("storyID", storyID),
			zap.Int("characterCount", len(characters)))
	}

	// Fetch story scenes
	s.logger.Debug("fetching story scenes",
		zap.String("storyID", storyID))
	scenes, err := s.repo.StoryScenes(ctx, storyID, 100, 0)
	if err != nil {
		s.logger.Warn("failed to fetch story scenes",
			zap.String("storyID", storyID),
			zap.Error(err))
	} else {
		story.Scenes = scenes
		s.logger.Debug("story scenes fetched",
			zap.String("storyID", storyID),
			zap.Int("sceneCount", len(scenes)))
	}

	// Fetch contributors
	s.logger.Debug("fetching story contributors",
		zap.String("storyID", storyID))
	contributors, err := s.repo.GetStoryContributors(ctx, storyID, 100, 0)
	if err != nil {
		s.logger.Warn("failed to fetch story contributors",
			zap.String("storyID", storyID),
			zap.Error(err))
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
		s.logger.Debug("story contributors fetched",
			zap.String("storyID", storyID),
			zap.Int("contributorCount", len(contributors)))

		// Record metrics - update participant count
		if s.metrics != nil {
			s.metrics.RecordStoryParticipantCount(storyID, float64(len(contributors)))
		}
	}

	s.logger.Info("story retrieved successfully",
		zap.String("storyID", storyID),
		zap.Int("characterCount", len(story.Characters)),
		zap.Int("sceneCount", len(story.Scenes)),
		zap.Int("contributorCount", len(story.Contributors)))

	return story, nil
}

// ListStories 获取故事列表
func (s *Service) ListStories(ctx context.Context, req StoryListRequest) ([]*domain.Story, int64, error) {
	s.logger.Info("listing stories",
		zap.String("status", req.Status),
		zap.String("genre", req.Genre),
		zap.String("authorID", req.AuthorID),
		zap.String("groupId", req.GroupID),
		zap.String("search", req.Search),
		zap.Int("limit", req.Limit),
		zap.Int("offset", req.Offset))

	// 设置默认分页参数
	if req.Limit == 0 {
		req.Limit = 20
		s.logger.Debug("using default limit",
			zap.Int("limit", req.Limit))
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

	s.logger.Debug("querying stories with filter",
		zap.String("status", filter.Status),
		zap.String("genre", filter.Genre),
		zap.Int("limit", filter.Limit),
		zap.Int("offset", filter.Offset))

	stories, total, err := s.repo.ListStories(ctx, filter)
	if err != nil {
		s.logger.Error("failed to list stories",
			zap.String("status", req.Status),
			zap.String("genre", req.Genre),
			zap.Error(err))
		return nil, 0, errors.New("failed to list stories")
	}

	s.logger.Info("stories listed successfully",
		zap.Int("count", len(stories)),
		zap.Int64("total", total),
		zap.Int("limit", req.Limit),
		zap.Int("offset", req.Offset))

	return stories, total, nil
}

// UpdateStory 更新故事
func (s *Service) UpdateStory(ctx context.Context, userID, storyID string, req UpdateStoryRequest) (*domain.Story, error) {
	s.logger.Info("updating story",
		zap.String("userID", userID),
		zap.String("storyID", storyID),
		zap.Bool("hasTitle", req.Title != nil),
		zap.Bool("hasDescription", req.Description != nil),
		zap.Bool("hasCoverImage", req.CoverImage != nil),
		zap.Bool("hasGenre", req.Genre != nil),
		zap.Bool("hasStatus", req.Status != nil))

	// 获取故事
	story, err := s.repo.StoryByID(ctx, storyID)
	if err != nil {
		if err == domain.ErrNotFound {
			s.logger.Warn("story not found for update",
				zap.String("storyID", storyID),
				zap.String("userID", userID))
			return nil, errors.New("story not found")
		}
		s.logger.Error("failed to get story for update",
			zap.String("storyID", storyID),
			zap.Error(err))
		return nil, errors.New("failed to get story")
	}

	s.logger.Debug("story retrieved for update",
		zap.String("storyID", storyID),
		zap.String("currentTitle", story.Title),
		zap.String("currentStatus", story.Status))

	// 验证权限
	if story.Author.ID != userID {
		s.logger.Warn("unauthorized story update attempt",
			zap.String("userID", userID),
			zap.String("storyID", storyID),
			zap.String("authorID", story.Author.ID))
		return nil, errors.New("unauthorized")
	}

	s.logger.Debug("authorization verified for story update",
		zap.String("storyID", storyID),
		zap.String("userID", userID))

	// 更新字段
	fieldsUpdated := []string{}
	if req.Title != nil {
		oldTitle := story.Title
		story.Title = *req.Title
		fieldsUpdated = append(fieldsUpdated, "title")
		s.logger.Debug("title updated",
			zap.String("storyID", storyID),
			zap.String("oldTitle", oldTitle),
			zap.String("newTitle", story.Title))
	}
	if req.Description != nil {
		story.Description = *req.Description
		fieldsUpdated = append(fieldsUpdated, "description")
		s.logger.Debug("description updated",
			zap.String("storyID", storyID),
			zap.Int("newLength", len(story.Description)))
	}
	if req.CoverImage != nil {
		story.CoverImage = *req.CoverImage
		fieldsUpdated = append(fieldsUpdated, "coverImage")
		s.logger.Debug("cover image updated",
			zap.String("storyID", storyID),
			zap.String("coverURL", story.CoverImage))
	}
	if req.Genre != nil {
		oldGenre := story.Genre
		story.Genre = *req.Genre
		fieldsUpdated = append(fieldsUpdated, "genre")
		s.logger.Debug("genre updated",
			zap.String("storyID", storyID),
			zap.String("oldGenre", oldGenre),
			zap.String("newGenre", story.Genre))
	}
	if req.Status != nil {
		oldStatus := story.Status
		story.Status = *req.Status
		fieldsUpdated = append(fieldsUpdated, "status")
		s.logger.Debug("status updated",
			zap.String("storyID", storyID),
			zap.String("oldStatus", oldStatus),
			zap.String("newStatus", story.Status))
	}
	if req.IsCollaborationOpen != nil {
		oldIsOpen := story.IsCollaborationOpen
		story.IsCollaborationOpen = *req.IsCollaborationOpen
		fieldsUpdated = append(fieldsUpdated, "isCollaborationOpen")
		s.logger.Debug("isCollaborationOpen updated",
			zap.String("storyID", storyID),
			zap.Bool("oldValue", oldIsOpen),
			zap.Bool("newValue", *req.IsCollaborationOpen))
	}

	if len(fieldsUpdated) == 0 {
		s.logger.Debug("no fields to update",
			zap.String("storyID", storyID))
		return story, nil
	}

	story.UpdatedAt = time.Now().Unix()

	s.logger.Debug("saving story updates to database",
		zap.String("storyID", storyID),
		zap.Strings("fieldsUpdated", fieldsUpdated))
	if err := s.repo.UpdateStory(ctx, story); err != nil {
		s.logger.Error("failed to update story in database",
			zap.String("storyID", storyID),
			zap.Strings("fieldsUpdated", fieldsUpdated),
			zap.Error(err))
		return nil, errors.New("failed to update story")
	}

	s.logger.Info("story updated successfully",
		zap.String("storyID", storyID),
		zap.Strings("fieldsUpdated", fieldsUpdated))
	return story, nil
}

// DeleteStory 删除故事
func (s *Service) DeleteStory(ctx context.Context, userID, storyID string) error {
	s.logger.Info("deleting story",
		zap.String("userID", userID),
		zap.String("storyID", storyID))

	// 获取故事
	story, err := s.repo.StoryByID(ctx, storyID)
	if err != nil {
		if err == domain.ErrNotFound {
			s.logger.Warn("story not found for deletion",
				zap.String("storyID", storyID),
				zap.String("userID", userID))
			return errors.New("story not found")
		}
		s.logger.Error("failed to get story for deletion",
			zap.String("storyID", storyID),
			zap.Error(err))
		return errors.New("failed to get story")
	}

	s.logger.Debug("story retrieved for deletion",
		zap.String("storyID", storyID),
		zap.String("title", story.Title),
		zap.String("status", story.Status))

	// 验证权限
	if story.Author.ID != userID {
		s.logger.Warn("unauthorized story delete attempt",
			zap.String("userID", userID),
			zap.String("storyID", storyID),
			zap.String("authorID", story.Author.ID))
		return errors.New("unauthorized")
	}

	s.logger.Debug("authorization verified for story deletion",
		zap.String("storyID", storyID),
		zap.String("userID", userID))

	s.logger.Debug("deleting story from database",
		zap.String("storyID", storyID))
	if err := s.repo.DeleteStory(ctx, storyID); err != nil {
		s.logger.Error("failed to delete story from database",
			zap.String("storyID", storyID),
			zap.Error(err))
		return errors.New("failed to delete story")
	}

	s.logger.Info("story deleted successfully",
		zap.String("storyID", storyID),
		zap.String("userID", userID))

	// 如果故事属于群组，减少群组的故事计数
	if story.GroupID != "" {
		if err := s.repo.DecrementGroupStoryCount(ctx, story.GroupID); err != nil {
			s.logger.Warn("failed to decrement group story count",
				zap.String("groupId", story.GroupID),
				zap.String("storyId", storyID),
				zap.Error(err))
		}
	}

	return nil
}

// LikeStory 点赞故事
func (s *Service) LikeStory(ctx context.Context, userID, storyID string) error {
	s.logger.Info("liking story",
		zap.String("userID", userID),
		zap.String("storyID", storyID))

	// 验证故事存在
	story, err := s.repo.StoryByID(ctx, storyID)
	if err != nil {
		if err == domain.ErrNotFound {
			s.logger.Warn("story not found for like",
				zap.String("storyID", storyID),
				zap.String("userID", userID))
			return errors.New("story not found")
		}
		s.logger.Error("failed to get story for like",
			zap.String("storyID", storyID),
			zap.Error(err))
		return errors.New("failed to get story")
	}

	s.logger.Debug("story verified for like",
		zap.String("storyID", storyID),
		zap.String("title", story.Title))

	s.logger.Debug("executing like operation",
		zap.String("userID", userID),
		zap.String("storyID", storyID))
	if err := s.repo.LikeStory(ctx, userID, storyID); err != nil {
		// Treat "already liked" as success (idempotent operation)
		if errors.Is(err, domain.ErrAlreadyLiked) {
			s.logger.Info("story already liked",
				zap.String("userID", userID),
				zap.String("storyID", storyID))
			return nil
		}
		s.logger.Error("failed to like story",
			zap.String("userID", userID),
			zap.String("storyID", storyID),
			zap.Error(err))
		return errors.New("failed to like story")
	}

	s.logger.Info("story liked successfully",
		zap.String("userID", userID),
		zap.String("storyID", storyID))
	return nil
}

// UnlikeStory 取消点赞故事
func (s *Service) UnlikeStory(ctx context.Context, userID, storyID string) error {
	s.logger.Info("unliking story",
		zap.String("userID", userID),
		zap.String("storyID", storyID))

	s.logger.Debug("executing unlike operation",
		zap.String("userID", userID),
		zap.String("storyID", storyID))
	if err := s.repo.UnlikeStory(ctx, userID, storyID); err != nil {
		// Treat "not liked" as success (idempotent operation)
		if errors.Is(err, domain.ErrNotFound) {
			s.logger.Info("story not liked (idempotent)",
				zap.String("userID", userID),
				zap.String("storyID", storyID))
			return nil
		}
		s.logger.Error("failed to unlike story",
			zap.String("userID", userID),
			zap.String("storyID", storyID),
			zap.Error(err))
		return errors.New("failed to unlike story")
	}

	s.logger.Info("story unliked successfully",
		zap.String("userID", userID),
		zap.String("storyID", storyID))
	return nil
}

// FollowStory 关注故事
func (s *Service) FollowStory(ctx context.Context, userID, storyID string) error {
	s.logger.Info("following story",
		zap.String("userID", userID),
		zap.String("storyID", storyID))

	// 验证故事存在
	story, err := s.repo.StoryByID(ctx, storyID)
	if err != nil {
		if err == domain.ErrNotFound {
			s.logger.Warn("story not found for follow",
				zap.String("storyID", storyID),
				zap.String("userID", userID))
			return errors.New("story not found")
		}
		s.logger.Error("failed to get story for follow",
			zap.String("storyID", storyID),
			zap.Error(err))
		return errors.New("failed to get story")
	}

	s.logger.Debug("story verified for follow",
		zap.String("storyID", storyID),
		zap.String("title", story.Title))

	s.logger.Debug("executing follow operation",
		zap.String("userID", userID),
		zap.String("storyID", storyID))
	if err := s.repo.FollowStory(ctx, userID, storyID); err != nil {
		// Treat "already following" as success (idempotent operation)
		if errors.Is(err, domain.ErrAlreadyExists) {
			s.logger.Info("story already followed",
				zap.String("userID", userID),
				zap.String("storyID", storyID))
			return nil
		}
		s.logger.Error("failed to follow story",
			zap.String("userID", userID),
			zap.String("storyID", storyID),
			zap.Error(err))
		return errors.New("failed to follow story")
	}

	s.logger.Info("story followed successfully",
		zap.String("userID", userID),
		zap.String("storyID", storyID))
	return nil
}

// UnfollowStory 取消关注故事
func (s *Service) UnfollowStory(ctx context.Context, userID, storyID string) error {
	s.logger.Info("unfollowing story",
		zap.String("userID", userID),
		zap.String("storyID", storyID))

	s.logger.Debug("executing unfollow operation",
		zap.String("userID", userID),
		zap.String("storyID", storyID))
	if err := s.repo.UnfollowStory(ctx, userID, storyID); err != nil {
		// Treat "not following" as success (idempotent operation)
		if errors.Is(err, domain.ErrNotFound) {
			s.logger.Info("story not followed (idempotent)",
				zap.String("userID", userID),
				zap.String("storyID", storyID))
			return nil
		}
		s.logger.Error("failed to unfollow story",
			zap.String("userID", userID),
			zap.String("storyID", storyID),
			zap.Error(err))
		return errors.New("failed to unfollow story")
	}

	s.logger.Info("story unfollowed successfully",
		zap.String("userID", userID),
		zap.String("storyID", storyID))
	return nil
}

// ========== 故事 AI 渲染功能（同步） ==========

// RenderStoryRequest AI渲染故事请求（同步模式）
// 用于丰富故事描述和生成背景图片
type RenderStoryRequest struct {
	// AI 丰富选项
	EnrichDescription  bool                `json:"enrichDescription"`     // 是否丰富故事描述
	GenerateBackground bool                `json:"generateBackground"`    // 是否生成背景图片
	GenerateCover      bool                `json:"generateCover"`         // 是否生成封面图片
	Style              *domain.StyleConfig `json:"style,omitempty"`       // AI 生成风格配置（完整信息，可为空）
	AspectRatio        string              `json:"aspectRatio,omitempty"` // 图片比例：1:1, 16:9, 9:16, etc.
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
			s.logger.Warn("story not found for rendering",
				zap.String("storyID", storyID),
				zap.String("userID", userID))
			return nil, errors.New("story not found")
		}
		s.logger.Error("failed to get story for rendering",
			zap.String("storyID", storyID),
			zap.String("userID", userID),
			zap.Error(err))
		return nil, errors.New("failed to get story")
	}

	s.logger.Debug("story retrieved for rendering",
		zap.String("storyID", storyID),
		zap.String("title", story.Title),
		zap.String("status", story.Status))

	if story.Author.ID != userID {
		s.logger.Warn("unauthorized render attempt",
			zap.String("userID", userID),
			zap.String("storyID", storyID),
			zap.String("authorID", story.Author.ID))
		return nil, errors.New("unauthorized: not story owner")
	}

	s.logger.Debug("authorization verified for rendering",
		zap.String("storyID", storyID),
		zap.String("userID", userID))

	// 检查是否有需要处理的任务
	if !req.EnrichDescription && !req.GenerateBackground && !req.GenerateCover {
		s.logger.Warn("no render options selected",
			zap.String("storyID", storyID),
			zap.Bool("enrichDescription", req.EnrichDescription),
			zap.Bool("generateBackground", req.GenerateBackground),
			zap.Bool("generateCover", req.GenerateCover))
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
	s.logger.Debug("fetching story for media rendering",
		zap.String("storyID", storyID),
		zap.String("userID", userID))
	story, err := s.repo.StoryByID(ctx, storyID)
	if err != nil {
		if err == domain.ErrNotFound {
			s.logger.Warn("story not found for media rendering",
				zap.String("storyID", storyID),
				zap.String("userID", userID))
			return nil, errors.New("story not found")
		}
		s.logger.Error("failed to get story for media rendering",
			zap.String("storyID", storyID),
			zap.String("userID", userID),
			zap.Error(err))
		return nil, errors.New("failed to get story")
	}

	s.logger.Debug("story retrieved for media rendering",
		zap.String("storyID", storyID),
		zap.String("title", story.Title),
		zap.Int("panels", story.Panels))

	if story.Author.ID != userID {
		s.logger.Warn("unauthorized media render attempt",
			zap.String("userID", userID),
			zap.String("storyID", storyID),
			zap.String("authorID", story.Author.ID))
		return nil, errors.New("unauthorized: not story owner")
	}

	s.logger.Debug("authorization verified for media rendering",
		zap.String("storyID", storyID),
		zap.String("userID", userID))

	// 检查故事是否有内容可以渲染
	if story.Panels == 0 {
		s.logger.Warn("story has no panels to render",
			zap.String("storyID", storyID),
			zap.String("userID", userID))
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
	s.logger.Debug("serializing render config",
		zap.String("storyID", storyID),
		zap.String("type", string(config.Type)))
	configJSON, err := utils.JSONMarshal(config)
	if err != nil {
		s.logger.Error("failed to serialize render config",
			zap.String("storyID", storyID),
			zap.String("type", string(config.Type)),
			zap.Error(err))
		return nil, errors.New("failed to serialize config")
	}

	// 创建渲染任务
	task := &domain.RenderTask{
		ID:        utils.GenerateID(),
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
	s.logger.Debug("creating render task in database",
		zap.String("storyID", storyID),
		zap.String("taskID", task.ID),
		zap.String("type", string(task.Type)))
	if err := s.repo.CreateRenderTask(ctx, task); err != nil {
		s.logger.Error("failed to create render task",
			zap.String("storyID", storyID),
			zap.String("taskID", task.ID),
			zap.String("type", string(task.Type)),
			zap.Error(err))
		return nil, errors.New("failed to create render task")
	}

	s.logger.Debug("render task created successfully",
		zap.String("taskID", task.ID),
		zap.String("storyID", storyID))

	// 更新故事状态为渲染中
	s.logger.Debug("updating story status to rendering",
		zap.String("storyID", storyID))
	_, err = s.UpdateStory(ctx, userID, storyID, UpdateStoryRequest{
		Status: utils.StringPtr("rendering"),
	})
	if err != nil {
		s.logger.Warn("failed to update story status to rendering",
			zap.String("storyID", storyID),
			zap.String("taskID", task.ID),
			zap.Error(err))
		// 不中断流程，任务已创建
	} else {
		s.logger.Debug("story status updated to rendering",
			zap.String("storyID", storyID))
	}

	// 启动异步渲染
	s.logger.Info("starting asynchronous media render task",
		zap.String("taskID", task.ID),
		zap.String("storyID", storyID),
		zap.String("type", string(task.Type)))
	go s.processMediaRenderTask(context.Background(), task)

	s.logger.Info("media render task created and started",
		zap.String("taskID", task.ID),
		zap.String("storyID", storyID),
		zap.String("type", string(task.Type)),
		zap.String("status", string(task.Status)))

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
	s.logger.Info("starting render process",
		zap.String("taskID", task.ID),
		zap.String("type", string(task.Type)),
		zap.Int("storyboardCount", len(storyboards)))
	var renderErr error
	switch task.Type {
	case domain.RenderTaskTypeVideo:
		s.logger.Debug("rendering video content",
			zap.String("taskID", task.ID))
		renderErr = s.renderVideoContent(ctx, task, storyboards, &config)
	case domain.RenderTaskTypeImageSet:
		s.logger.Debug("rendering image set content",
			zap.String("taskID", task.ID))
		renderErr = s.renderImageSetContent(ctx, task, storyboards, &config)
	case domain.RenderTaskTypeAnimation:
		s.logger.Debug("rendering animation content",
			zap.String("taskID", task.ID))
		renderErr = s.renderAnimationContent(ctx, task, storyboards, &config)
	default:
		s.logger.Error("unknown render type",
			zap.String("taskID", task.ID),
			zap.String("type", string(task.Type)))
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
	s.logger.Debug("generating output URLs",
		zap.String("taskID", task.ID),
		zap.String("type", string(task.Type)))
	task.OutputURL = fmt.Sprintf("/uploads/renders/%s.%s", task.ID, utils.FileExtensionForRenderType(task.Type))
	task.ThumbnailURL = fmt.Sprintf("/uploads/renders/%s_thumb.jpg", task.ID)
	task.FileSize = 10485760 // 10MB（示例）

	if task.Type == domain.RenderTaskTypeVideo {
		task.Duration = 120 // 2分钟（示例）
		task.Resolution = "1080p"
		s.logger.Debug("video render metadata set",
			zap.String("taskID", task.ID),
			zap.Int("duration", task.Duration),
			zap.String("resolution", task.Resolution))
	}

	// 完成
	s.logger.Info("marking render task as completed",
		zap.String("taskID", task.ID),
		zap.String("outputURL", task.OutputURL))
	task.Status = domain.RenderTaskStatusCompleted
	task.Progress = 100
	completedTime := time.Now().Unix()
	task.CompletedAt = &completedTime
	task.UpdatedAt = time.Now().Unix()
	if err := s.repo.UpdateRenderTask(ctx, task); err != nil {
		s.logger.Error("failed to update render task completion",
			zap.String("taskID", task.ID),
			zap.Error(err))
		return
	}

	s.logger.Debug("render task status updated to completed",
		zap.String("taskID", task.ID))

	// 更新故事状态为 rendered（渲染完成，等待用户发布）
	s.logger.Debug("updating story status to rendered",
		zap.String("taskID", task.ID),
		zap.String("storyID", task.StoryID))
	_, err = s.UpdateStory(ctx, task.UserID, task.StoryID, UpdateStoryRequest{
		Status: utils.StringPtr("rendered"),
	})
	if err != nil {
		s.logger.Warn("failed to update story status after render",
			zap.String("taskID", task.ID),
			zap.String("storyID", task.StoryID),
			zap.Error(err))
		// 不中断流程，任务已完成
	} else {
		s.logger.Debug("story status updated to rendered",
			zap.String("storyID", task.StoryID))
	}

	s.logger.Info("media render task completed",
		zap.String("taskID", task.ID),
		zap.Duration("duration", time.Duration(time.Now().Unix()-startTime)*time.Second),
	)
}

// renderVideoContent 渲染视频内容
func (s *Service) renderVideoContent(ctx context.Context, task *domain.RenderTask, storyboards []*domain.Storyboard, config *domain.RenderConfig) error {
	s.logger.Info("rendering video content",
		zap.String("taskID", task.ID),
		zap.Int("storyboardCount", len(storyboards)),
		zap.String("resolution", config.Resolution),
		zap.String("quality", config.Quality))
	totalSteps := len(storyboards)
	baseProgress := 20

	for i := range storyboards {
		// 检查取消
		select {
		case <-ctx.Done():
			s.logger.Warn("render context cancelled",
				zap.String("taskID", task.ID),
				zap.Error(ctx.Err()))
			return ctx.Err()
		default:
		}

		if cancelled := s.isRenderTaskCancelled(ctx, task.ID); cancelled {
			s.logger.Info("render task cancelled by user",
				zap.String("taskID", task.ID))
			return errors.New("task cancelled by user")
		}

		progress := baseProgress + ((i + 1) * 60 / totalSteps)
		if err := s.repo.UpdateRenderTaskProgress(ctx, task.ID, progress); err != nil {
			s.logger.Warn("failed to update render progress",
				zap.String("taskID", task.ID),
				zap.Int("progress", progress),
				zap.Error(err))
		} else {
			s.logger.Debug("render progress updated",
				zap.String("taskID", task.ID),
				zap.Int("progress", progress),
				zap.Int("currentStep", i+1),
				zap.Int("totalSteps", totalSteps))
		}

		// TODO: 实际的视频生成逻辑
		time.Sleep(500 * time.Millisecond)
	}

	s.logger.Debug("finalizing video render",
		zap.String("taskID", task.ID))
	if err := s.repo.UpdateRenderTaskProgress(ctx, task.ID, 90); err != nil {
		s.logger.Warn("failed to update final render progress",
			zap.String("taskID", task.ID),
			zap.Error(err))
	}

	s.logger.Info("video content rendered successfully",
		zap.String("taskID", task.ID))
	return nil
}

// renderImageSetContent 渲染图片集内容
func (s *Service) renderImageSetContent(ctx context.Context, task *domain.RenderTask, storyboards []*domain.Storyboard, config *domain.RenderConfig) error {
	s.logger.Info("rendering image set content",
		zap.String("taskID", task.ID),
		zap.Int("storyboardCount", len(storyboards)),
		zap.String("quality", config.Quality))
	totalSteps := len(storyboards)
	baseProgress := 20

	for i := range storyboards {
		// 检查取消
		select {
		case <-ctx.Done():
			s.logger.Warn("render context cancelled",
				zap.String("taskID", task.ID),
				zap.Error(ctx.Err()))
			return ctx.Err()
		default:
		}

		if cancelled := s.isRenderTaskCancelled(ctx, task.ID); cancelled {
			s.logger.Info("render task cancelled by user",
				zap.String("taskID", task.ID))
			return errors.New("task cancelled by user")
		}

		progress := baseProgress + ((i + 1) * 70 / totalSteps)
		if err := s.repo.UpdateRenderTaskProgress(ctx, task.ID, progress); err != nil {
			s.logger.Warn("failed to update render progress",
				zap.String("taskID", task.ID),
				zap.Int("progress", progress),
				zap.Error(err))
		} else {
			s.logger.Debug("render progress updated",
				zap.String("taskID", task.ID),
				zap.Int("progress", progress),
				zap.Int("currentStep", i+1),
				zap.Int("totalSteps", totalSteps))
		}

		// TODO: 实际的图片集生成逻辑
		time.Sleep(300 * time.Millisecond)
	}

	s.logger.Debug("finalizing image set render",
		zap.String("taskID", task.ID))
	if err := s.repo.UpdateRenderTaskProgress(ctx, task.ID, 90); err != nil {
		s.logger.Warn("failed to update final render progress",
			zap.String("taskID", task.ID),
			zap.Error(err))
	}

	s.logger.Info("image set content rendered successfully",
		zap.String("taskID", task.ID))
	return nil
}

// renderAnimationContent 渲染动画内容
func (s *Service) renderAnimationContent(ctx context.Context, task *domain.RenderTask, storyboards []*domain.Storyboard, config *domain.RenderConfig) error {
	s.logger.Info("rendering animation content",
		zap.String("taskID", task.ID),
		zap.Int("storyboardCount", len(storyboards)),
		zap.String("quality", config.Quality))
	totalSteps := len(storyboards)
	baseProgress := 20

	for i := range storyboards {
		// 检查取消
		select {
		case <-ctx.Done():
			s.logger.Warn("render context cancelled",
				zap.String("taskID", task.ID),
				zap.Error(ctx.Err()))
			return ctx.Err()
		default:
		}

		if cancelled := s.isRenderTaskCancelled(ctx, task.ID); cancelled {
			s.logger.Info("render task cancelled by user",
				zap.String("taskID", task.ID))
			return errors.New("task cancelled by user")
		}

		progress := baseProgress + ((i + 1) * 50 / totalSteps)
		if err := s.repo.UpdateRenderTaskProgress(ctx, task.ID, progress); err != nil {
			s.logger.Warn("failed to update render progress",
				zap.String("taskID", task.ID),
				zap.Int("progress", progress),
				zap.Error(err))
		} else {
			s.logger.Debug("render progress updated",
				zap.String("taskID", task.ID),
				zap.Int("progress", progress),
				zap.Int("currentStep", i+1),
				zap.Int("totalSteps", totalSteps))
		}

		// TODO: 实际的动画生成逻辑
		time.Sleep(400 * time.Millisecond)
	}

	s.logger.Debug("finalizing animation render",
		zap.String("taskID", task.ID))
	if err := s.repo.UpdateRenderTaskProgress(ctx, task.ID, 90); err != nil {
		s.logger.Warn("failed to update final render progress",
			zap.String("taskID", task.ID),
			zap.Error(err))
	}

	s.logger.Info("animation content rendered successfully",
		zap.String("taskID", task.ID))
	return nil
}

// isRenderTaskCancelled 检查渲染任务是否被用户取消
func (s *Service) isRenderTaskCancelled(ctx context.Context, taskID string) bool {
	currentTask, err := s.repo.GetRenderTask(ctx, taskID)
	if err != nil {
		s.logger.Debug("failed to check render task cancellation status",
			zap.String("taskID", taskID),
			zap.Error(err))
		return false
	}
	isCancelled := currentTask != nil && currentTask.Status == domain.RenderTaskStatusCancelled
	if isCancelled {
		s.logger.Debug("render task is cancelled",
			zap.String("taskID", taskID))
	}
	return isCancelled
}

// markRenderTaskFailed 将渲染任务标记为失败并恢复故事状态
func (s *Service) markRenderTaskFailed(ctx context.Context, task *domain.RenderTask, errMsg string) {
	s.logger.Info("marking render task as failed",
		zap.String("taskID", task.ID),
		zap.String("storyID", task.StoryID),
		zap.String("error", errMsg))
	task.Status = domain.RenderTaskStatusFailed
	task.ErrorMessage = errMsg
	task.UpdatedAt = time.Now().Unix()

	// 使用新的 context 防止原 context 已取消
	updateCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	s.logger.Debug("updating render task status to failed",
		zap.String("taskID", task.ID))
	if err := s.repo.UpdateRenderTask(updateCtx, task); err != nil {
		s.logger.Error("failed to mark render task as failed",
			zap.String("taskID", task.ID),
			zap.String("storyID", task.StoryID),
			zap.Error(err))
	} else {
		s.logger.Debug("render task marked as failed",
			zap.String("taskID", task.ID))
	}

	s.logger.Debug("restoring story status after render failure",
		zap.String("taskID", task.ID),
		zap.String("storyID", task.StoryID))
	s.restoreStoryStatus(updateCtx, task)
}

// restoreStoryStatus 恢复故事状态（从 rendering 恢复为 draft）
func (s *Service) restoreStoryStatus(ctx context.Context, task *domain.RenderTask) {
	s.logger.Debug("restoring story status to draft",
		zap.String("storyID", task.StoryID),
		zap.String("taskID", task.ID))
	_, err := s.UpdateStory(ctx, task.UserID, task.StoryID, UpdateStoryRequest{
		Status: utils.StringPtr("draft"),
	})
	if err != nil {
		s.logger.Error("failed to restore story status after render failure/cancellation",
			zap.String("storyID", task.StoryID),
			zap.String("taskID", task.ID),
			zap.Error(err))
	} else {
		s.logger.Debug("story status restored to draft",
			zap.String("storyID", task.StoryID))
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
		ID:          utils.GenerateID(),
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
		Status: utils.StringPtr("published"),
	})
	if err != nil {
		s.logger.Error("failed to update story status", zap.Error(err))
		return nil, errors.New("failed to update story status")
	}

	// 发布成功后刷新统计：
	// 1) 用户创建的故事板计数（User.StoryboardCount）
	// 2) 故事的故事板计数（Story.StoryboardCount）
	// 3) 参与故事板的角色统计（Character.Stories）
	if err := s.refreshPublishStoryStats(ctx, userID, storyID); err != nil {
		s.logger.Warn("failed to refresh publish story stats (non-fatal)",
			zap.String("userID", userID),
			zap.String("storyID", storyID),
			zap.Error(err))
	}

	s.logger.Info("story published successfully",
		zap.String("storyID", storyID),
		zap.String("publicationID", publication.ID),
		zap.Int("version", nextVersion),
	)

	return publication, nil
}

func (s *Service) refreshPublishStoryStats(ctx context.Context, userID, storyID string) error {
	// 1) Update user's storyboard count
	userStoryboardCount, err := s.repo.CountStoryboardsByCreator(ctx, userID)
	if err != nil {
		return fmt.Errorf("count storyboards by creator: %w", err)
	}
	if u, err := s.repo.UserByID(ctx, userID); err != nil {
		return fmt.Errorf("get user: %w", err)
	} else if u != nil {
		u.StoryboardCount = int(userStoryboardCount)
		if err := s.repo.UpdateUser(ctx, u); err != nil {
			return fmt.Errorf("update user storyboard count: %w", err)
		}
	}

	// 2) Update story's storyboard count
	storyboardCount, err := s.repo.CountStoryboardsByStory(ctx, storyID)
	if err != nil {
		return fmt.Errorf("count storyboards by story: %w", err)
	}
	story, err := s.repo.StoryByID(ctx, storyID)
	if err != nil {
		return fmt.Errorf("get story: %w", err)
	}
	if story != nil {
		story.StoryboardCount = int(storyboardCount)
		if err := s.repo.UpdateStory(ctx, story); err != nil {
			return fmt.Errorf("update story storyboard count: %w", err)
		}
	}

	// 3) Update character participation counts (Character.Stories)
	charCounts, err := s.repo.CharacterStoryboardCountsByStory(ctx, storyID)
	if err != nil {
		return fmt.Errorf("count character storyboard participation: %w", err)
	}
	chars, err := s.repo.CharactersByStory(ctx, storyID)
	if err != nil {
		return fmt.Errorf("list characters by story: %w", err)
	}
	for _, ch := range chars {
		if ch == nil {
			continue
		}
		target := int(charCounts[ch.ID]) // missing -> 0
		if ch.Stories == target {
			continue
		}
		ch.Stories = target
		if err := s.repo.UpdateCharacter(ctx, ch); err != nil {
			return fmt.Errorf("update character(%s) participation count: %w", ch.ID, err)
		}
	}

	// Cache invalidation (best-effort)
	c := s.getCache()
	if c != nil {
		_ = c.Delete(ctx, cache.UserKey(userID))
		_ = c.Delete(ctx, cache.StoryKey(storyID))
		for _, ch := range chars {
			if ch == nil {
				continue
			}
			_ = c.Delete(ctx, cache.CharacterKey(ch.ID))
		}
	}

	return nil
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
		Status: utils.StringPtr("draft"),
	})
	if err != nil {
		s.logger.Error("failed to update story status", zap.Error(err))
		return errors.New("failed to update story status")
	}

	// 取消发布成功后刷新统计（幂等重算，非致命）
	if err := s.refreshPublishStoryStats(ctx, userID, storyID); err != nil {
		s.logger.Warn("failed to refresh unpublish story stats (non-fatal)",
			zap.String("userID", userID),
			zap.String("storyID", storyID),
			zap.Error(err))
	}

	s.logger.Info("story unpublished successfully", zap.String("storyID", storyID))
	return nil
}

// ========== 辅助函数 ==========

// moved to internal/utils

// ========== AI 故事丰富功能 ==========

// EnrichStoryDescription 使用AI丰富故事描述
// 根据用户输入的原始描述，生成更加丰富、生动的故事背景描述
func (s *Service) EnrichStoryDescription(ctx context.Context, userID, storyID, originalDesc, genre string, style *domain.StyleConfig) (*StoryAIEnrichResponse, error) {
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
		UserID:         userID,
		OriginalPrompt: prompt,
		SystemPrompt: `你是一位资深故事创作作家兼编辑。
你的任务：根据用户提供的故事概念与约束，生成「可直接用于故事详情页展示」的背景描述，并给出标签建议。

输出格式要求（必须严格遵守）：
1. 只输出 **纯 JSON**，不要使用 markdown 代码块，不要输出解释或多余文本
2. JSON 必须包含以下字段：
   - "enrichedDescription": string  // 300-500字左右，中文为主，可少量专有名词
   - "suggestedTags": string[]      // 5-10个标签，简短、可检索、去重

写作要求：
- 保持原始描述的核心概念与主题，不要改写成另一个故事
- 强化环境描写、氛围、世界观设定与潜在冲突/悬念
- 与用户提供的 genre / style 保持一致（若未提供则保持中性、通用）
- 不要出现“作为AI/模型/我认为”等自我指代
- 避免敏感或违规内容；不要使用表情符号
`,
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

	// 解析 JSON 输出（若解析失败则降级为纯文本）
	type enrichOut struct {
		EnrichedDescription string   `json:"enrichedDescription"`
		SuggestedTags       []string `json:"suggestedTags"`
	}

	text := strings.TrimSpace(result.Text)
	out := enrichOut{}
	if err := json.Unmarshal([]byte(text), &out); err != nil || strings.TrimSpace(out.EnrichedDescription) == "" {
		// fallback: treat as plain enriched description
		out.EnrichedDescription = text
		out.SuggestedTags = nil
	}

	// Normalize tags (best-effort)
	if len(out.SuggestedTags) > 0 {
		seen := make(map[string]struct{}, len(out.SuggestedTags))
		tags := make([]string, 0, len(out.SuggestedTags))
		for _, t := range out.SuggestedTags {
			t = strings.TrimSpace(t)
			if t == "" {
				continue
			}
			if _, ok := seen[t]; ok {
				continue
			}
			seen[t] = struct{}{}
			tags = append(tags, t)
			if len(tags) >= 10 {
				break
			}
		}
		out.SuggestedTags = tags
	}

	return &StoryAIEnrichResponse{
		EnrichedDescription: strings.TrimSpace(out.EnrichedDescription),
		SuggestedTags:       out.SuggestedTags,
		TokensUsed:          result.TokensUsed,
		RecordID:            result.RecordID,
		DurationMs:          result.DurationMs,
	}, nil
}

// buildEnrichDescriptionPrompt 构建丰富描述的提示词
func (s *Service) buildEnrichDescriptionPrompt(originalDesc, genre string, style *domain.StyleConfig) string {
	prompt := fmt.Sprintf(`你将收到一个「故事概念/背景」的原始描述。请在不改变核心设定的前提下，将其扩写为适合展示在故事详情页的「故事背景描述」。

【原始描述】
%s

【上下文约束】
`, strings.TrimSpace(originalDesc))

	if genre != "" {
		prompt += fmt.Sprintf("- 故事类型(genre)：%s\n", strings.TrimSpace(genre))
	}

	if style != nil && style.Style != "" {
		prompt += fmt.Sprintf("- 风格偏好(style)：%s\n", strings.TrimSpace(style.Style))
		if style.Description != "" {
			prompt += fmt.Sprintf("- 风格说明：%s\n", strings.TrimSpace(style.Description))
		}
	}

	prompt += `
【写作目标】
- 输出一段 300-500 字左右的中文背景描述（自然段即可）
- 强化环境描写、氛围、世界观细节，让读者“能看到画面”
- 暗示潜在冲突/悬念，但不要剧透完整情节

【硬性规则】
- 保留原始描述的核心概念/人物关系/世界设定，不要改写成另一个故事
- 不要出现“作为AI/模型/我认为”等自我指代
- 不要输出 markdown、标题编号、解释说明

【标签建议（用于检索）】
- 给出 5-10 个简短标签
- 尽量覆盖：题材/时代或世界观/氛围基调/关键主题/主要冲突方向/重要元素（如学校、海盗、机甲、魔法、都市、末日等）
- 标签去重，避免过长句子

请按 SystemPrompt 要求输出对应 JSON。`

	return prompt
}

// GenerateStoryCoverRequest 生成故事封面请求
type GenerateStoryCoverRequest struct {
	Title              string              `json:"title"`
	Description        string              `json:"description"`
	Genre              string              `json:"genre"`
	Style              *domain.StyleConfig `json:"style,omitempty"` // AI 生成风格配置（完整信息，可为空）
	AspectRatio        string              `json:"aspectRatio,omitempty"`
	GenerateCover      bool                `json:"generateCover"`
	GeneratePoster     bool                `json:"generatePoster"`
	GenerateBackground bool                `json:"generateBackground"`
}

// GenerateStoryCover 使用AI生成故事封面/海报/背景图片（两步AI工作流）
// Step 1: 使用LLM生成封面概念JSON
// Step 2: 组装最终提示词，使用图像生成AI创建图片
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

	// Step 1: 生成封面概念（共享概念用于所有图片类型）
	var coverConcept *CoverConcept
	if req.GenerateCover || req.GeneratePoster || req.GenerateBackground {
		conceptResult, err := s.generateCoverConcept(ctx, userID, storyID, req.Title, req.Description, req.Genre, req.Style)
		if err != nil {
			s.logger.Warn("failed to generate cover concept, falling back to simple prompt",
				zap.String("storyID", storyID),
				zap.Error(err))
			// 降级到简单提示词
		} else {
			coverConcept = conceptResult.Concept
			totalTokens += conceptResult.TokensUsed
			s.logger.Info("cover concept generated successfully",
				zap.String("storyID", storyID),
				zap.String("conceptRecordID", conceptResult.RecordID))
		}
	}

	// Step 2: 生成封面图
	if req.GenerateCover {
		finalPrompt := s.assembleFinalCoverPrompt(coverConcept, req.Title, "cover")

		coverResult, err := s.aiGenService.GenerateImage(ctx, &GenerateImageRequest{
			UserID:            userID,
			Prompt:            finalPrompt,
			Provider:          "gemini",
			Model:             "",
			AspectRatio:       req.AspectRatio,
			Quality:           "high",
			OutputCount:       1,
			RelatedEntityID:   storyID,
			RelatedEntityType: "story_cover",
			Metadata: map[string]interface{}{
				"operation": "generate_cover",
				"storyId":   storyID,
				"step":      2,
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

	// Step 2: 生成海报图
	if req.GeneratePoster {
		finalPrompt := s.assembleFinalCoverPrompt(coverConcept, req.Title, "poster")

		posterResult, err := s.aiGenService.GenerateImage(ctx, &GenerateImageRequest{
			UserID:            userID,
			Prompt:            finalPrompt,
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
				"step":      2,
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

	// Step 2: 生成背景图
	if req.GenerateBackground {
		finalPrompt := s.assembleFinalCoverPrompt(coverConcept, req.Title, "background")

		bgResult, err := s.aiGenService.GenerateImage(ctx, &GenerateImageRequest{
			UserID:            userID,
			Prompt:            finalPrompt,
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
				"step":      2,
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

// CoverConcept LLM生成的封面概念结构
type CoverConcept struct {
	CoverConcept struct {
		VisualSubject      string `json:"visual_subject"`      // 视觉主体（主要角色/场景/元素）
		SceneEnvironment   string `json:"scene_environment"`   // 场景环境（背景、天气、道具）
		CompositionCamera  string `json:"composition_camera"`  // 构图和镜头（角度、取景、景深）
		LightingAtmosphere string `json:"lighting_atmosphere"` // 灯光和氛围（光照类型、色彩、情绪）
		ArtStyle           string `json:"art_style"`           // 艺术风格（媒介、渲染引擎、风格关键词）
	} `json:"cover_concept"`
	TypographyInstruction struct {
		TitleContent    string `json:"title_content"`    // 标题文本内容
		TitleStyle      string `json:"title_style"`      // 标题样式（字体、材质、颜色）
		TitlePosition   string `json:"title_position"`   // 标题位置
		SubtitleContent string `json:"subtitle_content"` // 副标题内容（可选）
		SubtitleStyle   string `json:"subtitle_style"`   // 副标题样式（可选）
	} `json:"typography_instruction"`
}

// coverConceptGenerationResult Step 1 结果
type coverConceptGenerationResult struct {
	RecordID    string
	ConceptJSON string
	Concept     *CoverConcept
	TokensUsed  int
}

// generateCoverConcept Step 1: 使用LLM生成封面概念
func (s *Service) generateCoverConcept(ctx context.Context, userID, storyID string, title, description, genre string, style *domain.StyleConfig) (*coverConceptGenerationResult, error) {
	// 构建System Prompt
	systemPrompt := `# Role
You are an expert Key Art / Book Cover Designer.
Your task is to generate a structured concept (JSON) for an image-generation model based on Story Information.

# Important Product Constraint (TEXT HANDLING)
- The final image will have typography overlaid later by the app.
- Do NOT require the image model to generate readable text.
- In your plan, reserve a clean typography SAFE AREA (negative space) where text can be placed later.

# Goal
Create a professional, cinematic key art concept that is:
- visually clear (strong focal point, readable silhouette)
- compositionally clean (no clutter where typography should go)
- consistent with genre and style preference

# Steps
1) Analyze Story: extract key visuals, symbols, setting, mood.
2) Design Composition: camera, framing, depth, spatial hierarchy.
3) Plan Typography Safe Area (no text rendering): choose placement and style references only.

# Output Rules
- Output ONLY valid JSON (no markdown, no explanations).
- Use concise but vivid English phrases suitable for image generation.
- Avoid sensitive/illegal content.

# JSON Output Schema
{
  "cover_concept": {
    "visual_subject": "string (main subject with concrete visual details)",
    "scene_environment": "string (setting + weather + props, if any)",
    "composition_camera": "string (camera angle + framing + depth of field + negative space guidance)",
    "lighting_atmosphere": "string (lighting + color palette + mood)",
    "art_style": "string (medium + rendering style keywords)"
  },
  "typography_instruction": {
    "title_content": "string (MUST be 'NONE' — do not generate text in image)",
    "title_style": "string (style reference for later overlay, e.g. 'minimal sans-serif, gold foil feel')",
    "title_position": "string (safe-area placement, e.g. 'top center with 20% margin')",
    "subtitle_content": "string ('NONE')",
    "subtitle_style": "string ('NONE' or optional style reference)"
  }
}`

	// 构建User Prompt
	var userPrompt strings.Builder
	userPrompt.WriteString("Please create a key art / book cover concept based on the following story information.\n")
	userPrompt.WriteString("Typography will be added later by the app, so reserve a clean safe area and DO NOT generate readable text.\n\n")

	// 故事信息
	userPrompt.WriteString("[Story Information]\n")
	userPrompt.WriteString("Title: ")
	userPrompt.WriteString(title)
	userPrompt.WriteString("\n")
	if description != "" {
		userPrompt.WriteString("Description: ")
		userPrompt.WriteString(description)
		userPrompt.WriteString("\n")
	}
	if genre != "" {
		userPrompt.WriteString("Genre: ")
		userPrompt.WriteString(genre)
		userPrompt.WriteString("\n")
	}
	if style != nil && style.Style != "" {
		userPrompt.WriteString("Style Preference: ")
		userPrompt.WriteString(style.Style)
		userPrompt.WriteString("\n")
		if style.Description != "" {
			userPrompt.WriteString("Style Description: ")
			userPrompt.WriteString(style.Description)
			userPrompt.WriteString("\n")
		}
	}
	userPrompt.WriteString("\n")

	// 调用AI生成服务
	genReq := &GenerateTextRequest{
		UserID:            userID,
		OriginalPrompt:    userPrompt.String(),
		SystemPrompt:      systemPrompt,
		Model:             "gemini-2.5-flash",
		Temperature:       0.8,
		MaxTokens:         2000,
		RelatedEntityID:   storyID,
		RelatedEntityType: "story_cover",
		Metadata: map[string]interface{}{
			"operation": "cover_concept_generation",
			"storyId":   storyID,
			"step":      1,
		},
	}

	result, err := s.aiGenService.GenerateText(ctx, genReq)
	if err != nil {
		return nil, err
	}

	// 解析生成的JSON
	concept, err := s.parseCoverConcept(result.Text)
	if err != nil {
		s.logger.Warn("failed to parse cover concept JSON, using raw text",
			zap.Error(err),
			zap.String("rawText", truncateForLog(result.Text, 500)))
	}

	return &coverConceptGenerationResult{
		RecordID:    result.RecordID,
		ConceptJSON: result.Text,
		Concept:     concept,
		TokensUsed:  result.TokensUsed,
	}, nil
}

// parseCoverConcept 解析封面概念JSON
func (s *Service) parseCoverConcept(text string) (*CoverConcept, error) {
	// 清理JSON文本
	cleanedText := strings.TrimSpace(text)

	// 处理markdown代码块
	if strings.HasPrefix(cleanedText, "```") {
		if idx := strings.Index(cleanedText, "\n"); idx != -1 {
			cleanedText = cleanedText[idx+1:]
		}
		if idx := strings.LastIndex(cleanedText, "```"); idx != -1 {
			cleanedText = strings.TrimSpace(cleanedText[:idx])
		}
	}

	// 提取JSON对象
	cleanedText = extractJSONFromText(cleanedText)

	var concept CoverConcept
	if err := json.Unmarshal([]byte(cleanedText), &concept); err != nil {
		return nil, err
	}

	return &concept, nil
}

// assembleFinalCoverPrompt 组装最终封面图像生成提示词
func (s *Service) assembleFinalCoverPrompt(concept *CoverConcept, title, imageType string) string {
	var prompt strings.Builder

	if concept != nil && concept.CoverConcept.VisualSubject != "" {
		// 使用结构化概念组装提示词
		c := concept.CoverConcept
		t := concept.TypographyInstruction

		prompt.WriteString("TASK:\n")
		prompt.WriteString("Create a high-quality, cinematic illustration for a story.\n")
		prompt.WriteString("This image will be used as ")
		prompt.WriteString(imageType)
		prompt.WriteString(" key art.\n\n")

		prompt.WriteString("STORY TITLE (for context only, do NOT render as text):\n")
		prompt.WriteString(title)
		prompt.WriteString("\n\n")

		prompt.WriteString("PRIMARY SUBJECT:\n")
		prompt.WriteString(c.VisualSubject)
		prompt.WriteString("\n\n")

		if c.SceneEnvironment != "" {
			prompt.WriteString("SCENE / ENVIRONMENT:\n")
			prompt.WriteString(c.SceneEnvironment)
			prompt.WriteString("\n\n")
		}

		if c.CompositionCamera != "" {
			prompt.WriteString("COMPOSITION & CAMERA:\n")
			prompt.WriteString(c.CompositionCamera)
			prompt.WriteString("\n\n")
		}

		if c.LightingAtmosphere != "" {
			prompt.WriteString("LIGHTING & MOOD:\n")
			prompt.WriteString(c.LightingAtmosphere)
			prompt.WriteString("\n\n")
		}

		if c.ArtStyle != "" {
			prompt.WriteString("ART STYLE / RENDERING:\n")
			prompt.WriteString(c.ArtStyle)
			prompt.WriteString("\n\n")
		}

		// 排版指令
		if t.TitleContent != "" && t.TitleContent != "NONE" {
			prompt.WriteString("TYPOGRAPHY SAFE AREA (IMPORTANT):\n")
			prompt.WriteString("- Do NOT render any readable text, letters, words, logos, or watermarks.\n")
			prompt.WriteString("- Reserve a clean, uncluttered area for later title overlay.\n")
			if t.TitlePosition != "" {
				prompt.WriteString("- Safe area position: ")
				prompt.WriteString(t.TitlePosition)
				prompt.WriteString("\n")
			}
			if t.TitleStyle != "" && t.TitleStyle != "NONE" {
				prompt.WriteString("- Title style reference (for layout mood only): ")
				prompt.WriteString(t.TitleStyle)
				prompt.WriteString("\n")
			}
			if t.SubtitleStyle != "" && t.SubtitleStyle != "NONE" {
				prompt.WriteString("- Subtitle style reference (optional): ")
				prompt.WriteString(t.SubtitleStyle)
				prompt.WriteString("\n")
			}
			prompt.WriteString("\n")
		}

		// 根据图片类型添加特定要求
		switch imageType {
		case "cover":
			prompt.WriteString("OUTPUT REQUIREMENTS (COVER):\n")
			prompt.WriteString("- Professional book cover key art\n")
			prompt.WriteString("- Strong focal point, readable silhouette, clean composition\n")
			prompt.WriteString("- Leave clear space for typography overlay\n")
		case "poster":
			prompt.WriteString("OUTPUT REQUIREMENTS (POSTER):\n")
			prompt.WriteString("- Movie poster style, dramatic lighting, epic composition\n")
			prompt.WriteString("- High contrast, punchy color grading, dynamic staging\n")
			prompt.WriteString("- Leave a small clean area for optional overlay text\n")
		case "background":
			prompt.WriteString("OUTPUT REQUIREMENTS (BACKGROUND):\n")
			prompt.WriteString("- Wide panoramic background suitable for UI background\n")
			prompt.WriteString("- Softer tones, unobtrusive details, avoid clutter in the center\n")
			prompt.WriteString("- No readable text or logos\n")
		}

		prompt.WriteString("\n\nNEGATIVE CONSTRAINTS:\n")
		prompt.WriteString("- No readable text, no logos, no watermarks, no signatures\n")
		prompt.WriteString("- No frames, no borders, no UI mockups\n")
		prompt.WriteString("- No low-res, blur, noise, compression artifacts\n")
	} else {
		// 降级：使用基础提示词
		prompt.WriteString("TASK:\n")
		prompt.WriteString("Create a high-quality cinematic illustration for a story.\n\n")
		prompt.WriteString("STORY TITLE (for context only, do NOT render as text):\n")
		prompt.WriteString(title)
		prompt.WriteString("\n\n")
		prompt.WriteString("KEY REQUIREMENTS:\n")
		prompt.WriteString("- Do NOT render any readable text, no logos, no watermarks\n")
		prompt.WriteString("- Leave a clean area for later title overlay\n")
		switch imageType {
		case "cover":
			prompt.WriteString("- Professional book cover key art, clean composition\n")
		case "poster":
			prompt.WriteString("- Movie poster style, dramatic lighting, epic composition\n")
		case "background":
			prompt.WriteString("- Wide panoramic background, soft tones, unobtrusive\n")
		}
	}

	return prompt.String()
}

// updateUserTokenUsage 更新用户的token使用量
func (s *Service) updateUserTokenUsage(ctx context.Context, userID string, tokensUsed int) error {
	if tokensUsed <= 0 {
		// 如果没有使用token，直接返回
		return nil
	}

	s.logger.Info("updating user token usage",
		zap.String("userID", userID),
		zap.Int("tokensUsed", tokensUsed))

	// 调用 repository 更新 token 余额
	// amount 为负数表示消费（扣除token）
	// source 标识使用来源
	// description 描述使用场景
	_, err := s.repo.UpdateTokenBalance(ctx, userID, -tokensUsed, "story_ai_generation", fmt.Sprintf("Story AI generation consumed %d tokens", tokensUsed))
	if err != nil {
		s.logger.Error("failed to update user token usage",
			zap.String("userID", userID),
			zap.Int("tokensUsed", tokensUsed),
			zap.Error(err))
		return fmt.Errorf("failed to update token usage: %w", err)
	}

	s.logger.Info("user token usage updated successfully",
		zap.String("userID", userID),
		zap.Int("tokensUsed", tokensUsed))

	// Record metrics
	if s.metrics != nil {
		s.metrics.RecordUserTokenConsumed(userID, float64(tokensUsed))
	}

	return nil
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
		ID:        utils.GenerateID(),
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

	// Record metrics - update participant count
	if s.metrics != nil {
		// Get updated contributor count
		contributors, err := s.repo.GetStoryContributors(ctx, storyID, 1000, 0)
		if err == nil {
			s.metrics.RecordStoryParticipantCount(storyID, float64(len(contributors)))
		}
	}

	return contributor, nil
}

// RemoveStoryContributor 移除故事贡献者
func (s *Service) RemoveStoryContributor(ctx context.Context, operatorID, storyID, contributorID string) error {
	// 参数验证
	if operatorID == "" || storyID == "" || contributorID == "" {
		return errors.New("invalid parameters: operatorID, storyID and contributorID are required")
	}

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
		return fmt.Errorf("failed to get story: %w", err)
	}

	// 验证操作者权限（作者可以移除任何人，贡献者可以移除自己）
	isAuthor := story.Author != nil && story.Author.ID == operatorID
	isSelfRemoval := operatorID == contributorID

	if !isAuthor && !isSelfRemoval {
		return errors.New("permission denied: only author can remove other contributors")
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

	// Record metrics - update participant count
	if s.metrics != nil {
		contributors, err := s.repo.GetStoryContributors(ctx, storyID, 1000, 0)
		if err != nil {
			s.logger.Warn("failed to get contributors for metrics", zap.Error(err))
		} else {
			s.metrics.RecordStoryParticipantCount(storyID, float64(len(contributors)))
		}
	}

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

// ========== 碎片转故事功能 ==========

// ConvertFragmentToStory 将碎片转换为故事
func (s *Service) ConvertFragmentToStory(ctx context.Context, userID string, fragmentID string, req domain.ConvertFragmentRequest) (*domain.ConvertFragmentResponse, error) {
	s.logger.Info("converting fragment to story",
		zap.String("userID", userID),
		zap.String("fragmentID", fragmentID),
		zap.String("title", req.Title),
	)

	// 1. 获取碎片信息 - 需要查询数据库获取完整的碎片信息
	// 从 repository 中获取 fragment repository
	fragment, err := s.getFragmentByID(ctx, fragmentID)
	if err != nil {
		s.logger.Error("failed to get fragment",
			zap.String("fragmentID", fragmentID),
			zap.Error(err))
		return nil, fmt.Errorf("fragment not found: %w", err)
	}

	// 2. 检查权限（只能转换自己的碎片）
	if fragment.AuthorID != userID && fragment.CreatorID != userID {
		s.logger.Warn("permission denied: not fragment owner",
			zap.String("userID", userID),
			zap.String("fragmentID", fragmentID),
			zap.String("authorID", fragment.AuthorID),
			zap.String("creatorID", fragment.CreatorID))
		return nil, errors.New("permission denied: not fragment owner")
	}

	s.logger.Debug("fragment retrieved",
		zap.String("fragmentID", fragmentID),
		zap.String("content", fragment.Content),
		zap.Int("mediaCount", len(fragment.MediaURLs)))

	// 3. 设置默认值
	sceneCount := req.SceneCount
	if sceneCount < 2 || sceneCount > 8 {
		sceneCount = 3 // 默认3个场景
	}

	isCollaborationOpen := req.CollaborationType == "open"

	now := time.Now().Unix()

	// 4. 创建故事
	story := &domain.Story{
		ID:                  utils.GenerateID(),
		Title:               req.Title,
		Description:         req.Description,
		CoverImage:          req.CoverImage,
		Genre:               req.Genre,
		AuthorID:            userID,
		Status:              "draft",
		IsCollaborationOpen: isCollaborationOpen,
		AIEnabled:           req.IsAIEnabled,
		DefaultSceneCount:   sceneCount,
		OriginalDescription: fragment.Content, // 保留原始内容
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	// 获取作者信息
	author, err := s.repo.UserByID(ctx, userID)
	if err != nil {
		s.logger.Error("failed to get author",
			zap.String("userID", userID),
			zap.Error(err))
		return nil, errors.New("author not found")
	}
	story.Author = author

	s.logger.Debug("creating story",
		zap.String("storyID", story.ID),
		zap.String("title", story.Title),
		zap.Int("sceneCount", sceneCount))

	if err := s.repo.CreateStory(ctx, story); err != nil {
		s.logger.Error("failed to create story",
			zap.String("storyID", story.ID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to create story: %w", err)
	}

	s.logger.Info("story created",
		zap.String("storyID", story.ID),
		zap.String("title", story.Title))

	// 5. 创建根故事板
	storyboard := &domain.Storyboard{
		ID:             utils.GenerateID(),
		StoryID:        story.ID,
		ParentID:       domain.StoryboardRootMarker, // "__root__"
		CreatorID:      userID,
		CreatorName:    author.DisplayName,
		CreatorAvatar:  author.Avatar,
		Title:          "Chapter 1",
		Content:        fragment.Content, // 碎片内容作为初始内容
		RawInput:       fragment.Content,
		SceneCount:     sceneCount,
		WorkflowStatus: "draft",
		CurrentStep:    1,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	s.logger.Debug("creating root storyboard",
		zap.String("storyboardID", storyboard.ID),
		zap.String("storyID", story.ID))

	if err := s.repo.CreateStoryboard(ctx, storyboard); err != nil {
		s.logger.Error("failed to create storyboard",
			zap.String("storyboardID", storyboard.ID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to create storyboard: %w", err)
	}

	s.logger.Info("root storyboard created",
		zap.String("storyboardID", storyboard.ID),
		zap.String("storyID", story.ID))

	// 6. 迁移碎片媒体资源到故事板场景
	scenesCreated := 0
	if len(fragment.MediaURLs) > 0 {
		s.logger.Debug("migrating fragment media to storyboard scenes",
			zap.Int("mediaCount", len(fragment.MediaURLs)),
			zap.Int("sceneCount", sceneCount))

		for i, mediaURL := range fragment.MediaURLs {
			if i >= sceneCount {
				break // 不超过设定的场景数
			}
			scene := &domain.StoryboardScene{
				ID:           utils.GenerateID(),
				StoryboardID: storyboard.ID,
				Sequence:     i + 1,
				Title:        fmt.Sprintf("Scene %d", i+1),
				Image:        mediaURL,
				Description:  "", // 可以从 fragment content 中解析或留空
				CreatedAt:    now,
				UpdatedAt:    now,
			}
			if err := s.repo.CreateStoryboardScenes(ctx, storyboard.ID, []*domain.StoryboardScene{scene}); err != nil {
				s.logger.Warn("failed to create scene",
					zap.String("storyboardID", storyboard.ID),
					zap.Int("sequence", i+1),
					zap.Error(err))
			} else {
				scenesCreated++
			}
		}
	}

	s.logger.Info("storyboard scenes created",
		zap.Int("scenesCreated", scenesCreated))

	// 7. 更新故事根故事板ID
	story.RootStoryboardID = storyboard.ID
	if err := s.repo.UpdateStory(ctx, story); err != nil {
		s.logger.Warn("failed to update story root storyboard",
			zap.String("storyID", story.ID),
			zap.String("storyboardID", storyboard.ID),
			zap.Error(err))
		// 非致命错误，继续返回成功
	} else {
		s.logger.Debug("story root storyboard updated",
			zap.String("storyID", story.ID),
			zap.String("rootStoryboardID", storyboard.ID))
	}

	// 8. 记录用户活动
	go s.RecordStoryCreated(context.Background(), userID, story.ID, story.Title)

	s.logger.Info("fragment converted to story successfully",
		zap.String("fragmentID", fragmentID),
		zap.String("storyID", story.ID),
		zap.String("storyboardID", storyboard.ID),
		zap.Int("scenesCreated", scenesCreated))

	return &domain.ConvertFragmentResponse{
		Story:      story,
		Storyboard: storyboard,
		FragmentID: fragmentID,
	}, nil
}

// getFragmentByID 从数据库获取碎片信息
func (s *Service) getFragmentByID(ctx context.Context, fragmentID string) (*domain.Fragment, error) {
	return s.repo.FragmentByID(ctx, fragmentID)
}
