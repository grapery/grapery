package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/common"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

// ========== Panel Request/Response Types ==========

// CreatePanelRequest 创建面板请求
type CreatePanelRequest struct {
	StoryID       string `json:"storyId"`
	StoryboardID  string `json:"storyboardId,omitempty"`
	Sequence      int    `json:"sequence"`
	Image         string `json:"img"`
	Text          string `json:"text"`
	TextPos       string `json:"textPos,omitempty"`
	TextRight     string `json:"textRight,omitempty"`
	IsAIGenerated bool   `json:"isAIGenerated,omitempty"`
	Prompt        string `json:"prompt,omitempty"`
}

// UpdatePanelRequest 更新面板请求
type UpdatePanelRequest struct {
	Sequence  *int    `json:"sequence,omitempty"`
	Image     *string `json:"img,omitempty"`
	Text      *string `json:"text,omitempty"`
	TextPos   *string `json:"textPos,omitempty"`
	TextRight *string `json:"textRight,omitempty"`
	Published *bool   `json:"isPublished,omitempty"`
}

// ReorderPanelsRequest 重排面板请求
type ReorderPanelsRequest struct {
	PanelIDs []string `json:"panelIds" binding:"required"`
}

// ========== Story Comment Request/Response Types ==========

// CreateStoryCommentRequest 创建故事评论请求
type CreateStoryCommentRequest struct {
	StoryID  string `json:"storyId"`
	Content  string `json:"content" binding:"required"`
	ParentID string `json:"parentId,omitempty"`
}

// CreateReplyRequest 创建回复请求
type CreateReplyRequest struct {
	CommentID string `json:"commentId"`
	Content   string `json:"content" binding:"required"`
}

// ========== Panel Service Methods ==========

// ListStoryPanels 列出故事面板
func (s *Service) ListStoryPanels(ctx context.Context, storyID string, limit, offset int) ([]domain.Panel, int, error) {
	panels, err := s.repo.PanelsByStory(ctx, storyID)
	if err != nil {
		s.logger.Error("failed to list story panels", zap.Error(err), zap.String("storyId", storyID))
		return nil, 0, err
	}

	// 转换为值类型
	result := make([]domain.Panel, len(panels))
	for i, p := range panels {
		result[i] = *p
	}

	return result, len(result), nil
}

// CreateStoryPanel 创建故事面板
func (s *Service) CreateStoryPanel(ctx context.Context, userID string, req CreatePanelRequest) (*domain.Panel, error) {
	// 验证故事所有权
	story, err := s.repo.StoryByID(ctx, req.StoryID)
	if err != nil {
		return nil, err
	}
	if story.UserID != userID {
		return nil, domain.ErrUnauthorized
	}

	// 如果没有指定序列号，则自动分配
	if req.Sequence == 0 {
		panels, _ := s.repo.PanelsByStory(ctx, req.StoryID)
		req.Sequence = len(panels) + 1
	}

	now := time.Now().Unix()
	panel := &domain.Panel{
		BaseModel: common.BaseModel{
			ID:        uuid.New().String(),
			CreatedAt: now,
			UpdatedAt: now,
		},
		StoryID:       req.StoryID,
		StoryboardID:  req.StoryboardID,
		Sequence:      req.Sequence,
		Image:         req.Image,
		Text:          req.Text,
		TextPos:       req.TextPos,
		TextRight:     req.TextRight,
		IsAIGenerated: req.IsAIGenerated,
		Prompt:        req.Prompt,
		Published:     false,
	}

	if err := s.repo.CreatePanel(ctx, panel); err != nil {
		s.logger.Error("failed to create panel", zap.Error(err))
		return nil, err
	}

	return panel, nil
}

// UpdateStoryPanel 更新故事面板
func (s *Service) UpdateStoryPanel(ctx context.Context, userID, storyID, panelID string, req UpdatePanelRequest) (*domain.Panel, error) {
	// 验证故事所有权
	story, err := s.repo.StoryByID(ctx, storyID)
	if err != nil {
		return nil, err
	}
	if story.UserID != userID {
		return nil, domain.ErrUnauthorized
	}

	panel, err := s.repo.PanelByID(ctx, panelID)
	if err != nil {
		return nil, err
	}

	// 更新字段
	if req.Sequence != nil {
		panel.Sequence = *req.Sequence
	}
	if req.Image != nil {
		panel.Image = *req.Image
	}
	if req.Text != nil {
		panel.Text = *req.Text
	}
	if req.TextPos != nil {
		panel.TextPos = *req.TextPos
	}
	if req.TextRight != nil {
		panel.TextRight = *req.TextRight
	}
	if req.Published != nil {
		panel.Published = *req.Published
	}
	panel.UpdatedAt = time.Now().Unix()

	if err := s.repo.UpdatePanel(ctx, panel); err != nil {
		s.logger.Error("failed to update panel", zap.Error(err))
		return nil, err
	}

	return panel, nil
}

