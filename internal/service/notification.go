package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/telemetry"
	"go.uber.org/zap"
)

// targetPathSegment maps domain target type keys to URL path segments (Voyager / API style).
func targetPathSegment(targetTypeKey string) string {
	switch targetTypeKey {
	case "story":
		return "stories"
	case "storyboard":
		return "storyboards"
	case "fragment":
		return "fragments"
	case "character":
		return "characters"
	default:
		return targetTypeKey
	}
}

func (s *Service) resolveStoryIDForCommentTarget(ctx context.Context, targetTypeKey, targetID string) string {
	switch targetTypeKey {
	case "story":
		return targetID
	case "storyboard":
		sb, err := s.repo.StoryboardByID(ctx, targetID)
		if err == nil && sb != nil {
			return sb.StoryID
		}
	case "character":
		ch, err := s.repo.CharacterByID(ctx, targetID)
		if err == nil && ch != nil {
			return ch.StoryID
		}
	}
	return ""
}

// userAllowsPushForNotificationType respects nested notificationSettings.push; defaults to allow if settings missing.
func (s *Service) userAllowsPushForNotificationType(ctx context.Context, userID, typ string) bool {
	settings, err := s.repo.UserSettings(ctx, userID)
	if err != nil || settings == nil {
		return true
	}
	raw := strings.TrimSpace(settings.NotificationSettings)
	if raw == "" {
		return settings.PushNotifications
	}
	var ns domain.NotificationSettings
	if err := json.Unmarshal([]byte(raw), &ns); err != nil {
		return settings.PushNotifications
	}
	if !ns.Push.Enabled {
		return false
	}
	switch typ {
	case "like":
		return ns.Push.NewLike
	case "comment", "reply", "mention":
		return ns.Push.NewComment
	case "follow":
		return ns.Push.NewFollower
	case "storyboard", "fork", "story_follow_storyboard", "storyboard_generation_complete":
		return ns.Push.StoryUpdate
	case "group_invite":
		return ns.Push.SystemAnnouncement
	case "system", "fragment_generation_complete":
		return ns.Push.SystemAnnouncement
	default:
		return true
	}
}

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
		case "storyboard", "fork", "story_follow_storyboard", "storyboard_generation_complete":
			category = "transactional"
		case "fragment_generation_complete":
			category = "system"
		}
		metrics.RecordNotificationByCategory(category)
	}

	return nil
}

// NotifyComment 评论通知（targetTypeKey 为 story / storyboard / character / fragment 等英文 key）
func (s *Service) NotifyComment(ctx context.Context, targetUserID, actorID, actorName, actorAvatar, targetTypeKey, targetID, commentID string) error {
	seg := targetPathSegment(targetTypeKey)
	notification := &domain.Notification{
		UserID:      targetUserID,
		Type:        "comment",
		Title:       "新评论",
		Content:     fmt.Sprintf("%s 评论了你的%s", actorName, s.getTargetTypeName(targetTypeKey)),
		Link:        fmt.Sprintf("/%s/%s#comment-%s", seg, targetID, commentID),
		ActorID:     actorID,
		ActorName:   actorName,
		ActorAvatar: actorAvatar,
		StoryID:     s.resolveStoryIDForCommentTarget(ctx, targetTypeKey, targetID),
	}
	return s.CreateNotificationWithPush(ctx, notification)
}

// NotifyLike 点赞通知（targetTypeKey 如 storyboard；storyID 在分镜点赞时传入故事 ID，其它可为空）
func (s *Service) NotifyLike(ctx context.Context, targetUserID, actorID, actorName, actorAvatar, targetTypeKey, targetID, storyID string) error {
	seg := targetPathSegment(targetTypeKey)
	notification := &domain.Notification{
		UserID:      targetUserID,
		Type:        "like",
		Title:       "新点赞",
		Content:     fmt.Sprintf("%s 赞了你的%s", actorName, s.getTargetTypeName(targetTypeKey)),
		Link:        fmt.Sprintf("/%s/%s", seg, targetID),
		ActorID:     actorID,
		ActorName:   actorName,
		ActorAvatar: actorAvatar,
		StoryID:     storyID,
	}
	return s.CreateNotificationWithPush(ctx, notification)
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
	return s.CreateNotificationWithPush(ctx, notification)
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
		StoryID:     storyID,
	}
	return s.CreateNotificationWithPush(ctx, notification)
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
		StoryID:     storyID,
	}
	return s.CreateNotificationWithPush(ctx, notification)
}

// NotifyCommentReply 评论回复通知（targetTypeKey 为 story / storyboard / character 等）
func (s *Service) NotifyCommentReply(ctx context.Context, targetUserID, actorID, actorName, actorAvatar, targetTypeKey, targetID, commentID string) error {
	// 不通知自己
	if targetUserID == actorID {
		return nil
	}

	seg := targetPathSegment(targetTypeKey)
	notification := &domain.Notification{
		UserID:      targetUserID,
		Type:        "reply",
		Title:       "新回复",
		Content:     fmt.Sprintf("%s 回复了你的评论", actorName),
		Link:        fmt.Sprintf("/%s/%s#comment-%s", seg, targetID, commentID),
		ActorID:     actorID,
		ActorName:   actorName,
		ActorAvatar: actorAvatar,
		StoryID:     s.resolveStoryIDForCommentTarget(ctx, targetTypeKey, targetID),
	}
	return s.CreateNotificationWithPush(ctx, notification)
}

