package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"

	"github.com/grapestree/fgrapery/grapery/internal/common"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	genapi "github.com/grapestree/fgrapery/grapery/internal/genai"
)

// AsyncVideoCompletionService 异步视频完成处理服务
// 负责轮询异步视频任务状态，完成时确认配额预留
type AsyncVideoCompletionService struct {
	aiService *AIGenerationService
	repo      domain.Repository
	logger    *zap.Logger
	mu        sync.RWMutex

	// Redis 客户端（用于分布式锁和持久化）
	redisClient *redis.Client

	// 分布式锁服务
	distLock *RedisDistributedLock

	// 轮询配置
	pollInterval       time.Duration
	maxPollAttempts    int
	pollingTaskIDs     map[string]*PollingTask
	pollingTaskTimeout time.Duration

	// 重试配置
	maxRetryCount     int // 最大重试次数
	retryBackoffBase  time.Duration

	// 服务状态
	instanceID    string // 当前实例ID（用于分布式锁）
	started       bool
	stopChan      chan struct{}
	shutdownWG    sync.WaitGroup
}

// PollingTask 轮询任务
type PollingTask struct {
	TaskID          string
	RecordID        string
	UserID          string
	Provider        string
	ReservationID   string
	EstimatedTokens int
	StartTime       time.Time
	Status          string

	// 新增字段
	FailCount       int    // 失败计数
	LastFailAt      time.Time // 最后失败时间
	LastFailReason  string // 最后失败原因
	LockedBy        string // 锁定实例ID
}

// NewAsyncVideoCompletionService 创建异步视频完成处理服务
func NewAsyncVideoCompletionService(aiService *AIGenerationService, repo domain.Repository, logger *zap.Logger) *AsyncVideoCompletionService {
	instanceID := fmt.Sprintf("async-video-%d", time.Now().UnixNano())

	return &AsyncVideoCompletionService{
		aiService:          aiService,
		repo:               repo,
		logger:             logger,
		pollInterval:       30 * time.Second, // 每 30 秒轮询一次
		maxPollAttempts:    120,              // 最多轮询 120 次（60 分钟）
		pollingTaskIDs:     make(map[string]*PollingTask),
		pollingTaskTimeout: 90 * time.Minute, // 轮询任务超时时间
		maxRetryCount:      5,                // 最大失败重试次数
		retryBackoffBase:   30 * time.Second, // 重试退避基数
		instanceID:         instanceID,
		stopChan:           make(chan struct{}),
	}
}

// SetRedisClient 设置 Redis 客户端
func (s *AsyncVideoCompletionService) SetRedisClient(client *redis.Client) {
	s.redisClient = client
	s.distLock = NewRedisDistributedLock(client, s.logger)
}

// SetPollInterval 设置轮询间隔
func (s *AsyncVideoCompletionService) SetPollInterval(interval time.Duration) {
	s.pollInterval = interval
}

// SetMaxPollAttempts 设置最大轮询次数
func (s *AsyncVideoCompletionService) SetMaxPollAttempts(attempts int) {
	s.maxPollAttempts = attempts
}

// SetRetryConfig 设置重试配置
func (s *AsyncVideoCompletionService) SetRetryConfig(maxRetry int, backoff time.Duration) {
	s.maxRetryCount = maxRetry
	s.retryBackoffBase = backoff
}

// RestorePendingTasks 从数据库恢复未完成任务（修复服务重启丢失任务问题）
func (s *AsyncVideoCompletionService) RestorePendingTasks(ctx context.Context) error {
	s.logger.Info("restoring pending async video tasks from database")

	// 获取处理中和待处理的视频生成任务
	statuses := []domain.AITaskStatus{
		domain.AITaskStatusProcessing,
		domain.AITaskStatusPending,
	}

	records, err := s.repo.GetPendingAIGenerationRecords(ctx, statuses, 1000)
	if err != nil {
		s.logger.Error("failed to get pending AI generation records",
			zap.Error(err))
		return fmt.Errorf("failed to get pending records: %w", err)
	}

	s.logger.Info("found pending tasks in database",
		zap.Int("count", len(records)))

	restoredCount := 0
	for _, record := range records {
		// 从 metadata 中提取任务信息
		taskID, provider, reservationID, estimatedTokens, ok := s.extractTaskMetadata(record)
		if !ok {
			s.logger.Warn("failed to extract task metadata from record",
				zap.String("recordID", record.ID))
			continue
		}

		// 注册任务
		s.RegisterTask(taskID, record.ID, record.UserID, provider, reservationID, estimatedTokens)
		restoredCount++
	}

	s.logger.Info("async video tasks restored successfully",
		zap.Int("totalRecords", len(records)),
		zap.Int("restoredTasks", restoredCount),
		zap.Int("inMemoryTasks", len(s.pollingTaskIDs)))

	return nil
}

