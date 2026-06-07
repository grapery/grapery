package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/telemetry"
	"go.uber.org/zap"
)

func notificationUnreadCountCacheKey(userID string) string {
	return "notif:unread:" + userID
}

func notificationPushBody(n *domain.Notification) string {
	if n == nil {
		return ""
	}
	body := strings.TrimSpace(n.Content)
	if n.TokensUsed <= 0 {
		return body
	}
	suffix := fmt.Sprintf("\n\n本次生成约消耗 %d Tokens（计费以实际结算为准）。", n.TokensUsed)
	if body == "" {
		return strings.TrimSpace(suffix)
	}
	return body + suffix
}

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
	case "storyboard", "fork", "story_follow_storyboard", "storyboard_generation_complete", "storyboard_generation_failed":
		return ns.Push.StoryUpdate
	case "group_invite":
		return ns.Push.SystemAnnouncement
	case "system", "fragment_generation_complete", "fragment_generation_failed", "character_generation_complete", "character_generation_failed",
		"moderation_report_received", "moderation_block_confirmed", "moderation_report_resolved":
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
	c := s.getCache()
	key := notificationUnreadCountCacheKey(userID)
	if c != nil {
		var cached int
		if err := c.Get(ctx, key, &cached); err == nil {
			return cached, nil
		}
	}
	count, err := s.repo.UnreadNotificationCount(ctx, userID)
	if err != nil {
		return 0, err
	}
	if c != nil {
		_ = c.Set(ctx, key, count, 30*time.Second)
	}
	return count, nil
}

func (s *Service) MarkAsRead(ctx context.Context, id, userID string) error {
	if err := s.repo.MarkNotificationRead(ctx, id); err != nil {
		return err
	}
	if c := s.getCache(); c != nil {
		_ = c.Delete(ctx, notificationUnreadCountCacheKey(userID))
	}
	return nil
}

func (s *Service) MarkAllAsRead(ctx context.Context, userID string) error {
	if err := s.repo.MarkAllNotificationsRead(ctx, userID); err != nil {
		return err
	}
	if c := s.getCache(); c != nil {
		_ = c.Delete(ctx, notificationUnreadCountCacheKey(userID))
	}
	return nil
}

func (s *Service) DeleteNotification(ctx context.Context, id, userID string) error {
	if err := s.repo.DeleteNotification(ctx, id); err != nil {
		return err
	}
	if c := s.getCache(); c != nil {
		_ = c.Delete(ctx, notificationUnreadCountCacheKey(userID))
	}
	return nil
}

// CreateNotification 创建通知（内部使用）
func (s *Service) CreateNotification(ctx context.Context, notification *domain.Notification) error {
	if err := s.repo.CreateNotification(ctx, notification); err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}
	if c := s.getCache(); c != nil {
		_ = c.Delete(ctx, notificationUnreadCountCacheKey(notification.UserID))
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
		case "storyboard", "fork", "story_follow_storyboard", "storyboard_generation_complete", "storyboard_generation_failed":
			category = "transactional"
		case "fragment_generation_complete", "fragment_generation_failed", "character_generation_complete", "character_generation_failed",
			"moderation_report_received", "moderation_block_confirmed", "moderation_report_resolved":
			category = "system"
		}
		metrics.RecordNotificationByCategory(category)
	}

	return nil
}

// NotifyComment 评论通知（targetTypeKey 为 story / storyboard / character / fragment 等英文 key）
func (s *Service) NotifyComment(ctx context.Context, targetUserID, actorID, actorName, actorAvatar, targetTypeKey, targetID, commentID string) error {
	seg := targetPathSegment(targetTypeKey)
	n := &domain.Notification{
		UserID:           targetUserID,
		Type:             "comment",
		Link:             fmt.Sprintf("/%s/%s#comment-%s", seg, targetID, commentID),
		ActorID:          actorID,
		ActorName:        actorName,
		ActorAvatar:      actorAvatar,
		RelatedCommentID: commentID,
	}
	s.populateTargetRichContext(ctx, n, targetTypeKey, targetID)
	if n.StoryID == "" {
		n.StoryID = s.resolveStoryIDForCommentTarget(ctx, targetTypeKey, targetID)
	}
	if cmt, err := s.repo.CommentByID(ctx, commentID); err == nil && cmt != nil {
		t := strings.TrimSpace(cmt.Content)
		if t != "" {
			n.CommentText = truncateNotificationText(t, 400)
		}
	}
	sum := s.targetSummaryPhrase(n, targetTypeKey)
	n.Title = "新评论"
	if n.CommentText != "" {
		n.Content = fmt.Sprintf("%s 在 %s 下留言：\n\n%s", actorName, sum, n.CommentText)
	} else {
		n.Content = fmt.Sprintf("%s 评论了%s", actorName, sum)
	}
	return s.CreateNotificationWithPush(ctx, n)
}

