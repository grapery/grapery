package asynctask

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hibiken/asynq"

	"github.com/grapery/grapery/config"
)

var (
	producerMu     sync.Mutex
	producerClient *asynq.Client
)

func getProducerClient() (*asynq.Client, error) {
	producerMu.Lock()
	defer producerMu.Unlock()

	if producerClient != nil {
		return producerClient, nil
	}

	cfg := config.GlobalConfig
	if cfg == nil || cfg.Redis == nil {
		return nil, fmt.Errorf("asynctask: redis config not initialized")
	}

	addr := strings.TrimSpace(cfg.Redis.Address)
	if addr == "" {
		return nil, fmt.Errorf("asynctask: redis address not configured")
	}

	redisDB := 0
	if dbStr := strings.TrimSpace(cfg.Redis.Database); dbStr != "" {
		value, err := strconv.Atoi(dbStr)
		if err != nil {
			return nil, fmt.Errorf("asynctask: parse redis db: %w", err)
		}
		redisDB = value
	}

	producerClient = asynq.NewClient(asynq.RedisClientOpt{
		Addr:     addr,
		Password: cfg.Redis.Password,
		DB:       redisDB,
	})
	return producerClient, nil
}

// SubmitVideoGenerationTask enqueues a video generation task to Asynq.
func SubmitVideoGenerationTask(ctx context.Context, payload *VideoGeneratePayload, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	client, err := getProducerClient()
	if err != nil {
		return nil, err
	}

	task, err := NewVideoGenerateTask(payload)
	if err != nil {
		return nil, err
	}

	defaultOpts := []asynq.Option{
		asynq.Queue(VideoQueueName),
		asynq.MaxRetry(defaultMaxRetry),
		asynq.Timeout(60 * time.Minute),
	}
	options := append(defaultOpts, opts...)
	return client.EnqueueContext(ctx, task, options...)
}