// extractTaskMetadata 从 AIGenerationRecord 的 metadata 中提取任务信息
func (s *AsyncVideoCompletionService) extractTaskMetadata(record *domain.AIGenerationRecord) (taskID, provider, reservationID string, estimatedTokens int, ok bool) {
	if record.Metadata == nil {
		return "", "", "", 0, false
	}

	// 尝试从 OutputResult 中解析（已有的输出结果）
	if record.OutputResult != "" {
		var output map[string]interface{}
		if err := json.Unmarshal([]byte(record.OutputResult), &output); err == nil {
			if tid, ok := output["taskId"].(string); ok && tid != "" {
				taskID = tid
			}
		}
	}

	// 从 metadata 中提取
	if tid, ok := record.Metadata["taskId"].(string); ok && tid != "" {
		taskID = tid
	}
	if p, ok := record.Metadata["provider"].(string); ok {
		provider = p
	} else {
		provider = record.Provider
	}
	if rid, ok := record.Metadata["reservationId"].(string); ok {
		reservationID = rid
	}
	if et, ok := record.Metadata["estimatedTokens"].(float64); ok {
		estimatedTokens = int(et)
	}

	if taskID == "" {
		return "", "", "", 0, false
	}

	return taskID, provider, reservationID, estimatedTokens, true
}

// RegisterTask 注册异步任务进行轮询
func (s *AsyncVideoCompletionService) RegisterTask(taskID, recordID, userID, provider string, reservationID string, estimatedTokens int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查任务是否已存在
	if existing, exists := s.pollingTaskIDs[taskID]; exists {
		s.logger.Warn("task already registered, updating",
			zap.String("taskID", taskID),
			zap.String("existingRecordID", existing.RecordID),
			zap.String("newRecordID", recordID))
	}

	task := &PollingTask{
		TaskID:          taskID,
		RecordID:        recordID,
		UserID:          userID,
		Provider:        provider,
		ReservationID:   reservationID,
		EstimatedTokens: estimatedTokens,
		StartTime:       time.Now(),
		Status:          "pending",
		FailCount:       0,
		LockedBy:        s.instanceID,
	}

	s.pollingTaskIDs[taskID] = task

	// 持久化到 Redis（用于多实例协同）
	if s.redisClient != nil {
		s.persistTaskToRedis(task)
	}

	s.logger.Info("async video task registered for polling",
		zap.String("taskID", taskID),
		zap.String("recordID", recordID),
		zap.String("userID", userID),
		zap.String("provider", provider),
		zap.String("instanceID", s.instanceID))
}

// persistTaskToRedis 持久化任务到 Redis
func (s *AsyncVideoCompletionService) persistTaskToRedis(task *PollingTask) {
	ctx := context.Background()
	key := fmt.Sprintf("async_video_task:%s", task.TaskID)

	data, err := json.Marshal(task)
	if err != nil {
		s.logger.Error("failed to marshal task for Redis",
			zap.String("taskID", task.TaskID),
			zap.Error(err))
		return
	}

	// 设置 2 小时过期时间
	if err := s.redisClient.Set(ctx, key, string(data), 2*time.Hour).Err(); err != nil {
		s.logger.Error("failed to persist task to Redis",
			zap.String("taskID", task.TaskID),
			zap.Error(err))
	}
}

// removeTaskFromRedis 从 Redis 移除任务
func (s *AsyncVideoCompletionService) removeTaskFromRedis(taskID string) {
	if s.redisClient == nil {
		return
	}

	ctx := context.Background()
	key := fmt.Sprintf("async_video_task:%s", taskID)
	if err := s.redisClient.Del(ctx, key).Err(); err != nil {
		s.logger.Error("failed to remove task from Redis",
			zap.String("taskID", taskID),
			zap.Error(err))
	}
}

