package service

import (
	"context"
	"fmt"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/telemetry"
	"go.uber.org/zap"
)

func (s *Service) ListNotifications(ctx context.Context, userID string, limit, offset int) ([]*domain.Notification, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.NotificationsByUser(ctx, userID, limit, offset)
}

func (s *Service) UnreadCount(ctx context.Context, userID string) (int, error) {
	return s.repo.UnreadNotificationCount(ctx, userID)
}

func (s *Service) MarkAsRead(ctx context.Context, id, userID string) error {
	return s.repo.MarkNotificationRead(ctx, id)
}

func (s *Service) MarkAllAsRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllNotificationsRead(ctx, userID)
}

func (s *Service) DeleteNotification(ctx context.Context, id, userID string) error {
	return s.repo.DeleteNotification(ctx, id)
}

// CreateNotification 创建通知（内部使用）
func (s *Service) CreateNotification(ctx context.Context, notification *domain.Notification) error {
	if err := s.repo.CreateNotification(ctx, notification); err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}
	s.logger.Info("notification created", zap.String("userId", notification.UserID), zap.String("type", notification.Type))

	// 记录通知创建指标
	if metrics := telemetry.GetDefaultMetrics(); metrics != nil {
		metrics.RecordNotificationSentSimple("in_app", "database", "success")
		// 根据通知类型分类
		category := "system"
		switch notification.Type {
		case "comment", "like", "follow", "reply", "mention":
			category = "social"
		case "storyboard", "fork":
			category = "transactional"
		}
		metrics.RecordNotificationByCategory(category)
	}

	return nil
}

// NotifyComment 评论通知
func (s *Service) NotifyComment(ctx context.Context, targetUserID, actorID, actorName, actorAvatar, targetType, targetID, commentID string) error {
	notification := &domain.Notification{
		UserID:      targetUserID,
		Type:        "comment",
		Title:       "新评论",
		Content:     fmt.Sprintf("%s 评论了你的%s", actorName, targetType),
		Link:        fmt.Sprintf("/%s/%s#comment-%s", targetType, targetID, commentID),
		ActorID:     actorID,
		ActorName:   actorName,
		ActorAvatar: actorAvatar,
	}
	return s.CreateNotification(ctx, notification)
}

// NotifyLike 点赞通知
func (s *Service) NotifyLike(ctx context.Context, targetUserID, actorID, actorName, actorAvatar, targetType, targetID string) error {
	notification := &domain.Notification{
		UserID:      targetUserID,
		Type:        "like",
		Title:       "新点赞",
		Content:     fmt.Sprintf("%s 赞了你的%s", actorName, targetType),
		Link:        fmt.Sprintf("/%s/%s", targetType, targetID),
		ActorID:     actorID,
		ActorName:   actorName,
		ActorAvatar: actorAvatar,
	}
	return s.CreateNotification(ctx, notification)
}

// NotifyFollow 关注通知
func (s *Service) NotifyFollow(ctx context.Context, targetUserID, actorID, actorName, actorAvatar string) error {
	notification := &domain.Notification{
		UserID:      targetUserID,
		Type:        "follow",
		Title:       "新关注",
		Content:     fmt.Sprintf("%s 关注了你", actorName),
		Link:        fmt.Sprintf("/users/%s", actorID),
		ActorID:     actorID,
		ActorName:   actorName,
		ActorAvatar: actorAvatar,
	}
	return s.CreateNotification(ctx, notification)
}

// NotifyStoryboardCreated 新 storyboard 创建通知
func (s *Service) NotifyStoryboardCreated(ctx context.Context, storyOwnerID, creatorID, creatorName, creatorAvatar, storyID, storyboardID string) error {
	// 如果创建者是故事作者本人，不需要通知
	if storyOwnerID == creatorID {
		return nil
	}

	notification := &domain.Notification{
		UserID:      storyOwnerID,
		Type:        "storyboard",
		Title:       "新 Storyboard",
		Content:     fmt.Sprintf("%s 为你的故事创建了新的分镜", creatorName),
		Link:        fmt.Sprintf("/stories/%s/storyboards/%s", storyID, storyboardID),
		ActorID:     creatorID,
		ActorName:   creatorName,
		ActorAvatar: creatorAvatar,
	}
	return s.CreateNotification(ctx, notification)
}

