package mysql

import (
	"context"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/gorm"
)

// ========== Render Task operations ==========

// CreateRenderTask 创建渲染任务
func (r *Repository) CreateRenderTask(ctx context.Context, task *domain.RenderTask) error {
	dbTask := RenderTaskToModel(task)
	dbTask.RetryCount = 0
	dbTask.MaxRetries = 3
	return r.db.WithContext(ctx).Create(dbTask).Error
}

// GetRenderTask 获取渲染任务
func (r *Repository) GetRenderTask(ctx context.Context, taskID string) (*domain.RenderTask, error) {
	var dbTask RenderTask
	if err := r.db.WithContext(ctx).Where("id = ?", taskID).First(&dbTask).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return r.renderTaskToDomain(&dbTask), nil
}

// GetRenderTaskByStoryID 获取故事的最新渲染任务
func (r *Repository) GetRenderTaskByStoryID(ctx context.Context, storyID string) (*domain.RenderTask, error) {
	var dbTask RenderTask
	err := r.db.WithContext(ctx).
		Where("story_id = ?", storyID).
		Order("created_at DESC").
		First(&dbTask).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return r.renderTaskToDomain(&dbTask), nil
}

// UpdateRenderTask 更新渲染任务
func (r *Repository) UpdateRenderTask(ctx context.Context, task *domain.RenderTask) error {
	updates := map[string]interface{}{
		"status":        task.Status,
		"progress":      task.Progress,
		"output_url":    task.OutputURL,
		"thumbnail_url": task.ThumbnailURL,
		"error_message": task.ErrorMessage,
		"file_size":     task.FileSize,
		"duration":      task.Duration,
		"resolution":    task.Resolution,
		"updated_at":    time.Now(),
	}

	if task.StartedAt != nil {
		updates["started_at"] = time.Unix(*task.StartedAt, 0)
	}
	if task.CompletedAt != nil {
		updates["completed_at"] = time.Unix(*task.CompletedAt, 0)
	}

	return r.db.WithContext(ctx).
		Model(&RenderTask{}).
		Where("id = ?", task.ID).
		Updates(updates).Error
}

// UpdateRenderTaskStatus 更新任务状态
func (r *Repository) UpdateRenderTaskStatus(ctx context.Context, taskID string, status domain.RenderTaskStatus, progress int) error {
	return r.db.WithContext(ctx).
		Model(&RenderTask{}).
		Where("id = ?", taskID).
		Updates(map[string]interface{}{
			"status":     status,
			"progress":   progress,
			"updated_at": time.Now(),
		}).Error
}

// UpdateRenderTaskProgress 更新任务进度
func (r *Repository) UpdateRenderTaskProgress(ctx context.Context, taskID string, progress int) error {
	return r.db.WithContext(ctx).
		Model(&RenderTask{}).
		Where("id = ?", taskID).
		Updates(map[string]interface{}{
			"progress":   progress,
			"updated_at": time.Now(),
		}).Error
}

// ListRenderTasks 列出用户的渲染任务
func (r *Repository) ListRenderTasks(ctx context.Context, userID string, limit, offset int) ([]*domain.RenderTask, error) {
	var dbTasks []RenderTask

	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&dbTasks).Error

	if err != nil {
		return nil, err
	}

	tasks := make([]*domain.RenderTask, len(dbTasks))
	for i, dbTask := range dbTasks {
		tasks[i] = r.renderTaskToDomain(&dbTask)
	}
	return tasks, nil
}

// ListPendingRenderTasks 列出待处理的渲染任务
func (r *Repository) ListPendingRenderTasks(ctx context.Context, limit int) ([]*domain.RenderTask, error) {
	var dbTasks []RenderTask

	err := r.db.WithContext(ctx).
		Where("status = ?", domain.RenderTaskStatusPending).
		Order("created_at ASC").
		Limit(limit).
		Find(&dbTasks).Error

	if err != nil {
		return nil, err
	}

	tasks := make([]*domain.RenderTask, len(dbTasks))
	for i, dbTask := range dbTasks {
		tasks[i] = r.renderTaskToDomain(&dbTask)
	}
	return tasks, nil
}

// DeleteRenderTask 删除渲染任务
func (r *Repository) DeleteRenderTask(ctx context.Context, taskID string) error {
	return r.db.WithContext(ctx).
		Where("id = ?", taskID).
		Delete(&RenderTask{}).Error
}

// QueueRenderTask 将任务加入队列
func (r *Repository) QueueRenderTask(ctx context.Context, taskID, queueName string) error {
	return r.db.WithContext(ctx).
		Model(&RenderTask{}).
		Where("id = ?", taskID).
		Update("queue_name", queueName).Error
}

// DequeueRenderTask 从队列获取任务
func (r *Repository) DequeueRenderTask(ctx context.Context, queueName string) (*domain.RenderTask, error) {
	var dbTask RenderTask

	// 使用事务锁定一条记录
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 查找并锁定一条待处理的任务
		if err := tx.
			Where("queue_name = ? AND status = ?", queueName, domain.RenderTaskStatusPending).
			Order("created_at ASC").
			First(&dbTask).Error; err != nil {
			return err
		}

		// 立即更新为处理中状态
		return tx.Model(&RenderTask{}).
			Where("id = ?", dbTask.ID).
			Updates(map[string]interface{}{
				"status":     domain.RenderTaskStatusProcessing,
				"started_at": time.Now(),
			}).Error
	})

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return r.renderTaskToDomain(&dbTask), nil
}

