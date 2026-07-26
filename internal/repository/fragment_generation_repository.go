package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/common"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/repository/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

func fragmentGenerationTasksFromRows(rows []*mysql.FragmentGenerationTaskDB) ([]*domain.FragmentGenerationTask, error) {
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

	tasks, err := fragmentGenerationTasksFromRows(rows)
	if err != nil {
		return nil, 0, err
	}

	return tasks, total, nil
}

func (r *FragmentGenerationRepository) FindByClientMessageID(ctx context.Context, userID, clientMessageID string) (*domain.FragmentGenerationTask, error) {
	if userID == "" || clientMessageID == "" {
		return nil, nil
	}
	var rows []*mysql.FragmentGenerationTaskDB
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(100).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		task, err := taskFromDB(row)
		if err != nil {
			return nil, err
		}
		if task.Request.ClientMessageID == clientMessageID {
			return task, nil
		}
	}
	return nil, nil
}

func (r *FragmentGenerationRepository) FindActiveByDraftID(ctx context.Context, userID, draftID string) (*domain.FragmentGenerationTask, error) {
	if userID == "" || draftID == "" {
		return nil, nil
	}
	var rows []*mysql.FragmentGenerationTaskDB
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND status IN ?", userID, []string{
			string(common.TaskStatusPending),
			string(common.TaskStatusProcessing),
		}).
		Order("created_at DESC").
		Limit(100).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		task, err := taskFromDB(row)
		if err != nil {
			return nil, err
		}
		if task.Request.TargetDraftFragmentID == draftID {
			return task, nil
		}
	}
	return nil, nil
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

	query := r.db.WithContext(ctx).Model(&mysql.FragmentGenerationTaskDB{}).Where("id = ?", id)
	if status == string(common.TaskStatusCancelled) {
		query = query.Where("status IN ?", []string{string(common.TaskStatusPending), string(common.TaskStatusProcessing)})
	} else {
		query = query.Where("status NOT IN ?", []string{
			string(common.TaskStatusCompleted),
			string(common.TaskStatusFailed),
			string(common.TaskStatusCancelled),
		})
	}
	return query.Updates(updates).Error
}

// UpdateResult 更新任务结果
func (r *FragmentGenerationRepository) UpdateResult(ctx context.Context, id string, result *domain.FragmentGenerationResult) error {
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	return r.db.WithContext(ctx).Model(&mysql.FragmentGenerationTaskDB{}).
		Where("id = ? AND status NOT IN ?", id, []string{
			string(common.TaskStatusCompleted),
			string(common.TaskStatusFailed),
			string(common.TaskStatusCancelled),
		}).
		Updates(map[string]interface{}{
			"result_json": string(resultJSON),
			"tokens_used": result.TokensUsed,
			"updated_at":  time.Now().Unix(),
		}).Error
}

// UpsertImageSlots stores the durable per-image state for a fragment generation task.
func (r *FragmentGenerationRepository) UpsertImageSlots(ctx context.Context, taskID, fragmentID string, slots []domain.FragmentGenerationImageSlot) error {
	if len(slots) == 0 {
		return nil
	}
	existingByIndex, err := r.fragmentGenerationImageSlotsByIndex(ctx, taskID)
	if err != nil {
		return err
	}
	rows := make([]*mysql.FragmentGenerationImageSlotDB, 0, len(slots))
	now := time.Now().Unix()
	for _, slot := range slots {
		if slot.Index <= 0 {
			continue
		}
		id := slot.ID
		if id == "" {
			id = stableFragmentGenerationImageSlotID(taskID, slot.Index)
		}
		slotTaskID := slot.TaskID
		if slotTaskID == "" {
			slotTaskID = taskID
		}
		slotFragmentID := slot.FragmentID
		if slotFragmentID == "" {
			slotFragmentID = fragmentID
		}
		status := slot.Status
		if status == "" {
			status = "planned"
		}
		imageURL := slot.ImageURL
		assetID := slot.AssetID
		errorMessage := slot.ErrorMessage
		if existing, ok := existingByIndex[slot.Index]; ok {
			if assetID == "" {
				assetID = existing.AssetID
			}
			if imageURL == "" && existing.ImageURL != "" {
				imageURL = existing.ImageURL
			}
			if status != "completed" && existing.Status == "completed" && existing.ImageURL != "" && imageURL != "" {
				status = existing.Status
			}
			if errorMessage == "" {
				errorMessage = existing.ErrorMessage
			}
		}
		rows = append(rows, &mysql.FragmentGenerationImageSlotDB{
			ID:           id,
			TaskID:       slotTaskID,
			FragmentID:   slotFragmentID,
			Index:        slot.Index,
			Title:        slot.Title,
			Caption:      slot.Caption,
			Status:       status,
			ImageURL:     imageURL,
			AssetID:      assetID,
			ErrorMessage: errorMessage,
			MetadataJSON: "{}",
			UpdatedAt:    now,
		})
	}
	if len(rows) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "task_id"}, {Name: "slot_index"}},
			UpdateAll: true,
		}).
		Create(&rows).Error
}

func (r *FragmentGenerationRepository) fragmentGenerationImageSlotsByIndex(ctx context.Context, taskID string) (map[int]mysql.FragmentGenerationImageSlotDB, error) {
	out := map[int]mysql.FragmentGenerationImageSlotDB{}
	if taskID == "" {
		return out, nil
	}
	var rows []mysql.FragmentGenerationImageSlotDB
	if err := r.db.WithContext(ctx).
		Where("task_id = ?", taskID).
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("load existing image slots: %w", err)
	}
	for _, row := range rows {
		out[row.Index] = row
	}
	return out, nil
}

// ListImageSlots returns durable image slots for a task in display order.
func (r *FragmentGenerationRepository) ListImageSlots(ctx context.Context, taskID string) ([]domain.FragmentGenerationImageSlot, error) {
	var rows []mysql.FragmentGenerationImageSlotDB
	if err := r.db.WithContext(ctx).
		Where("task_id = ?", taskID).
		Order("slot_index ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	slots := make([]domain.FragmentGenerationImageSlot, 0, len(rows))
	for _, row := range rows {
		slots = append(slots, domain.FragmentGenerationImageSlot{
			ID:           row.ID,
			TaskID:       row.TaskID,
			FragmentID:   row.FragmentID,
			Index:        row.Index,
			Title:        row.Title,
			Caption:      row.Caption,
			Status:       row.Status,
			ImageURL:     row.ImageURL,
			AssetID:      row.AssetID,
			ErrorMessage: row.ErrorMessage,
		})
	}
	return slots, nil
}

func stableFragmentGenerationImageSlotID(taskID string, index int) string {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprintf("%s|%d", taskID, index))).String()
}

// UpdateError 更新错误信息
func (r *FragmentGenerationRepository) UpdateError(ctx context.Context, id string, status string, errorMsg string) error {
	now := time.Now().Unix()
	return r.db.WithContext(ctx).Model(&mysql.FragmentGenerationTaskDB{}).
		Where("id = ? AND status NOT IN ?", id, []string{
			string(common.TaskStatusCompleted),
			string(common.TaskStatusFailed),
			string(common.TaskStatusCancelled),
		}).
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
	return fragmentGenerationTasksFromRows(rows)
}
