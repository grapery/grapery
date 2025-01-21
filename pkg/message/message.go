package message

import (
	"context"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"time"

	ali_mns "github.com/aliyun/aliyun-mns-go-sdk"
)

type ChatMessageService interface {
	SendMessage(ctx context.Context, message string) error
	ReceiveMessage(ctx context.Context, message string) error
}

func SendMessage(ctx context.Context, message string) error {
	return nil
}

func ReceiveMessage(ctx context.Context, message string) error {
	return nil
}

const (
	// 阿里云消息队列服务地址
	endpoint = "http://1866841989078847.mns.cn-hangzhou.aliyuncs.com"
	// 队列名称
	queueName = "test-queue"
	// 主题名称
	topicName = "test-topic"
)

func main() {
	go func() {
		log.Println(http.ListenAndServe("localhost:8080", nil))
	}()

	// Replace with your own endpoint.
	client := ali_mns.NewClient(endpoint)
	msg := ali_mns.MessageSendRequest{
		MessageBody:  "hello <\"aliyun-mns-go-sdk\">",
		DelaySeconds: 0,
		Priority:     8,
	}

	queueManager := ali_mns.NewMNSQueueManager(client)
	err := queueManager.CreateQueue(queueName, 0, 65536, 345600, 30, 0, 3)
	time.Sleep(time.Duration(2) * time.Second)
	if err != nil && !ali_mns.ERR_MNS_QUEUE_ALREADY_EXIST_AND_HAVE_SAME_ATTR.IsEqual(err) {
		fmt.Println(err)
		return
	}

	queue := ali_mns.NewMNSQueue(queueName, client)
	for i := 1; i < 10000; i++ {
		ret, err := queue.SendMessage(msg)
		go func() {
			fmt.Println(queue.QPSMonitor().QPS())
		}()

		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println("response: ", ret)
		}

		endChan := make(chan int)
		respChan := make(chan ali_mns.MessageReceiveResponse)
		errChan := make(chan error)
		go func() {
			select {
			case resp := <-respChan:
				{
					fmt.Println("response: ", resp)
					fmt.Println("change the visibility: ", resp.ReceiptHandle)
					if ret, e := queue.ChangeMessageVisibility(resp.ReceiptHandle, 5); e != nil {
						fmt.Println(e)
					} else {
						fmt.Println("visibility changed", ret)
						fmt.Println("delete it now: ", ret.ReceiptHandle)
						if e := queue.DeleteMessage(ret.ReceiptHandle); e != nil {
							fmt.Println(e)
						}
						endChan <- 1
					}
				}
			case err := <-errChan:
				{
					fmt.Println(err)
					endChan <- 1
				}
			}
		}()

		queue.ReceiveMessage(respChan, errChan, 30)
		<-endChan
	}
}

func TopicManage() {
	// Replace with your own endpoint.
	queueSubName := "test-sub-queue"
	httpSubName := "test-sub-http"
	client := ali_mns.NewClient(endpoint)

	// 1. create a queue for receiving pushed messages
	queueManager := ali_mns.NewMNSQueueManager(client)
	err := queueManager.CreateSimpleQueue(queueName)
	if err != nil && !ali_mns.ERR_MNS_QUEUE_ALREADY_EXIST_AND_HAVE_SAME_ATTR.IsEqual(err) {
		fmt.Println(err)
		return
	}

	// 2. create the topic
	topicManager := ali_mns.NewMNSTopicManager(client)
	// topicManager.DeleteTopic("testTopic")
	err = topicManager.CreateSimpleTopic(topicName)
	if err != nil && !ali_mns.ERR_MNS_TOPIC_ALREADY_EXIST_AND_HAVE_SAME_ATTR.IsEqual(err) {
		fmt.Println(err)
		return
	}

	topic := ali_mns.NewMNSTopic(topicName, client)
	// 3. subscribe to topic, the endpoint is queue
	queueSub := ali_mns.MessageSubsribeRequest{
		Endpoint:            topic.GenerateQueueEndpoint(queueName),
		NotifyContentFormat: ali_mns.SIMPLIFIED,
	}

	// 4. subscribe to topic, the endpoint is HTTP(S)
	httpSub := ali_mns.MessageSubsribeRequest{
		Endpoint:            "http://www.baidu.com",
		NotifyContentFormat: ali_mns.SIMPLIFIED,
	}

	err = topic.Subscribe(queueSubName, queueSub)
	if err != nil && !ali_mns.ERR_MNS_SUBSCRIPTION_ALREADY_EXIST_AND_HAVE_SAME_ATTR.IsEqual(err) {
		fmt.Println(err)
		return
	}

	err = topic.Subscribe(httpSubName, httpSub)
	if err != nil && !ali_mns.ERR_MNS_SUBSCRIPTION_ALREADY_EXIST_AND_HAVE_SAME_ATTR.IsEqual(err) {
		fmt.Println(err)
		return
	}

	time.Sleep(time.Duration(2) * time.Second)

	// 5. now publish message
	msg := ali_mns.MessagePublishRequest{
		MessageBody: "hello topic <\"aliyun-mns-go-sdk\">",
		MessageAttributes: &ali_mns.MessageAttributes{
			MailAttributes: &ali_mns.MailAttributes{
				Subject:     "AAA中文",
				AccountName: "BBB",
			},
		},
	}
	_, err = topic.PublishMessage(msg)
	if err != nil {
		fmt.Println(err)
		return
	}

	// 6. receive the message from queue
	queue := ali_mns.NewMNSQueue(queueName, client)
	endChan := make(chan int)
	respChan := make(chan ali_mns.MessageReceiveResponse)
	errChan := make(chan error)
	go func() {
		select {
		case resp := <-respChan:
			{
				fmt.Println("response: ", resp)
				fmt.Println("change the visibility: ", resp.ReceiptHandle)
				if ret, e := queue.ChangeMessageVisibility(resp.ReceiptHandle, 5); e != nil {
					fmt.Println(e)
				} else {
					fmt.Println("visibility changed", ret)
					fmt.Println("delete it now: ", ret.ReceiptHandle)
					if e := queue.DeleteMessage(ret.ReceiptHandle); e != nil {
						fmt.Println(e)
					}
					endChan <- 1
				}
			}
		case err := <-errChan:
			{
				fmt.Println(err)
				endChan <- 1
			}
		}
	}()

	queue.ReceiveMessage(respChan, errChan, 30)
	<-endChan
}