// NotifyLike 点赞通知（targetTypeKey 如 storyboard；storyID 在分镜点赞时传入故事 ID，其它可为空）
func (s *Service) NotifyLike(ctx context.Context, targetUserID, actorID, actorName, actorAvatar, targetTypeKey, targetID, storyID string) error {
	seg := targetPathSegment(targetTypeKey)
	n := &domain.Notification{
		UserID:      targetUserID,
		Type:        "like",
		Link:        fmt.Sprintf("/%s/%s", seg, targetID),
		ActorID:     actorID,
		ActorName:   actorName,
		ActorAvatar: actorAvatar,
		StoryID:     storyID,
	}
	if strings.TrimSpace(n.StoryID) == "" {
		n.StoryID = s.resolveStoryIDForCommentTarget(ctx, targetTypeKey, targetID)
	}
	s.populateTargetRichContext(ctx, n, targetTypeKey, targetID)
	sum := s.targetSummaryPhrase(n, targetTypeKey)
	n.Title = likeNotificationTitle(targetTypeKey)
	n.Content = fmt.Sprintf("%s 赞了%s", actorName, sum)
	return s.CreateNotificationWithPush(ctx, n)
}

// NotifyFollow 关注通知
func (s *Service) NotifyFollow(ctx context.Context, targetUserID, actorID, actorName, actorAvatar string) error {
	n := &domain.Notification{
		UserID:      targetUserID,
		Type:        "follow",
		Title:       "新关注",
		Content:     fmt.Sprintf("%s 关注了你。可在对方主页查看动态与作品。", actorName),
		Link:        fmt.Sprintf("/users/%s", actorID),
		ActorID:     actorID,
		ActorName:   actorName,
		ActorAvatar: actorAvatar,
	}
	return s.CreateNotificationWithPush(ctx, n)
}

func (s *Service) NotifyCharacterGenerationComplete(ctx context.Context, userID, storyID, characterID, characterName string) error {
	title := "角色生成完成"
	name := strings.TrimSpace(characterName)
	if name == "" {
		name = "故事角色"
	}
	n := &domain.Notification{
		UserID:             userID,
		Type:               "character_generation_complete",
		Title:              title,
		Content:            fmt.Sprintf("「%s」已完成资料、立绘和三视图生成。", name),
		Link:               fmt.Sprintf("/characters/%s", characterID),
		StoryID:            storyID,
		RelatedCharacterID: characterID,
	}
	return s.CreateNotificationWithPush(ctx, n)
}

func (s *Service) NotifyCharacterGenerationFailed(ctx context.Context, userID, storyID, taskID, reason string) error {
	msg := strings.TrimSpace(reason)
	if msg == "" {
		msg = "请稍后在草稿箱重试。"
	}
	n := &domain.Notification{
		UserID:  userID,
		Type:    "character_generation_failed",
		Title:   "角色生成失败",
		Content: msg,
		Link:    "/drafts",
		StoryID: storyID,
	}
	if strings.TrimSpace(taskID) != "" {
		n.Link = fmt.Sprintf("/drafts?characterTaskId=%s", taskID)
	}
	return s.CreateNotificationWithPush(ctx, n)
}

