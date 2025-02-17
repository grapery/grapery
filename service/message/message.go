package message

import (
	"fmt"
	"io"
	"sync"
	"time"

	api "github.com/grapery/common-protoc/gen"
)

// 添加一些有用的常量和类型
const (
	MaxMessageSize   = 4 * 1024 * 1024 // 4MB
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
}

// 添加心跳检测
func (s *MessageService) cleanupInactiveClients() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for id, client := range s.clients {
			client.mu.RLock()
			if now.Sub(client.LastSeen) > 5*time.Minute {
				close(client.Messages)
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
	// 启动清理程序
	go ms.cleanupInactiveClients()
	return ms
}

// 用户建立连接的消息，初始化本次连接，并管理此次的连接会话。如果用户长时间没有消息流，则关闭此次连接。

func (s *MessageService) initChatContext(stream api.StreamMessageService_StreamChatMessageServer) error {
	return nil
}

func (s *MessageService) StreamChatMessage(stream api.StreamMessageService_StreamChatMessageServer) error {
	for {
		// 从客户端接收消息
		req, err := stream.Recv()
		if err == io.EOF {
			return nil // 如果客户端关闭了流，则退出
		}
		if err != nil {
			return err // 处理接收错误
		}
		// 处理消息逻辑（例如：记录消息，存储数据库等）
		fmt.Printf("Received message: %s\n", req.Message.Content)
		time.Sleep(10 * time.Second)
		// 构建响应消息
		response := &api.StreamChatMessageResponse{
			Code:      1,
			Message:   fmt.Sprintf("现在时间: ", time.Now()),
			Timestamp: time.Now().Unix(),
			RequestId: req.RequestId,
		}

		// 发送响应
		if err := stream.Send(response); err != nil {
			return err
		}
	}
}
