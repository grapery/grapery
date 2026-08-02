package models

import (
	"context"
	"fmt"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// SystemNotificationType defines notification categories.
type SystemNotificationType string

const (
	SystemNotificationTypeLike            SystemNotificationType = "like"
	SystemNotificationTypeComment         SystemNotificationType = "comment"
	SystemNotificationTypeFollow          SystemNotificationType = "follow"
	SystemNotificationTypeStoryCreated    SystemNotificationType = "story_created"
	SystemNotificationTypeStoryPublished  SystemNotificationType = "story_published"
	SystemNotificationTypeVIPSubscription SystemNotificationType = "vip_subscription"
	SystemNotificationTypeSystemUpdate    SystemNotificationType = "system_update"
	SystemNotificationTypeMaintenance     SystemNotificationType = "maintenance"
	SystemNotificationTypeAchievement     SystemNotificationType = "achievement"
)

// SystemNotification represents a persisted system notification entry.
type SystemNotification struct {
	ID                  int64                  `gorm:"primaryKey;column:id" json:"id"`
	UserID              int64                  `gorm:"column:user_id;index" json:"user_id"`
	Type                SystemNotificationType `gorm:"column:type;size:32" json:"type"`
	Title               string                 `gorm:"column:title;size:255" json:"title"`
	Content             string                 `gorm:"column:content;type:text" json:"content"`
	IsRead              bool                   `gorm:"column:is_read" json:"is_read"`
	RelatedUserID       *int64                 `gorm:"column:related_user_id" json:"related_user_id"`
	RelatedStoryID      *int64                 `gorm:"column:related_story_id" json:"related_story_id"`
	RelatedStoryBoardID *int64                 `gorm:"column:related_storyboard_id" json:"related_storyboard_id"`
	RelatedCommentID    *int64                 `gorm:"column:related_comment_id" json:"related_comment_id"`
	ExtraData           datatypes.JSONMap      `gorm:"column:extra_data;type:json" json:"extra_data"`
	CreatedAt           time.Time              `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt           time.Time              `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	Deleted             bool                   `gorm:"column:deleted" json:"deleted"`
}

// TableName returns the underlying table name.
func (SystemNotification) TableName() string {
	return "system_notifications"
}

// ListSystemNotificationsByUser fetches latest notifications for a user.
func ListSystemNotificationsByUser(ctx context.Context, userID int64, limit int) ([]*SystemNotification, bool, error) {
	if limit <= 0 {
		limit = 20
	}
	queryLimit := limit + 1
	if queryLimit > 100 {
		queryLimit = 100
	}
	var notifications []*SystemNotification
	err := DataBase().WithContext(ctx).
		Model(&SystemNotification{}).
		Where("user_id = ? AND deleted = 0", userID).
		Order("created_at desc").
		Limit(queryLimit).
		Find(&notifications).Error
	if err != nil {
		return nil, false, fmt.Errorf("list system notifications failed: %w", err)
	}
	hasMore := len(notifications) > limit
	if hasMore {
		notifications = notifications[:limit]
	}
	return notifications, hasMore, nil
}

// CountSystemNotificationsByUser returns total notifications for the user.
func CountSystemNotificationsByUser(ctx context.Context, userID int64) (int64, error) {
	var count int64
	err := DataBase().WithContext(ctx).
		Model(&SystemNotification{}).
		Where("user_id = ? AND deleted = 0", userID).
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("count system notifications failed: %w", err)
	}
	return count, nil
}

// MarkAllSystemNotificationsRead marks all notifications for the user as read.
func MarkAllSystemNotificationsRead(ctx context.Context, userID int64) error {
	err := DataBase().WithContext(ctx).
		Model(&SystemNotification{}).
		Where("user_id = ? AND deleted = 0 AND is_read = 0", userID).
		Updates(map[string]interface{}{
			"is_read":    true,
			"updated_at": time.Now(),
		}).Error
	if err != nil {
		return fmt.Errorf("mark system notifications read failed: %w", err)
	}
	return nil
}

// MarkSystemNotificationRead marks a single notification as read for the given user.
func MarkSystemNotificationRead(ctx context.Context, userID, notificationID int64) error {
	res := DataBase().WithContext(ctx).
		Model(&SystemNotification{}).
		Where("id = ? AND user_id = ? AND deleted = 0", notificationID, userID).
		Updates(map[string]interface{}{
			"is_read":    true,
			"updated_at": time.Now(),
		})
	if res.Error != nil {
		return fmt.Errorf("mark system notification read failed: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