// NotifyStoryboardCreated 新 storyboard 创建通知
func (s *Service) NotifyStoryboardCreated(ctx context.Context, storyOwnerID, creatorID, creatorName, creatorAvatar, storyID, storyboardID string) error {
	// 如果创建者是故事作者本人，不需要通知
	if storyOwnerID == creatorID {
		return nil
	}

	n := &domain.Notification{
		UserID:       storyOwnerID,
		Type:         "storyboard",
		Link:         fmt.Sprintf("/stories/%s/storyboards/%s", storyID, storyboardID),
		ActorID:      creatorID,
		ActorName:    creatorName,
		ActorAvatar:  creatorAvatar,
		StoryID:      storyID,
		StoryboardID: storyboardID,
	}
	s.populateTargetRichContext(ctx, n, "storyboard", storyboardID)
	sum := s.targetSummaryPhrase(n, "storyboard")
	n.Title = storyboardCreatedTitle()
	n.Content = fmt.Sprintf("%s 在 %s 发布了新的分镜。", creatorName, sum)
	return s.CreateNotificationWithPush(ctx, n)
}

// NotifyStoryboardForked Fork storyboard 通知
func (s *Service) NotifyStoryboardForked(ctx context.Context, parentCreatorID, forkerID, forkerName, forkerAvatar, storyID, parentStoryboardID, newStoryboardID string) error {
	// 如果 fork 者是父节点创建者本人，不需要通知
	if parentCreatorID == forkerID {
		return nil
	}

	parentTitle := ""
	if sb, err := s.repo.StoryboardByID(ctx, parentStoryboardID); err == nil && sb != nil {
		parentTitle = strings.TrimSpace(sb.Title)
	}
	if parentTitle == "" {
		parentTitle = "原分镜"
	}

	n := &domain.Notification{
		UserID:       parentCreatorID,
		Type:         "fork",
		Link:         fmt.Sprintf("/stories/%s/storyboards/%s", storyID, newStoryboardID),
		ActorID:      forkerID,
		ActorName:    forkerName,
		ActorAvatar:  forkerAvatar,
		StoryID:      storyID,
		StoryboardID: newStoryboardID,
	}
	s.populateTargetRichContext(ctx, n, "storyboard", newStoryboardID)
	child := strings.TrimSpace(n.StoryboardTitle)
	if child == "" {
		child = "新版本"
	}
	n.Title = "分镜被 Fork"
	n.Content = fmt.Sprintf("%s 从你的分镜《%s》衍生出新分镜《%s》，所属故事《%s》。",
		forkerName, parentTitle, child, strings.TrimSpace(n.StoryTitle))
	if strings.TrimSpace(n.StoryTitle) == "" {
		n.Content = fmt.Sprintf("%s 从你的分镜《%s》衍生出新分镜《%s》。", forkerName, parentTitle, child)
	}
	return s.CreateNotificationWithPush(ctx, n)
}

// NotifyCommentReply 评论回复通知（targetTypeKey 为 story / storyboard / character 等）
func (s *Service) NotifyCommentReply(ctx context.Context, targetUserID, actorID, actorName, actorAvatar, targetTypeKey, targetID, commentID string) error {
	// 不通知自己
	if targetUserID == actorID {
		return nil
	}

	seg := targetPathSegment(targetTypeKey)
	n := &domain.Notification{
		UserID:           targetUserID,
		Type:             "reply",
		Link:             fmt.Sprintf("/%s/%s#comment-%s", seg, targetID, commentID),
		ActorID:          actorID,
		ActorName:        actorName,
		ActorAvatar:      actorAvatar,
		RelatedCommentID: commentID,
	}
	s.populateTargetRichContext(ctx, n, targetTypeKey, targetID)
	if n.StoryID == "" {
		n.StoryID = s.resolveStoryIDForCommentTarget(ctx, targetTypeKey, targetID)
	}
	if cmt, err := s.repo.CommentByID(ctx, commentID); err == nil && cmt != nil {
		t := strings.TrimSpace(cmt.Content)
		if t != "" {
			n.CommentText = truncateNotificationText(t, 400)
		}
	}
	sum := s.targetSummaryPhrase(n, targetTypeKey)
	n.Title = "新回复"
	if n.CommentText != "" {
		n.Content = fmt.Sprintf("%s 在 %s 回复了你：\n\n%s", actorName, sum, n.CommentText)
	} else {
		n.Content = fmt.Sprintf("%s 在 %s 回复了你的评论", actorName, sum)
	}
	return s.CreateNotificationWithPush(ctx, n)
}