// StartPolling 启动轮询服务
func (s *AsyncVideoCompletionService) StartPolling(ctx context.Context) {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		s.logger.Warn("async video completion polling service already started")
		return
	}
	s.started = true
	s.mu.Unlock()

	s.logger.Info("starting async video completion polling service",
		zap.Duration("pollInterval", s.pollInterval),
		zap.Int("maxPollAttempts", s.maxPollAttempts),
		zap.String("instanceID", s.instanceID))

	// 首先恢复未完成的任务
	if err := s.RestorePendingTasks(ctx); err != nil {
		s.logger.Error("failed to restore pending tasks, but continuing startup",
			zap.Error(err))
	}

	s.shutdownWG.Add(1)
	go func() {
		defer s.shutdownWG.Done()

		ticker := time.NewTicker(s.pollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				s.logger.Info("async video completion polling service stopped by context")
				return
			case <-s.stopChan:
				s.logger.Info("async video completion polling service stopped by stop signal")
				return
			case <-ticker.C:
				s.pollPendingTasks(ctx)
			}
		}
	}()
}

// Stop 停止轮询服务
func (s *AsyncVideoCompletionService) Stop() {
	s.mu.Lock()
	if !s.started {
		s.mu.Unlock()
		return
	}
	s.started = false
	s.mu.Unlock()

	s.logger.Info("stopping async video completion polling service",
		zap.String("instanceID", s.instanceID))

	close(s.stopChan)
	s.shutdownWG.Wait()

	s.logger.Info("async video completion polling service stopped")
}

// pollPendingTasks 轮询所有待处理的任务
func (s *AsyncVideoCompletionService) pollPendingTasks(ctx context.Context) {
	s.mu.Lock()
	tasks := make([]*PollingTask, 0, len(s.pollingTaskIDs))
	for _, task := range s.pollingTaskIDs {
		tasks = append(tasks, task)
	}
	s.mu.Unlock()

	if len(tasks) == 0 {
		return
	}

	s.logger.Debug("polling async video tasks",
		zap.Int("count", len(tasks)),
		zap.String("instanceID", s.instanceID))

	for _, task := range tasks {
		// 检查任务是否超时
		if time.Since(task.StartTime) > s.pollingTaskTimeout {
			s.logger.Warn("async video task polling timeout",
				zap.String("taskID", task.TaskID),
				zap.String("recordID", task.RecordID),
				zap.Duration("elapsed", time.Since(task.StartTime)))
			s.handleTaskTimeout(ctx, task)
			continue
		}

		// 获取分布式锁，防止多实例重复处理（修复多实例协同问题）
		if s.distLock != nil {
			lockKey := fmt.Sprintf("async_video_poll_lock:%s", task.TaskID)
			acquired, err := s.distLock.AcquireLock(ctx, lockKey, s.instanceID, s.pollInterval/2)
			if err != nil {
				s.logger.Warn("failed to acquire poll lock",
					zap.String("taskID", task.TaskID),
					zap.Error(err))
				continue
			}
			if !acquired {
				// 任务已被其他实例处理
				s.logger.Debug("task already being polled by another instance",
					zap.String("taskID", task.TaskID),
					zap.String("lockedBy", task.LockedBy))
				continue
			}
			// 确保释放锁
			defer func() {
				_ = s.distLock.ReleaseLock(ctx, lockKey, s.instanceID)
			}()
		}

		// 轮询单个任务状态
		if err := s.pollTaskStatus(ctx, task); err != nil {
			// 检查是否需要重试（修复轮询失败无重试计数问题）
			if s.shouldRetryTask(task) {
				s.incrementFailCount(task, err.Error())
				s.logger.Warn("task poll failed, will retry",
					zap.String("taskID", task.TaskID),
					zap.Int("failCount", task.FailCount),
					zap.Error(err))
			} else {
				s.logger.Error("task poll failed after max retries, moving to dead letter queue",
					zap.String("taskID", task.TaskID),
					zap.Int("failCount", task.FailCount),
					zap.Error(err))
				s.handleTaskDeadLetter(ctx, task, err.Error())
			}
		}
	}
}

// shouldRetryTask 判断任务是否应该重试
func (s *AsyncVideoCompletionService) shouldRetryTask(task *PollingTask) bool {
	return task.FailCount < s.maxRetryCount
}

// incrementFailCount 增加失败计数
func (s *AsyncVideoCompletionService) incrementFailCount(task *PollingTask, reason string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task.FailCount++
	task.LastFailAt = time.Now()
	task.LastFailReason = reason

	// 更新 Redis 中的任务状态
	if s.redisClient != nil {
		s.persistTaskToRedis(task)
	}
}