// NotifyStoryboardForked Fork storyboard 通知
func (s *Service) NotifyStoryboardForked(ctx context.Context, parentCreatorID, forkerID, forkerName, forkerAvatar, storyID, parentStoryboardID, newStoryboardID string) error {
	// 如果 fork 者是父节点创建者本人，不需要通知
	if parentCreatorID == forkerID {
		return nil
	}

	notification := &domain.Notification{
		UserID:      parentCreatorID,
		Type:        "fork",
		Title:       "Storyboard 被 Fork",
		Content:     fmt.Sprintf("%s fork 了你的分镜并创建了新版本", forkerName),
		Link:        fmt.Sprintf("/stories/%s/storyboards/%s", storyID, newStoryboardID),
		ActorID:     forkerID,
		ActorName:   forkerName,
		ActorAvatar: forkerAvatar,
	}
	return s.CreateNotification(ctx, notification)
}

// NotifyCommentReply 评论回复通知
func (s *Service) NotifyCommentReply(ctx context.Context, targetUserID, actorID, actorName, actorAvatar, targetType, targetID, commentID string) error {
	// 不通知自己
	if targetUserID == actorID {
		return nil
	}

	notification := &domain.Notification{
		UserID:      targetUserID,
		Type:        "reply",
		Title:       "新回复",
		Content:     fmt.Sprintf("%s 回复了你的评论", actorName),
		Link:        fmt.Sprintf("/%s/%s#comment-%s", targetType, targetID, commentID),
		ActorID:     actorID,
		ActorName:   actorName,
		ActorAvatar: actorAvatar,
	}
	return s.CreateNotification(ctx, notification)
}

// NotifyCommentLike 评论点赞通知
func (s *Service) NotifyCommentLike(ctx context.Context, targetUserID, actorID, actorName, actorAvatar, targetType, targetID, commentID string) error {
	// 不通知自己
	if targetUserID == actorID {
		return nil
	}

	notification := &domain.Notification{
		UserID:      targetUserID,
		Type:        "like",
		Title:       "评论被赞",
		Content:     fmt.Sprintf("%s 赞了你的评论", actorName),
		Link:        fmt.Sprintf("/%s/%s#comment-%s", targetType, targetID, commentID),
		ActorID:     actorID,
		ActorName:   actorName,
		ActorAvatar: actorAvatar,
	}
	return s.CreateNotification(ctx, notification)
}

// NotifyMention 提及通知
func (s *Service) NotifyMention(ctx context.Context, targetUserID, actorID, actorName, actorAvatar, targetType, targetID, commentID string) error {
	// 不通知自己
	if targetUserID == actorID {
		return nil
	}

	notification := &domain.Notification{
		UserID:      targetUserID,
		Type:        "mention",
		Title:       "有人提及了你",
		Content:     fmt.Sprintf("%s 在评论中提及了你", actorName),
		Link:        fmt.Sprintf("/%s/%s#comment-%s", targetType, targetID, commentID),
		ActorID:     actorID,
		ActorName:   actorName,
		ActorAvatar: actorAvatar,
	}
	return s.CreateNotification(ctx, notification)
}

// NotifyGroupInvitation 群组邀请通知
func (s *Service) NotifyGroupInvitation(ctx context.Context, inviteeID, inviterID, inviterName, inviterAvatar, groupID, groupName string) error {
	notification := &domain.Notification{
		UserID:      inviteeID,
		Type:        "group_invite",
		Title:       "群组邀请",
		Content:     fmt.Sprintf("%s 邀请你加入群组「%s」", inviterName, groupName),
		Link:        fmt.Sprintf("/groups/%s", groupID),
		ActorID:     inviterID,
		ActorName:   inviterName,
		ActorAvatar: inviterAvatar,
	}
	return s.CreateNotification(ctx, notification)
}
