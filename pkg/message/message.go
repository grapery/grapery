package message

import (
	"context"
	"fmt"
	"log"
	_ "net/http/pprof"
	"os"
	"strconv"
	"time"

	rmq_client "github.com/apache/rocketmq-clients/golang/v5"
	"github.com/apache/rocketmq-clients/golang/v5/credentials"
)

type ChatMessageService interface {
	SendNormalMessage(ctx context.Context, message string) error
	SendDelayMessage(ctx context.Context, message string) error
	SendFiFoMessage(ctx context.Context, message string) error
	ReceiveMessage(ctx context.Context, message string) error
}

func SendMessage(ctx context.Context, message string) error {
	return nil
}

func ReceiveMessage(ctx context.Context, message string) error {
	return nil
}

const (
	RocketMQTopic         = "xxxxxx"
	RocketMQConsumerGroup = "xxxxxx"
	RocketMQEndpoint      = "xxxxxx"
	RocketMQAccessKey     = "xxxxxx"
	RocketMQSecretKey     = "xxxxxx"
)

var (
	// maximum waiting time for receive func
	AwaitDuration = time.Second * 5
	// maximum number of messages received at one time
	MaxMessageNum int32 = 16
	// invisibleDuration should > 20s
	InvisibleDuration = time.Second * 20
	// receive messages in a loop
)

func ConsumeMessage(ctx context.Context, message string) {
	// log to console
	os.Setenv("mq.consoleAppender.enabled", "true")
	rmq_client.ResetLogger()
	// In most case, you don't need to create many consumers, singleton pattern is more recommended.
	simpleConsumer, err := rmq_client.NewSimpleConsumer(&rmq_client.Config{
		Endpoint:      RocketMQEndpoint,
		ConsumerGroup: RocketMQConsumerGroup,
		Credentials: &credentials.SessionCredentials{
			AccessKey:    RocketMQAccessKey,
			AccessSecret: RocketMQSecretKey,
		},
	},
		rmq_client.WithAwaitDuration(AwaitDuration),
		rmq_client.WithSubscriptionExpressions(map[string]*rmq_client.FilterExpression{
			RocketMQTopic: rmq_client.SUB_ALL,
		}),
	)
	if err != nil {
		log.Fatal(err)
	}
	// start simpleConsumer
	err = simpleConsumer.Start()
	if err != nil {
		log.Fatal(err)
	}
	// graceful stop simpleConsumer
	defer simpleConsumer.GracefulStop()

	go func() {
		for {
			fmt.Println("start receive message")
			mvs, err := simpleConsumer.Receive(context.TODO(), MaxMessageNum, InvisibleDuration)
			if err != nil {
				fmt.Println(err)
			}
			// ack message
			for _, mv := range mvs {
				simpleConsumer.Ack(context.TODO(), mv)
				fmt.Println(mv)
			}
			fmt.Println("wait a moment")
			fmt.Println()
			time.Sleep(time.Second * 3)
		}
	}()
}

func NormalSendMessage(ctx context.Context, message string) {
	os.Setenv("mq.consoleAppender.enabled", "true")
	rmq_client.ResetLogger()
	// In most case, you don't need to create many producers, singleton pattern is more recommended.
	producer, err := rmq_client.NewProducer(&rmq_client.Config{
		Endpoint: RocketMQEndpoint,
		Credentials: &credentials.SessionCredentials{
			AccessKey:    RocketMQAccessKey,
			AccessSecret: RocketMQSecretKey,
		},
	},
		rmq_client.WithTopics(RocketMQTopic),
	)
	if err != nil {
		log.Fatal(err)
	}
	// start producer
	err = producer.Start()
	if err != nil {
		log.Fatal(err)
	}
	// graceful stop producer
	defer producer.GracefulStop()

	for i := 0; i < 10; i++ {
		// new a message
		msg := &rmq_client.Message{
			Topic: RocketMQTopic,
			Body:  []byte("this is a message : " + strconv.Itoa(i)),
		}
		// set keys and tag
		msg.SetKeys("a", "b")
		msg.SetTag("ab")
		// send message in sync
		resp, err := producer.Send(context.TODO(), msg)
		if err != nil {
			log.Fatal(err)
		}
		for i := 0; i < len(resp); i++ {
			fmt.Printf("%#v\n", resp[i])
		}
		// wait a moment
		time.Sleep(time.Second * 1)
	}
}

func DelaySendMessage(ctx context.Context, message string) {
	// log to console
	os.Setenv("mq.consoleAppender.enabled", "true")
	rmq_client.ResetLogger()
	// In most case, you don't need to create many producers, singleton pattern is more recommended.
	producer, err := rmq_client.NewProducer(&rmq_client.Config{
		Endpoint: RocketMQEndpoint,
		Credentials: &credentials.SessionCredentials{
			AccessKey:    RocketMQAccessKey,
			AccessSecret: RocketMQSecretKey,
		},
	},
		rmq_client.WithTopics(RocketMQTopic),
	)
	if err != nil {
		log.Fatal(err)
	}
	// start producer
	err = producer.Start()
	if err != nil {
		log.Fatal(err)
	}
	// graceful stop producer
	defer producer.GracefulStop()
	for i := 0; i < 10; i++ {
		// new a message
		msg := &rmq_client.Message{
			Topic: RocketMQTopic,
			Body:  []byte("this is a message : " + strconv.Itoa(i)),
		}
		// set keys and tag
		msg.SetKeys("a", "b")
		msg.SetTag("ab")
		// set delay timestamp
		msg.SetDelayTimestamp(time.Now().Add(time.Second * 10))
		// send message in sync
		resp, err := producer.Send(context.TODO(), msg)
		if err != nil {
			log.Fatal(err)
		}
		for i := 0; i < len(resp); i++ {
			fmt.Printf("%#v\n", resp[i])
		}
		// wait a moment
		time.Sleep(time.Second * 1)
	}

}

func FiFoSendMessage(ctx context.Context, message string) {
	os.Setenv("mq.consoleAppender.enabled", "true")
	rmq_client.ResetLogger()
	// In most case, you don't need to create many producers, singleton pattern is more recommended.
	producer, err := rmq_client.NewProducer(&rmq_client.Config{
		Endpoint: RocketMQEndpoint,
		Credentials: &credentials.SessionCredentials{
			AccessKey:    RocketMQAccessKey,
			AccessSecret: RocketMQSecretKey,
		},
	},
		rmq_client.WithTopics(RocketMQTopic),
	)
	if err != nil {
		log.Fatal(err)
	}
	// start producer
	err = producer.Start()
	if err != nil {
		log.Fatal(err)
	}
	// graceful stop producer
	defer producer.GracefulStop()
	for i := 0; i < 10; i++ {
		// new a message
		msg := &rmq_client.Message{
			Topic: RocketMQTopic,
			Body:  []byte("this is a message : " + strconv.Itoa(i)),
		}
		// set keys and tag
		msg.SetKeys("a", "b")
		msg.SetTag("ab")
		msg.SetMessageGroup("fifo")
		// send message in sync
		resp, err := producer.Send(context.TODO(), msg)
		if err != nil {
			log.Fatal(err)
		}
		for i := 0; i < len(resp); i++ {
			fmt.Printf("%#v\n", resp[i])
		}
		// wait a moment
		time.Sleep(time.Second * 1)
	}
}
