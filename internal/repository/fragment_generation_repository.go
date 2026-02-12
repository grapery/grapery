package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/common"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/gorm"
)

type FragmentGenerationRepository struct {
	db *gorm.DB
}

func NewFragmentGenerationRepository(db *gorm.DB) *FragmentGenerationRepository {
	return &FragmentGenerationRepository{db: db}
}

// Create 创建碎片生成任务
func (r *FragmentGenerationRepository) Create(ctx context.Context, task *domain.FragmentGenerationTask) error {
	return r.db.WithContext(ctx).Create(task).Error
}

// GetByID 获取任务详情
func (r *FragmentGenerationRepository) GetByID(ctx context.Context, id string) (*domain.FragmentGenerationTask, error) {
	var task domain.FragmentGenerationTask
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&task).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// GetByUserID 获取用户的任务列表
func (r *FragmentGenerationRepository) GetByUserID(ctx context.Context, userID string, limit, offset int) ([]*domain.FragmentGenerationTask, int64, error) {
	var tasks []*domain.FragmentGenerationTask
	var total int64

	query := r.db.WithContext(ctx).Model(&domain.FragmentGenerationTask{}).Where("user_id = ?", userID)

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&tasks).Error

	if err != nil {
		return nil, 0, err
	}

	return tasks, total, nil
}

// Update 更新任务
func (r *FragmentGenerationRepository) Update(ctx context.Context, task *domain.FragmentGenerationTask) error {
	return r.db.WithContext(ctx).Save(task).Error
}

// UpdateStatus 更新任务状态
func (r *FragmentGenerationRepository) UpdateStatus(ctx context.Context, id string, status string, progress int, currentStep string) error {
	updates := map[string]interface{}{
		"status":       status,
		"progress":     progress,
		"current_step": currentStep,
		"updated_at":   time.Now().Unix(),
	}

	if status == string(common.TaskStatusProcessing) && currentStep != "" {
		updates["started_at"] = time.Now().Unix()
	}

	if status == string(common.TaskStatusCompleted) || status == string(common.TaskStatusFailed) {
		now := time.Now().Unix()
		updates["completed_at"] = &now
	}

	return r.db.WithContext(ctx).Model(&domain.FragmentGenerationTask{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// UpdateResult 更新任务结果
func (r *FragmentGenerationRepository) UpdateResult(ctx context.Context, id string, result *domain.FragmentGenerationResult) error {
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	return r.db.WithContext(ctx).Model(&domain.FragmentGenerationTask{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"result":     string(resultJSON),
			"tokens_used": gorm.Expr("tokens_used + ?", result.TokensUsed),
			"updated_at": time.Now().Unix(),
		}).Error
}

// UpdateError 更新错误信息
func (r *FragmentGenerationRepository) UpdateError(ctx context.Context, id string, status string, errorMsg string) error {
	now := time.Now().Unix()
	return r.db.WithContext(ctx).Model(&domain.FragmentGenerationTask{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":         status,
			"error_message":  errorMsg,
			"completed_at":   &now,
			"updated_at":     time.Now().Unix(),
		}).Error
}

// Delete 删除任务
func (r *FragmentGenerationRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&domain.FragmentGenerationTask{}).Error
}

// GetPendingTasks 获取待处理的任务
func (r *FragmentGenerationRepository) GetPendingTasks(ctx context.Context, limit int) ([]*domain.FragmentGenerationTask, error) {
	var tasks []*domain.FragmentGenerationTask
	err := r.db.WithContext(ctx).
		Where("status = ?", string(common.TaskStatusPending)).
		Order("created_at ASC").
		Limit(limit).
		Find(&tasks).Error
	return tasks, err
}