// NotifyCommentLike 评论点赞通知
func (s *Service) NotifyCommentLike(ctx context.Context, targetUserID, actorID, actorName, actorAvatar, targetTypeKey, targetID, commentID string) error {
	// 不通知自己
	if targetUserID == actorID {
		return nil
	}

	seg := targetPathSegment(targetTypeKey)
	notification := &domain.Notification{
		UserID:      targetUserID,
		Type:        "like",
		Title:       "评论被赞",
		Content:     fmt.Sprintf("%s 赞了你的评论", actorName),
		Link:        fmt.Sprintf("/%s/%s#comment-%s", seg, targetID, commentID),
		ActorID:     actorID,
		ActorName:   actorName,
		ActorAvatar: actorAvatar,
		StoryID:     s.resolveStoryIDForCommentTarget(ctx, targetTypeKey, targetID),
	}
	return s.CreateNotificationWithPush(ctx, notification)
}

// NotifyMention 提及通知
func (s *Service) NotifyMention(ctx context.Context, targetUserID, actorID, actorName, actorAvatar, targetTypeKey, targetID, commentID string) error {
	// 不通知自己
	if targetUserID == actorID {
		return nil
	}

	seg := targetPathSegment(targetTypeKey)
	notification := &domain.Notification{
		UserID:      targetUserID,
		Type:        "mention",
		Title:       "有人提及了你",
		Content:     fmt.Sprintf("%s 在评论中提及了你", actorName),
		Link:        fmt.Sprintf("/%s/%s#comment-%s", seg, targetID, commentID),
		ActorID:     actorID,
		ActorName:   actorName,
		ActorAvatar: actorAvatar,
		StoryID:     s.resolveStoryIDForCommentTarget(ctx, targetTypeKey, targetID),
	}
	return s.CreateNotificationWithPush(ctx, notification)
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
	return s.CreateNotificationWithPush(ctx, notification)
}

// NotifyFragmentGenerationCompleted AI 故事碎片生成完成（通知作者本人）
func (s *Service) NotifyFragmentGenerationCompleted(ctx context.Context, userID, fragmentID, previewTitle string) error {
	title := strings.TrimSpace(previewTitle)
	if title == "" {
		title = "你的故事碎片"
	}
	if utf8.RuneCountInString(title) > 48 {
		title = string([]rune(title)[:48]) + "…"
	}
	notification := &domain.Notification{
		UserID:  userID,
		Type:    "fragment_generation_complete",
		Title:   "碎片生成完成",
		Content: fmt.Sprintf("「%s」已生成完成，可前往查看与编辑", title),
		Link:    fmt.Sprintf("/fragments/%s", fragmentID),
	}
	return s.CreateNotificationWithPush(ctx, notification)
}

// NotifyStoryboardInitialGenerationCompleted 分镜初版 AI 生成完成（内容+分镜落库，通知创建者）
func (s *Service) NotifyStoryboardInitialGenerationCompleted(ctx context.Context, userID, storyboardID, storyID string) error {
	notification := &domain.Notification{
		UserID:  userID,
		Type:    "storyboard_generation_complete",
		Title:   "分镜生成完成",
		Content: "AI 已生成分镜内容与场景，可继续编辑或生成配图",
		Link:    fmt.Sprintf("/stories/%s/storyboards/%s", storyID, storyboardID),
		StoryID: storyID,
	}
	return s.CreateNotificationWithPush(ctx, notification)
}

// NotifyFollowedStoryPublishedStoryboard 关注的故事发布了新分镜（通知单个关注者）
func (s *Service) NotifyFollowedStoryPublishedStoryboard(ctx context.Context, followerUserID, storyID, storyboardID, publisherID, publisherName, publisherAvatar, storyTitle string) error {
	st := strings.TrimSpace(storyTitle)
	if st == "" {
		st = "故事"
	}
	if utf8.RuneCountInString(st) > 36 {
		st = string([]rune(st)[:36]) + "…"
	}
	notification := &domain.Notification{
		UserID:      followerUserID,
		Type:        "story_follow_storyboard",
		Title:       "关注的故事更新了",
		Content:     fmt.Sprintf("你关注的「%s」发布了新分镜", st),
		Link:        fmt.Sprintf("/stories/%s/storyboards/%s", storyID, storyboardID),
		ActorID:     publisherID,
		ActorName:   publisherName,
		ActorAvatar: publisherAvatar,
		StoryID:     storyID,
	}
	return s.CreateNotificationWithPush(ctx, notification)
}
