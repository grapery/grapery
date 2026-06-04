package service

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/grapestree/fgrapery/grapery/internal/cache"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

// Business rules (see voyager/docs/FEATURES.md SOC-003 / SOC-004)
const (
	maxCommentContentRunes = 500
	// max comment thread depth: 0 = top-level, 1, 2 => three levels; no 4th level.
	maxRootThreadDepth = 2
)

// deleteCommentsListCacheForTarget invalidates list cache for all sort/limit/offset variants we use when caching.
func deleteCommentsListCacheForTarget(ctx context.Context, c cache.Cache, targetType, targetID string) {
	if c == nil {
		return
	}
	sorts := []string{"newest", "hot"}
	for _, sort := range sorts {
		for limit := 20; limit <= 100; limit += 20 {
			for offset := 0; offset < 200; offset += limit {
				_ = c.Delete(ctx, cache.CommentsListKey(targetType, targetID, sort, limit, offset))
			}
		}
	}
}

// normalizeCommentListSort returns "newest" or "hot".
func normalizeCommentListSort(sort string) string {
	if sort == "hot" {
		return "hot"
	}
	return "newest"
}

// CreateComment 创建评论
func (s *Service) CreateComment(ctx context.Context, comment *domain.Comment) error {
	normalized, err := normalizeAndValidateCommentContent(comment.Content)
	if err != nil {
		return err
	}
	comment.Content = normalized
	// 验证目标存在与评论策略
	if err := s.validateCommentTarget(ctx, comment.TargetType, comment.TargetID, comment.UserID); err != nil {
		return err
	}

	// 获取评论者信息
	author, err := s.repo.UserByID(ctx, comment.UserID)
	if err != nil {
		return fmt.Errorf("failed to get comment author: %w", err)
	}

	var parentComment *domain.Comment

	// 如果是回复，验证父评论存在
	if comment.ParentID != "" {
		parentComment, err = s.repo.CommentByID(ctx, comment.ParentID)
		if err != nil {
			return fmt.Errorf("parent comment not found: %w", err)
		}

		// 验证父评论的目标与当前评论相同
		if parentComment.TargetType != comment.TargetType || parentComment.TargetID != comment.TargetID {
			return fmt.Errorf("parent comment belongs to different target")
		}
		depth, err := s.commentThreadDepthFromRoot(ctx, parentComment)
		if err != nil {
			return err
		}
		// 新评论深度 = parentDepth+1 不得超过 maxRootThreadDepth
		if depth >= maxRootThreadDepth {
			return fmt.Errorf("comment would exceed maximum nesting depth (max 3 levels): %w", domain.ErrInvalidInput)
		}
	}

	// 创建评论
	if err := s.repo.CreateComment(ctx, comment); err != nil {
		s.logger.Error("failed to create comment",
			zap.String("commentId", comment.ID),
			zap.Error(err))
		return fmt.Errorf("failed to create comment: %w", err)
	}

	// 使相关缓存失效
	c := s.getCache()
	if c != nil {
		deleteCommentsListCacheForTarget(ctx, c, comment.TargetType, comment.TargetID)
		// 清除目标对象的评论计数缓存（如果有）
		_ = c.Delete(ctx, cache.StoryCommentsKey(comment.TargetID))
		s.logger.Debug("comment cache invalidated",
			zap.String("targetType", comment.TargetType),
			zap.String("targetId", comment.TargetID))
	}

	s.logger.Info("comment created",
		zap.String("id", comment.ID),
		zap.String("authorId", comment.UserID),
		zap.String("targetType", comment.TargetType),
		zap.String("targetId", comment.TargetID))

	// 异步发送通知（不阻塞主流程）
	go s.sendCommentNotifications(context.Background(), comment, author, parentComment)

	return nil
}

