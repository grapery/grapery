package message

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	api "github.com/grapery/common-protoc/gen"
	"github.com/grapery/grapery/models"
	"github.com/grapery/grapery/pkg/client"
	"github.com/grapery/grapery/utils/log"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// 添加一些有用的常量和类型
const (
	MaxMessageSize   = 8 * 1024 * 1024 // 8MB
	ClientBufferSize = 1000
)

type Client struct {
	ID       string
	Messages chan *api.StreamChatMessage
	LastSeen time.Time
	mu       sync.RWMutex
}

// 实现消息服务
type MessageService struct {
	api.UnimplementedStreamMessageServiceServer
	// 管理活跃的客户端连接
	clients map[string]*Client
	mu      sync.RWMutex
	client  *client.ZhipuStoryClient
}

// 添加心跳检测
func (s *MessageService) cleanupInactiveClients() {
	log.Log().Info("start cleanup inactive clients")
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for id, client := range s.clients {
			client.mu.RLock()
			if now.Sub(client.LastSeen) > 30*time.Minute {
				close(client.Messages)
				log.Log().Info("close client", zap.String("client_id", id))
				delete(s.clients, id)
			}
			client.mu.RUnlock()
		}
		s.mu.Unlock()
	}
}

// 更新客户端最后活跃时间
func (c *Client) updateLastSeen() {
	c.mu.Lock()
	c.LastSeen = time.Now()
	c.mu.Unlock()
}

func NewMessageService() *MessageService {
	ms := &MessageService{
		clients: make(map[string]*Client),
	}
	ms.client = client.NewStoryClient(
		client.PlatformZhipu,
	)
	// 启动清理程序
	go ms.cleanupInactiveClients()
	return ms
}

// 用户建立连接的消息，初始化本次连接，并管理此次的连接会话。如果用户长时间没有消息流，则关闭此次连接。

func (s *MessageService) InitChatContext(stream api.StreamMessageService_StreamChatMessageServer) error {
	userId := stream.Context().Value("user_id").(int64)
	roleId := stream.Context().Value("role_id").(int64)
	chatCtx, err := models.GetChatContextByUserIDAndRoleID(stream.Context(), userId, roleId)
	if err != nil {
		log.Log().Error("get user chat context failed", zap.Error(err))
		return err
	}
	chatCtx.Status = 1
	chatCtx.UserID = userId
	chatCtx.RoleID = roleId
	err = models.CreateChatContext(stream.Context(), chatCtx)
	if err != nil {
		log.Log().Error("create user chat context failed", zap.Error(err))
		return err
	}
	return nil
}

func (s *MessageService) ChatRecieveMessage(ctx context.Context, req *api.StreamChatMessageRequest, replyChan chan *models.ChatMessage) error {
	chatCtx, err := models.GetChatContextByUserIDAndRoleID(ctx, int64(req.Message.GetUserId()), int64(req.Message.GetRoleId()))
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Log().Error("get user chat context failed", zap.Error(err))
		return err
	}
	if chatCtx == nil {
		return errors.New("没有开启聊天")
	}
	var (
		userId int64 = int64(req.Message.GetUserId())
	)
	fmt.Printf("userId %d ChatWithStoryRole req %s \n", userId, req.String())
	chatMessage := new(models.ChatMessage)
	for _, message := range req.Message.Messages {
		chatMessage.ChatContextID = int64(chatCtx.ID)
		chatMessage.UserID = int64(message.GetUserId())
		chatMessage.Content = message.GetMessage()
		chatMessage.Status = 1
		chatMessage.RoleID = int64(message.GetRoleId())
		chatMessage.Sender = int64(message.GetSender())
		chatMessage.UUID = message.GetUuid()
		err = models.CreateChatMessage(ctx, chatMessage)
		if err != nil {
			log.Log().Error("create story role chat message failed", zap.Error(err))
			return err
		}
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case replyChan <- chatMessage:
		log.Log().Info("reply message", zap.String("message", chatMessage.Content))
	}
	return nil
}

