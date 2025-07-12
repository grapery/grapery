package llmchat

import (
	"context"
	"fmt"
	"strconv"
	"time"

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
	err := models.Init(cfg.SqlDB.Username, cfg.SqlDB.Password, cfg.SqlDB.Address, cfg.SqlDB.Database)
	if err != nil {
		logrus.Errorf("init sql database failed : [%s]", err.Error())
		return err
	}
	return nil
}

// CreateSessionService 创建会话，供handler调用
func CreateSessionService(ctx context.Context, userID int64, name, roleId, botId string) (interface{}, error) {
	sessionId := uuid.NewString()
	logrus.Infof("CreateSessionService: userID: %d, name: %s, roleId: %s, botId: %s", userID, name, roleId, botId)
	return llmchatpkg.GetLLMChatEngine().CreateSession(ctx, userID, name, sessionId, roleId, botId)
}

// SendMessageService 发送消息，供handler调用
func SendMessageService(ctx context.Context, sessionID string, userID int64, content string) (interface{}, error) {
	logrus.Infof("SendMessageService: sessionID: %s, userID: %d, content: %s", sessionID, userID, content)
	return llmchatpkg.GetLLMChatEngine().SendMessage(ctx, sessionID, userID, content)
}

// RetryMessageService 重试消息，供handler调用
func RetryMessageService(ctx context.Context, msgID int64) (interface{}, error) {
	logrus.Infof("RetryMessageService: msgID: %d", msgID)
	return llmchatpkg.GetLLMChatEngine().RetryMessage(ctx, msgID)
}

// FeedbackMessageService 消息反馈，供handler调用
func FeedbackMessageService(ctx context.Context, msgID, userID int64, feedbackType int) (interface{}, error) {
	logrus.Infof("FeedbackMessageService: msgID: %d, userID: %d, feedbackType: %s, content: %s", msgID, userID, feedbackType)
	return llmchatpkg.GetLLMChatEngine().FeedbackMessage(ctx, msgID, userID, feedbackType)
}

// InterruptMessageService 中断消息，供handler调用
func InterruptMessageService(ctx context.Context, msgID int64) error {
	logrus.Infof("InterruptMessageService: msgID: %d", msgID)
	return llmchatpkg.GetLLMChatEngine().InterruptMessage(ctx, msgID)
}

func ClearSessionService(ctx context.Context, sessionID string) error {
	logrus.Infof("ClearSessionService: sessionID: %s", sessionID)
	return llmchatpkg.GetLLMChatEngine().ConversationClear(ctx, sessionID)
}

func SessionMessageService(ctx context.Context, sessionID string, page, pageSize int) ([]*llmchatpkg.LLMChatMessage, bool, error) {
	msgs, hasMore, err := llmchatpkg.GetLLMChatEngine().SessionMessages(ctx, sessionID, page, pageSize)
	if err != nil {
		return nil, false, err
	}
	return msgs, hasMore, nil
}

// CopyMessageService 根据旧消息ID复制一条新消息，生成新messageId
func CopyMessageService(ctx context.Context, oldMessageID int64, userID int64) (*models.LLMChatMsg, error) {
	// 声明类型！详细注释
	// 1. 查询原消息
	var oldMsg models.LLMChatMsg
	if err := oldMsg.GetById(ctx, oldMessageID); err != nil {
		return nil, err
	}
	if oldMsg.Deleted {
		return nil, fmt.Errorf("原消息已删除")
	}
	// 2. 生成新messageId
	newMessageId := uuid.New().String()
	// 3. 构造新消息，内容与原消息一致，messageId不同，时间更新
	newMsg := &models.LLMChatMsg{
		SessionID:      oldMsg.SessionID,
		MessageId:      newMessageId,
		ConversationId: oldMsg.ConversationId,
		UserID:         userID, // 归属当前用户
		Content:        oldMsg.Content,
		LLmContent:     oldMsg.LLmContent,
		MsgType:        oldMsg.MsgType,
		Status:         "pending",
		Like:           0,
		Attachments:    oldMsg.Attachments,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		Deleted:        false,
	}
	// 4. 保存新消息
	if err := newMsg.Create(ctx); err != nil {
		return nil, err
	}
	return newMsg, nil
}

// GetSessionService 根据用户ID和角色ID获取指定session
func GetSessionService(ctx context.Context, userID int64, roleID int64) (*llmchatpkg.LLMChatSession, error) {
	// 声明类型！详细注释
	sess, err := models.GetUserSessionByUserIDAndRoleID(ctx, userID, roleID)
	if err != nil {
		return nil, err
	}
	if sess == nil {
		return nil, nil
	}
	session := &llmchatpkg.LLMChatSession{
		UserID:    sess.UserID,
		Name:      sess.Name,
		SessionId: sess.SessionId,
		RoleId:    strconv.FormatInt(roleID, 10),
		BotId:     sess.BotId,
		MsgCount:  sess.MsgCount,
		StartTime: sess.StartTime,
		EndTime:   sess.EndTime,
		CreatedAt: sess.CreatedAt,
		UpdatedAt: sess.UpdatedAt,
	}
	return session, nil
}