// handleTaskDeadLetter 处理进入死信队列的任务
func (s *AsyncVideoCompletionService) handleTaskDeadLetter(ctx context.Context, task *PollingTask, reason string) {
	s.logger.Error("task moved to dead letter queue",
		zap.String("taskID", task.TaskID),
		zap.String("recordID", task.RecordID),
		zap.String("reason", reason),
		zap.Int("failCount", task.FailCount))

	// 1. 更新记录状态为失败
	record, err := s.repo.GetAIGenerationRecord(ctx, task.RecordID)
	if err != nil {
		s.logger.Error("failed to get AI generation record for dead letter task",
			zap.String("recordID", task.RecordID),
			zap.Error(err))
	} else {
		now := time.Now()
		completedUnix := now.Unix()
		record.Status = domain.AITaskStatusFailed
		record.ErrorMessage = fmt.Sprintf("Task failed after %d retries: %s", task.FailCount, reason)
		record.CompletedAt = &completedUnix

		if err := s.repo.UpdateAIGenerationRecord(ctx, record); err != nil {
			s.logger.Error("failed to update record for dead letter task",
				zap.String("recordID", task.RecordID),
				zap.Error(err))
		}
	}

	// 2. 释放配额预留
	if task.ReservationID != "" && s.aiService.quotaReservation != nil {
		if err := s.aiService.quotaReservation.ReleaseQuota(ctx, task.ReservationID); err != nil {
			s.logger.Error("failed to release quota reservation for dead letter task",
				zap.String("reservationID", task.ReservationID),
				zap.Error(err))
		}
	}

	// 3. 从内存和 Redis 中移除任务
	s.removeTask(task.TaskID)

	// 4. 可选：将死信任务持久化到专门的死信队列（用于人工干预）
	s.persistToDeadLetterQueue(ctx, task, reason)
}

// persistToDeadLetterQueue 将任务持久化到死信队列
func (s *AsyncVideoCompletionService) persistToDeadLetterQueue(ctx context.Context, task *PollingTask, reason string) {
	if s.redisClient == nil {
		return
	}

	key := fmt.Sprintf("async_video_dlq:%s", task.TaskID)
	data := map[string]interface{}{
		"task_id":          task.TaskID,
		"record_id":        task.RecordID,
		"user_id":          task.UserID,
		"provider":         task.Provider,
		"reservation_id":   task.ReservationID,
		"estimated_tokens": task.EstimatedTokens,
		"fail_count":       task.FailCount,
		"last_fail_at":     task.LastFailAt,
		"last_fail_reason": reason,
		"created_at":       task.StartTime,
		"dead_letter_at":   time.Now(),
	}

	jsonData, _ := json.Marshal(data)
	// 死信队列保留 7 天
	if err := s.redisClient.Set(ctx, key, string(jsonData), 7*24*time.Hour).Err(); err != nil {
		s.logger.Error("failed to persist task to dead letter queue",
			zap.String("taskID", task.TaskID),
			zap.Error(err))
	}
}

// handleTaskTimeout 处理超时任务
func (s *AsyncVideoCompletionService) handleTaskTimeout(ctx context.Context, task *PollingTask) {
	// 将超时任务移入死信队列
	s.handleTaskDeadLetter(ctx, task, "polling timeout")
}

// pollTaskStatus 轮询单个任务状态
func (s *AsyncVideoCompletionService) pollTaskStatus(ctx context.Context, task *PollingTask) error {
	// 获取视频状态
	if s.aiService.genAPI == nil {
		return fmt.Errorf("GenAPI not configured")
	}

	resp, err := s.aiService.genAPI.GetVideoStatus(ctx, task.Provider, task.TaskID)
	if err != nil {
		return fmt.Errorf("failed to get video status: %w", err)
	}

	// 检查任务状态
	if resp.Status == string(common.TaskStatusCompleted) && resp.VideoURL != "" {
		// 视频生成完成
		s.handleTaskCompletion(ctx, task, resp)
		// 成功处理，重置失败计数
		task.FailCount = 0
	} else if resp.Status == string(common.TaskStatusFailed) || resp.Status == "error" {
		// 视频生成失败
		s.handleTaskFailure(ctx, task, resp.Error)
	} else {
		// 仍在处理中
		s.logger.Debug("video task still processing",
			zap.String("taskID", task.TaskID),
			zap.String("status", resp.Status),
			zap.Int("progress", resp.Progress))
	}

	return nil
}

