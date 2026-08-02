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
func SendMessageService(ctx context.Context, sessionID string, userID int64, content string, assists map[string]string) (interface{}, error) {
	logrus.Infof("SendMessageService: sessionID: %s, userID: %d, content: %s", sessionID, userID, content)
	return llmchatpkg.GetLLMChatEngine().SendMessage(ctx, sessionID, userID, content, assists)
}

// RetryMessageService 重试消息，供handler调用
func RetryMessageService(ctx context.Context, msgID int64) (interface{}, error) {
	logrus.Infof("RetryMessageService: msgID: %d", msgID)
	return llmchatpkg.GetLLMChatEngine().RetryMessage(ctx, msgID)
}

// FeedbackMessageService 消息反馈，供handler调用
func FeedbackMessageService(ctx context.Context, msgID string, userID int64, feedbackType int) (interface{}, error) {
	logrus.Infof("FeedbackMessageService: msgID: %s, userID: %d, feedbackType: %d", msgID, userID, feedbackType)
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
	lastMessage, err := models.GetLastMessageBySessionID(ctx, sess.SessionId)
	if err != nil {
		return nil, err
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
		LastMessage: llmchatpkg.LLMChatMessage{
			MessageId:      lastMessage.MessageId,
			SessionID:      lastMessage.SessionID,
			UserID:         lastMessage.UserID,
			Content:        lastMessage.Content,
			MsgType:        lastMessage.MsgType,
			Status:         lastMessage.Status,
			CreatedAt:      lastMessage.CreatedAt,
			UpdatedAt:      lastMessage.UpdatedAt,
			Deleted:        lastMessage.Deleted,
			ConversationId: lastMessage.ConversationId,
			LLmContent:     lastMessage.LLmContent,
			Like:           lastMessage.Like,
			Attachments:    lastMessage.Attachments,
		},
	}
	return session, nil
}

// SessionMessagePageByMessageID 根据message_id倒序分页获取消息
func SessionMessagePageByMessageID(ctx context.Context, sessionID, messageID string, pageSize int) ([]*llmchatpkg.LLMChatMessage, bool, error) {
	msgs, hasMore, err := llmchatpkg.GetLLMChatEngine().SessionMessagesByMessageID(ctx, sessionID, messageID, pageSize)
	if err != nil {
		return nil, false, err
	}
	if len(msgs) > pageSize {
		hasMore = true
		msgs = msgs[:pageSize]
	} else {
		hasMore = false
	}
	llmMsgs := make([]*llmchatpkg.LLMChatMessage, 0, len(msgs))
	for _, msg := range msgs {
		llmMsgs = append(llmMsgs, &llmchatpkg.LLMChatMessage{
			MessageId:      msg.MessageId,
			SessionID:      msg.SessionID,
			UserID:         msg.UserID,
			Content:        msg.Content,
			MsgType:        msg.MsgType,
			Status:         msg.Status,
			CreatedAt:      msg.CreatedAt,
			UpdatedAt:      msg.UpdatedAt,
			Deleted:        msg.Deleted,
			ConversationId: msg.ConversationId,
			LLmContent:     msg.LLmContent,
			Like:           msg.Like,
			Attachments:    msg.Attachments,
		})
	}
	return llmMsgs, hasMore, nil
}

func GetSessionBySessionID(ctx context.Context, sessionID string) (*llmchatpkg.LLMChatSession, error) {
	session, err := models.GetUserSessionBySessionId(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, nil
	}
	llmSession := &llmchatpkg.LLMChatSession{
		UserID:    session.UserID,
		Name:      session.Name,
		SessionId: session.SessionId,
		RoleId:    session.RoleId,
		BotId:     session.BotId,
		MsgCount:  session.MsgCount,
		StartTime: session.StartTime,
		EndTime:   session.EndTime,
		CreatedAt: session.CreatedAt,
		UpdatedAt: session.UpdatedAt,
	}
	return llmSession, nil
}

