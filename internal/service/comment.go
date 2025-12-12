package service

import (
	"context"
	"fmt"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

// CreateComment 创建评论
func (s *Service) CreateComment(ctx context.Context, comment *domain.Comment) error {
	// 验证目标存在
	if err := s.validateCommentTarget(ctx, comment.TargetType, comment.TargetID); err != nil {
		return err
	}

	// 获取评论者信息
	author, err := s.repo.UserByID(ctx, comment.AuthorID)
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
	}

	// 创建评论
	if err := s.repo.CreateComment(ctx, comment); err != nil {
		return fmt.Errorf("failed to create comment: %w", err)
	}

	s.logger.Info("comment created",
		zap.String("id", comment.ID),
		zap.String("authorId", comment.AuthorID),
		zap.String("targetType", comment.TargetType),
		zap.String("targetId", comment.TargetID),
	)

	// 异步发送通知（不阻塞主流程）
	go s.sendCommentNotifications(context.Background(), comment, author, parentComment)

	return nil
}

// sendCommentNotifications 发送评论相关通知
func (s *Service) sendCommentNotifications(ctx context.Context, comment *domain.Comment, author *domain.User, parentComment *domain.Comment) {
	// 1. 如果是回复，通知父评论作者
	if parentComment != nil && parentComment.AuthorID != comment.AuthorID {
		if err := s.NotifyCommentReply(ctx, parentComment.AuthorID, author.ID, author.DisplayName, author.Avatar, comment.TargetType, comment.TargetID, comment.ID); err != nil {
			s.logger.Error("failed to send reply notification",
				zap.Error(err),
				zap.String("commentId", comment.ID),
				zap.String("parentCommentAuthor", parentComment.AuthorID),
			)
		}
	}

	// 2. 通知目标对象的作者
	targetAuthorID, targetAuthorName := s.getTargetAuthorInfo(ctx, comment.TargetType, comment.TargetID)
	if targetAuthorID != "" && targetAuthorID != comment.AuthorID {
		// 如果是回复，且父评论作者就是目标作者，不重复通知
		if parentComment != nil && parentComment.AuthorID == targetAuthorID {
			return
		}

		if err := s.NotifyComment(ctx, targetAuthorID, author.ID, author.DisplayName, author.Avatar, s.getTargetTypeName(comment.TargetType), comment.TargetID, comment.ID); err != nil {
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
			return comment.AuthorID, ""
		}
	}
	return "", ""
}

