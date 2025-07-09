package llmchat

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/grapery/grapery/models"
)

// CreateSession 创建会话
func CreateSession(ctx context.Context, userID int64, name string) (*models.Session, error) {
	s := &models.Session{
		UserID: userID,
		Name:   name,
	}
	if err := s.Create(ctx); err != nil {
		return nil, err
	}
	return s, nil
}

// SendMessage 在会话中发送消息
func SendMessage(ctx context.Context, sessionID, userID int64, content string, parentID *int64) (*models.LLMChatMsg, error) {
	msg := &models.LLMChatMsg{
		SessionID: sessionID,
		UserID:    userID,
		Content:   content,
		MsgType:   "user",
		Status:    "pending",
		ParentID:  parentID,
	}
	if err := msg.Create(ctx); err != nil {
		return nil, err
	}
	return msg, nil
}

// RetryMessage 重试用户消息（复制原消息内容，parentID指向原消息）
func RetryMessage(ctx context.Context, msgID int64) (*models.LLMChatMsg, error) {
	var oldMsg models.LLMChatMsg
	if err := oldMsg.GetById(ctx, msgID); err != nil {
		return nil, err
	}
	if oldMsg.MsgType != "user" {
		return nil, errors.New("只能重试用户消息")
	}
	return SendMessage(ctx, oldMsg.SessionID, oldMsg.UserID, oldMsg.Content, &oldMsg.ID)
}

// FeedbackMessage 对消息进行反馈
func FeedbackMessage(ctx context.Context, msgID, userID int64, feedbackType, content string) (*models.LLMMsgFeedback, error) {
	fb := &models.LLMMsgFeedback{
		MsgID:   msgID,
		UserID:  userID,
		Type:    feedbackType, // like/dislike/complaint
		Content: content,
	}
	if err := fb.Create(ctx); err != nil {
		return nil, err
	}
	return fb, nil
}

// InterruptMessage 中断消息（将状态置为interrupted）
func InterruptMessage(ctx context.Context, msgID int64) error {
	var msg models.LLMChatMsg
	if err := msg.GetById(ctx, msgID); err != nil {
		return err
	}
	return msg.UpdateStatus(ctx, "interrupted")
}

// LLMStreamCallback 回调类型
// 每生成一段内容就调用一次
// 声明类型！
type LLMStreamCallback func(chunk string) error

// LLMStreamChat 流式推理，支持context中断
func LLMStreamChat(ctx context.Context, prompt string, cb LLMStreamCallback) error {
	for i := 1; i <= 5; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := cb(fmt.Sprintf("AI响应片段%d", i)); err != nil {
				return err
			}
			time.Sleep(500 * time.Millisecond)
		}
	}
	return nil
}