// GetUserSessionListService 获取指定用户的所有会话，按更新时间倒序，转换为 LLMChatSession 并补充 last_message 字段
func GetUserSessionListService(ctx context.Context, userID int64) ([]*llmchatpkg.LLMChatSession, error) {
	sessions, err := models.ListUserSessionsByUserId(ctx, userID)
	if err != nil {
		return nil, err
	}
	result := make([]*llmchatpkg.LLMChatSession, 0, len(sessions))
	for _, s := range sessions {
		sess := llmchatpkg.ToLLMChatSession(s)
		// 查询最后一条消息
		msg, err := models.GetLastMessageBySessionID(ctx, s.SessionId)
		if err == nil && msg != nil {
			sess.LastMessage = *llmchatpkg.ToLLMChatMessage(msg)
		}
		roleInt, err := strconv.ParseInt(s.RoleId, 10, 64)
		if err != nil {
			return nil, err
		}
		role, err := models.GetStoryRoleByID(ctx, roleInt)
		if err == nil && role != nil {
			sess.RoleDetail = llmchatpkg.RoleDetail{
				RoleId:          strconv.FormatInt(roleInt, 10),
				RoleName:        role.CharacterName,
				RoleDescription: role.CharacterDescription,
				RoleAvatar:      role.CharacterAvatar,
			}
		}
		result = append(result, sess)
	}
	return result, nil
}

func CreatBotService(ctx context.Context, userID int64, roleId int64, name, iconFileID string) (string, error) {
	logrus.Infof("CreatBotService: userID: %d, roleId: %d, name: %s, iconFileID: %s", userID, roleId, name, iconFileID)
	botId, err := llmchatpkg.GetLLMChatEngine().CreateBot(ctx, userID, roleId, name, iconFileID)
	if err != nil {
		logrus.Errorf("CreatBotService failed: userID: %d, roleId: %d, name: %s, iconFileID: %s, err: %v", userID, roleId, name, iconFileID, err)
		return "", err
	}
	logrus.Infof("CreatBotService success: userID: %d, roleId: %d, name: %s, iconFileID: %s, botId: %s", userID, roleId, name, iconFileID, botId)
	return botId, nil
}

func PublishBotService(ctx context.Context, userID int64, roleId int64, botId string) (string, error) {
	logrus.Infof("PublishBotService: userID: %d, roleId: %d, botId: %s", userID, roleId, botId)
	botId, err := llmchatpkg.GetLLMChatEngine().PublishBot(ctx, userID, roleId, botId)
	if err != nil {
		logrus.Errorf("PublishBotService failed: userID: %d, roleId: %d, botId: %s, err: %v", userID, roleId, botId, err)
		return "", err
	}
	logrus.Infof("PublishBotService: userID: %d, roleId: %d, botId: %s", userID, roleId, botId)
	return botId, nil
}

func UpdateBotService(ctx context.Context, userID int64, roleId int64, botId string) (string, error) {
	logrus.Infof("UpdateBotService: userID: %d, roleId: %d, botId: %s", userID, roleId, botId)
	botId, err := llmchatpkg.GetLLMChatEngine().UpdateBot(ctx, userID, roleId, botId)
	if err != nil {
		logrus.Errorf("UpdateBotService failed: userID: %d, roleId: %d, botId: %s, err: %v", userID, roleId, botId, err)
		return "", err
	}
	logrus.Infof("UpdateBotService: userID: %d, roleId: %d, botId: %s", userID, roleId, botId)
	return botId, nil
}

func DeleteBotService(ctx context.Context, userID int64, roleId int64, botId string) error {
	logrus.Infof("DeleteBotService: userID: %d, roleId: %d, botId: %s", userID, roleId, botId)
	err := llmchatpkg.GetLLMChatEngine().DeleteBot(ctx, userID, roleId, botId)
	if err != nil {
		logrus.Errorf("DeleteBotService failed: userID: %d, roleId: %d, botId: %s, err: %v", userID, roleId, botId, err)
		return err
	}
	logrus.Infof("DeleteBotService: userID: %d, roleId: %d, botId: %s", userID, roleId, botId)
	return nil
}

func GetBotService(ctx context.Context, userId int64, roleId int64, botId string) (string, error) {
	logrus.Infof("GetBotService: userID: %d, roleId: %d, botId: %s", userId, roleId, botId)
	botId, err := llmchatpkg.GetLLMChatEngine().GetBot(ctx, userId, roleId, botId)
	if err != nil {
		logrus.Errorf("GetBotService failed: userID: %d, roleId: %d, botId: %s, err: %v", userId, roleId, botId, err)
		return "", err
	}
	logrus.Infof("GetBotService: userID: %d, roleId: %d, botId: %s", userId, roleId, botId)
	return botId, nil
}
