package service

import (
	"context"
	"testing"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

func TestGenerationAuditService_RecordValidation(t *testing.T) {
	svc := NewGenerationAuditService(nil, nil)
	err := svc.Record(context.Background(), &domain.GenerationStepAuditRecord{RunID: "run_1", StepName: "plan"})
	if err == nil {
		t.Fatal("expected error without repo")
	}
}

func TestGenerationAuditService_RecordBatchEmpty(t *testing.T) {
	svc := NewGenerationAuditService(nil, nil)
	if err := svc.RecordBatch(context.Background(), nil); err == nil {
		t.Fatal("expected error without repo")
	}
}