// DeleteStoryPanel 删除故事面板
func (s *Service) DeleteStoryPanel(ctx context.Context, userID, storyID, panelID string) error {
	// 验证故事所有权
	story, err := s.repo.StoryByID(ctx, storyID)
	if err != nil {
		return err
	}
	if story.UserID != userID {
		return domain.ErrUnauthorized
	}

	if err := s.repo.DeletePanel(ctx, panelID); err != nil {
		s.logger.Error("failed to delete panel", zap.Error(err))
		return err
	}

	return nil
}

// ReorderStoryPanels 重排故事面板
func (s *Service) ReorderStoryPanels(ctx context.Context, userID, storyID string, panelIDs []string) error {
	// 验证故事所有权
	story, err := s.repo.StoryByID(ctx, storyID)
	if err != nil {
		return err
	}
	if story.UserID != userID {
		return domain.ErrUnauthorized
	}

	// 批量更新面板序列
	for i, panelID := range panelIDs {
		panel, err := s.repo.PanelByID(ctx, panelID)
		if err != nil {
			continue
		}
		panel.Sequence = i + 1
		panel.UpdatedAt = time.Now().Unix()
		if err := s.repo.UpdatePanel(ctx, panel); err != nil {
			s.logger.Error("failed to update panel sequence", zap.Error(err), zap.String("panelId", panelID))
			return err
		}
	}

	return nil
}

// ========== Story Comment Service Methods ==========

// ListStoryComments 列出故事评论
func (s *Service) ListStoryComments(ctx context.Context, storyID, userID string, limit, offset int, sortBy string) ([]domain.StoryComment, int, error) {
	// 使用现有的 CommentsByTarget 方法
	sort := "newest"
	if sortBy == "hot" || sortBy == "hottest" {
		sort = "hot"
	}
	comments, total, err := s.repo.CommentsByTarget(ctx, "story", storyID, sort, limit, offset)
	if err != nil {
		s.logger.Error("failed to list story comments", zap.Error(err), zap.String("storyId", storyID))
		return nil, 0, err
	}

	// 转换为 StoryComment 格式
	result := make([]domain.StoryComment, len(comments))
	for i, c := range comments {
		userTag := ""
		if c.Author != nil {
			// 检查是否是故事作者
			story, _ := s.repo.StoryByID(ctx, storyID)
			if story != nil && story.UserID == c.UserID {
				userTag = "作者"
			}
		}

		result[i] = domain.StoryComment{
			BaseModel:  c.BaseModel,
			StoryID:    storyID,
			UserID:     c.UserID,
			Content:    c.Content,
			ParentID:   c.ParentID,
			UserName:   getAuthorName(c.Author),
			UserAvatar: getAuthorAvatar(c.Author),
			UserTag:    userTag,
			Likes:      c.Likes,
			ReplyCount: c.ReplyCount,
			IsLiked:    c.IsLiked,
		}
	}

	return result, int(total), nil
}

// CreateStoryComment 创建故事评论
func (s *Service) CreateStoryComment(ctx context.Context, userID string, req CreateStoryCommentRequest) (*domain.StoryComment, error) {
	// 获取用户信息
	user, err := s.repo.UserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 检查是否是故事作者
	story, _ := s.repo.StoryByID(ctx, req.StoryID)
	userTag := ""
	if story != nil && story.UserID == userID {
		userTag = "作者"
	}

	// 评论开关：作者本人不受限；关闭评论时其他用户不可评论。
	if story != nil && !story.AllowComments && story.UserID != userID {
		s.logger.Warn("comment rejected: comments disabled for story",
			zap.String("storyID", req.StoryID),
			zap.String("userID", userID))
		return nil, domain.ErrForbidden
	}

	comment := &domain.Comment{
		UserID:     userID,
		Content:    req.Content,
		TargetType: "story",
		TargetID:   req.StoryID,
		ParentID:   req.ParentID,
		Author:     user,
	}

	if err := s.CreateComment(ctx, comment); err != nil {
		s.logger.Error("failed to create story comment", zap.Error(err))
		return nil, err
	}

	return &domain.StoryComment{
		BaseModel:  comment.BaseModel,
		StoryID:    req.StoryID,
		UserID:     userID,
		Content:    req.Content,
		ParentID:   req.ParentID,
		UserName:   user.DisplayName,
		UserAvatar: user.Avatar,
		UserTag:    userTag,
		Likes:      0,
		ReplyCount: 0,
		IsLiked:    false,
	}, nil
}

