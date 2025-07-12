package llmchat

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/grapery/grapery/models"
	"github.com/grapery/grapery/pkg/cloud/coze"
	"github.com/grapery/grapery/utils/log"
	"go.uber.org/zap"
	"gorm.io/datatypes"
)

// LLMChatEngine 封装LLM流式推理相关逻辑
// 声明类型！
type LLMChatEngine struct {
}

// NewLLMChatEngine 初始化LLMChatEngine
func NewLLMChatEngine() *LLMChatEngine {
	return &LLMChatEngine{}
}

var llmEngine *LLMChatEngine

func init() {
	llmEngine = NewLLMChatEngine()
}

func GetLLMChatEngine() *LLMChatEngine {
	return llmEngine
}

// ================= 对外暴露结构体定义 =================
// LLMChatSession 对外暴露的会话结构体，避免直接暴露 models.UserSession
// 仅包含业务需要的字段
// 声明类型！
type LLMChatSession struct {
	UserID         int64     `json:"user_id"`
	Name           string    `json:"name"`
	SessionId      string    `json:"session_id"`
	ConversationId string    `json:"conversation_id"`
	RoleId         string    `json:"role_id"`
	BotId          string    `json:"bot_id"`
	MsgCount       int       `json:"msg_count"`
	StartTime      int64     `json:"start_time"`
	EndTime        int64     `json:"end_time"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// LLMChatMessage 对外暴露的消息结构体，避免直接暴露 models.LLMChatMsg
// 声明类型！
type LLMChatMessage struct {
	ID             int64          `json:"id"`
	MessageId      string         `json:"message_id"`
	SessionID      string         `json:"session_id"`
	UserID         int64          `json:"user_id"`
	Content        string         `json:"content"`
	MsgType        string         `json:"msg_type"`
	Status         string         `json:"status"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	Deleted        bool           `json:"deleted"`
	ConversationId string         `json:"conversation_id"`
	LLmContent     string         `json:"llm_content"`
	Like           int            `json:"like"`
	Attachments    datatypes.JSON `json:"attachments"`
}

// LLMMsgFeedbackResp 对外暴露的反馈结构体，避免直接暴露 models.LLMMsgFeedback
// 声明类型！
type LLMMsgFeedbackResp struct {
	ID      int64  `json:"id"`
	MsgID   string `json:"msg_id"`
	UserID  int64  `json:"user_id"`
	Type    int    `json:"type"`
	Content string `json:"content"`
}

// ================= 工具函数：models -> 对外结构体转换 =================
// toLLMChatSession 将 models.UserSession 转为 LLMChatSession
func toLLMChatSession(s *models.UserSession) *LLMChatSession {
	if s == nil {
		return nil
	}
	return &LLMChatSession{
		UserID:         s.UserID,
		Name:           s.Name,
		SessionId:      s.SessionId,
		ConversationId: s.ConversationId,
		RoleId:         s.RoleId,
		BotId:          s.BotId,
		MsgCount:       s.MsgCount,
		StartTime:      s.StartTime,
		EndTime:        s.EndTime,
		CreatedAt:      s.CreatedAt,
		UpdatedAt:      s.UpdatedAt,
	}
}

// toLLMChatMessage 将 models.LLMChatMsg 转为 LLMChatMessage
func toLLMChatMessage(m *models.LLMChatMsg) *LLMChatMessage {
	if m == nil {
		return nil
	}
	return &LLMChatMessage{
		ID:             m.ID,
		MessageId:      m.MessageId,
		SessionID:      m.SessionID,
		UserID:         m.UserID,
		Content:        m.Content,
		MsgType:        m.MsgType,
		Status:         m.Status,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
		Deleted:        m.Deleted,
		ConversationId: m.ConversationId,
		LLmContent:     m.LLmContent,
		Like:           m.Like,
		Attachments:    m.Attachments,
	}
}

// toLLMMsgFeedbackResp 将 models.LLMMsgFeedback 转为 LLMMsgFeedbackResp
func toLLMMsgFeedbackResp(f *models.LLMMsgFeedback) *LLMMsgFeedbackResp {
	if f == nil {
		return nil
	}
	return &LLMMsgFeedbackResp{
		ID:      f.ID,
		MsgID:   f.MsgID,
		UserID:  f.UserID,
		Type:    f.Type,
		Content: f.Content,
	}
}

// toLLMChatMessageSlice 批量转换
func toLLMChatMessageSlice(msgs []*models.LLMChatMsg) []*LLMChatMessage {
	res := make([]*LLMChatMessage, 0, len(msgs))
	for _, m := range msgs {
		res = append(res, toLLMChatMessage(m))
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].CreatedAt.Before(res[j].CreatedAt)
	})
	return res
}