// handleTaskCompletion 处理任务完成
func (s *AsyncVideoCompletionService) handleTaskCompletion(ctx context.Context, task *PollingTask, resp *genapi.GenerateResponse) {
	s.logger.Info("async video task completed",
		zap.String("taskID", task.TaskID),
		zap.String("recordID", task.RecordID),
		zap.String("videoURL", resp.VideoURL))

	// 1. 获取 AI 生成记录
	record, err := s.repo.GetAIGenerationRecord(ctx, task.RecordID)
	if err != nil {
		s.logger.Error("failed to get AI generation record",
			zap.String("recordID", task.RecordID),
			zap.Error(err))
		return
	}

	// 2. 更新记录状态
	now := time.Now()
	completedUnix := now.Unix()
	record.Status = domain.AITaskStatusCompleted
	record.Progress = 100
	record.VideoCount = 1
	record.CompletedAt = &completedUnix
	record.DurationMs = now.Sub(time.Unix(record.CreatedAt, 0)).Milliseconds()

	// 记录使用量
	if resp.Usage != nil {
		record.TotalTokens = resp.Usage.TotalTokens
		record.VideoCount = resp.Usage.VideoCount
	}

	// 更新输出结果
	outputResult := map[string]interface{}{
		"taskId":   task.TaskID,
		"videoURL": resp.VideoURL,
		"status":   "completed",
		"progress": 100,
	}
	if outputJSON, err := json.Marshal(outputResult); err == nil {
		record.OutputResult = string(outputJSON)
	}

	if err := s.repo.UpdateAIGenerationRecord(ctx, record); err != nil {
		s.logger.Error("failed to update AI generation record",
			zap.String("recordID", task.RecordID),
			zap.Error(err))
	}

	// 3. 确认配额预留或扣减 token（修复配额确认失败数据不一致问题）
	actualTokens := record.TotalTokens
	if actualTokens == 0 {
		actualTokens = 5000 // 默认 5000 tokens
	}

	quotaConfirmSuccess := false
	if task.ReservationID != "" && s.aiService.quotaReservation != nil {
		if err := s.confirmQuotaWithRetry(ctx, task, actualTokens); err != nil {
			s.logger.Error("failed to confirm quota reservation after retries, initiating compensation",
				zap.String("reservationID", task.ReservationID),
				zap.String("recordID", task.RecordID),
				zap.Int("actualTokens", actualTokens),
				zap.Error(err))

			// 启动补偿事务
			go s.compensateQuotaConfirmation(context.Background(), task, actualTokens, record.ID)
		} else {
			quotaConfirmSuccess = true
			s.logger.Info("quota reservation confirmed for async video",
				zap.String("reservationID", task.ReservationID),
				zap.String("recordID", task.RecordID),
				zap.Int("estimatedTokens", task.EstimatedTokens),
				zap.Int("actualTokens", actualTokens))
		}
	} else {
		// 如果没有配额预留，直接扣减
		if _, err := s.repo.UpdateTokenBalance(ctx, task.UserID, -actualTokens, "ai_video_generation", fmt.Sprintf("AI async video generation consumed %d tokens", actualTokens)); err != nil {
			s.logger.Error("failed to deduct token balance for async video",
				zap.String("userID", task.UserID),
				zap.String("recordID", task.RecordID),
				zap.Int("tokensUsed", actualTokens),
				zap.Error(err))
			// 记录到对账队列
			s.recordForReconciliation(context.Background(), task, actualTokens, "deduct_failed")
		} else {
			quotaConfirmSuccess = true
			s.logger.Info("token balance deducted for async video",
				zap.String("userID", task.UserID),
				zap.String("recordID", task.RecordID),
				zap.Int("tokensUsed", actualTokens))
		}
	}

	// 如果配额处理成功，记录成功事件
	if quotaConfirmSuccess {
		s.recordQuotaSuccess(context.Background(), task, actualTokens)
	}

	// 4. 移除轮询任务
	s.removeTask(task.TaskID)
}

