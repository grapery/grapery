package service

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/email"
	"go.uber.org/zap"
)

const (
	notificationTypeModerationReportReceived = "moderation_report_received"
	notificationTypeModerationBlockConfirmed = "moderation_block_confirmed"
	notificationTypeModerationReportResolved = "moderation_report_resolved"
)

func moderationNotifyEmails() []string {
	raw := strings.TrimSpace(os.Getenv("MODERATION_NOTIFY_EMAILS"))
	if raw == "" {
		raw = "putaoshuyunying@grapery.xyz"
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func (s *Service) notifyModerationReportReceived(ctx context.Context, reporterID, targetKind, targetID, reason string) {
	n := &domain.Notification{
		UserID: reporterID,
		Type:   notificationTypeModerationReportReceived,
		Title:  "举报已收到",
		Link:   moderationReporterLink(targetKind, targetID),
	}
	switch strings.ToLower(strings.TrimSpace(targetKind)) {
	case "user":
		if u, err := s.repo.UserByID(ctx, targetID); err == nil && u != nil {
			name := displayNameForUser(u)
			n.Content = fmt.Sprintf("我们已收到你对用户「%s」的举报，审核团队将在 24 小时内处理。感谢你为社区安全所做的贡献。", name)
			n.ActorID = targetID
			n.ActorName = name
			n.ActorAvatar = strings.TrimSpace(u.Avatar)
		} else {
			n.Content = "我们已收到你的用户举报，审核团队将在 24 小时内处理。感谢你为社区安全所做的贡献。"
		}
	default:
		n.Content = "我们已收到你的内容举报，审核团队将在 24 小时内处理。感谢你为社区安全所做的贡献。"
		s.populateTargetRichContext(ctx, n, targetKind, targetID)
	}
	if r := strings.TrimSpace(reason); r != "" {
		n.Content += "\n\n举报原因：" + truncateNotificationText(r, 200)
	}
	if err := s.CreateNotificationWithPush(ctx, n); err != nil {
		s.logger.Warn("failed to send moderation report confirmation",
			zap.Error(err),
			zap.String("reporterId", reporterID))
	}
}

func (s *Service) notifyModerationBlockConfirmed(ctx context.Context, blockerID, blockedID string) {
	n := &domain.Notification{
		UserID: blockerID,
		Type:   notificationTypeModerationBlockConfirmed,
		Title:  "已屏蔽用户",
		Link:   "/settings/blocked-users",
	}
	if u, err := s.repo.UserByID(ctx, blockedID); err == nil && u != nil {
		name := displayNameForUser(u)
		n.Content = fmt.Sprintf("你已成功屏蔽用户「%s」。对方的动态与内容将不再出现在你的信息流中；我们已记录该操作并将进行必要复核。", name)
		n.ActorID = blockedID
		n.ActorName = name
		n.ActorAvatar = strings.TrimSpace(u.Avatar)
	} else {
		n.Content = "你已成功屏蔽该用户。对方的动态与内容将不再出现在你的信息流中；我们已记录该操作并将进行必要复核。"
	}
	if err := s.CreateNotificationWithPush(ctx, n); err != nil {
		s.logger.Warn("failed to send block confirmation notification",
			zap.Error(err),
			zap.String("blockerId", blockerID))
	}
}

func (s *Service) notifyModerationTeamAsync(event string, fields map[string]string) {
	recipients := moderationNotifyEmails()
	if len(recipients) == 0 {
		return
	}
	go func() {
		if err := email.ModerationAlertEmail(recipients, event, fields); err != nil {
			s.logger.Warn("failed to send moderation alert email",
				zap.Error(err),
				zap.String("event", event))
		}
	}()
}

func moderationReporterLink(targetKind, targetID string) string {
	switch strings.ToLower(strings.TrimSpace(targetKind)) {
	case "user":
		return fmt.Sprintf("/users/%s", targetID)
	default:
		seg := targetPathSegment(targetKind)
		if seg == "" || targetID == "" {
			return "/notifications"
		}
		return fmt.Sprintf("/%s/%s", seg, targetID)
	}
}

func displayNameForUser(u *domain.User) string {
	if u == nil {
		return "用户"
	}
	if n := strings.TrimSpace(u.DisplayName); n != "" {
		return n
	}
	if n := strings.TrimSpace(u.Username); n != "" {
		return n
	}
	return "用户"
}
