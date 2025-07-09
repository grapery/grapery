package llmchat

import (
	"context"

	"github.com/google/uuid"
	"github.com/grapery/grapery/config"
	"github.com/grapery/grapery/models"
	llmchatpkg "github.com/grapery/grapery/pkg/llmchat"
	"github.com/grapery/grapery/utils/cache"
	"github.com/sirupsen/logrus"
)

func Init(cfg *config.Config) error {
	llmchatpkg.GetLLMChatEngine()
	cache.NewRedisClient(cfg)
	err := models.Init(cfg.SqlDB.Username, cfg.SqlDB.Password, cfg.SqlDB.Database)
	if err != nil {
		logrus.Errorf("init sql database failed : [%s]", err.Error())
		return err
	}
	return nil
}

// CreateSessionService 创建会话，供handler调用
func CreateSessionService(ctx context.Context, userID int64, name, roleId, botId string) (interface{}, error) {
	sessionId := uuid.NewString()
	return llmchatpkg.GetLLMChatEngine().CreateSession(ctx, userID, name, sessionId, roleId, botId)
}

// SendMessageService 发送消息，供handler调用
func SendMessageService(ctx context.Context, sessionID string, userID int64, content string) (interface{}, error) {
	return llmchatpkg.GetLLMChatEngine().SendMessage(ctx, sessionID, userID, content)
}

// RetryMessageService 重试消息，供handler调用
func RetryMessageService(ctx context.Context, msgID int64) (interface{}, error) {
	return llmchatpkg.GetLLMChatEngine().RetryMessage(ctx, msgID)
}

// FeedbackMessageService 消息反馈，供handler调用
func FeedbackMessageService(ctx context.Context, msgID, userID int64, feedbackType, content string) (interface{}, error) {
	return llmchatpkg.GetLLMChatEngine().FeedbackMessage(ctx, msgID, userID, feedbackType, content)
}

// InterruptMessageService 中断消息，供handler调用
func InterruptMessageService(ctx context.Context, msgID int64) error {
	return llmchatpkg.GetLLMChatEngine().InterruptMessage(ctx, msgID)
}

func ClearSessionService(ctx context.Context, sessionID string) error {
	return llmchatpkg.GetLLMChatEngine().ConversationClear(ctx, sessionID)
}

func SessionMessageService(ctx context.Context, sessionID string, page, pageSize int) ([]*models.LLMChatMsg, bool, error) {
	msgs, hasMore, err := llmchatpkg.GetLLMChatEngine().SessionMessages(ctx, sessionID, page, pageSize)
	if err != nil {
		return nil, false, err
	}
	return msgs, hasMore, nil
}
