package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis"
	log "github.com/sirupsen/logrus"
)

const (
	// 任务队列相关的Redis Key前缀
	VideoTaskQueueKey          = "video:task:queue"            // 待处理任务队列
	VideoTaskProcessingKey     = "video:task:processing:"      // 正在处理的任务 Hash
	VideoTaskStatusKey         = "video:task:status:"          // 任务状态缓存
	VideoTaskResultKey         = "video:task:result:"          // 任务结果缓存
	VideoTaskLockKey           = "video:task:lock:"            // 任务锁
	VideoTaskUserProcessingKey = "video:task:user:processing:" // 用户正在处理的任务ID

	// 任务状态
	TaskStatusPending    = "pending"    // 等待中
	TaskStatusProcessing = "processing" // 处理中
	TaskStatusCompleted  = "completed"  // 已完成
	TaskStatusFailed     = "failed"     // 失败

	// TTL设置
	TaskStatusTTL = 3600 * 24 * 7 // 7天
	TaskResultTTL = 3600 * 24 * 7 // 7天
	TaskLockTTL   = 300           // 5分钟
	UserTaskTTL   = 3600 * 2      // 2小时（用户任务标记）
)

// VideoTaskQueue 视频任务队列管理器
type VideoTaskQueue struct {
	client *RedisClient
}

// VideoTaskInfo 视频任务信息
type VideoTaskInfo struct {
	TaskID    string    `json:"task_id"`    // 任务UUID
	UserID    int64     `json:"user_id"`    // 用户ID
	StoryID   int64     `json:"story_id"`   // 故事ID
	BoardID   int64     `json:"board_id"`   // 故事板ID
	SceneID   int64     `json:"scene_id"`   // 场景ID
	Prompt    string    `json:"prompt"`     // 提示词
	Status    string    `json:"status"`     // 任务状态
	CreatedAt time.Time `json:"created_at"` // 创建时间
	StartedAt time.Time `json:"started_at"` // 开始时间
	UpdatedAt time.Time `json:"updated_at"` // 更新时间
	Priority  int64     `json:"priority"`   // 优先级 (分数，越大越优先)
}

// VideoTaskResult 视频任务结果
type VideoTaskResult struct {
	TaskID    string    `json:"task_id"`    // 任务UUID
	Status    string    `json:"status"`     // 任务状态
	VideoURL  string    `json:"video_url"`  // 视频URL
	Message   string    `json:"message"`    // 错误或成功消息
	Code      string    `json:"code"`       // 错误码
	Tokens    int       `json:"tokens"`     // 消耗的tokens
	StartTime int64     `json:"start_time"` // 开始时间戳
	EndTime   int64     `json:"end_time"`   // 结束时间戳
	UpdatedAt time.Time `json:"updated_at"` // 更新时间
}

// NewVideoTaskQueue 创建视频任务队列管理器
func NewVideoTaskQueue() *VideoTaskQueue {
	return &VideoTaskQueue{
		client: GetCacheClient(),
	}
}

// EnqueueTask 将任务加入队列
func (q *VideoTaskQueue) EnqueueTask(ctx context.Context, task *VideoTaskInfo) error {
	// 设置任务状态为pending
	task.Status = TaskStatusPending
	task.CreatedAt = time.Now()
	task.UpdatedAt = time.Now()

	// 序列化任务信息
	taskData, err := json.Marshal(task)
	if err != nil {
		log.Errorf("Failed to marshal task info: %v", err)
		return err
	}

	// 使用ZAdd添加到有序集合，使用优先级作为分数
	// 分数越大，优先级越高；相同优先级按时间戳排序
	score := float64(task.Priority)*1e10 + float64(task.CreatedAt.Unix())
	err = q.client.ZAdd(VideoTaskQueueKey, redis.Z{
		Score:  score,
		Member: task.TaskID,
	}).Err()
	if err != nil {
		log.Errorf("Failed to enqueue task: %v", err)
		return err
	}

	// 缓存任务详细信息
	statusKey := fmt.Sprintf("%s%s", VideoTaskStatusKey, task.TaskID)
	err = SetBytes(ctx, statusKey, taskData, TaskStatusTTL)
	if err != nil {
		log.Errorf("Failed to cache task status: %v", err)
		return err
	}

	log.Infof("✅ 任务成功加入队列: %s (优先级: %.0f, 用户: %d)", task.TaskID, score, task.UserID)
	return nil
}