// NotifyCommentLike 评论点赞通知
func (s *Service) NotifyCommentLike(ctx context.Context, targetUserID, actorID, actorName, actorAvatar, targetTypeKey, targetID, commentID string) error {
	// 不通知自己
	if targetUserID == actorID {
		return nil
	}

	seg := targetPathSegment(targetTypeKey)
	n := &domain.Notification{
		UserID:           targetUserID,
		Type:             "like",
		Link:             fmt.Sprintf("/%s/%s#comment-%s", seg, targetID, commentID),
		ActorID:          actorID,
		ActorName:        actorName,
		ActorAvatar:      actorAvatar,
		RelatedCommentID: commentID,
	}
	s.populateTargetRichContext(ctx, n, targetTypeKey, targetID)
	if n.StoryID == "" {
		n.StoryID = s.resolveStoryIDForCommentTarget(ctx, targetTypeKey, targetID)
	}
	sum := s.targetSummaryPhrase(n, targetTypeKey)
	n.Title = "评论被赞"
	n.Content = fmt.Sprintf("%s 赞了你在 %s 下的评论", actorName, sum)
	return s.CreateNotificationWithPush(ctx, n)
}

// NotifyMention 提及通知
func (s *Service) NotifyMention(ctx context.Context, targetUserID, actorID, actorName, actorAvatar, targetTypeKey, targetID, commentID string) error {
	// 不通知自己
	if targetUserID == actorID {
		return nil
	}

	seg := targetPathSegment(targetTypeKey)
	n := &domain.Notification{
		UserID:           targetUserID,
		Type:             "mention",
		Link:             fmt.Sprintf("/%s/%s#comment-%s", seg, targetID, commentID),
		ActorID:          actorID,
		ActorName:        actorName,
		ActorAvatar:      actorAvatar,
		RelatedCommentID: commentID,
	}
	s.populateTargetRichContext(ctx, n, targetTypeKey, targetID)
	if n.StoryID == "" {
		n.StoryID = s.resolveStoryIDForCommentTarget(ctx, targetTypeKey, targetID)
	}
	if cmt, err := s.repo.CommentByID(ctx, commentID); err == nil && cmt != nil {
		t := strings.TrimSpace(cmt.Content)
		if t != "" {
			n.CommentText = truncateNotificationText(t, 400)
		}
	}
	sum := s.targetSummaryPhrase(n, targetTypeKey)
	n.Title = "有人提及了你"
	if n.CommentText != "" {
		n.Content = fmt.Sprintf("%s 在 %s 的评论中提到你：\n\n%s", actorName, sum, n.CommentText)
	} else {
		n.Content = fmt.Sprintf("%s 在 %s 的评论中提及了你", actorName, sum)
	}
	return s.CreateNotificationWithPush(ctx, n)
}

// NotifyGroupInvitation 群组邀请通知
func (s *Service) NotifyGroupInvitation(ctx context.Context, inviteeID, inviterID, inviterName, inviterAvatar, groupID, groupName string) error {
	gn := strings.TrimSpace(groupName)
	if gn == "" {
		gn = "未命名群组"
	}
	n := &domain.Notification{
		UserID:      inviteeID,
		Type:        "group_invite",
		Title:       "群组邀请",
		Content:     fmt.Sprintf("%s 邀请你加入群组「%s」。", inviterName, gn),
		Link:        fmt.Sprintf("/groups/%s", groupID),
		ActorID:     inviterID,
		ActorName:   inviterName,
		ActorAvatar: inviterAvatar,
	}
	return s.CreateNotificationWithPush(ctx, n)
}

// NotifyFragmentGenerationCompleted AI 故事碎片生成完成（通知作者本人）
func (s *Service) NotifyFragmentGenerationCompleted(ctx context.Context, userID, fragmentID, previewTitle string, tokensUsed int) error {
	title := strings.TrimSpace(previewTitle)
	if title == "" {
		title = "你的故事碎片"
	}
	if utf8.RuneCountInString(title) > 48 {
		title = string([]rune(title)[:48]) + "…"
	}
	n := &domain.Notification{
		UserID:     userID,
		Type:       "fragment_generation_complete",
		Title:      "碎片生成完成",
		Link:       fmt.Sprintf("/fragments/%s", fragmentID),
		FragmentID: fragmentID,
		TokensUsed: tokensUsed,
	}
	s.populateTargetRichContext(ctx, n, "fragment", fragmentID)
	if strings.TrimSpace(n.StoryTitle) == "" {
		n.StoryTitle = title
	}
	n.Content = fmt.Sprintf("「%s」已生成完成，包含文案与配图。点按下方按钮可打开碎片继续编辑。", title)
	return s.CreateNotificationWithPush(ctx, n)
}