// sendCommentNotifications 发送评论相关通知
func (s *Service) sendCommentNotifications(ctx context.Context, comment *domain.Comment, author *domain.User, parentComment *domain.Comment) {
	// 1. 如果是回复，通知父评论作者
	if parentComment != nil && parentComment.UserID != comment.UserID {
		if err := s.NotifyCommentReply(ctx, parentComment.UserID, author.ID, author.DisplayName, author.Avatar, comment.TargetType, comment.TargetID, comment.ID); err != nil {
			s.logger.Error("failed to send reply notification",
				zap.Error(err),
				zap.String("commentId", comment.ID),
				zap.String("parentCommentAuthor", parentComment.UserID),
			)
		}
	}

	// 2. 通知目标对象的作者
	targetAuthorID, targetAuthorName := s.getTargetAuthorInfo(ctx, comment.TargetType, comment.TargetID)
	if targetAuthorID != "" && targetAuthorID != comment.UserID {
		// 如果是回复，且父评论作者就是目标作者，不重复通知
		if parentComment != nil && parentComment.UserID == targetAuthorID {
			return
		}

		if err := s.NotifyComment(ctx, targetAuthorID, author.ID, author.DisplayName, author.Avatar, comment.TargetType, comment.TargetID, comment.ID); err != nil {
			s.logger.Error("failed to send comment notification to target author",
				zap.Error(err),
				zap.String("commentId", comment.ID),
				zap.String("targetAuthor", targetAuthorID),
				zap.String("targetAuthorName", targetAuthorName),
			)
		}
	}
}

// getTargetAuthorInfo 获取目标对象作者信息
func (s *Service) getTargetAuthorInfo(ctx context.Context, targetType, targetID string) (authorID, authorName string) {
	switch targetType {
	case "story":
		story, err := s.repo.StoryByID(ctx, targetID)
		if err == nil && story.Author != nil {
			return story.Author.ID, story.Author.DisplayName
		}
	case "fragment":
		fragment, err := s.repo.FragmentByID(ctx, targetID)
		if err == nil && fragment.Author != nil {
			return fragment.Author.ID, fragment.Author.DisplayName
		}
	case "storyboard":
		storyboard, err := s.repo.StoryboardByID(ctx, targetID)
		if err == nil && storyboard.Creator != nil {
			return storyboard.Creator.ID, storyboard.Creator.DisplayName
		}
	case "character":
		character, err := s.repo.CharacterByID(ctx, targetID)
		if err == nil && character.Author != nil {
			return character.Author.ID, character.Author.DisplayName
		}
	case "comment":
		comment, err := s.repo.CommentByID(ctx, targetID)
		if err == nil {
			return comment.UserID, ""
		}
	}
	return "", ""
}

// getTargetTypeName 获取目标类型的中文名称
func (s *Service) getTargetTypeName(targetType string) string {
	switch targetType {
	case "story":
		return "故事"
	case "fragment":
		return "故事碎片"
	case "storyboard":
		return "分镜"
	case "character":
		return "角色"
	case "comment":
		return "评论"
	default:
		return targetType
	}
}

// GetComment 获取评论详情（带缓存）
func (s *Service) GetComment(ctx context.Context, id string) (*domain.Comment, error) {
	s.logger.Debug("getting comment",
		zap.String("commentId", id))

	// 尝试从缓存获取
	c := s.getCache()
	if c != nil {
		key := cache.CommentKey(id)
		var cachedComment domain.Comment
		if err := c.Get(ctx, key, &cachedComment); err == nil {
			s.logger.Debug("comment cache hit",
				zap.String("commentId", id))
			return &cachedComment, nil
		} else {
			s.logger.Debug("comment cache miss",
				zap.String("commentId", id),
				zap.Error(err))
		}
	}

	// 从数据库获取
	comment, err := s.repo.CommentByID(ctx, id)
	if err != nil {
		s.logger.Error("failed to get comment",
			zap.String("commentId", id),
			zap.Error(err))
		return nil, err
	}

	// 写入缓存
	if c != nil {
		key := cache.CommentKey(id)
		if err := c.Set(ctx, key, comment, commentCacheTTL); err != nil {
			s.logger.Warn("failed to cache comment",
				zap.String("commentId", id),
				zap.Error(err))
		} else {
			s.logger.Debug("comment cached",
				zap.String("commentId", id))
		}
	}

	return comment, nil
}