// IncrementRenderRetryCount 增加重试次数
func (r *Repository) IncrementRenderRetryCount(ctx context.Context, taskID string) error {
	return r.db.WithContext(ctx).
		Model(&RenderTask{}).
		Where("id = ?", taskID).
		Update("retry_count", gorm.Expr("retry_count + 1")).Error
}

// GetRenderTaskRetryInfo 获取任务重试信息
func (r *Repository) GetRenderTaskRetryInfo(ctx context.Context, taskID string) (retryCount, maxRetries int, err error) {
	var dbTask RenderTask
	err = r.db.WithContext(ctx).
		Select("retry_count, max_retries").
		Where("id = ?", taskID).
		First(&dbTask).Error

	if err != nil {
		return 0, 0, err
	}
	return dbTask.RetryCount, dbTask.MaxRetries, nil
}

// ListStaleRenderTasks 列出超时的任务（处理中但长时间未更新）
func (r *Repository) ListStaleRenderTasks(ctx context.Context, timeoutMinutes int) ([]*domain.RenderTask, error) {
	var dbTasks []RenderTask
	timeout := time.Now().Add(-time.Duration(timeoutMinutes) * time.Minute)

	err := r.db.WithContext(ctx).
		Where("status = ? AND updated_at < ?", domain.RenderTaskStatusProcessing, timeout).
		Find(&dbTasks).Error

	if err != nil {
		return nil, err
	}

	tasks := make([]*domain.RenderTask, len(dbTasks))
	for i, dbTask := range dbTasks {
		tasks[i] = r.renderTaskToDomain(&dbTask)
	}
	return tasks, nil
}

// ========== Story Publication operations ==========

// CreateStoryPublication 创建故事发布记录
func (r *Repository) CreateStoryPublication(ctx context.Context, publication *domain.StoryPublication) error {
	dbPub := &StoryPublication{
		ID:           publication.ID,
		StoryID:      publication.StoryID,
		Version:      publication.Version,
		Status:       publication.Status,
		RenderTaskID: publication.RenderTaskID,
		PublishedAt:  time.Unix(publication.PublishedAt, 0),
	}

	return r.db.WithContext(ctx).Create(dbPub).Error
}

// GetStoryPublication 获取故事发布记录
func (r *Repository) GetStoryPublication(ctx context.Context, publicationID string) (*domain.StoryPublication, error) {
	var dbPub StoryPublication
	if err := r.db.WithContext(ctx).Where("id = ?", publicationID).First(&dbPub).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return r.storyPublicationToDomain(&dbPub), nil
}

// GetLatestStoryPublication 获取故事的最新发布版本
func (r *Repository) GetLatestStoryPublication(ctx context.Context, storyID string) (*domain.StoryPublication, error) {
	var dbPub StoryPublication
	err := r.db.WithContext(ctx).
		Where("story_id = ?", storyID).
		Order("version DESC").
		First(&dbPub).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return r.storyPublicationToDomain(&dbPub), nil
}

// UpdateStoryPublication 更新故事发布记录
func (r *Repository) UpdateStoryPublication(ctx context.Context, publication *domain.StoryPublication) error {
	updates := map[string]interface{}{
		"status":         publication.Status,
		"render_task_id": publication.RenderTaskID,
		"updated_at":     time.Now(),
	}

	if publication.UnpublishedAt != nil {
		updates["unpublished_at"] = time.Unix(*publication.UnpublishedAt, 0)
	}

	return r.db.WithContext(ctx).
		Model(&StoryPublication{}).
		Where("id = ?", publication.ID).
		Updates(updates).Error
}

// ListStoryPublications 列出故事的所有发布版本
func (r *Repository) ListStoryPublications(ctx context.Context, storyID string) ([]*domain.StoryPublication, error) {
	var dbPubs []StoryPublication

	err := r.db.WithContext(ctx).
		Where("story_id = ?", storyID).
		Order("version DESC").
		Find(&dbPubs).Error

	if err != nil {
		return nil, err
	}

	pubs := make([]*domain.StoryPublication, len(dbPubs))
	for i, dbPub := range dbPubs {
		pubs[i] = r.storyPublicationToDomain(&dbPub)
	}
	return pubs, nil
}

// renderTaskToDomain 已移至 converters.go 中的 ModelToRenderTask
func (r *Repository) renderTaskToDomain(dbTask *RenderTask) *domain.RenderTask {
	return ModelToRenderTask(dbTask)
}

// storyPublicationToDomain 已移至 converters.go 中的 ModelToStoryPublication
func (r *Repository) storyPublicationToDomain(dbPub *StoryPublication) *domain.StoryPublication {
	return ModelToStoryPublication(dbPub)
}