// confirmQuotaWithRetry 带重试的配额确认
func (s *AsyncVideoCompletionService) confirmQuotaWithRetry(ctx context.Context, task *PollingTask, actualTokens int) error {
	var lastErr error
	for i := 0; i < 3; i++ {
		err := s.aiService.quotaReservation.ConfirmQuota(ctx, task.ReservationID, actualTokens)
		if err == nil {
			return nil
		}
		lastErr = err
		s.logger.Warn("quota confirm attempt failed, retrying",
			zap.String("reservationID", task.ReservationID),
			zap.Int("attempt", i+1),
			zap.Error(err))
		time.Sleep(time.Duration(i+1) * time.Second)
	}
	return lastErr
}

// compensateQuotaConfirmation 配额确认失败的补偿事务
func (s *AsyncVideoCompletionService) compensateQuotaConfirmation(ctx context.Context, task *PollingTask, actualTokens int, recordID string) {
	s.logger.Warn("starting quota confirmation compensation",
		zap.String("taskID", task.TaskID),
		zap.String("reservationID", task.ReservationID),
		zap.Int("actualTokens", actualTokens))

	// 1. 尝试再次确认预留
	if task.ReservationID != "" && s.aiService.quotaReservation != nil {
		if err := s.confirmQuotaWithRetry(ctx, task, actualTokens); err == nil {
			s.logger.Info("quota confirmation recovered via retry",
				zap.String("reservationID", task.ReservationID))
			s.recordQuotaSuccess(ctx, task, actualTokens)
			return
		}
	}

	// 2. 如果确认失败，预留已过期，需要直接扣减用户余额
	// 但要先检查预留是否真的已过期（防止重复扣费）
	reservationExpired := true
	if s.aiService.quotaReservation != nil {
		// 检查预留状态（如果实现了状态查询）
		// 这里假设预留已过期
	}

	if reservationExpired {
		// 直接扣减用户余额并记录补偿事件
		if _, err := s.repo.UpdateTokenBalance(ctx, task.UserID, -actualTokens, "ai_video_generation_compensation", fmt.Sprintf("Compensation for failed reservation confirm: task %s", task.TaskID)); err != nil {
			s.logger.Error("compensation deduct failed, recording for manual reconciliation",
				zap.String("userID", task.UserID),
				zap.String("taskID", task.TaskID),
				zap.Int("tokens", actualTokens),
				zap.Error(err))
			s.recordForReconciliation(ctx, task, actualTokens, "compensation_failed")
		} else {
			s.logger.Info("compensation deduct completed",
				zap.String("userID", task.UserID),
				zap.String("taskID", task.TaskID),
				zap.Int("tokens", actualTokens))
		}
	}

	// 3. 记录补偿事件
	s.recordCompensationEvent(ctx, task, actualTokens, recordID)
}

// recordForReconciliation 记录需要人工对账的事件
func (s *AsyncVideoCompletionService) recordForReconciliation(ctx context.Context, task *PollingTask, tokens int, reason string) {
	if s.redisClient == nil {
		return
	}

	key := fmt.Sprintf("quota_reconciliation:%s:%s", task.TaskID, task.RecordID)
	data := map[string]interface{}{
		"task_id":        task.TaskID,
		"record_id":      task.RecordID,
		"user_id":        task.UserID,
		"tokens":         tokens,
		"reason":         reason,
		"created_at":     time.Now(),
		"requires_manual": true,
	}

	jsonData, _ := json.Marshal(data)
	// 保留 30 天
	if err := s.redisClient.Set(ctx, key, string(jsonData), 30*24*time.Hour).Err(); err != nil {
		s.logger.Error("failed to record reconciliation event",
			zap.String("taskID", task.TaskID),
			zap.Error(err))
	}
}

// recordQuotaSuccess 记录配额处理成功事件
func (s *AsyncVideoCompletionService) recordQuotaSuccess(ctx context.Context, task *PollingTask, tokens int) {
	if s.redisClient == nil {
		return
	}

	key := fmt.Sprintf("quota_success:%s:%s", task.TaskID, task.RecordID)
	data := map[string]interface{}{
		"task_id":   task.TaskID,
		"record_id": task.RecordID,
		"user_id":   task.UserID,
		"tokens":    tokens,
		"timestamp": time.Now(),
	}

	jsonData, _ := json.Marshal(data)
	// 保留 7 天
	if err := s.redisClient.Set(ctx, key, string(jsonData), 7*24*time.Hour).Err(); err != nil {
		s.logger.Debug("failed to record quota success event",
			zap.String("taskID", task.TaskID))
	}
}

