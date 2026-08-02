package message

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// MessageTopic 消息主题
type MessageTopic struct {
	gorm.Model
	Name        string `gorm:"uniqueIndex;size:255"`
	Description string
	CreatedBy   string `gorm:"size:64"`   // 创建者ID
	LastMsgID   int64  `gorm:"default:0"` // 主题下最新消息ID
}

// MessageRecord 消息记录
type MessageRecord struct {
	ID        uint      `gorm:"primarykey"`
	TopicID   uint      `gorm:"index:idx_topic_msg"` // 关联的主题ID
	MsgID     int64     `gorm:"index:idx_topic_msg"` // 消息在主题下的序号
	SenderID  string    `gorm:"size:64;index"`       // 发送者ID
	Content   []byte    `gorm:"type:blob"`           // 消息内容
	CreatedAt time.Time `gorm:"index"`               // 发送时间
}

// ConsumerOffset 消费者偏移量记录
type ConsumerOffset struct {
	gorm.Model
	TopicID    uint   `gorm:"uniqueIndex:idx_consumer_topic"`
	ConsumerID string `gorm:"size:64;uniqueIndex:idx_consumer_topic"` // 消费者ID
	LastMsgID  int64  `gorm:"default:0"`                              // 最后消费的消息ID
}

// Repository 定义存储库接口
type Repository interface {
	// Topic 相关操作
	CreateTopic(topic *MessageTopic) error
	GetTopic(name string) (*MessageTopic, error)

	// Message 相关操作
	SaveMessage(record *MessageRecord) error
	GetMessages(topicID uint, startMsgID int64, limit int) ([]*MessageRecord, error)

	// Offset 相关操作
	UpdateOffset(offset *ConsumerOffset) error
	GetOffset(topicID uint, consumerID string) (*ConsumerOffset, error)
}

type GormRepository struct {
	db *gorm.DB
}

func NewGormRepository(db *gorm.DB) (*GormRepository, error) {
	// 自动迁移表结构
	err := db.AutoMigrate(&MessageTopic{}, &MessageRecord{}, &ConsumerOffset{})
	if err != nil {
		return nil, err
	}
	return &GormRepository{db: db}, nil
}

// Topic 相关实现
func (r *GormRepository) CreateTopic(topic *MessageTopic) error {
	return r.db.Create(topic).Error
}

func (r *GormRepository) GetTopic(name string) (*MessageTopic, error) {
	var topic MessageTopic
	err := r.db.Where("name = ?", name).First(&topic).Error
	if err != nil {
		return nil, err
	}
	return &topic, nil
}

// Message 相关实现
func (r *GormRepository) SaveMessage(record *MessageRecord) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 获取主题
		var topic MessageTopic
		if err := tx.First(&topic, record.TopicID).Error; err != nil {
			return err
		}

		// 设置消息ID并更新主题的最新消息ID
		topic.LastMsgID++
		record.MsgID = topic.LastMsgID

		// 保存消息
		if err := tx.Create(record).Error; err != nil {
			return err
		}

		// 更新主题的最新消息ID
		return tx.Model(&topic).Update("last_msg_id", topic.LastMsgID).Error
	})
}

func (r *GormRepository) GetMessages(topicID uint, startMsgID int64, limit int) ([]*MessageRecord, error) {
	var messages []*MessageRecord
	err := r.db.Where("topic_id = ? AND msg_id > ?", topicID, startMsgID).
		Order("msg_id asc").
		Limit(limit).
		Find(&messages).Error
	return messages, err
}

// Offset 相关实现
func (r *GormRepository) UpdateOffset(offset *ConsumerOffset) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var existing ConsumerOffset
		err := tx.Where("topic_id = ? AND consumer_id = ?",
			offset.TopicID, offset.ConsumerID).First(&existing).Error

		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// 创建新记录
				return tx.Create(offset).Error
			}
			return err
		}

		// 更新现有记录
		return tx.Model(&existing).
			Update("last_msg_id", offset.LastMsgID).Error
	})
}

func (r *GormRepository) GetOffset(topicID uint, consumerID string) (*ConsumerOffset, error) {
	var offset ConsumerOffset
	err := r.db.Where("topic_id = ? AND consumer_id = ?",
		topicID, consumerID).First(&offset).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 返回新的 offset 记录
			return &ConsumerOffset{
				TopicID:    topicID,
				ConsumerID: consumerID,
				LastMsgID:  0,
			}, nil
		}
		return nil, err
	}
	return &offset, nil
}