// UpdateComment 更新评论
func (s *Service) UpdateComment(ctx context.Context, comment *domain.Comment, userID string) error {
	normalized, err := normalizeAndValidateCommentContent(comment.Content)
	if err != nil {
		return err
	}
	comment.Content = normalized
	// 验证权限
	existing, err := s.repo.CommentByID(ctx, comment.ID)
	if err != nil {
		return err
	}

	if existing.UserID != userID {
		return fmt.Errorf("permission denied: not the author")
	}

	// 更新
	if err := s.repo.UpdateComment(ctx, comment); err != nil {
		s.logger.Error("failed to update comment",
			zap.String("commentId", comment.ID),
			zap.Error(err))
		return fmt.Errorf("failed to update comment: %w", err)
	}

	// 使缓存失效并重新缓存
	c := s.getCache()
	if c != nil {
		key := cache.CommentKey(comment.ID)
		if err := c.Delete(ctx, key); err != nil {
			s.logger.Warn("failed to invalidate comment cache",
				zap.String("commentId", comment.ID),
				zap.Error(err))
		}
		// 重新缓存
		if err := c.Set(ctx, key, comment, commentCacheTTL); err != nil {
			s.logger.Warn("failed to cache updated comment",
				zap.String("commentId", comment.ID),
				zap.Error(err))
		}
		deleteCommentsListCacheForTarget(ctx, c, comment.TargetType, comment.TargetID)
	}

	s.logger.Info("comment updated",
		zap.String("id", comment.ID),
		zap.String("userId", userID))

	return nil
}

// DeleteComment 删除评论
func (s *Service) DeleteComment(ctx context.Context, id, userID string) error {
	// 验证权限
	comment, err := s.repo.CommentByID(ctx, id)
	if err != nil {
		return err
	}

	if comment.UserID != userID {
		return fmt.Errorf("permission denied: not the author")
	}

	// 删除
	if err := s.repo.DeleteComment(ctx, id); err != nil {
		s.logger.Error("failed to delete comment",
			zap.String("commentId", id),
			zap.Error(err))
		return fmt.Errorf("failed to delete comment: %w", err)
	}

	// 使缓存失效
	c := s.getCache()
	if c != nil {
		key := cache.CommentKey(id)
		if err := c.Delete(ctx, key); err != nil {
			s.logger.Warn("failed to invalidate comment cache",
				zap.String("commentId", id),
				zap.Error(err))
		}
		deleteCommentsListCacheForTarget(ctx, c, comment.TargetType, comment.TargetID)
	}

	s.logger.Info("comment deleted",
		zap.String("id", id),
		zap.String("userId", userID))

	return nil
}

// ListComments 获取目标的评论列表（带缓存）
func (s *Service) ListComments(ctx context.Context, targetType, targetID string, limit, offset int, userID, sort string) ([]*domain.Comment, int64, error) {
	sort = normalizeCommentListSort(sort)
	s.logger.Debug("listing comments",
		zap.String("targetType", targetType),
		zap.String("targetID", targetID),
		zap.String("sort", sort),
		zap.Int("limit", limit),
		zap.Int("offset", offset))

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	// 尝试从缓存获取（注意：点赞状态是用户相关的，不缓存）
	c := s.getCache()
	if c != nil && userID == "" {
		// 只有未登录用户才使用缓存（因为登录用户的点赞状态不同）
		cacheKey := cache.CommentsListKey(targetType, targetID, sort, limit, offset)
		var cachedComments []*domain.Comment
		var cachedTotal int64
		if err := c.Get(ctx, cacheKey, &cachedComments); err == nil {
			// 尝试获取总数缓存
			totalKey := cacheKey + ":total"
			_ = c.Get(ctx, totalKey, &cachedTotal)
			s.logger.Debug("comments list cache hit",
				zap.String("targetType", targetType),
				zap.String("targetID", targetID),
				zap.Int("count", len(cachedComments)))
			return cachedComments, cachedTotal, nil
		}
		s.logger.Debug("comments list cache miss",
			zap.String("targetType", targetType),
			zap.String("targetID", targetID),
		)
	}

	comments, total, err := s.repo.CommentsByTarget(ctx, targetType, targetID, sort, limit, offset)
	if err != nil {
		s.logger.Error("failed to list comments",
			zap.String("targetType", targetType),
			zap.String("targetID", targetID),
			zap.Error(err))
		return nil, 0, fmt.Errorf("failed to list comments: %w", err)
	}

	// 如果用户已登录，填充点赞状态
	if userID != "" {
		for _, comment := range comments {
			hasLiked, isLike, err := s.repo.IsCommentLiked(ctx, userID, comment.ID)
			if err == nil && hasLiked {
				comment.IsLiked = isLike
				comment.IsDisliked = !isLike
			}
		}
	}

	// 写入缓存（仅未登录用户）
	if c != nil && userID == "" && len(comments) > 0 {
		cacheKey := cache.CommentsListKey(targetType, targetID, sort, limit, offset)
		if err := c.Set(ctx, cacheKey, comments, commentCacheTTL); err != nil {
			s.logger.Warn("failed to cache comments list",
				zap.String("targetType", targetType),
				zap.String("targetID", targetID),
				zap.Error(err))
		} else {
			// 缓存总数
			totalKey := cacheKey + ":total"
			_ = c.Set(ctx, totalKey, total, commentCacheTTL)
			s.logger.Debug("comments list cached",
				zap.String("targetType", targetType),
				zap.String("targetID", targetID),
				zap.Int("count", len(comments)))
		}
	}

	return comments, total, nil
}

