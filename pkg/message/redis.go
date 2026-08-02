package message

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/go-redis/redis"
)

var (
	redisClient *redis.Client
)

func init() {
	redisClient = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
}

// MessageQueue 定义消息队列接口
type MessageQueue interface {
	// Publish 发布消息到指定主题
	Publish(ctx context.Context, topic string, message []byte) error
	// Subscribe 订阅指定主题
	Subscribe(ctx context.Context, topic string) (<-chan Message, error)
	// Close 关闭队列连接
	Close() error
}

// Message 表示一条消息
type Message struct {
	Topic   string
	Payload []byte
	ID      string
	Time    time.Time
}

// Config Redis配置
type Config struct {
	Addr     string
	Password string
	DB       int
}

type RedisQueue struct {
	client  *redis.Client
	pubsub  map[string]*redis.PubSub
	streams map[string]chan Message
	mu      sync.RWMutex
	done    chan struct{}
}

// NewRedisQueue 创建新的Redis消息队列实例
func NewRedisQueue(cfg *Config) (*RedisQueue, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// 测试连接
	_, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping().Err(); err != nil {
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	return &RedisQueue{
		client:  client,
		pubsub:  make(map[string]*redis.PubSub),
		streams: make(map[string]chan Message),
		done:    make(chan struct{}),
	}, nil
}

// Publish 发布消息
func (rq *RedisQueue) Publish(ctx context.Context, topic string, payload []byte) error {
	msg := Message{
		Topic:   topic,
		Payload: payload,
		ID:      generateMessageID(),
		Time:    time.Now(),
	}

	// 序列化消息
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("message serialization failed: %w", err)
	}

	// 使用Redis Stream存储消息
	streamKey := fmt.Sprintf("stream:%s", topic)
	if err := rq.client.XAdd(&redis.XAddArgs{
		Stream: streamKey,
		Values: map[string]interface{}{
			"data": data,
		},
	}).Err(); err != nil {
		return fmt.Errorf("redis stream add failed: %w", err)
	}

	// 发布消息通知
	if err := rq.client.Publish(topic, data).Err(); err != nil {
		return fmt.Errorf("redis publish failed: %w", err)
	}

	return nil
}

// Subscribe 订阅主题
func (rq *RedisQueue) Subscribe(ctx context.Context, topic string) (<-chan Message, error) {
	rq.mu.Lock()
	defer rq.mu.Unlock()

	// 检查是否已经订阅
	if ch, exists := rq.streams[topic]; exists {
		return ch, nil
	}

	// 创建消息通道
	msgChan := make(chan Message, 100)
	rq.streams[topic] = msgChan

	// 订阅Redis Pub/Sub
	pubsub := rq.client.Subscribe(topic)
	rq.pubsub[topic] = pubsub

	// 处理消息
	go rq.handleMessages(ctx, topic, pubsub, msgChan)

	// 处理历史消息
	go rq.handleHistoricalMessages(ctx, topic, msgChan)

	return msgChan, nil
}

// handleMessages 处理实时消息
func (rq *RedisQueue) handleMessages(ctx context.Context, topic string, pubsub *redis.PubSub, msgChan chan Message) {
	defer func() {
		rq.mu.Lock()
		delete(rq.pubsub, topic)
		delete(rq.streams, topic)
		close(msgChan)
		rq.mu.Unlock()
	}()

	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case <-rq.done:
			return
		case msg := <-ch:
			if msg == nil {
				return
			}

			var message Message
			if err := json.Unmarshal([]byte(msg.Payload), &message); err != nil {
				continue
			}

			select {
			case msgChan <- message:
			default:
				// 通道已满，跳过消息
			}
		}
	}
}

// handleHistoricalMessages 处理历史消息
func (rq *RedisQueue) handleHistoricalMessages(ctx context.Context, topic string, msgChan chan Message) {
	streamKey := fmt.Sprintf("stream:%s", topic)
	lastID := "0-0" // 从最早的消息开始

	for {
		// 读取历史消息
		streams, err := rq.client.XRead(&redis.XReadArgs{
			Streams: []string{streamKey, lastID},
			Count:   100,
			Block:   0,
		}).Result()

		if err != nil {
			if err != redis.Nil {
				time.Sleep(time.Second)
			}
			continue
		}

		for _, stream := range streams {
			for _, message := range stream.Messages {
				if data, ok := message.Values["data"].(string); ok {
					var msg Message
					if err := json.Unmarshal([]byte(data), &msg); err != nil {
						continue
					}

					select {
					case msgChan <- msg:
					default:
						// 通道已满，跳过消息
					}
				}
				lastID = message.ID
			}
		}
	}
}

// Close 关闭队列
func (rq *RedisQueue) Close() error {
	close(rq.done)

	rq.mu.Lock()
	defer rq.mu.Unlock()

	// 关闭所有订阅
	for _, pubsub := range rq.pubsub {
		_ = pubsub.Close()
	}

	// 关闭所有流
	for _, stream := range rq.streams {
		close(stream)
	}

	// 清空映射
	rq.pubsub = make(map[string]*redis.PubSub)
	rq.streams = make(map[string]chan Message)

	// 关闭Redis连接
	return rq.client.Close()
}

// generateMessageID 生成消息ID
func generateMessageID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), rand.Int63())
}
