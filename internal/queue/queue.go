package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// TaskQueue Redis任务队列
type TaskQueue struct {
	client *redis.Client
	logger *zap.Logger
}

// TaskMessage 任务消息
type TaskMessage struct {
	TaskID              string                 `json:"taskId"`
	TaskType            string                 `json:"taskType"` // ai_task, render_task
	Data                map[string]interface{} `json:"data"`
	CreatedAt           time.Time              `json:"createdAt"`
	ProcessingStartedAt *time.Time             `json:"processingStartedAt,omitempty"`
	Retry               int                    `json:"retry"`
}

const (
	// 任务处理超时时间（超过此时间认为任务卡住了）
	TaskProcessingTimeout = 10 * time.Minute
)

const (
	// 队列名称
	QueueAITask     = "queue:ai:tasks"
	QueueRenderTask = "queue:render:tasks"

	// 处理中的任务集合
	ProcessingAITasks     = "processing:ai:tasks"
	ProcessingRenderTasks = "processing:render:tasks"

	// 失败任务队列
	FailedAITasks     = "failed:ai:tasks"
	FailedRenderTasks = "failed:render:tasks"
)

// NewTaskQueue 创建任务队列
func NewTaskQueue(client *redis.Client, logger *zap.Logger) *TaskQueue {
	return &TaskQueue{
		client: client,
		logger: logger,
	}
}

// ========== 任务入队 ==========

// EnqueueAITask 将AI任务放入队列
func (q *TaskQueue) EnqueueAITask(ctx context.Context, taskID string, data map[string]interface{}) error {
	message := &TaskMessage{
		TaskID:    taskID,
		TaskType:  "ai_task",
		Data:      data,
		CreatedAt: time.Now(),
		Retry:     0,
	}

	return q.enqueue(ctx, QueueAITask, message)
}

// EnqueueRenderTask 将渲染任务放入队列
func (q *TaskQueue) EnqueueRenderTask(ctx context.Context, taskID string, data map[string]interface{}) error {
	message := &TaskMessage{
		TaskID:    taskID,
		TaskType:  "render_task",
		Data:      data,
		CreatedAt: time.Now(),
		Retry:     0,
	}

	return q.enqueue(ctx, QueueRenderTask, message)
}

// enqueue 通用入队方法
func (q *TaskQueue) enqueue(ctx context.Context, queueName string, message *TaskMessage) error {
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	// 使用 RPUSH 将任务推入队列尾部
	err = q.client.RPush(ctx, queueName, data).Err()
	if err != nil {
		q.logger.Error("failed to enqueue task",
			zap.String("queue", queueName),
			zap.String("taskId", message.TaskID),
			zap.Error(err),
		)
		return fmt.Errorf("rpush: %w", err)
	}

	q.logger.Info("task enqueued",
		zap.String("queue", queueName),
		zap.String("taskId", message.TaskID),
		zap.String("taskType", message.TaskType),
	)

	return nil
}

// ========== 任务出队 ==========

// DequeueAITask 从队列中取出AI任务（阻塞）
func (q *TaskQueue) DequeueAITask(ctx context.Context, timeout time.Duration) (*TaskMessage, error) {
	return q.dequeue(ctx, QueueAITask, ProcessingAITasks, timeout)
}

// DequeueRenderTask 从队列中取出渲染任务（阻塞）
func (q *TaskQueue) DequeueRenderTask(ctx context.Context, timeout time.Duration) (*TaskMessage, error) {
	return q.dequeue(ctx, QueueRenderTask, ProcessingRenderTasks, timeout)
}