func (s *MessageService) ChatReplyMessage(ctx context.Context, message models.ChatMessage, replyChan chan *models.ChatMessage) error {
	roleInfo, err := models.GetStoryRoleByID(ctx, message.RoleID)
	if err != nil {
		log.Log().Error("get story role by id failed", zap.Error(err))
		return nil
	}
	var chatParams = &client.ChatWithRoleParams{
		MessageContent: message.Content,
		Background:     roleInfo.CharacterDescription,
		SenseDesc:      "", // sence
		RolePositive:   "", // 角色的描述
		RoleNegative:   "",
		RequestId:      message.UUID,
		UserId:         fmt.Sprintf("grapery_chat_ctx_%d_user_%d", message.ChatContextID, message.UserID),
	}
	select {
	case <-ctx.Done():
		return nil
	default:
	}
	chatResp, err := s.client.ChatWithRole(ctx, chatParams)
	if err != nil {
		log.Log().Error("chat with role failed", zap.Error(err))
		return nil
	}
	roleReplyMessage := new(models.ChatMessage)
	roleReplyMessage.ChatContextID = int64(message.ChatContextID)
	roleReplyMessage.UserID = int64(message.UserID)
	roleReplyMessage.Content = chatResp.Content
	roleReplyMessage.Status = 1
	roleReplyMessage.RoleID = int64(message.RoleID)
	roleReplyMessage.Sender = int64(message.RoleID)
	roleReplyMessage.UUID = message.UUID
	err = models.CreateChatMessage(ctx, roleReplyMessage)
	if err != nil {
		log.Log().Error("create story role chat message failed", zap.Error(err))
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case replyChan <- roleReplyMessage:
		log.Log().Info("reply message", zap.String("message", roleReplyMessage.Content))
	}
	return nil
}

func (s *MessageService) StreamChatMessage(stream api.StreamMessageService_StreamChatMessageServer) error {
	for {
		// 从客户端接收消息
		req, err := stream.Recv()
		if err == io.EOF {
			log.Log().Info("client closed the stream")
			return nil // 如果客户端关闭了流，则退出
		}
		if err != nil {
			log.Log().Info("recv message error:" + err.Error())
			return err // 处理接收错误
		}
		// 处理消息逻辑（例如：记录消息，存储数据库等）
		recRet := make(chan *models.ChatMessage, 1)
		response := &api.StreamChatMessageResponse{
			Code:      0,
			Message:   "",
			Timestamp: time.Now().Unix(),
			RequestId: req.RequestId,
		}
		err = s.ChatRecieveMessage(stream.Context(), req, recRet)
		if err != nil {
			// 发送失败响应
			response.Code = -1
			response.Message = "message send error: " + err.Error()
			response.Timestamp = time.Now().Unix()
			response.RequestId = req.RequestId
			if err := stream.Send(response); err != nil {
				continue
			}
		} else {
			// 发送成功响应
			response.Code = 0
			response.Message = "message send success"
			response.Timestamp = time.Now().Unix()
			response.RequestId = req.RequestId
			if err := stream.Send(response); err != nil {
				continue
			}
			// TODO: 发送成功后，等待回复消息
			select {
			case roleReplyMessage := <-recRet:
				if roleReplyMessage == nil {
					log.Log().Info("role reply message is nil")
					continue
				}
				replyMsg := make([]*api.StreamChatMessage, 0)
				realMsg := &api.StreamChatMessage{
					UserId:   roleReplyMessage.UserID,
					RoleId:   roleReplyMessage.RoleID,
					Messages: make([]*api.ChatMessage, 0),
				}
				realMsg.Messages = append(realMsg.Messages, &api.ChatMessage{
					UserId:    roleReplyMessage.UserID,
					RoleId:    roleReplyMessage.RoleID,
					Message:   roleReplyMessage.Content,
					Sender:    int32(roleReplyMessage.RoleID),
					Uuid:      roleReplyMessage.UUID,
					ChatId:    roleReplyMessage.ChatContextID,
					Timestamp: time.Now().Unix(),
					User:      &api.UserInfo{},
					Role:      &api.StoryRole{},
				})
				replyMsg = append(replyMsg, realMsg)
				roleReplyMessageResponse := &api.StreamChatMessageResponse{
					Code:          0,
					ReplyMessages: replyMsg,
					Timestamp:     time.Now().Unix(),
					RequestId:     req.RequestId,
				}
				if err := stream.Send(roleReplyMessageResponse); err != nil {
					continue
				}
			}
		}
	}
}
