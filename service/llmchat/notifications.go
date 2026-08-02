package llmchat

import (
	"context"

	"github.com/grapery/grapery/models"
)

// Notification represents the service level DTO for system notifications.
type Notification struct {
	ID                  int64                         `json:"id"`
	Type                models.SystemNotificationType `json:"type"`
	Title               string                        `json:"title"`
	Content             string                        `json:"content"`
	IsRead              bool                          `json:"is_read"`
	RelatedUserID       *int64                        `json:"related_user_id,omitempty"`
	RelatedStoryID      *int64                        `json:"related_story_id,omitempty"`
	RelatedStoryBoardID *int64                        `json:"related_storyboard_id,omitempty"`
	RelatedCommentID    *int64                        `json:"related_comment_id,omitempty"`
	ExtraData           map[string]interface{}        `json:"extra_data,omitempty"`
	CreatedAt           int64                         `json:"created_at"`
	UpdatedAt           int64                         `json:"updated_at"`
}

// GetRecentNotifications returns latest notifications for a user with capped limit.
func GetRecentNotifications(ctx context.Context, userID int64, limit int) ([]Notification, bool, int64, error) {
	if limit <= 0 || limit > 20 {
		limit = 20
	}
	items, hasMore, err := models.ListSystemNotificationsByUser(ctx, userID, limit)
	if err != nil {
		return nil, false, 0, err
	}
	total, err := models.CountSystemNotificationsByUser(ctx, userID)
	if err != nil {
		return nil, false, 0, err
	}
	result := make([]Notification, 0, len(items))
	for _, item := range items {
		dto := Notification{
			ID:                  item.ID,
			Type:                item.Type,
			Title:               item.Title,
			Content:             item.Content,
			IsRead:              item.IsRead,
			RelatedUserID:       item.RelatedUserID,
			RelatedStoryID:      item.RelatedStoryID,
			RelatedStoryBoardID: item.RelatedStoryBoardID,
			RelatedCommentID:    item.RelatedCommentID,
			CreatedAt:           item.CreatedAt.Unix(),
			UpdatedAt:           item.UpdatedAt.Unix(),
		}
		if item.ExtraData != nil {
			dto.ExtraData = map[string]interface{}(item.ExtraData)
		}
		result = append(result, dto)
	}
	return result, hasMore, total, nil
}

// MarkAllNotificationsRead marks all notifications for the user.
func MarkAllNotificationsRead(ctx context.Context, userID int64) error {
	return models.MarkAllSystemNotificationsRead(ctx, userID)
}

// MarkNotificationRead marks a single notification as read for the user.
func MarkNotificationRead(ctx context.Context, userID, notificationID int64) error {
	return models.MarkSystemNotificationRead(ctx, userID, notificationID)
}

// CreateNotificationParams defines parameters for creating a notification.
type CreateNotificationParams struct {
	UserID              int64
	Type                models.SystemNotificationType
	Title               string
	Content             string
	RelatedUserID       *int64
	RelatedStoryID      *int64
	RelatedStoryBoardID *int64
	RelatedCommentID    *int64
	ExtraData           map[string]interface{}
}

// CreateSystemNotification creates a new system notification.
func CreateSystemNotification(ctx context.Context, params CreateNotificationParams) error {
	notification := &models.SystemNotification{
		UserID:              params.UserID,
		Type:                params.Type,
		Title:               params.Title,
		Content:             params.Content,
		IsRead:              false,
		RelatedUserID:       params.RelatedUserID,
		RelatedStoryID:      params.RelatedStoryID,
		RelatedStoryBoardID: params.RelatedStoryBoardID,
		RelatedCommentID:    params.RelatedCommentID,
	}
	if params.ExtraData != nil {
		notification.ExtraData = params.ExtraData
	}
	return models.DataBase().WithContext(ctx).Create(notification).Error
}