// DequeueTask 从队列中取出一个任务（优先级最高的）
func (q *VideoTaskQueue) DequeueTask(ctx context.Context) (*VideoTaskInfo, error) {
	// 使用ZRevRange获取分数最高的任务（优先级最高）
	// 使用ZPOPMAX更安全，但需要Redis 5.0+
	result, err := q.client.ZRevRangeWithScores(VideoTaskQueueKey, 0, 0).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // 队列为空
		}
		log.Errorf("Failed to dequeue task: %v", err)
		return nil, err
	}

	if len(result) == 0 {
		return nil, nil // 队列为空
	}

	taskID := result[0].Member.(string)

	// 尝试获取任务锁
	lockKey := fmt.Sprintf("%s%s", VideoTaskLockKey, taskID)
	locked, err := q.client.SetNX(lockKey, "1", time.Second*time.Duration(TaskLockTTL)).Result()
	if err != nil {
		log.Errorf("Failed to acquire task lock: %v", err)
		return nil, err
	}

	if !locked {
		// 任务已被其他worker获取
		return nil, nil
	}

	// 从队列中移除
	err = q.client.ZRem(VideoTaskQueueKey, taskID).Err()
	if err != nil {
		log.Errorf("Failed to remove task from queue: %v", err)
		// 释放锁
		q.client.Del(lockKey)
		return nil, err
	}

	// 获取任务详情
	statusKey := fmt.Sprintf("%s%s", VideoTaskStatusKey, taskID)
	taskData, err := GetBytes(ctx, statusKey)
	if err != nil {
		log.Errorf("Failed to get task status: %v", err)
		// 释放锁
		q.client.Del(lockKey)
		return nil, err
	}

	var task VideoTaskInfo
	err = json.Unmarshal(taskData, &task)
	if err != nil {
		log.Errorf("Failed to unmarshal task info: %v", err)
		// 释放锁
		q.client.Del(lockKey)
		return nil, err
	}

	// 更新任务状态为processing
	task.Status = TaskStatusProcessing
	task.StartedAt = time.Now()
	task.UpdatedAt = time.Now()

	// 添加到processing集合
	processingKey := fmt.Sprintf("%s%s", VideoTaskProcessingKey, taskID)
	taskData, _ = json.Marshal(task)
	err = SetBytes(ctx, processingKey, taskData, TaskStatusTTL)
	if err != nil {
		log.Errorf("Failed to mark task as processing: %v", err)
	}

	// 更新任务状态缓存
	err = SetBytes(ctx, statusKey, taskData, TaskStatusTTL)
	if err != nil {
		log.Errorf("Failed to update task status: %v", err)
	}

	log.Infof("🎬 成功取出任务: %s (用户: %d, 状态: %s)", taskID, task.UserID, task.Status)
	return &task, nil
}

// CompleteTask 标记任务为完成
func (q *VideoTaskQueue) CompleteTask(ctx context.Context, result *VideoTaskResult) error {
	result.Status = TaskStatusCompleted
	result.UpdatedAt = time.Now()

	// 保存任务结果
	resultKey := fmt.Sprintf("%s%s", VideoTaskResultKey, result.TaskID)
	resultData, err := json.Marshal(result)
	if err != nil {
		log.Errorf("Failed to marshal task result: %v", err)
		return err
	}

	err = SetBytes(ctx, resultKey, resultData, TaskResultTTL)
	if err != nil {
		log.Errorf("Failed to save task result: %v", err)
		return err
	}

	// 更新任务状态
	statusKey := fmt.Sprintf("%s%s", VideoTaskStatusKey, result.TaskID)
	taskData, err := GetBytes(ctx, statusKey)
	if err != nil {
		log.Warnf("Failed to get task status for completion: %v", err)
	} else {
		var task VideoTaskInfo
		if err := json.Unmarshal(taskData, &task); err == nil {
			task.Status = TaskStatusCompleted
			task.UpdatedAt = time.Now()
			taskData, _ = json.Marshal(task)
			SetBytes(ctx, statusKey, taskData, TaskStatusTTL)
		}
	}

	// 从processing集合中移除
	processingKey := fmt.Sprintf("%s%s", VideoTaskProcessingKey, result.TaskID)
	DelCache(ctx, processingKey)

	// 释放任务锁
	lockKey := fmt.Sprintf("%s%s", VideoTaskLockKey, result.TaskID)
	q.client.Del(lockKey)

	log.Infof("Task completed successfully: %s", result.TaskID)
	return nil
}

