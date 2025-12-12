package mysql

import (
	"context"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/gorm"
)

// ========== AI Task operations ==========

// CreateAITask 创建AI任务
func (r *Repository) CreateAITask(ctx context.Context, task *domain.AITask) error {
	dbTask := AITaskToModel(task)
	dbTask.RetryCount = 0
	dbTask.MaxRetries = 3
	return r.db.WithContext(ctx).Create(dbTask).Error
}

// GetAITask 获取AI任务
func (r *Repository) GetAITask(ctx context.Context, taskID string) (*domain.AITask, error) {
	var dbTask AITask
	if err := r.db.WithContext(ctx).Where("id = ?", taskID).First(&dbTask).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return r.aiTaskToDomain(&dbTask), nil
}

// UpdateAITask 更新AI任务
func (r *Repository) UpdateAITask(ctx context.Context, task *domain.AITask) error {
	updates := map[string]interface{}{
		"status":        task.Status,
		"progress":      task.Progress,
		"tokens_used":   task.TokensUsed,
		"error_message": task.ErrorMessage,
		"updated_at":    time.Now(),
	}

	if task.Output != "" {
		updates["output"] = task.Output
	}
	if task.StartedAt != nil {
		updates["started_at"] = time.Unix(*task.StartedAt, 0)
	}
	if task.CompletedAt != nil {
		updates["completed_at"] = time.Unix(*task.CompletedAt, 0)
	}

	return r.db.WithContext(ctx).
		Model(&AITask{}).
		Where("id = ?", task.ID).
		Updates(updates).Error
}

// UpdateAITaskStatus 更新任务状态
func (r *Repository) UpdateAITaskStatus(ctx context.Context, taskID string, status domain.AITaskStatus, progress int) error {
	return r.db.WithContext(ctx).
		Model(&AITask{}).
		Where("id = ?", taskID).
		Updates(map[string]interface{}{
			"status":     status,
			"progress":   progress,
			"updated_at": time.Now(),
		}).Error
}

// UpdateAITaskProgress 更新任务进度
func (r *Repository) UpdateAITaskProgress(ctx context.Context, taskID string, progress int) error {
	return r.db.WithContext(ctx).
		Model(&AITask{}).
		Where("id = ?", taskID).
		Updates(map[string]interface{}{
			"progress":   progress,
			"updated_at": time.Now(),
		}).Error
}

// ListAITasks 列出用户的AI任务
func (r *Repository) ListAITasks(ctx context.Context, userID string, limit, offset int) ([]*domain.AITask, error) {
	var dbTasks []AITask

	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&dbTasks).Error

	if err != nil {
		return nil, err
	}

	tasks := make([]*domain.AITask, len(dbTasks))
	for i, dbTask := range dbTasks {
		tasks[i] = r.aiTaskToDomain(&dbTask)
	}
	return tasks, nil
}

// ListPendingAITasks 列出待处理的AI任务
func (r *Repository) ListPendingAITasks(ctx context.Context, limit int) ([]*domain.AITask, error) {
	var dbTasks []AITask

	err := r.db.WithContext(ctx).
		Where("status = ?", domain.AITaskStatusPending).
		Order("created_at ASC").
		Limit(limit).
		Find(&dbTasks).Error

	if err != nil {
		return nil, err
	}

	tasks := make([]*domain.AITask, len(dbTasks))
	for i, dbTask := range dbTasks {
		tasks[i] = r.aiTaskToDomain(&dbTask)
	}
	return tasks, nil
}

// DeleteAITask 删除AI任务
func (r *Repository) DeleteAITask(ctx context.Context, taskID string) error {
	return r.db.WithContext(ctx).
		Where("id = ?", taskID).
		Delete(&AITask{}).Error
}

// QueueAITask 将任务加入队列
func (r *Repository) QueueAITask(ctx context.Context, taskID, queueName string) error {
	return r.db.WithContext(ctx).
		Model(&AITask{}).
		Where("id = ?", taskID).
		Update("queue_name", queueName).Error
}

// DequeueAITask 从队列获取任务
func (r *Repository) DequeueAITask(ctx context.Context, queueName string) (*domain.AITask, error) {
	var dbTask AITask

	// 使用事务锁定一条记录
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 查找并锁定一条待处理的任务
		if err := tx.
			Where("queue_name = ? AND status = ?", queueName, domain.AITaskStatusPending).
			Order("created_at ASC").
			First(&dbTask).Error; err != nil {
			return err
		}

		// 立即更新为处理中状态
		return tx.Model(&AITask{}).
			Where("id = ?", dbTask.ID).
			Updates(map[string]interface{}{
				"status":     domain.AITaskStatusProcessing,
				"started_at": time.Now(),
			}).Error
	})

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return r.aiTaskToDomain(&dbTask), nil
}

// IncrementRetryCount 增加重试次数
func (r *Repository) IncrementRetryCount(ctx context.Context, taskID string) error {
	return r.db.WithContext(ctx).
		Model(&AITask{}).
		Where("id = ?", taskID).
		Update("retry_count", gorm.Expr("retry_count + 1")).Error
}

// GetAITaskRetryInfo 获取任务重试信息
func (r *Repository) GetAITaskRetryInfo(ctx context.Context, taskID string) (retryCount, maxRetries int, err error) {
	var dbTask AITask
	err = r.db.WithContext(ctx).
		Select("retry_count, max_retries").
		Where("id = ?", taskID).
		First(&dbTask).Error

	if err != nil {
		return 0, 0, err
	}
	return dbTask.RetryCount, dbTask.MaxRetries, nil
}

// ListStaleAITasks 列出超时的任务（处理中但长时间未更新）
func (r *Repository) ListStaleAITasks(ctx context.Context, timeoutMinutes int) ([]*domain.AITask, error) {
	var dbTasks []AITask
	timeout := time.Now().Add(-time.Duration(timeoutMinutes) * time.Minute)

	err := r.db.WithContext(ctx).
		Where("status = ? AND updated_at < ?", domain.AITaskStatusProcessing, timeout).
		Find(&dbTasks).Error

	if err != nil {
		return nil, err
	}

	tasks := make([]*domain.AITask, len(dbTasks))
	for i, dbTask := range dbTasks {
		tasks[i] = r.aiTaskToDomain(&dbTask)
	}
	return tasks, nil
}

// aiTaskToDomain 已移至 converters.go 中的 ModelToAITask
func (r *Repository) aiTaskToDomain(dbTask *AITask) *domain.AITask {
	return ModelToAITask(dbTask)
}
