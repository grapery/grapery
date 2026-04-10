package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/repository/mysql"
	"gorm.io/gorm"
)

type FragmentPanelGenerationRepository struct {
	db *gorm.DB
}

func NewFragmentPanelGenerationRepository(db *gorm.DB) *FragmentPanelGenerationRepository {
	return &FragmentPanelGenerationRepository{db: db}
}

// DB exposes the underlying GORM handle for cross-repository transactions.
func (r *FragmentPanelGenerationRepository) DB() *gorm.DB {
	return r.db
}

func panelTaskToDB(t *domain.FragmentPanelGenerationTask) (*mysql.FragmentPanelGenerationTaskDB, error) {
	reqJSON, err := json.Marshal(t.Request)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	row := &mysql.FragmentPanelGenerationTaskDB{
		ID:              t.ID,
		UserID:          t.UserID,
		Status:          t.Status,
		RequestJSON:     string(reqJSON),
		Progress:        t.Progress,
		CurrentStep:     t.CurrentStep,
		ErrorMessage:    t.ErrorMessage,
		DraftFragmentID: t.DraftFragmentID,
		CreatedAt:       t.CreatedAt,
		StartedAt:       t.StartedAt,
		CompletedAt:     t.CompletedAt,
		UpdatedAt:       t.UpdatedAt,
	}
	if len(t.Plan) > 0 {
		b, err := json.Marshal(t.Plan)
		if err != nil {
			return nil, err
		}
		row.PlanJSON = string(b)
	}
	if t.Result != nil {
		b, err := json.Marshal(t.Result)
		if err != nil {
			return nil, err
		}
		row.ResultJSON = string(b)
	}
	if t.Metrics != nil {
		b, err := json.Marshal(t.Metrics)
		if err != nil {
			return nil, err
		}
		row.MetricsJSON = string(b)
	}
	return row, nil
}

func panelTaskFromDB(row *mysql.FragmentPanelGenerationTaskDB) (*domain.FragmentPanelGenerationTask, error) {
	t := &domain.FragmentPanelGenerationTask{
		ID:              row.ID,
		UserID:          row.UserID,
		Status:          row.Status,
		Progress:        row.Progress,
		CurrentStep:     row.CurrentStep,
		ErrorMessage:    row.ErrorMessage,
		DraftFragmentID: row.DraftFragmentID,
		CreatedAt:       row.CreatedAt,
		StartedAt:       row.StartedAt,
		CompletedAt:     row.CompletedAt,
		UpdatedAt:       row.UpdatedAt,
	}
	if row.RequestJSON != "" {
		if err := json.Unmarshal([]byte(row.RequestJSON), &t.Request); err != nil {
			return nil, fmt.Errorf("unmarshal request: %w", err)
		}
	}
	if row.PlanJSON != "" {
		if err := json.Unmarshal([]byte(row.PlanJSON), &t.Plan); err != nil {
			return nil, fmt.Errorf("unmarshal plan: %w", err)
		}
	}
	if row.ResultJSON != "" {
		var res domain.FragmentPanelResultData
		if err := json.Unmarshal([]byte(row.ResultJSON), &res); err != nil {
			return nil, fmt.Errorf("unmarshal result: %w", err)
		}
		t.Result = &res
	}
	if row.MetricsJSON != "" {
		var m domain.FragmentPanelMetricsData
		if err := json.Unmarshal([]byte(row.MetricsJSON), &m); err != nil {
			return nil, fmt.Errorf("unmarshal metrics: %w", err)
		}
		t.Metrics = &m
	}
	return t, nil
}

func (r *FragmentPanelGenerationRepository) Create(ctx context.Context, task *domain.FragmentPanelGenerationTask) error {
	return r.CreateWithTx(ctx, r.db, task)
}

// CreateWithTx persists a panel generation task using the given transaction.
func (r *FragmentPanelGenerationRepository) CreateWithTx(ctx context.Context, tx *gorm.DB, task *domain.FragmentPanelGenerationTask) error {
	if tx == nil {
		return fmt.Errorf("tx is nil")
	}
	row, err := panelTaskToDB(task)
	if err != nil {
		return err
	}
	return tx.WithContext(ctx).Create(row).Error
}

func (r *FragmentPanelGenerationRepository) GetByID(ctx context.Context, id string) (*domain.FragmentPanelGenerationTask, error) {
	var row mysql.FragmentPanelGenerationTaskDB
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return nil, err
	}
	return panelTaskFromDB(&row)
}

func (r *FragmentPanelGenerationRepository) Save(ctx context.Context, task *domain.FragmentPanelGenerationTask) error {
	row, err := panelTaskToDB(task)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Save(row).Error
}

func (r *FragmentPanelGenerationRepository) UpdateStatus(ctx context.Context, id string, status string, progress int, currentStep string) error {
	updates := map[string]interface{}{
		"status":       status,
		"progress":     progress,
		"current_step": currentStep,
		"updated_at":   time.Now().Unix(),
	}
	if status == "completed" || status == "failed" {
		now := time.Now().Unix()
		updates["completed_at"] = now
	}
	return r.db.WithContext(ctx).Model(&mysql.FragmentPanelGenerationTaskDB{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// UpdateError marks the task failed and sets error_message + completed_at.
func (r *FragmentPanelGenerationRepository) UpdateError(ctx context.Context, id string, errorMsg string) error {
	now := time.Now().Unix()
	return r.db.WithContext(ctx).Model(&mysql.FragmentPanelGenerationTaskDB{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":         "failed",
			"error_message":  errorMsg,
			"completed_at":   now,
			"updated_at":     now,
		}).Error
}

// TryAcquireResumeProcessing atomically moves a task from failed or pending to processing.
// Clears error_message and completed_at. Returns true if a row was updated (caller won the race).
func (r *FragmentPanelGenerationRepository) TryAcquireResumeProcessing(ctx context.Context, taskID, userID string, progress int, currentStep string) (bool, error) {
	now := time.Now().Unix()
	res := r.db.WithContext(ctx).Model(&mysql.FragmentPanelGenerationTaskDB{}).
		Where("id = ? AND user_id = ? AND status IN ?", taskID, userID, []string{"failed", "pending"}).
		Updates(map[string]interface{}{
			"status":         "processing",
			"error_message":  "",
			"completed_at":   nil,
			"progress":       progress,
			"current_step":   currentStep,
			"updated_at":     now,
		})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// RevertProcessingToFailed sets status back to failed when resume setup fails (e.g. draft reset).
// Only affects rows still in processing for this user.
func (r *FragmentPanelGenerationRepository) RevertProcessingToFailed(ctx context.Context, taskID, userID, errMsg string) error {
	now := time.Now().Unix()
	return r.db.WithContext(ctx).Model(&mysql.FragmentPanelGenerationTaskDB{}).
		Where("id = ? AND user_id = ? AND status = ?", taskID, userID, "processing").
		Updates(map[string]interface{}{
			"status":         "failed",
			"error_message":  errMsg,
			"completed_at":   now,
			"updated_at":     now,
		}).Error
}