// getTargetTypeName 获取目标类型的中文名称
func (s *Service) getTargetTypeName(targetType string) string {
	switch targetType {
	case "story":
		return "故事"
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

// GetComment 获取评论详情
func (s *Service) GetComment(ctx context.Context, id string) (*domain.Comment, error) {
	comment, err := s.repo.CommentByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return comment, nil
}

// UpdateComment 更新评论
func (s *Service) UpdateComment(ctx context.Context, comment *domain.Comment, userID string) error {
	// 验证权限
	existing, err := s.repo.CommentByID(ctx, comment.ID)
	if err != nil {
		return err
	}

	if existing.AuthorID != userID {
		return fmt.Errorf("permission denied: not the author")
	}

	// 更新
	if err := s.repo.UpdateComment(ctx, comment); err != nil {
		return fmt.Errorf("failed to update comment: %w", err)
	}

	s.logger.Info("comment updated",
		zap.String("id", comment.ID),
		zap.String("userId", userID),
	)

	return nil
}

// DeleteComment 删除评论
func (s *Service) DeleteComment(ctx context.Context, id, userID string) error {
	// 验证权限
	comment, err := s.repo.CommentByID(ctx, id)
	if err != nil {
		return err
	}

	if comment.AuthorID != userID {
		return fmt.Errorf("permission denied: not the author")
	}

	// 删除
	if err := s.repo.DeleteComment(ctx, id); err != nil {
		return fmt.Errorf("failed to delete comment: %w", err)
	}

	s.logger.Info("comment deleted",
		zap.String("id", id),
		zap.String("userId", userID),
	)

	return nil
}

// ListComments 获取目标的评论列表
func (s *Service) ListComments(ctx context.Context, targetType, targetID string, limit, offset int, userID string) ([]*domain.Comment, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	comments, total, err := s.repo.CommentsByTarget(ctx, targetType, targetID, limit, offset)
	if err != nil {
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

	return comments, total, nil
}

// GetCommentReplies 获取评论的回复
func (s *Service) GetCommentReplies(ctx context.Context, parentID string, limit, offset int) ([]*domain.Comment, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	replies, err := s.repo.CommentReplies(ctx, parentID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get replies: %w", err)
	}

	return replies, nil
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

// ToggleLikeComment 切换点赞状态
func (s *Service) ToggleLikeComment(ctx context.Context, userID, commentID string) (*ToggleLikeResult, error) {
	// 获取评论信息
	comment, err := s.repo.CommentByID(ctx, commentID)
	if err != nil {
		return nil, fmt.Errorf("comment not found: %w", err)
	}

	// 检查当前是否已点赞
	hasLiked, isLike, err := s.repo.IsCommentLiked(ctx, userID, commentID)
	if err != nil {
		return nil, fmt.Errorf("failed to check like status: %w", err)
	}

	var newIsLiked bool
	var newLikes int

	if hasLiked && isLike {
		// 已点赞，取消点赞
		if err := s.repo.UnlikeComment(ctx, userID, commentID); err != nil {
			return nil, fmt.Errorf("failed to unlike comment: %w", err)
		}
		newIsLiked = false
		newLikes = comment.Likes - 1
		if newLikes < 0 {
			newLikes = 0
		}
		s.logger.Info("comment unliked",
			zap.String("userId", userID),
			zap.String("commentId", commentID),
		)
	} else {
		// 未点赞或是踩，执行点赞
		if err := s.repo.LikeComment(ctx, userID, commentID, true); err != nil {
			return nil, fmt.Errorf("failed to like comment: %w", err)
		}
		newIsLiked = true
		newLikes = comment.Likes + 1
		s.logger.Info("comment liked",
			zap.String("userId", userID),
			zap.String("commentId", commentID),
		)

		// 异步发送通知（不通知自己点赞自己的评论）
		if comment.AuthorID != userID {
			go func() {
				liker, err := s.repo.UserByID(context.Background(), userID)
				if err != nil {
					s.logger.Error("failed to get liker info for notification", zap.Error(err))
					return
				}

				if err := s.NotifyCommentLike(context.Background(), comment.AuthorID, liker.ID, liker.DisplayName, liker.Avatar, comment.TargetType, comment.TargetID, commentID); err != nil {
					s.logger.Error("failed to send comment like notification",
						zap.Error(err),
						zap.String("commentId", commentID),
						zap.String("commentAuthor", comment.AuthorID),
					)
				}
			}()
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
		return fmt.Errorf("failed to like comment: %w", err)
	}

	s.logger.Info("comment liked",
		zap.String("userId", userID),
		zap.String("commentId", commentID),
	)

	// 异步发送通知（不通知自己点赞自己的评论）
	if comment.AuthorID != userID {
		go func() {
			// 获取点赞者信息
			liker, err := s.repo.UserByID(context.Background(), userID)
			if err != nil {
				s.logger.Error("failed to get liker info for notification", zap.Error(err))
				return
			}

			if err := s.NotifyCommentLike(context.Background(), comment.AuthorID, liker.ID, liker.DisplayName, liker.Avatar, comment.TargetType, comment.TargetID, commentID); err != nil {
				s.logger.Error("failed to send comment like notification",
					zap.Error(err),
					zap.String("commentId", commentID),
					zap.String("commentAuthor", comment.AuthorID),
				)
			}
		}()
	}

	return nil
}

// DislikeComment 踩评论
func (s *Service) DislikeComment(ctx context.Context, userID, commentID string) error {
	if err := s.repo.LikeComment(ctx, userID, commentID, false); err != nil {
		return fmt.Errorf("failed to dislike comment: %w", err)
	}

	s.logger.Info("comment disliked",
		zap.String("userId", userID),
		zap.String("commentId", commentID),
	)

	return nil
}

// UnlikeComment 取消点赞/踩
func (s *Service) UnlikeComment(ctx context.Context, userID, commentID string) error {
	if err := s.repo.UnlikeComment(ctx, userID, commentID); err != nil {
		return fmt.Errorf("failed to unlike comment: %w", err)
	}

	s.logger.Info("comment unliked",
		zap.String("userId", userID),
		zap.String("commentId", commentID),
	)

	return nil
}

// validateCommentTarget 验证评论目标是否存在
func (s *Service) validateCommentTarget(ctx context.Context, targetType, targetID string) error {
	switch targetType {
	case "story":
		_, err := s.repo.StoryByID(ctx, targetID)
		if err != nil {
			return fmt.Errorf("story not found: %w", err)
		}
	case "storyboard":
		_, err := s.repo.StoryboardByID(ctx, targetID)
		if err != nil {
			return fmt.Errorf("storyboard not found: %w", err)
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
