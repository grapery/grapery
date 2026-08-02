package asynctask

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/hibiken/asynq"
)

// TaskManagerConfig describes settings for the Asynq server.
type TaskManagerConfig struct {
	Concurrency int
	Queues      map[string]int
	RetryDelay  asynq.RetryDelayFunc
	Strict      bool
	TaskCheck   time.Duration
}

// TaskManager orchestrates Asynq client/server lifecycle.
type TaskManager struct {
	client    *asynq.Client
	inspector *asynq.Inspector
	server    *asynq.Server
	mux       *asynq.ServeMux
	once      sync.Once
}

// NewTaskManager creates a new task manager and registers the video handler.
func NewTaskManager(redisOpt asynq.RedisClientOpt, handler asynq.Handler, cfg TaskManagerConfig) (*TaskManager, error) {
	if handler == nil {
		return nil, fmt.Errorf("asynctask: handler cannot be nil")
	}
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = runtime.NumCPU()
	}
	if cfg.Queues == nil {
		cfg.Queues = map[string]int{
			VideoQueueName: 1,
			"default":      1,
		}
	}
	if cfg.TaskCheck <= 0 {
		cfg.TaskCheck = time.Second
	}

	client := asynq.NewClient(redisOpt)
	inspector := asynq.NewInspector(redisOpt)

	serverCfg := asynq.Config{
		Concurrency:       cfg.Concurrency,
		Queues:            cfg.Queues,
		StrictPriority:    cfg.Strict,
		TaskCheckInterval: cfg.TaskCheck,
	}
	if cfg.RetryDelay != nil {
		serverCfg.RetryDelayFunc = cfg.RetryDelay
	}
	server := asynq.NewServer(redisOpt, serverCfg)

	mux := asynq.NewServeMux()
	mux.Handle(VideoTaskType, handler)

	return &TaskManager{
		client:    client,
		inspector: inspector,
		server:    server,
		mux:       mux,
	}, nil
}

// Start starts the Asynq server asynchronously.
func (m *TaskManager) Start(ctx context.Context) error {
	if err := m.server.Start(m.mux); err != nil {
		return err
	}
	go func() {
		<-ctx.Done()
		m.Shutdown()
	}()
	return nil
}

// Shutdown gracefully stops the server and releases resources.
func (m *TaskManager) Shutdown() {
	m.once.Do(func() {
		m.server.Shutdown()
		_ = m.client.Close()
		_ = m.inspector.Close()
	})
}

// EnqueueVideoTask enqueues a video generation job.
func (m *TaskManager) EnqueueVideoTask(ctx context.Context, payload *VideoGeneratePayload, opts ...asynq.Option) (*asynq.TaskInfo, error) {
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
	return m.client.EnqueueContext(ctx, task, options...)
}

// Client exposes the underlying Asynq client.
func (m *TaskManager) Client() *asynq.Client {
	return m.client
}

// Inspector exposes the Asynq inspector for monitoring.
func (m *TaskManager) Inspector() *asynq.Inspector {
	return m.inspector
}
