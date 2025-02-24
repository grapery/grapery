package main

import (
	"context"
	"io"
	"log"
	"time"

	"github.com/google/uuid"

	api "github.com/grapery/common-protoc/gen"
	"google.golang.org/grpc"
)

const Address string = "127.0.0.1:12307"

var streamClient api.StreamMessageServiceClient

func main() {
	// 连接服务器
	conn, err := grpc.Dial(Address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("net.Connect err: %v", err)
	}
	defer conn.Close()

	// 建立gRPC连接
	streamClient = api.NewStreamMessageServiceClient(conn)
	conversations()
}

func conversations() {
	stream, err := streamClient.StreamChatMessage(context.Background())
	if err != nil {
		log.Fatalf("get conversations stream err: %v", err)
	}

	go func() {
		for {
			res, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Fatalf("Conversations get stream err: %v", err)
			}
			// 打印返回值
			log.Println("res: ", res.Message)
		}
	}()
	for n := 0; n < 5; n++ {
		err := stream.Send(&api.StreamChatMessageRequest{
			Message: &api.StreamChatMessage{
				RoleId: 1,
				UserId: 666,
				Messages: []*api.ChatMessage{
					{
						RoleId:  1,
						UserId:  666,
						Sender:  1,
						Message: "hello grapery",
					},
				},
			},
			Timestamp: time.Now().Unix(),
			RequestId: uuid.New().String(),
			Token:     "1234567890",
		})
		if err != nil {
			log.Fatalf("stream request err: %v", err)
		}

		time.Sleep(time.Second * 1)
	}
	time.Sleep(time.Hour * 10)
	//最后关闭流
	err = stream.CloseSend()
	if err != nil {
		log.Fatalf("Conversations close stream err: %v", err)
	}
}