// GetCommentReplies 获取评论的回复及该父评论下回复总数
func (s *Service) GetCommentReplies(ctx context.Context, parentID string, limit, offset int, userID string) ([]*domain.Comment, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	replies, total, err := s.repo.CommentReplies(ctx, parentID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get replies: %w", err)
	}

	if userID != "" {
		for _, reply := range replies {
			hasLiked, isLike, err := s.repo.IsCommentLiked(ctx, userID, reply.ID)
			if err == nil && hasLiked {
				reply.IsLiked = isLike
				reply.IsDisliked = !isLike
			}
		}
	}

	return replies, total, nil
}

// GetCommentTree 获取完整的评论树
func (s *Service) GetCommentTree(ctx context.Context, rootID string) ([]*domain.Comment, error) {
	tree, err := s.repo.CommentTree(ctx, rootID)
	if err != nil {
		return nil, fmt.Errorf("failed to get comment tree: %w", err)
	}

	return tree, nil
}

// ToggleLikeResult 点赞切换结果
type ToggleLikeResult struct {
	IsLiked bool `json:"isLiked"`
	Likes   int  `json:"likes"`
}

// ToggleLikeComment 切换点赞状态（带缓存失效）
func (s *Service) ToggleLikeComment(ctx context.Context, userID, commentID string) (*ToggleLikeResult, error) {
	s.logger.Info("toggling comment like",
		zap.String("userID", userID),
		zap.String("commentID", commentID))

	// 获取评论信息
	comment, err := s.repo.CommentByID(ctx, commentID)
	if err != nil {
		s.logger.Error("failed to get comment for like toggle",
			zap.String("commentID", commentID),
			zap.Error(err))
		return nil, fmt.Errorf("comment not found: %w", err)
	}

	// 检查当前是否已点赞
	hasLiked, isLike, err := s.repo.IsCommentLiked(ctx, userID, commentID)
	if err != nil {
		s.logger.Error("failed to check like status",
			zap.String("userID", userID),
			zap.String("commentID", commentID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to check like status: %w", err)
	}

	var newIsLiked bool
	var newLikes int

	if hasLiked && isLike {
		// 已点赞，取消点赞
		if err := s.repo.UnlikeComment(ctx, userID, commentID); err != nil {
			s.logger.Error("failed to unlike comment",
				zap.String("userID", userID),
				zap.String("commentID", commentID),
				zap.Error(err))
			return nil, fmt.Errorf("failed to unlike comment: %w", err)
		}
		newIsLiked = false
		newLikes = comment.Likes - 1
		if newLikes < 0 {
			newLikes = 0
		}
		s.logger.Info("comment unliked",
			zap.String("userId", userID),
			zap.String("commentId", commentID))
	} else {
		// 未点赞或是踩，执行点赞
		if err := s.repo.LikeComment(ctx, userID, commentID, true); err != nil {
			s.logger.Error("failed to like comment",
				zap.String("userID", userID),
				zap.String("commentID", commentID),
				zap.Error(err))
			return nil, fmt.Errorf("failed to like comment: %w", err)
		}
		newIsLiked = true
		newLikes = comment.Likes + 1
		s.logger.Info("comment liked",
			zap.String("userId", userID),
			zap.String("commentId", commentID))

		// 异步发送通知（不通知自己点赞自己的评论）
		if comment.UserID != userID {
			go func() {
				liker, err := s.repo.UserByID(context.Background(), userID)
				if err != nil {
					s.logger.Error("failed to get liker info for notification", zap.Error(err))
					return
				}

				if err := s.NotifyCommentLike(context.Background(), comment.UserID, liker.ID, liker.DisplayName, liker.Avatar, comment.TargetType, comment.TargetID, commentID); err != nil {
					s.logger.Error("failed to send comment like notification",
						zap.Error(err),
						zap.String("commentId", commentID),
						zap.String("commentAuthor", comment.UserID))
				}
			}()
		}
	}

	// 使评论缓存失效（因为点赞数变化）
	c := s.getCache()
	if c != nil {
		key := cache.CommentKey(commentID)
		if err := c.Delete(ctx, key); err != nil {
			s.logger.Warn("failed to invalidate comment cache after like toggle",
				zap.String("commentID", commentID),
				zap.Error(err))
		}
		deleteCommentsListCacheForTarget(ctx, c, comment.TargetType, comment.TargetID)
	}

	// Re-read comment to get authoritative likes count (avoid stale value from pre-read)
	updatedComment, err := s.repo.CommentByID(ctx, commentID)
	if err != nil {
		s.logger.Warn("failed to re-read comment after like toggle, using computed value",
			zap.String("commentID", commentID),
			zap.Error(err))
	} else if updatedComment != nil {
		newLikes = updatedComment.Likes
		if newLikes < 0 {
			newLikes = 0
		}
	}

	return &ToggleLikeResult{
		IsLiked: newIsLiked,
		Likes:   newLikes,
	}, nil
}

// LikeComment 点赞评论
func (s *Service) LikeComment(ctx context.Context, userID, commentID string) error {
	// 获取评论信息
	comment, err := s.repo.CommentByID(ctx, commentID)
	if err != nil {
		return fmt.Errorf("comment not found: %w", err)
	}

	// 执行点赞
	if err := s.repo.LikeComment(ctx, userID, commentID, true); err != nil {
		s.logger.Error("failed to like comment",
			zap.String("userID", userID),
			zap.String("commentID", commentID),
			zap.Error(err))
		return fmt.Errorf("failed to like comment: %w", err)
	}

	// 使评论缓存失效（因为点赞数变化）
	c := s.getCache()
	if c != nil {
		key := cache.CommentKey(commentID)
		if err := c.Delete(ctx, key); err != nil {
			s.logger.Warn("failed to invalidate comment cache after like",
				zap.String("commentID", commentID),
				zap.Error(err))
		}
		deleteCommentsListCacheForTarget(ctx, c, comment.TargetType, comment.TargetID)
	}

	s.logger.Info("comment liked",
		zap.String("userId", userID),
		zap.String("commentId", commentID))

	// 异步发送通知（不通知自己点赞自己的评论）
	if comment.UserID != userID {
		go func() {
			// 获取点赞者信息
			liker, err := s.repo.UserByID(context.Background(), userID)
			if err != nil {
				s.logger.Error("failed to get liker info for notification", zap.Error(err))
				return
			}

			if err := s.NotifyCommentLike(context.Background(), comment.UserID, liker.ID, liker.DisplayName, liker.Avatar, comment.TargetType, comment.TargetID, commentID); err != nil {
				s.logger.Error("failed to send comment like notification",
					zap.Error(err),
					zap.String("commentId", commentID),
					zap.String("commentAuthor", comment.UserID),
				)
			}
		}()
	}

	return nil
}

// DislikeComment 踩评论（带缓存失效）
func (s *Service) DislikeComment(ctx context.Context, userID, commentID string) error {
	s.logger.Info("disliking comment",
		zap.String("userID", userID),
		zap.String("commentID", commentID))

	// 获取评论信息（用于缓存失效）
	comment, err := s.repo.CommentByID(ctx, commentID)
	if err != nil {
		s.logger.Error("failed to get comment for dislike",
			zap.String("commentID", commentID),
			zap.Error(err))
		return fmt.Errorf("comment not found: %w", err)
	}

	if err := s.repo.LikeComment(ctx, userID, commentID, false); err != nil {
		s.logger.Error("failed to dislike comment",
			zap.String("userID", userID),
			zap.String("commentID", commentID),
			zap.Error(err))
		return fmt.Errorf("failed to dislike comment: %w", err)
	}

	// 使评论缓存失效
	c := s.getCache()
	if c != nil {
		key := cache.CommentKey(commentID)
		if err := c.Delete(ctx, key); err != nil {
			s.logger.Warn("failed to invalidate comment cache after dislike",
				zap.String("commentID", commentID),
				zap.Error(err))
		}
		deleteCommentsListCacheForTarget(ctx, c, comment.TargetType, comment.TargetID)
	}

	s.logger.Info("comment disliked",
		zap.String("userId", userID),
		zap.String("commentId", commentID))

	return nil
}

// UnlikeComment 取消点赞/踩（带缓存失效）
func (s *Service) UnlikeComment(ctx context.Context, userID, commentID string) error {
	s.logger.Info("unliking comment",
		zap.String("userID", userID),
		zap.String("commentID", commentID))

	// 获取评论信息（用于缓存失效）
	comment, err := s.repo.CommentByID(ctx, commentID)
	if err != nil {
		s.logger.Error("failed to get comment for unlike",
			zap.String("commentID", commentID),
			zap.Error(err))
		return fmt.Errorf("comment not found: %w", err)
	}

	if err := s.repo.UnlikeComment(ctx, userID, commentID); err != nil {
		s.logger.Error("failed to unlike comment",
			zap.String("userID", userID),
			zap.String("commentID", commentID),
			zap.Error(err))
		return fmt.Errorf("failed to unlike comment: %w", err)
	}

	// 使评论缓存失效
	c := s.getCache()
	if c != nil {
		key := cache.CommentKey(commentID)
		if err := c.Delete(ctx, key); err != nil {
			s.logger.Warn("failed to invalidate comment cache after unlike",
				zap.String("commentID", commentID),
				zap.Error(err))
		}
		deleteCommentsListCacheForTarget(ctx, c, comment.TargetType, comment.TargetID)
	}

	s.logger.Info("comment unliked",
		zap.String("userId", userID),
		zap.String("commentId", commentID))

	return nil
}

// validateCommentTarget 验证评论目标是否存在
func (s *Service) validateCommentTarget(ctx context.Context, targetType, targetID, actorID string) error {
	switch targetType {
	case "story":
		story, err := s.repo.StoryByID(ctx, targetID)
		if err != nil {
			return fmt.Errorf("story not found: %w", err)
		}
		if !story.AllowComments && story.UserID != actorID {
			return domain.ErrForbidden
		}
		if story.Visibility == string(domain.StoryVisibilityPrivate) && story.UserID != actorID {
			return domain.ErrForbidden
		}
	case "fragment":
		fragment, err := s.repo.FragmentByID(ctx, targetID)
		if err != nil {
			return fmt.Errorf("fragment not found: %w", err)
		}
		visibility := domain.NormalizeFragmentVisibility(fragment.Visibility)
		if visibility == domain.FragmentVisibilityPrivate && fragment.UserID != actorID {
			return domain.ErrForbidden
		}
	case "storyboard":
		storyboard, err := s.repo.StoryboardByID(ctx, targetID)
		if err != nil {
			return fmt.Errorf("storyboard not found: %w", err)
		}
		if storyboard.UserID != actorID && storyboard.StoryID != "" {
			story, storyErr := s.repo.StoryByID(ctx, storyboard.StoryID)
			if storyErr == nil {
				if !story.AllowComments && story.UserID != actorID {
					return domain.ErrForbidden
				}
				if story.Visibility == string(domain.StoryVisibilityPrivate) && story.UserID != actorID {
					return domain.ErrForbidden
				}
			}
		}
	case "character":
		_, err := s.repo.CharacterByID(ctx, targetID)
		if err != nil {
			return fmt.Errorf("character not found: %w", err)
		}
	case "comment":
		_, err := s.repo.CommentByID(ctx, targetID)
		if err != nil {
			return fmt.Errorf("target comment not found: %w", err)
		}
	default:
		return fmt.Errorf("invalid target type: %s", targetType)
	}

	return nil
}

func normalizeAndValidateCommentContent(content string) (string, error) {
	trimmed := strings.TrimSpace(content)
	if utf8.RuneCountInString(trimmed) == 0 {
		return "", fmt.Errorf("comment content is empty: %w", domain.ErrInvalidInput)
	}
	if utf8.RuneCountInString(trimmed) > maxCommentContentRunes {
		return "", fmt.Errorf("comment exceeds %d characters: %w", maxCommentContentRunes, domain.ErrInvalidInput)
	}
	return trimmed, nil
}

// commentThreadDepthFromRoot returns 0 for a top-level comment, 1 for a direct reply, 2 for a reply to a reply, etc.
func (s *Service) commentThreadDepthFromRoot(ctx context.Context, c *domain.Comment) (int, error) {
	if c == nil || c.ParentID == "" {
		return 0, nil
	}
	parent, err := s.repo.CommentByID(ctx, c.ParentID)
	if err != nil {
		return 0, err
	}
	pd, err := s.commentThreadDepthFromRoot(ctx, parent)
	if err != nil {
		return 0, err
	}
	return pd + 1, nil
}
