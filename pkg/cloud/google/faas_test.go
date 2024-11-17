package google

import (
	"context"
	"fmt"
	"log"
)

func Test_gen() {
	ctx := context.Background()
	apiKey := "your-api-key"

	client, err := NewGeminiClient(ctx, apiKey)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// 生成文本
	response, err := client.GenerateText(ctx, "Tell me a joke")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(response)

	// 开始对话
	chat, err := client.Chat(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// 发送消息
	reply, err := client.SendMessage(ctx, chat, "Hello!")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(reply)
}