// CreateSession 创建会话
func (e *LLMChatEngine) CreateSession(ctx context.Context, userID int64, name, sessionId, roleId, botId string) (*LLMChatSession, error) {
	roleIdInt, err := strconv.ParseInt(roleId, 10, 64)
	if err != nil {
		log.Log().Error("[CreateSession] 转换roleId失败", zap.Error(err), zap.String("roleId", roleId))
		return nil, err
	}
	isExist, err := models.GetUserSessionByUserIDAndRoleID(ctx, userID, roleIdInt)
	if err != nil {
		log.Log().Error("[CreateSession] 获取会话失败", zap.Error(err), zap.Int64("userID", userID), zap.String("roleId", roleId))
		return nil, err
	}
	if isExist != nil {
		return toLLMChatSession(isExist), nil
	}
	log.Log().Info("[CreateSession] 开始创建会话", zap.Int64("userID", userID), zap.String("sessionId", sessionId), zap.String("roleId", roleId), zap.String("botId", botId), zap.String("name", name))
	conversationId, err := coze.GetCozeClient().ConversationCreate(ctx, coze.CozeConversationCreateParams{
		BotID: "7525037470162141226",
		MetaData: map[string]string{
			"uuid": sessionId,
		},
		Messages: []coze.EnterMessage{
			{
				Role:        "user",
				Content:     "你好",
				ContentType: "text",
			},
		},
	})
	if err != nil {
		log.Log().Error("[CreateSession] 创建会话失败", zap.Error(err), zap.Int64("userID", userID), zap.String("sessionId", sessionId))
		return nil, err
	}
	log.Log().Info("[CreateSession] 创建会话成功", zap.String("conversationId", conversationId))
	s := &models.UserSession{
		UserID:         userID,
		Name:           name,
		SessionId:      sessionId,
		ConversationId: conversationId,
		RoleId:         roleId,
		BotId:          botId,
		MsgCount:       0,
		StartTime:      time.Now().Unix(),
		EndTime:        time.Now().Unix(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	if err := s.Create(ctx); err != nil {
		log.Log().Error("[CreateSession] 创建会话失败", zap.Error(err), zap.Int64("userID", userID), zap.String("sessionId", sessionId))
		return nil, err
	}
	log.Log().Info("[CreateSession] 创建会话成功", zap.Int64("userID", userID), zap.String("sessionId", sessionId))
	return toLLMChatSession(s), nil
}

// SendMessage 在会话中发送消息
func (e *LLMChatEngine) SendMessage(ctx context.Context, sessionID string, userID int64, content string) (*LLMChatMessage, error) {
	log.Log().Info("[SendMessage] 用户发送消息", zap.Int64("userID", userID), zap.String("sessionID", sessionID), zap.String("content", content))
	var session models.UserSession
	if err := session.GetBySessionId(ctx, sessionID); err != nil {
		log.Log().Error("[SendMessage] 获取会话失败", zap.Error(err), zap.String("sessionID", sessionID))
		return nil, err
	}
	session.MsgCount++
	session.UpdatedAt = time.Now()
	if err := session.UpdateBySessionId(ctx, sessionID, map[string]interface{}{
		"msg_count":  session.MsgCount,
		"updated_at": time.Now(),
	}); err != nil {
		log.Log().Error("[SendMessage] 更新会话失败", zap.Error(err), zap.String("sessionID", sessionID))
		return nil, err
	}
	msg := &models.LLMChatMsg{
		MessageId:      uuid.New().String(),
		SessionID:      sessionID,
		UserID:         session.UserID,
		Content:        content,
		MsgType:        "user",
		Status:         "pending",
		Like:           0,
		Attachments:    datatypes.JSON{},
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		Deleted:        false,
		ConversationId: session.ConversationId,
	}
	if err := msg.Create(ctx); err != nil {
		log.Log().Error("[SendMessage] 消息写入失败", zap.Error(err), zap.Int64("userID", userID), zap.String("sessionID", sessionID))
		return nil, err
	}
	log.Log().Info("[SendMessage] 消息写入成功", zap.Int64("userID", userID), zap.String("sessionID", sessionID), zap.Int64("msgID", msg.ID))
	return toLLMChatMessage(msg), nil
}

// RetryMessage 重试用户消息（复制原消息内容）
func (e *LLMChatEngine) RetryMessage(ctx context.Context, msgID int64) (*LLMChatMessage, error) {
	log.Log().Info("[RetryMessage] 开始重试消息", zap.Int64("msgID", msgID))
	var oldMsg models.LLMChatMsg
	if err := oldMsg.GetById(ctx, msgID); err != nil {
		log.Log().Error("[RetryMessage] 获取原消息失败", zap.Error(err), zap.Int64("msgID", msgID))
		return nil, err
	}
	if oldMsg.MsgType != "user" {
		log.Log().Warn("[RetryMessage] 只能重试用户消息", zap.Int64("msgID", msgID))
		return nil, errors.New("只能重试用户消息")
	}
	msg, err := e.SendMessage(ctx, oldMsg.SessionID, oldMsg.UserID, oldMsg.Content)
	if err != nil {
		log.Log().Error("[RetryMessage] 重试消息失败", zap.Error(err), zap.Int64("msgID", msgID))
		return nil, err
	}
	log.Log().Info("[RetryMessage] 重试消息成功", zap.Int64("msgID", msgID), zap.Int64("newMsgID", msg.ID))
	return msg, nil
}

// FeedbackMessage 对消息进行反馈
func (e *LLMChatEngine) FeedbackMessage(ctx context.Context, msgID string, userID int64, feedbackType int) (*LLMMsgFeedbackResp, error) {
	log.Log().Info("[FeedbackMessage] 用户反馈消息", zap.String("msgID", msgID), zap.Int64("userID", userID), zap.Int("type", feedbackType))
	msg := &models.LLMChatMsg{}
	if err := msg.GetByMessageId(ctx, msgID); err != nil {
		log.Log().Error("[FeedbackMessage] 获取消息失败", zap.Error(err), zap.String("msgID", msgID))
		return nil, err
	}
	if msg.Like != feedbackType {
		msg.Like = feedbackType
		if err := msg.UpdateByMessageId(ctx, msg.MessageId, map[string]interface{}{
			"like": feedbackType,
		}); err != nil {
			log.Log().Error("[FeedbackMessage] 更新消息失败", zap.Error(err), zap.String("msgID", msgID))
			return nil, err
		}
	}
	fb := &models.LLMMsgFeedback{
		MsgID:  msgID,
		UserID: userID,
		Type:   feedbackType, // like/dislike/complaint
	}
	if feedbackType == 1 {
		fb.Content = "like"
	} else if feedbackType == 2 {
		fb.Content = "dislike"
	} else {
		fb.Content = ""
	}
	if err := fb.Create(ctx); err != nil {
		log.Log().Error("[FeedbackMessage] 反馈写入失败", zap.Error(err), zap.String("msgID", msgID), zap.Int64("userID", userID))
		return nil, err
	}
	log.Log().Info("[FeedbackMessage] 反馈写入成功", zap.String("msgID", msgID), zap.Int64("userID", userID))
	return toLLMMsgFeedbackResp(fb), nil
}

// InterruptMessage 中断消息（将状态置为interrupted）
func (e *LLMChatEngine) InterruptMessage(ctx context.Context, msgID int64) error {
	log.Log().Info("[InterruptMessage] 开始中断消息", zap.Int64("msgID", msgID))
	var msg models.LLMChatMsg
	if err := msg.GetById(ctx, msgID); err != nil {
		log.Log().Error("[InterruptMessage] 获取消息失败", zap.Error(err), zap.Int64("msgID", msgID))
		return err
	}
	err := msg.UpdateStatus(ctx, "interrupted")
	if err != nil {
		log.Log().Error("[InterruptMessage] 更新消息状态失败", zap.Error(err), zap.Int64("msgID", msgID))
		return err
	}
	log.Log().Info("[InterruptMessage] 消息已中断", zap.Int64("msgID", msgID))
	return nil
}

// SendMessageStream 流式发送消息，实时返回AI回复内容
// 返回：流式内容channel，error
func (e *LLMChatEngine) SendMessageStream(ctx context.Context, sessionID string, messageId string, userID int64, content string, msgChan chan string, answerMap map[string][]coze.AnswerOrFollowUp) error {
	log.Log().Info("[SendMessageStream] 入口参数", zap.Int64("userID", userID), zap.String("sessionID", sessionID), zap.String("content", content))

	realMsg := &models.LLMChatMsg{
		MessageId: messageId,
	}
	err := realMsg.GetByMessageId(ctx, messageId)
	if err != nil {
		log.Log().Error("[SendMessageStream] 获取消息失败", zap.Error(err), zap.String("messageId", messageId))
		return err
	}
	if realMsg.Status != "pending" {
		log.Log().Error("[SendMessageStream] 消息状态错误", zap.String("messageId", messageId), zap.String("status", realMsg.Status))
		return errors.New("消息状态错误")
	}
	// 2. 构造Coze流式参数（此处RoleName等可根据业务调整）
	cozeClient := coze.GetCozeClient()
	params := coze.CozeChatWithRoleStreamParams{
		ConversationID: realMsg.ConversationId,
		BotID:          "7525037470162141226",
		AdditionalMessages: []coze.CozeAdditionalMessage{
			{
				Content:     content,
				ContentType: "text",
				Role:        "user",
				Type:        "question",
			},
		},
		Stream:          true,
		AutoSaveHistory: true,
		UserID:          realMsg.MessageId,
	}
	log.Log().Info("[SendMessageStream] 构造Coze流式参数", zap.Any("params", params))
	go func(ctx context.Context) {
		// 3. 调用Coze流式API
		defer close(msgChan)
		log.Log().Info("[SendMessageStream] 调用Coze流式API", zap.String("sessionID", sessionID))
		err := cozeClient.ContinueChatWithAssistantStream(ctx, params, msgChan, answerMap)
		if err != nil {
			log.Log().Error("[SendMessageStream] Coze流式API调用失败", zap.Error(err), zap.Int64("userID", userID), zap.String("sessionID", sessionID))
			return
		}
		realMsg.Status = "sent"
		if len(answerMap["answer"]) > 0 {
			realMsg.LLmContent = answerMap["answer"][0].Content
		} else {
			realMsg.LLmContent = "AI回复内容为空"
		}
		err = realMsg.UpdateByMessageId(ctx, realMsg.MessageId, map[string]interface{}{
			"status":      realMsg.Status,
			"llm_content": realMsg.LLmContent,
		})
		if err != nil {
			log.Log().Error("[SendMessageStream] 更新消息失败", zap.Error(err), zap.Int64("msgID", realMsg.ID))
		}
	}(ctx)
	log.Log().Info("[SendMessageStream] 流式发送消息启动成功", zap.Int64("userID", userID), zap.String("sessionID", sessionID))
	return nil
}

func (e *LLMChatEngine) ConversationClear(ctx context.Context, sessionID string) error {
	log.Log().Info("[ConversationClear] 开始清空会话", zap.String("sessionID", sessionID))
	err := coze.GetCozeClient().ConversationClear(ctx, sessionID)
	if err != nil {
		log.Log().Error("[ConversationClear] 清空会话失败", zap.Error(err), zap.String("sessionID", sessionID))
		return err
	}
	log.Log().Info("[ConversationClear] 清空会话成功", zap.String("sessionID", sessionID))
	return nil
}

func (e *LLMChatEngine) SessionMessages(ctx context.Context, sessionID string, page, pageSize int) ([]*LLMChatMessage, bool, error) {
	log.Log().Info("[SessionMessages] 开始获取会话消息", zap.String("sessionID", sessionID), zap.Int("page", page), zap.Int("pageSize", pageSize))
	msgs, total, err := models.ListLLMChatMsgsBySessionIDWithPage(ctx, sessionID, page, pageSize)
	if err != nil {
		log.Log().Error("[SessionMessages] 获取会话消息失败", zap.Error(err), zap.String("sessionID", sessionID), zap.Int("page", page), zap.Int("pageSize", pageSize))
		return nil, false, err
	}
	return toLLMChatMessageSlice(msgs), total > int64(page*pageSize), nil
}

func (e *LLMChatEngine) SessionMessagesByMessageID(ctx context.Context, sessionID string, messageID string, pageSize int) ([]*LLMChatMessage, bool, error) {
	log.Log().Info("[SessionMessagesByMessageID] 开始获取会话消息", zap.String("sessionID", sessionID), zap.String("messageID", messageID), zap.Int("pageSize", pageSize))
	msgs, err := models.ListMsgsBySessionIdBeforeMessageId(ctx, sessionID, messageID, pageSize+1)
	if err != nil {
		log.Log().Error("[SessionMessagesByMessageID] 获取会话消息失败", zap.Error(err), zap.String("sessionID", sessionID), zap.String("messageID", messageID), zap.Int("pageSize", pageSize))
		return nil, false, err
	}
	var hasMore bool
	if len(msgs) > pageSize {
		msgs = msgs[:pageSize]
		hasMore = true
	} else {
		hasMore = false
	}
	return toLLMChatMessageSlice(msgs), hasMore, nil
}

// parseSSEData 解析SSE格式的data字段
// 输入示例："data: hello world\n\n"，返回"hello world"
func parseSSEData(chunk string) string {
	prefix := "data: "
	if strings.HasPrefix(chunk, prefix) {
		// 去除前缀和结尾换行
		return strings.TrimSpace(strings.TrimPrefix(chunk, prefix))
	}
	return ""
}