// FailTask 标记任务为失败
func (q *VideoTaskQueue) FailTask(ctx context.Context, result *VideoTaskResult) error {
	result.Status = TaskStatusFailed
	result.UpdatedAt = time.Now()

	// 保存任务结果
	resultKey := fmt.Sprintf("%s%s", VideoTaskResultKey, result.TaskID)
	resultData, err := json.Marshal(result)
	if err != nil {
		log.Errorf("Failed to marshal task result: %v", err)
		return err
	}

	err = SetBytes(ctx, resultKey, resultData, TaskResultTTL)
	if err != nil {
		log.Errorf("Failed to save task result: %v", err)
		return err
	}

	// 更新任务状态
	statusKey := fmt.Sprintf("%s%s", VideoTaskStatusKey, result.TaskID)
	taskData, err := GetBytes(ctx, statusKey)
	if err != nil {
		log.Warnf("Failed to get task status for failure: %v", err)
	} else {
		var task VideoTaskInfo
		if err := json.Unmarshal(taskData, &task); err == nil {
			task.Status = TaskStatusFailed
			task.UpdatedAt = time.Now()
			taskData, _ = json.Marshal(task)
			SetBytes(ctx, statusKey, taskData, TaskStatusTTL)
		}
	}

	// 从processing集合中移除
	processingKey := fmt.Sprintf("%s%s", VideoTaskProcessingKey, result.TaskID)
	DelCache(ctx, processingKey)

	// 释放任务锁
	lockKey := fmt.Sprintf("%s%s", VideoTaskLockKey, result.TaskID)
	q.client.Del(lockKey)

	log.Infof("Task failed: %s, error: %s", result.TaskID, result.Message)
	return nil
}

// GetTaskStatus 获取任务状态
func (q *VideoTaskQueue) GetTaskStatus(ctx context.Context, taskID string) (*VideoTaskInfo, error) {
	statusKey := fmt.Sprintf("%s%s", VideoTaskStatusKey, taskID)
	taskData, err := GetBytes(ctx, statusKey)
	if err != nil {
		log.Errorf("Failed to get task status: %v", err)
		return nil, err
	}

	var task VideoTaskInfo
	err = json.Unmarshal(taskData, &task)
	if err != nil {
		log.Errorf("Failed to unmarshal task info: %v", err)
		return nil, err
	}

	return &task, nil
}

// GetTaskResult 获取任务结果
func (q *VideoTaskQueue) GetTaskResult(ctx context.Context, taskID string) (*VideoTaskResult, error) {
	resultKey := fmt.Sprintf("%s%s", VideoTaskResultKey, taskID)
	resultData, err := GetBytes(ctx, resultKey)
	if err != nil {
		log.Errorf("Failed to get task result: %v", err)
		return nil, err
	}

	var result VideoTaskResult
	err = json.Unmarshal(resultData, &result)
	if err != nil {
		log.Errorf("Failed to unmarshal task result: %v", err)
		return nil, err
	}

	return &result, nil
}

// GetQueueLength 获取队列长度
func (q *VideoTaskQueue) GetQueueLength(ctx context.Context) (int64, error) {
	count, err := q.client.ZCard(VideoTaskQueueKey).Result()
	if err != nil {
		log.Errorf("Failed to get queue length: %v", err)
		return 0, err
	}
	return count, nil
}

