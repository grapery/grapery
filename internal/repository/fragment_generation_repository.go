package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/common"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/repository/mysql"
	"gorm.io/gorm"
)

type FragmentGenerationRepository struct {
	db *gorm.DB
}

func NewFragmentGenerationRepository(db *gorm.DB) *FragmentGenerationRepository {
	return &FragmentGenerationRepository{db: db}
}

func taskToDB(task *domain.FragmentGenerationTask) (*mysql.FragmentGenerationTaskDB, error) {
	requestJSON, err := json.Marshal(task.Request)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resultJSON := ""
	if task.Result != nil {
		b, err := json.Marshal(task.Result)
		if err != nil {
			return nil, fmt.Errorf("marshal result: %w", err)
		}
		resultJSON = string(b)
	}

	return &mysql.FragmentGenerationTaskDB{
		ID:           task.ID,
		UserID:       task.UserID,
		Status:       task.Status,
		RequestJSON:  string(requestJSON),
		ResultJSON:   resultJSON,
		Progress:     task.Progress,
		CurrentStep:  task.CurrentStep,
		ErrorMessage: task.ErrorMessage,
		TokensUsed:   task.TokensUsed,
		CreatedAt:    task.CreatedAt,
		StartedAt:    task.StartedAt,
		CompletedAt:  task.CompletedAt,
		UpdatedAt:    task.UpdatedAt,
	}, nil
}

func taskFromDB(row *mysql.FragmentGenerationTaskDB) (*domain.FragmentGenerationTask, error) {
	task := &domain.FragmentGenerationTask{
		ID:           row.ID,
		UserID:       row.UserID,
		Status:       row.Status,
		Progress:     row.Progress,
		CurrentStep:  row.CurrentStep,
		ErrorMessage: row.ErrorMessage,
		TokensUsed:   row.TokensUsed,
		CreatedAt:    row.CreatedAt,
		StartedAt:    row.StartedAt,
		CompletedAt:  row.CompletedAt,
		UpdatedAt:    row.UpdatedAt,
	}

	if row.RequestJSON != "" {
		if err := json.Unmarshal([]byte(row.RequestJSON), &task.Request); err != nil {
			return nil, fmt.Errorf("unmarshal request: %w", err)
		}
	}
	if row.ResultJSON != "" {
		var result domain.FragmentGenerationResult
		if err := json.Unmarshal([]byte(row.ResultJSON), &result); err != nil {
			return nil, fmt.Errorf("unmarshal result: %w", err)
		}
		task.Result = &result
	}
	return task, nil
}

// Create 创建碎片生成任务
func (r *FragmentGenerationRepository) Create(ctx context.Context, task *domain.FragmentGenerationTask) error {
	dbTask, err := taskToDB(task)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Create(dbTask).Error
}

// GetByID 获取任务详情
func (r *FragmentGenerationRepository) GetByID(ctx context.Context, id string) (*domain.FragmentGenerationTask, error) {
	var row mysql.FragmentGenerationTaskDB
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&row).Error
	if err != nil {
		return nil, err
	}
	return taskFromDB(&row)
}

// GetByUserID 获取用户的任务列表
func (r *FragmentGenerationRepository) GetByUserID(ctx context.Context, userID string, limit, offset int) ([]*domain.FragmentGenerationTask, int64, error) {
	var rows []*mysql.FragmentGenerationTaskDB
	var total int64

	query := r.db.WithContext(ctx).Model(&mysql.FragmentGenerationTaskDB{}).Where("user_id = ?", userID)

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&rows).Error

	if err != nil {
		return nil, 0, err
	}

	tasks := make([]*domain.FragmentGenerationTask, 0, len(rows))
	for _, row := range rows {
		task, err := taskFromDB(row)
		if err != nil {
			return nil, 0, err
		}
		tasks = append(tasks, task)
	}

	return tasks, total, nil
}

// Update 更新任务
func (r *FragmentGenerationRepository) Update(ctx context.Context, task *domain.FragmentGenerationTask) error {
	dbTask, err := taskToDB(task)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Save(dbTask).Error
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

	return r.db.WithContext(ctx).Model(&mysql.FragmentGenerationTaskDB{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// UpdateResult 更新任务结果
func (r *FragmentGenerationRepository) UpdateResult(ctx context.Context, id string, result *domain.FragmentGenerationResult) error {
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	return r.db.WithContext(ctx).Model(&mysql.FragmentGenerationTaskDB{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"result_json": string(resultJSON),
			"tokens_used": result.TokensUsed,
			"updated_at":  time.Now().Unix(),
		}).Error
}

// UpdateError 更新错误信息
func (r *FragmentGenerationRepository) UpdateError(ctx context.Context, id string, status string, errorMsg string) error {
	now := time.Now().Unix()
	return r.db.WithContext(ctx).Model(&mysql.FragmentGenerationTaskDB{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":        status,
			"error_message": errorMsg,
			"completed_at":  &now,
			"updated_at":    time.Now().Unix(),
		}).Error
}

// Delete 删除任务
func (r *FragmentGenerationRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&mysql.FragmentGenerationTaskDB{}).Error
}

// GetPendingTasks 获取待处理的任务
func (r *FragmentGenerationRepository) GetPendingTasks(ctx context.Context, limit int) ([]*domain.FragmentGenerationTask, error) {
	var rows []*mysql.FragmentGenerationTaskDB
	err := r.db.WithContext(ctx).
		Where("status = ?", string(common.TaskStatusPending)).
		Order("created_at ASC").
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	tasks := make([]*domain.FragmentGenerationTask, 0, len(rows))
	for _, row := range rows {
		task, err := taskFromDB(row)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}