// NotifyStoryboardInitialGenerationCompleted 分镜初版 AI 生成完成（内容+分镜落库，通知创建者）
func (s *Service) NotifyStoryboardInitialGenerationCompleted(ctx context.Context, userID, storyboardID, storyID string, tokensUsed int) error {
	n := &domain.Notification{
		UserID:       userID,
		Type:         "storyboard_generation_complete",
		Title:        "分镜生成完成",
		Link:         fmt.Sprintf("/stories/%s/storyboards/%s", storyID, storyboardID),
		StoryID:      storyID,
		StoryboardID: storyboardID,
		TokensUsed:   tokensUsed,
	}
	s.populateTargetRichContext(ctx, n, "storyboard", storyboardID)
	sum := s.targetSummaryPhrase(n, "storyboard")
	n.Content = fmt.Sprintf("AI 已为 %s 生成剧情与场景结构，可继续编辑文案、调整分镜或生成配图/视频。", sum)
	return s.CreateNotificationWithPush(ctx, n)
}

// NotifyFragmentGenerationFailed AI 故事碎片生成失败（通知作者本人）
func (s *Service) NotifyFragmentGenerationFailed(ctx context.Context, userID, fragmentID, reason string) error {
	reason = strings.TrimSpace(truncateNotificationText(reason, 280))
	n := &domain.Notification{
		UserID:     userID,
		Type:       "fragment_generation_failed",
		Title:      "碎片生成失败",
		Link:       fmt.Sprintf("/fragments/%s", fragmentID),
		FragmentID: fragmentID,
	}
	s.populateTargetRichContext(ctx, n, "fragment", fragmentID)
	if reason != "" {
		n.Content = fmt.Sprintf("生成未能完成。原因说明：\n\n%s\n\n可打开草稿查看详情或稍后重试。", reason)
	} else {
		n.Content = "生成未能完成。可打开草稿查看详情或稍后重试。"
	}
	return s.CreateNotificationWithPush(ctx, n)
}

// NotifyStoryboardGenerationFailed 分镜相关 AI 生成失败（初版、Fork、续写等；storyboardId 须为已存在的分镜 ID，用于跳转）
func (s *Service) NotifyStoryboardGenerationFailed(ctx context.Context, userID, storyID, storyboardID, reason string) error {
	reason = strings.TrimSpace(truncateNotificationText(reason, 280))
	n := &domain.Notification{
		UserID:       userID,
		Type:         "storyboard_generation_failed",
		Title:        "分镜生成失败",
		Link:         fmt.Sprintf("/stories/%s/storyboards/%s", storyID, storyboardID),
		StoryID:      storyID,
		StoryboardID: storyboardID,
	}
	s.populateTargetRichContext(ctx, n, "storyboard", storyboardID)
	if reason != "" {
		n.Content = fmt.Sprintf("AI 未能完成该分镜的生成。原因说明：\n\n%s\n\n请稍后重试或手动编辑分镜。", reason)
	} else {
		n.Content = "AI 未能完成该分镜的生成。请稍后重试或手动编辑分镜。"
	}
	return s.CreateNotificationWithPush(ctx, n)
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
	n := &domain.Notification{
		UserID:       followerUserID,
		Type:         "story_follow_storyboard",
		Title:        "关注的故事更新了",
		Link:         fmt.Sprintf("/stories/%s/storyboards/%s", storyID, storyboardID),
		ActorID:      publisherID,
		ActorName:    publisherName,
		ActorAvatar:  publisherAvatar,
		StoryID:      storyID,
		StoryboardID: storyboardID,
	}
	s.populateTargetRichContext(ctx, n, "storyboard", storyboardID)
	sbTitle := strings.TrimSpace(n.StoryboardTitle)
	if sbTitle == "" {
		sbTitle = "新分镜"
	}
	n.StoryTitle = st
	n.Content = fmt.Sprintf("你关注的「%s」刚发布了分镜《%s》，由 %s 更新。", st, sbTitle, publisherName)
	return s.CreateNotificationWithPush(ctx, n)
}
