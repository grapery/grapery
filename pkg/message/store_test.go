package message

import (
	"fmt"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func ExampleUsage() {
	// 初始化数据库连接
	db, err := gorm.Open(mysql.Open("dsn"), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	// 创建存储库
	repo, err := NewGormRepository(db)
	if err != nil {
		log.Fatal(err)
	}

	// 创建主题
	topic := &MessageTopic{
		Name:        "chat_room_1",
		Description: "Chat Room 1",
		CreatedBy:   "admin",
	}
	if err := repo.CreateTopic(topic); err != nil {
		log.Fatal(err)
	}

	// 保存消息
	msg := &MessageRecord{
		TopicID:  topic.ID,
		SenderID: "user1",
		Content:  []byte("Hello, World!"),
	}
	if err := repo.SaveMessage(msg); err != nil {
		log.Fatal(err)
	}

	// 更新消费者偏移量
	offset := &ConsumerOffset{
		TopicID:    topic.ID,
		ConsumerID: "user2",
		LastMsgID:  msg.MsgID,
	}
	if err := repo.UpdateOffset(offset); err != nil {
		log.Fatal(err)
	}

	// 获取消息
	messages, err := repo.GetMessages(topic.ID, 0, 10)
	if err != nil {
		log.Fatal(err)
	}
	for _, m := range messages {
		fmt.Printf("Message %d: %s\n", m.MsgID, string(m.Content))
	}
}
