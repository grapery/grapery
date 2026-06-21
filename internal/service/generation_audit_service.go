package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

// GenerationAuditService 持久化 agent 上报的 GenerationStepAudit。
type GenerationAuditService struct {
	repo   domain.Repository
	logger *zap.Logger
}

func NewGenerationAuditService(repo domain.Repository, logger *zap.Logger) *GenerationAuditService {
	return &GenerationAuditService{repo: repo, logger: logger}
}

func (s *GenerationAuditService) Record(ctx context.Context, rec *domain.GenerationStepAuditRecord) error {
	if s == nil || s.repo == nil {
		return fmt.Errorf("generation audit service unavailable")
	}
	if rec == nil {
		return fmt.Errorf("audit record is required")
	}
	if strings.TrimSpace(rec.RunID) == "" {
		return fmt.Errorf("runId is required")
	}
	if strings.TrimSpace(rec.StepName) == "" {
		return fmt.Errorf("stepName is required")
	}
	if rec.Status == "" {
		rec.Status = "started"
	}
	if rec.Attempt <= 0 {
		rec.Attempt = 1
	}
	if rec.CreatedAt == 0 {
		rec.CreatedAt = time.Now().UnixMilli()
	}
	return s.repo.CreateGenerationStepAuditRecord(ctx, rec)
}

func (s *GenerationAuditService) RecordBatch(ctx context.Context, records []*domain.GenerationStepAuditRecord) error {
	if s == nil || s.repo == nil {
		return fmt.Errorf("generation audit service unavailable")
	}
	for _, rec := range records {
		if rec == nil {
			continue
		}
		if rec.Status == "" {
			rec.Status = "started"
		}
		if rec.Attempt <= 0 {
			rec.Attempt = 1
		}
		if rec.CreatedAt == 0 {
			rec.CreatedAt = time.Now().UnixMilli()
		}
	}
	return s.repo.CreateGenerationStepAuditRecords(ctx, records)
}

func (s *GenerationAuditService) ListByRunID(ctx context.Context, runID string, limit int) ([]*domain.GenerationStepAuditRecord, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("generation audit service unavailable")
	}
	return s.repo.ListGenerationStepAuditsByRunID(ctx, strings.TrimSpace(runID), limit)
}

func (s *GenerationAuditService) ListByTaskID(ctx context.Context, taskID string, limit int) ([]*domain.GenerationStepAuditRecord, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("generation audit service unavailable")
	}
	return s.repo.ListGenerationStepAuditsByTaskID(ctx, strings.TrimSpace(taskID), limit)
}

func (s *GenerationAuditService) ListForUser(ctx context.Context, userID, runID, taskID string, limit int) ([]*domain.GenerationStepAuditRecord, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("generation audit service unavailable")
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, fmt.Errorf("userId is required")
	}
	return s.repo.ListGenerationStepAuditsByUserID(ctx, userID, strings.TrimSpace(runID), strings.TrimSpace(taskID), limit)
}