// dequeue 通用出队方法（使用 BLMOVE 实现可靠队列）
func (q *TaskQueue) dequeue(ctx context.Context, sourceQueue, processingSet string, timeout time.Duration) (*TaskMessage, error) {
	// 使用 BLMOVE 原子性地将任务从队列移到处理中集合
	// 这样即使 worker 崩溃，任务也不会丢失
	result, err := q.client.BLMove(
		ctx,
		sourceQueue,
		processingSet,
		"LEFT",
		"RIGHT",
		timeout,
	).Result()

	if err != nil {
		if err == redis.Nil {
			// 超时，没有任务
			return nil, nil
		}
		return nil, fmt.Errorf("blmove: %w", err)
	}

	var message TaskMessage
	if err := json.Unmarshal([]byte(result), &message); err != nil {
		q.logger.Error("failed to unmarshal task message",
			zap.Error(err),
			zap.String("data", result),
		)
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	// 设置开始处理时间
	now := time.Now()
	message.ProcessingStartedAt = &now

	// 更新处理中集合中的任务（包含处理开始时间）
	updatedData, _ := json.Marshal(message)
	// 移除旧的，添加新的（带有处理时间戳）
	q.client.LRem(ctx, processingSet, 1, result)
	q.client.RPush(ctx, processingSet, updatedData)

	q.logger.Debug("task dequeued",
		zap.String("queue", sourceQueue),
		zap.String("taskId", message.TaskID),
	)

	return &message, nil
}

// ========== 任务完成 ==========

// CompleteAITask 标记AI任务完成（从处理中集合移除）
func (q *TaskQueue) CompleteAITask(ctx context.Context, taskID string) error {
	return q.completeTask(ctx, ProcessingAITasks, taskID)
}

// CompleteRenderTask 标记渲染任务完成
func (q *TaskQueue) CompleteRenderTask(ctx context.Context, taskID string) error {
	return q.completeTask(ctx, ProcessingRenderTasks, taskID)
}

// completeTask 通用完成方法
func (q *TaskQueue) completeTask(ctx context.Context, processingSet, taskID string) error {
	// 从处理中的集合中移除任务
	// 注意：这里需要遍历列表找到对应的任务
	// 在生产环境中，建议使用 sorted set 或 hash 来优化查找

	length, err := q.client.LLen(ctx, processingSet).Result()
	if err != nil {
		return fmt.Errorf("llen: %w", err)
	}

	// 获取所有处理中的任务
	items, err := q.client.LRange(ctx, processingSet, 0, length-1).Result()
	if err != nil {
		return fmt.Errorf("lrange: %w", err)
	}

	// 查找并移除对应的任务
	for _, item := range items {
		var msg TaskMessage
		if err := json.Unmarshal([]byte(item), &msg); err != nil {
			continue
		}

		if msg.TaskID == taskID {
			// 使用 LREM 移除任务
			err := q.client.LRem(ctx, processingSet, 1, item).Err()
			if err != nil {
				return fmt.Errorf("lrem: %w", err)
			}

			q.logger.Info("task completed and removed from processing set",
				zap.String("taskId", taskID),
				zap.String("set", processingSet),
			)
			return nil
		}
	}

	q.logger.Warn("task not found in processing set",
		zap.String("taskId", taskID),
		zap.String("set", processingSet),
	)

	return nil
}

// ========== 任务失败 ==========

// FailAITask 标记AI任务失败
func (q *TaskQueue) FailAITask(ctx context.Context, taskID string, message *TaskMessage, err error) error {
	return q.failTask(ctx, ProcessingAITasks, FailedAITasks, QueueAITask, taskID, message, err)
}

// FailRenderTask 标记渲染任务失败
func (q *TaskQueue) FailRenderTask(ctx context.Context, taskID string, message *TaskMessage, err error) error {
	return q.failTask(ctx, ProcessingRenderTasks, FailedRenderTasks, QueueRenderTask, taskID, message, err)
}

// failTask 通用失败处理
func (q *TaskQueue) failTask(ctx context.Context, processingSet, failedQueue, retryQueue, taskID string, message *TaskMessage, taskErr error) error {
	// 从处理中集合移除
	if err := q.completeTask(ctx, processingSet, taskID); err != nil {
		q.logger.Error("failed to remove task from processing set",
			zap.Error(err),
		)
	}

	// 增加重试次数
	message.Retry++

	// 最多重试3次
	if message.Retry <= 3 {
		// 重新放回队列
		q.logger.Info("task failed, retrying",
			zap.String("taskId", taskID),
			zap.Int("retry", message.Retry),
			zap.Error(taskErr),
		)

		data, _ := json.Marshal(message)
		return q.client.RPush(ctx, retryQueue, data).Err()
	}

	// 超过重试次数，放入失败队列
	q.logger.Error("task failed after max retries",
		zap.String("taskId", taskID),
		zap.Int("retries", message.Retry),
		zap.Error(taskErr),
	)

	data, _ := json.Marshal(message)
	return q.client.RPush(ctx, failedQueue, data).Err()
}

// ========== 队列监控 ==========

// GetQueueLength 获取队列长度
func (q *TaskQueue) GetQueueLength(ctx context.Context, queueName string) (int64, error) {
	return q.client.LLen(ctx, queueName).Result()
}

// GetProcessingCount 获取处理中的任务数量
func (q *TaskQueue) GetProcessingCount(ctx context.Context, processingSet string) (int64, error) {
	return q.client.LLen(ctx, processingSet).Result()
}

// GetFailedCount 获取失败任务数量
func (q *TaskQueue) GetFailedCount(ctx context.Context, failedQueue string) (int64, error) {
	return q.client.LLen(ctx, failedQueue).Result()
}

// GetQueueStats 获取队列统计信息
func (q *TaskQueue) GetQueueStats(ctx context.Context) (map[string]interface{}, error) {
	aiQueueLen, _ := q.GetQueueLength(ctx, QueueAITask)
	aiProcessingLen, _ := q.GetProcessingCount(ctx, ProcessingAITasks)
	aiFailedLen, _ := q.GetFailedCount(ctx, FailedAITasks)

	renderQueueLen, _ := q.GetQueueLength(ctx, QueueRenderTask)
	renderProcessingLen, _ := q.GetProcessingCount(ctx, ProcessingRenderTasks)
	renderFailedLen, _ := q.GetFailedCount(ctx, FailedRenderTasks)

	return map[string]interface{}{
		"ai_tasks": map[string]int64{
			"queued":     aiQueueLen,
			"processing": aiProcessingLen,
			"failed":     aiFailedLen,
		},
		"render_tasks": map[string]int64{
			"queued":     renderQueueLen,
			"processing": renderProcessingLen,
			"failed":     renderFailedLen,
		},
	}, nil
}

// ========== 清理和维护 ==========

// ClearQueue 清空队列（慎用！）
func (q *TaskQueue) ClearQueue(ctx context.Context, queueName string) error {
	return q.client.Del(ctx, queueName).Err()
}

// RecoverStuckTasks 恢复卡住的任务（将处理超时的任务重新放回队列）
func (q *TaskQueue) RecoverStuckTasks(ctx context.Context) error {
	q.logger.Info("recovering stuck tasks...")

	// 恢复 AI 任务
	aiRecovered, err := q.recoverStuckTasksFromSet(ctx, ProcessingAITasks, QueueAITask)
	if err != nil {
		q.logger.Error("failed to recover stuck AI tasks", zap.Error(err))
	}

	// 恢复渲染任务
	renderRecovered, err := q.recoverStuckTasksFromSet(ctx, ProcessingRenderTasks, QueueRenderTask)
	if err != nil {
		q.logger.Error("failed to recover stuck render tasks", zap.Error(err))
	}

	if aiRecovered > 0 || renderRecovered > 0 {
		q.logger.Info("stuck tasks recovered",
			zap.Int("aiTasks", aiRecovered),
			zap.Int("renderTasks", renderRecovered),
		)
	}

	return nil
}

// recoverStuckTasksFromSet 从指定处理集合中恢复卡住的任务
func (q *TaskQueue) recoverStuckTasksFromSet(ctx context.Context, processingSet, targetQueue string) (int, error) {
	// 获取处理中集合的长度
	length, err := q.client.LLen(ctx, processingSet).Result()
	if err != nil {
		return 0, fmt.Errorf("llen: %w", err)
	}

	if length == 0 {
		return 0, nil
	}

	// 获取所有处理中的任务
	items, err := q.client.LRange(ctx, processingSet, 0, length-1).Result()
	if err != nil {
		return 0, fmt.Errorf("lrange: %w", err)
	}

	recovered := 0
	now := time.Now()

	for _, item := range items {
		var msg TaskMessage
		if err := json.Unmarshal([]byte(item), &msg); err != nil {
			q.logger.Warn("failed to unmarshal stuck task, removing",
				zap.Error(err),
				zap.String("data", item),
			)
			// 移除无法解析的任务
			q.client.LRem(ctx, processingSet, 1, item)
			continue
		}

		// 检查任务是否超时
		isStuck := false
		if msg.ProcessingStartedAt != nil {
			// 如果有处理开始时间，检查是否超时
			if now.Sub(*msg.ProcessingStartedAt) > TaskProcessingTimeout {
				isStuck = true
			}
		} else {
			// 如果没有处理开始时间，使用创建时间判断
			// 如果创建时间超过2倍超时时间，认为卡住了
			if now.Sub(msg.CreatedAt) > TaskProcessingTimeout*2 {
				isStuck = true
			}
		}

		if isStuck {
			q.logger.Warn("found stuck task, recovering",
				zap.String("taskId", msg.TaskID),
				zap.String("taskType", msg.TaskType),
				zap.Int("retry", msg.Retry),
			)

			// 从处理中集合移除
			if err := q.client.LRem(ctx, processingSet, 1, item).Err(); err != nil {
				q.logger.Error("failed to remove stuck task from processing set",
					zap.String("taskId", msg.TaskID),
					zap.Error(err),
				)
				continue
			}

			// 增加重试次数
			msg.Retry++
			msg.ProcessingStartedAt = nil // 清除处理开始时间

			// 最多重试3次
			if msg.Retry <= 3 {
				// 重新放回队列
				data, _ := json.Marshal(msg)
				if err := q.client.RPush(ctx, targetQueue, data).Err(); err != nil {
					q.logger.Error("failed to requeue stuck task",
						zap.String("taskId", msg.TaskID),
						zap.Error(err),
					)
					continue
				}
				recovered++
			} else {
				// 超过重试次数，放入失败队列
				failedQueue := FailedAITasks
				if msg.TaskType == "render_task" {
					failedQueue = FailedRenderTasks
				}

				data, _ := json.Marshal(msg)
				if err := q.client.RPush(ctx, failedQueue, data).Err(); err != nil {
					q.logger.Error("failed to move stuck task to failed queue",
						zap.String("taskId", msg.TaskID),
						zap.Error(err),
					)
				}

				q.logger.Error("stuck task moved to failed queue after max retries",
					zap.String("taskId", msg.TaskID),
					zap.Int("retries", msg.Retry),
				)
			}
		}
	}

	return recovered, nil
}
