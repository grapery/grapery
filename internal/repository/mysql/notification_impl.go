package mysql

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

func (r *Repository) NotificationsByUser(ctx context.Context, userID string, limit, offset int) ([]*domain.Notification, error) {
	var notifications []Notification
	query := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}
	if err := query.Find(&notifications).Error; err != nil {
		return nil, fmt.Errorf("failed to get notifications: %w", err)
	}
	result := make([]*domain.Notification, len(notifications))
	for i := range notifications {
		result[i] = ModelToNotification(&notifications[i])
	}
	return result, nil
}

func (r *Repository) UnreadNotificationCount(ctx context.Context, userID string) (int, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&Notification{}).Where("user_id = ? AND `read` = ?", userID, false).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count unread notifications: %w", err)
	}
	return int(count), nil
}

func (r *Repository) CreateNotification(ctx context.Context, notification *domain.Notification) error {
	dbNotif := Notification{
		ID:          uuid.New().String(),
		UserID:      notification.UserID,
		Type:        notification.Type,
		Title:       notification.Title,
		Content:     notification.Content,
		Link:        notification.Link,
		Read:        false,
		ActorID:     notification.ActorID,
		ActorName:   notification.ActorName,
		ActorAvatar: notification.ActorAvatar,
		// Story context
		StoryTitle:         notification.StoryTitle,
		StoryCover:         notification.StoryCover,
		StoryID:            notification.StoryID,
		CommentText:        notification.CommentText,
		RelatedCommentID:   notification.RelatedCommentID,
		StoryboardID:       notification.StoryboardID,
		StoryboardTitle:    notification.StoryboardTitle,
		FragmentID:         notification.FragmentID,
		RelatedCharacterID: notification.RelatedCharacterID,
		TokensUsed:         notification.TokensUsed,
		// System notification
		SysTitle: notification.SysTitle,
		SysBody:  notification.SysBody,
		SysIcon:  notification.SysIcon,
	}
	if err := r.db.WithContext(ctx).Create(&dbNotif).Error; err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}
	notification.ID = dbNotif.ID
	notification.CreatedAt = dbNotif.CreatedAt.Unix()
	return nil
}

func (r *Repository) MarkNotificationRead(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Model(&Notification{}).Where("id = ?", id).Update("`read`", true).Error; err != nil {
		return fmt.Errorf("failed to mark notification as read: %w", err)
	}
	return nil
}

func (r *Repository) MarkAllNotificationsRead(ctx context.Context, userID string) error {
	if err := r.db.WithContext(ctx).Model(&Notification{}).Where("user_id = ?", userID).Update("`read`", true).Error; err != nil {
		return fmt.Errorf("failed to mark all notifications as read: %w", err)
	}
	return nil
}

func (r *Repository) DeleteNotification(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Delete(&Notification{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("failed to delete notification: %w", err)
	}
	return nil
}