// GetProcessingCount 获取正在处理的任务数
func (q *VideoTaskQueue) GetProcessingCount(ctx context.Context) (int64, error) {
	// 通过扫描processing前缀的key来统计
	pattern := fmt.Sprintf("%s*", VideoTaskProcessingKey)
	keys, err := q.client.Keys(pattern).Result()
	if err != nil {
		log.Errorf("Failed to get processing count: %v", err)
		return 0, err
	}
	return int64(len(keys)), nil
}

// ReleaseLock 释放任务锁（用于异常情况）
func (q *VideoTaskQueue) ReleaseLock(ctx context.Context, taskID string) error {
	lockKey := fmt.Sprintf("%s%s", VideoTaskLockKey, taskID)
	return q.client.Del(lockKey).Err()
}

// RequeueTask 重新将任务加入队列（用于失败重试）
func (q *VideoTaskQueue) RequeueTask(ctx context.Context, taskID string, priority int64) error {
	// 获取任务信息
	statusKey := fmt.Sprintf("%s%s", VideoTaskStatusKey, taskID)
	taskData, err := GetBytes(ctx, statusKey)
	if err != nil {
		log.Errorf("Failed to get task for requeue: %v", err)
		return err
	}

	var task VideoTaskInfo
	err = json.Unmarshal(taskData, &task)
	if err != nil {
		log.Errorf("Failed to unmarshal task info: %v", err)
		return err
	}

	// 更新优先级和状态
	task.Priority = priority
	task.Status = TaskStatusPending
	task.UpdatedAt = time.Now()

	// 重新入队
	return q.EnqueueTask(ctx, &task)
}

// Global instance
var videoTaskQueue *VideoTaskQueue

// InitVideoTaskQueue 初始化视频任务队列
func InitVideoTaskQueue() *VideoTaskQueue {
	videoTaskQueue = NewVideoTaskQueue()
	return videoTaskQueue
}

// GetVideoTaskQueue 获取全局视频任务队列实例
func GetVideoTaskQueue() *VideoTaskQueue {
	if videoTaskQueue == nil {
		videoTaskQueue = NewVideoTaskQueue()
	}
	return videoTaskQueue
}

// CheckUserHasProcessingTask 检查用户是否有正在处理的任务
func (q *VideoTaskQueue) CheckUserHasProcessingTask(ctx context.Context, userID int64) (bool, string, error) {
	key := fmt.Sprintf("%s%d", VideoTaskUserProcessingKey, userID)
	taskID, err := q.client.Get(key).Result()
	if err != nil {
		if err == redis.Nil {
			// 用户没有正在处理的任务
			return false, "", nil
		}
		log.Errorf("Failed to check user processing task: %v", err)
		return false, "", err
	}

	// 用户有正在处理的任务
	return true, taskID, nil
}

// SetUserProcessingTask 设置用户正在处理的任务
func (q *VideoTaskQueue) SetUserProcessingTask(ctx context.Context, userID int64, taskID string) error {
	key := fmt.Sprintf("%s%d", VideoTaskUserProcessingKey, userID)
	err := q.client.Set(key, taskID, time.Second*time.Duration(UserTaskTTL)).Err()
	if err != nil {
		log.Errorf("Failed to set user processing task: %v", err)
		return err
	}

	log.Infof("Set user processing task: userID=%d, taskID=%s", userID, taskID)
	return nil
}

// ClearUserProcessingTask 清除用户正在处理的任务标记
func (q *VideoTaskQueue) ClearUserProcessingTask(ctx context.Context, userID int64) error {
	key := fmt.Sprintf("%s%d", VideoTaskUserProcessingKey, userID)
	err := q.client.Del(key).Err()
	if err != nil {
		log.Errorf("Failed to clear user processing task: %v", err)
		return err
	}

	log.Infof("Cleared user processing task: userID=%d", userID)
	return nil
}

// GetUserProcessingTaskID 获取用户正在处理的任务ID
func (q *VideoTaskQueue) GetUserProcessingTaskID(ctx context.Context, userID int64) (string, error) {
	key := fmt.Sprintf("%s%d", VideoTaskUserProcessingKey, userID)
	taskID, err := q.client.Get(key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", nil
		}
		log.Errorf("Failed to get user processing task: %v", err)
		return "", err
	}

	return taskID, nil
}
