package mysql

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

func generationStepAuditToModel(d *domain.GenerationStepAuditRecord) *GenerationStepAuditRecordDB {
	if d == nil {
		return nil
	}
	id := d.ID
	if id == "" {
		id = "audit_" + uuid.New().String()
	}
	m := &GenerationStepAuditRecordDB{
		ID:           id,
		Sequence:     d.Sequence,
		RunID:        d.RunID,
		TaskID:       d.TaskID,
		BusinessType: d.BusinessType,
		BusinessID:   d.BusinessID,
		AgentVersion: d.AgentVersion,
		StepName:     d.StepName,
		Attempt:      d.Attempt,
		Status:       d.Status,
		Provider:     d.Provider,
		Model:        d.Model,
		Prompt:       d.Prompt,
		RawOutput:    d.RawOutput,
		ErrorCode:    d.ErrorCode,
		ErrorMessage: d.ErrorMessage,
		InputTokens:  d.InputTokens,
		OutputTokens: d.OutputTokens,
		TotalTokens:  d.TotalTokens,
		DurationMs:   d.DurationMs,
		UserID:       d.UserID,
	}
	if len(d.InputRefs) > 0 {
		b, _ := json.Marshal(d.InputRefs)
		m.InputRefsJSON = string(b)
	}
	if len(d.ParsedOutput) > 0 {
		b, _ := json.Marshal(d.ParsedOutput)
		m.ParsedOutputJSON = string(b)
	}
	if len(d.Metadata) > 0 {
		b, _ := json.Marshal(d.Metadata)
		m.MetadataJSON = string(b)
	}
	if d.StartedAt > 0 {
		t := time.UnixMilli(d.StartedAt)
		m.StartedAt = &t
	}
	if d.EndedAt > 0 {
		t := time.UnixMilli(d.EndedAt)
		m.EndedAt = &t
	}
	return m
}

func modelToGenerationStepAudit(m *GenerationStepAuditRecordDB) *domain.GenerationStepAuditRecord {
	if m == nil {
		return nil
	}
	out := &domain.GenerationStepAuditRecord{
		ID:           m.ID,
		Sequence:     m.Sequence,
		RunID:        m.RunID,
		TaskID:       m.TaskID,
		BusinessType: m.BusinessType,
		BusinessID:   m.BusinessID,
		AgentVersion: m.AgentVersion,
		StepName:     m.StepName,
		Attempt:      m.Attempt,
		Status:       m.Status,
		Provider:     m.Provider,
		Model:        m.Model,
		Prompt:       m.Prompt,
		RawOutput:    m.RawOutput,
		ErrorCode:    m.ErrorCode,
		ErrorMessage: m.ErrorMessage,
		InputTokens:  m.InputTokens,
		OutputTokens: m.OutputTokens,
		TotalTokens:  m.TotalTokens,
		DurationMs:   m.DurationMs,
		CreatedAt:    m.CreatedAt.UnixMilli(),
		UserID:       m.UserID,
	}
	if m.InputRefsJSON != "" {
		_ = json.Unmarshal([]byte(m.InputRefsJSON), &out.InputRefs)
	}
	if m.ParsedOutputJSON != "" {
		_ = json.Unmarshal([]byte(m.ParsedOutputJSON), &out.ParsedOutput)
	}
	if m.MetadataJSON != "" {
		_ = json.Unmarshal([]byte(m.MetadataJSON), &out.Metadata)
	}
	if m.StartedAt != nil {
		out.StartedAt = m.StartedAt.UnixMilli()
	}
	if m.EndedAt != nil {
		out.EndedAt = m.EndedAt.UnixMilli()
	}
	return out
}

func (r *Repository) CreateGenerationStepAuditRecord(ctx context.Context, record *domain.GenerationStepAuditRecord) error {
	return r.db.WithContext(ctx).Create(generationStepAuditToModel(record)).Error
}

func (r *Repository) CreateGenerationStepAuditRecords(ctx context.Context, records []*domain.GenerationStepAuditRecord) error {
	if len(records) == 0 {
		return nil
	}
	models := make([]GenerationStepAuditRecordDB, 0, len(records))
	for _, rec := range records {
		if rec == nil {
			continue
		}
		models = append(models, *generationStepAuditToModel(rec))
	}
	if len(models) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&models).Error
}

func (r *Repository) ListGenerationStepAuditsByRunID(ctx context.Context, runID string, limit int) ([]*domain.GenerationStepAuditRecord, error) {
	if limit <= 0 {
		limit = 200
	}
	if limit > 500 {
		limit = 500
	}
	var models []GenerationStepAuditRecordDB
	if err := r.db.WithContext(ctx).Where("run_id = ?", runID).Order("sequence ASC, created_at ASC").Limit(limit).Find(&models).Error; err != nil {
		return nil, err
	}
	out := make([]*domain.GenerationStepAuditRecord, len(models))
	for i := range models {
		out[i] = modelToGenerationStepAudit(&models[i])
	}
	return out, nil
}

func (r *Repository) ListGenerationStepAuditsByTaskID(ctx context.Context, taskID string, limit int) ([]*domain.GenerationStepAuditRecord, error) {
	if limit <= 0 {
		limit = 200
	}
	if limit > 500 {
		limit = 500
	}
	var models []GenerationStepAuditRecordDB
	if err := r.db.WithContext(ctx).Where("task_id = ?", taskID).Order("sequence ASC, created_at ASC").Limit(limit).Find(&models).Error; err != nil {
		return nil, err
	}
	out := make([]*domain.GenerationStepAuditRecord, len(models))
	for i := range models {
		out[i] = modelToGenerationStepAudit(&models[i])
	}
	return out, nil
}

func (r *Repository) ListGenerationStepAuditsByUserID(ctx context.Context, userID string, runID, taskID string, limit int) ([]*domain.GenerationStepAuditRecord, error) {
	if limit <= 0 {
		limit = 200
	}
	if limit > 500 {
		limit = 500
	}
	q := r.db.WithContext(ctx).Where("user_id = ?", userID)
	if runID != "" {
		q = q.Where("run_id = ?", runID)
	}
	if taskID != "" {
		q = q.Where("task_id = ?", taskID)
	}
	var models []GenerationStepAuditRecordDB
	if err := q.Order("sequence ASC, created_at ASC").Limit(limit).Find(&models).Error; err != nil {
		return nil, err
	}
	out := make([]*domain.GenerationStepAuditRecord, len(models))
	for i := range models {
		out[i] = modelToGenerationStepAudit(&models[i])
	}
	return out, nil
}