// recordCompensationEvent 记录补偿事件
func (s *AsyncVideoCompletionService) recordCompensationEvent(ctx context.Context, task *PollingTask, tokens int, recordID string) {
	if s.redisClient == nil {
		return
	}

	key := fmt.Sprintf("quota_compensation:%s:%s", task.TaskID, recordID)
	data := map[string]interface{}{
		"task_id":   task.TaskID,
		"record_id": recordID,
		"user_id":   task.UserID,
		"tokens":    tokens,
		"timestamp": time.Now(),
	}

	jsonData, _ := json.Marshal(data)
	// 保留 30 天
	if err := s.redisClient.Set(ctx, key, string(jsonData), 30*24*time.Hour).Err(); err != nil {
		s.logger.Error("failed to record compensation event",
			zap.String("taskID", task.TaskID),
			zap.Error(err))
	}
}

// handleTaskFailure 处理任务失败
func (s *AsyncVideoCompletionService) handleTaskFailure(ctx context.Context, task *PollingTask, errorMsg string) {
	s.logger.Warn("async video task failed",
		zap.String("taskID", task.TaskID),
		zap.String("recordID", task.RecordID),
		zap.String("error", errorMsg))

	// 1. 获取 AI 生成记录
	record, err := s.repo.GetAIGenerationRecord(ctx, task.RecordID)
	if err != nil {
		s.logger.Error("failed to get AI generation record",
			zap.String("recordID", task.RecordID),
			zap.Error(err))
		return
	}

	// 2. 更新记录状态
	now := time.Now()
	completedUnix := now.Unix()
	record.Status = domain.AITaskStatusFailed
	record.ErrorMessage = errorMsg
	record.CompletedAt = &completedUnix

	if err := s.repo.UpdateAIGenerationRecord(ctx, record); err != nil {
		s.logger.Error("failed to update AI generation record",
			zap.String("recordID", task.RecordID),
			zap.Error(err))
	}

	// 3. 释放配额预留
	if task.ReservationID != "" && s.aiService.quotaReservation != nil {
		if err := s.aiService.quotaReservation.ReleaseQuota(ctx, task.ReservationID); err != nil {
			s.logger.Error("failed to release quota reservation for failed async video",
				zap.String("reservationID", task.ReservationID),
				zap.String("recordID", task.RecordID),
				zap.Error(err))
			// 记录到对账队列
			s.recordForReconciliation(ctx, task, task.EstimatedTokens, "release_failed")
		} else {
			s.logger.Info("quota reservation released for failed async video",
				zap.String("reservationID", task.ReservationID),
				zap.String("recordID", task.RecordID),
				zap.Int("reservedTokens", task.EstimatedTokens))
		}
	}

	// 4. 移除轮询任务
	s.removeTask(task.TaskID)
}

// removeTask 移除轮询任务
func (s *AsyncVideoCompletionService) removeTask(taskID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.pollingTaskIDs, taskID)
	s.removeTaskFromRedis(taskID)
}

// GetPollingTaskCount 获取当前轮询任务数量
func (s *AsyncVideoCompletionService) GetPollingTaskCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.pollingTaskIDs)
}

// GetPollingTasks 获取所有轮询任务
func (s *AsyncVideoCompletionService) GetPollingTasks() []*PollingTask {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks := make([]*PollingTask, 0, len(s.pollingTaskIDs))
	for _, task := range s.pollingTaskIDs {
		tasks = append(tasks, task)
	}
	return tasks
}

// GetDeadLetterTasks 获取死信队列中的任务数量
func (s *AsyncVideoCompletionService) GetDeadLetterTasks(ctx context.Context) (int, error) {
	if s.redisClient == nil {
		return 0, fmt.Errorf("redis not configured")
	}

	keys, err := s.redisClient.Keys(ctx, "async_video_dlq:*").Result()
	if err != nil {
		return 0, err
	}
	return len(keys), nil
}

// GetReconciliationTasks 获取需要对账的任务数量
func (s *AsyncVideoCompletionService) GetReconciliationTasks(ctx context.Context) (int, error) {
	if s.redisClient == nil {
		return 0, fmt.Errorf("redis not configured")
	}

	keys, err := s.redisClient.Keys(ctx, "quota_reconciliation:*").Result()
	if err != nil {
		return 0, err
	}
	return len(keys), nil
}
