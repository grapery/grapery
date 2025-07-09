package llmchat

import (
	"context"

	llmchatpkg "github.com/grapery/grapery/pkg/llmchat"
)

// CreateSessionService 创建会话，供handler调用
func CreateSessionService(ctx context.Context, userID int64, name string) (interface{}, error) {
	return llmchatpkg.CreateSession(ctx, userID, name)
}

// SendMessageService 发送消息，供handler调用
func SendMessageService(ctx context.Context, sessionID, userID int64, content string, parentID *int64) (interface{}, error) {
	return llmchatpkg.SendMessage(ctx, sessionID, userID, content, parentID)
}

// RetryMessageService 重试消息，供handler调用
func RetryMessageService(ctx context.Context, msgID int64) (interface{}, error) {
	return llmchatpkg.RetryMessage(ctx, msgID)
}

// FeedbackMessageService 消息反馈，供handler调用
func FeedbackMessageService(ctx context.Context, msgID, userID int64, feedbackType, content string) (interface{}, error) {
	return llmchatpkg.FeedbackMessage(ctx, msgID, userID, feedbackType, content)
}

// InterruptMessageService 中断消息，供handler调用
func InterruptMessageService(ctx context.Context, msgID int64) error {
	return llmchatpkg.InterruptMessage(ctx, msgID)
}
