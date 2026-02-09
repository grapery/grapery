package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	genapi "github.com/grapestree/fgrapery/grapery/internal/genai"
	"go.uber.org/zap"
)

// AsyncVideoCompletionService 异步视频完成处理服务
// 负责轮询异步视频任务状态，完成时确认配额预留
type AsyncVideoCompletionService struct {
	aiService *AIGenerationService
	repo      domain.Repository
	logger    *zap.Logger
	mu        sync.RWMutex

	// 轮询配置
	pollInterval       time.Duration
	maxPollAttempts    int
	pollingTaskIDs     map[string]*PollingTask
	pollingTaskTimeout time.Duration
}

// PollingTask 轮询任务
type PollingTask struct {
	TaskID        string
	RecordID      string
	UserID        string
	Provider      string
	ReservationID string
	EstimatedTokens int
	StartTime     time.Time
	Status        string
}

// NewAsyncVideoCompletionService 创建异步视频完成处理服务
func NewAsyncVideoCompletionService(aiService *AIGenerationService, repo domain.Repository, logger *zap.Logger) *AsyncVideoCompletionService {
	return &AsyncVideoCompletionService{
		aiService:          aiService,
		repo:               repo,
		logger:             logger,
		pollInterval:       30 * time.Second, // 每 30 秒轮询一次
		maxPollAttempts:    120,               // 最多轮询 120 次（60 分钟）
		pollingTaskIDs:     make(map[string]*PollingTask),
		pollingTaskTimeout: 90 * time.Minute,  // 轮询任务超时时间
	}
}

// SetPollInterval 设置轮询间隔
func (s *AsyncVideoCompletionService) SetPollInterval(interval time.Duration) {
	s.pollInterval = interval
}

// SetMaxPollAttempts 设置最大轮询次数
func (s *AsyncVideoCompletionService) SetMaxPollAttempts(attempts int) {
	s.maxPollAttempts = attempts
}

// RegisterTask 注册异步任务进行轮询
func (s *AsyncVideoCompletionService) RegisterTask(taskID, recordID, userID, provider string, reservationID string, estimatedTokens int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task := &PollingTask{
		TaskID:          taskID,
		RecordID:        recordID,
		UserID:          userID,
		Provider:        provider,
		ReservationID:   reservationID,
		EstimatedTokens: estimatedTokens,
		StartTime:       time.Now(),
		Status:          "pending",
	}

	s.pollingTaskIDs[taskID] = task

	s.logger.Info("async video task registered for polling",
		zap.String("taskID", taskID),
		zap.String("recordID", recordID),
		zap.String("userID", userID),
		zap.String("provider", provider))
}

// StartPolling 启动轮询服务
func (s *AsyncVideoCompletionService) StartPolling(ctx context.Context) {
	s.logger.Info("starting async video completion polling service",
		zap.Duration("pollInterval", s.pollInterval),
		zap.Int("maxPollAttempts", s.maxPollAttempts))

	ticker := time.NewTicker(s.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("async video completion polling service stopped")
			return
		case <-ticker.C:
			s.pollPendingTasks(ctx)
		}
	}
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

	s.logger.Debug("polling async video tasks", zap.Int("count", len(tasks)))

	for _, task := range tasks {
		// 检查任务是否超时
		if time.Since(task.StartTime) > s.pollingTaskTimeout {
			s.logger.Warn("async video task polling timeout",
				zap.String("taskID", task.TaskID),
				zap.String("recordID", task.RecordID),
				zap.Duration("elapsed", time.Since(task.StartTime)))
			s.removeTask(task.TaskID)
			continue
		}

		// 轮询单个任务状态
		if err := s.pollTaskStatus(ctx, task); err != nil {
			s.logger.Warn("failed to poll task status",
				zap.String("taskID", task.TaskID),
				zap.Error(err))
		}
	}
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
	if resp.Status == "completed" && resp.VideoURL != "" {
		// 视频生成完成
		s.handleTaskCompletion(ctx, task, resp)
	} else if resp.Status == "failed" || resp.Status == "error" {
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
func (s *AsyncVideoCompletionService) handleTaskCompletion(ctx context.Context, task *PollingTask, resp *genapi.GenerateVideoResponse) {
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

	// 3. 确认配额预留或扣减 token
	actualTokens := record.TotalTokens
	if actualTokens == 0 {
		actualTokens = 5000 // 默认 5000 tokens
	}

	// 如果有配额预留，确认实际使用量
	if task.ReservationID != "" && s.aiService.quotaReservation != nil {
		if err := s.aiService.quotaReservation.ConfirmQuota(ctx, task.ReservationID, actualTokens); err != nil {
			s.logger.Error("failed to confirm quota reservation for async video",
				zap.String("reservationID", task.ReservationID),
				zap.String("recordID", task.RecordID),
				zap.Int("actualTokens", actualTokens),
				zap.Error(err))
			// 确认失败，预留会在过期时自动释放
		} else {
			s.logger.Info("quota reservation confirmed for async video",
				zap.String("reservationID", task.ReservationID),
				zap.String("recordID", task.RecordID),
				zap.Int("estimatedTokens", task.EstimatedTokens),
				zap.Int("actualTokens", actualTokens))
		}
	} else {
		// 如果没有配额预留，直接扣减
		_, err := s.repo.UpdateTokenBalance(ctx, task.UserID, -actualTokens, "ai_video_generation", fmt.Sprintf("AI async video generation consumed %d tokens", actualTokens))
		if err != nil {
			s.logger.Error("failed to deduct token balance for async video",
				zap.String("userID", task.UserID),
				zap.String("recordID", task.RecordID),
				zap.Int("tokensUsed", actualTokens),
				zap.Error(err))
		} else {
			s.logger.Info("token balance deducted for async video",
				zap.String("userID", task.UserID),
				zap.String("recordID", task.RecordID),
				zap.Int("tokensUsed", actualTokens))
		}
	}

	// 4. 移除轮询任务
	s.removeTask(task.TaskID)
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