// CreateCommentReply 创建评论回复
func (s *Service) CreateCommentReply(ctx context.Context, userID string, req CreateReplyRequest) (*domain.StoryReply, error) {
	// 获取用户信息
	user, err := s.repo.UserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 获取原评论
	comment, err := s.repo.CommentByID(ctx, req.CommentID)
	if err != nil {
		return nil, err
	}
	if comment.TargetType != "story" {
		return nil, domain.ErrInvalidInput
	}

	// 检查是否是故事作者
	var userTag string
	if comment.TargetType == "story" {
		story, _ := s.repo.StoryByID(ctx, comment.TargetID)
		if story != nil {
			if story.UserID == userID {
				userTag = "作者"
			} else if !story.AllowComments {
				// 评论关闭时，非作者不可回复该故事下的评论。
				s.logger.Warn("reply rejected: comments disabled for story",
					zap.String("storyID", comment.TargetID),
					zap.String("userID", userID))
				return nil, domain.ErrForbidden
			}
		}
	}

	reply := &domain.Comment{
		UserID:     userID,
		Content:    req.Content,
		TargetType: comment.TargetType,
		TargetID:   comment.TargetID,
		ParentID:   req.CommentID,
		Author:     user,
	}

	if err := s.CreateComment(ctx, reply); err != nil {
		s.logger.Error("failed to create comment reply", zap.Error(err))
		return nil, err
	}

	return &domain.StoryReply{
		BaseModel:  reply.BaseModel,
		CommentID:  req.CommentID,
		UserID:     userID,
		Content:    req.Content,
		UserName:   user.DisplayName,
		UserAvatar: user.Avatar,
		UserTag:    userTag,
		Likes:      0,
		IsLiked:    false,
	}, nil
}

// Helper functions
func getAuthorName(author *domain.User) string {
	if author == nil {
		return "Unknown"
	}
	return author.DisplayName
}

func getAuthorAvatar(author *domain.User) string {
	if author == nil {
		return ""
	}
	return author.Avatar
}

// ========== Storyboard Panel Service Methods ==========

// ListStoryboardPanels 列出分镜面板
func (s *Service) ListStoryboardPanels(ctx context.Context, storyboardID string, limit, offset int) ([]domain.StoryboardPanel, int, error) {
	panels, err := s.repo.PanelsByStoryboard(ctx, storyboardID)
	if err != nil {
		s.logger.Error("failed to list storyboard panels", zap.Error(err), zap.String("storyboardId", storyboardID))
		return nil, 0, err
	}

	// 转换为值类型
	result := make([]domain.StoryboardPanel, len(panels))
	for i, p := range panels {
		result[i] = *p
	}

	return result, len(result), nil
}

// CreateStoryboardPanel 创建分镜面板
func (s *Service) CreateStoryboardPanel(ctx context.Context, userID, storyboardID string, panel *domain.StoryboardPanel) (*domain.StoryboardPanel, error) {
	// 验证分镜所有权
	storyboard, err := s.repo.StoryboardByID(ctx, storyboardID)
	if err != nil {
		return nil, err
	}
	if storyboard.UserID != userID {
		return nil, domain.ErrUnauthorized
	}

	// 如果没有指定序列号，则自动分配
	if panel.Sequence == 0 {
		panels, _ := s.repo.PanelsByStoryboard(ctx, storyboardID)
		panel.Sequence = len(panels) + 1
	}

	now := time.Now().Unix()
	panel.ID = uuid.New().String()
	panel.StoryboardID = storyboardID
	panel.CreatedAt = now
	panel.UpdatedAt = now

	if err := s.repo.CreateStoryboardPanel(ctx, panel); err != nil {
		s.logger.Error("failed to create storyboard panel", zap.Error(err))
		return nil, err
	}

	return panel, nil
}
