package mysql

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GenerationRuntimeRepository struct{ db *gorm.DB }

func NewGenerationRuntimeRepository(db *gorm.DB) *GenerationRuntimeRepository {
	return &GenerationRuntimeRepository{db: db}
}

func (r *GenerationRuntimeRepository) SaveGenerationExecution(ctx context.Context, run *domain.GenerationExecution, eventType string) (*domain.GenerationEvent, error) {
	if run == nil || strings.TrimSpace(run.ID) == "" {
		return nil, fmt.Errorf("generation execution id is required")
	}
	if existing, err := r.findGenerationExecutionByIdempotency(ctx, run); err == nil && existing != nil && existing.ID != run.ID {
		*run = *existing
		return nil, nil
	}
	var event *domain.GenerationEvent
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current GenerationExecutionDB
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", run.ID).Take(&current).Error
		switch {
		case err == nil:
			if isTerminalGenerationStatus(current.Status) {
				existing, decodeErr := generationExecutionFromDB(&current)
				if decodeErr != nil {
					return decodeErr
				}
				*run = *existing
				return nil
			}
			run.Sequence = current.Sequence + 1
			if run.CreatedAt.IsZero() {
				run.CreatedAt = current.CreatedAt
			}
		case errors.Is(err, gorm.ErrRecordNotFound):
			run.Sequence = 1
		default:
			return fmt.Errorf("lock generation execution: %w", err)
		}
		run.UpdatedAt = time.Now().UTC()
		if run.CreatedAt.IsZero() {
			run.CreatedAt = run.UpdatedAt
		}
		row, err := generationExecutionToDB(run)
		if err != nil {
			return err
		}
		if err := tx.Save(row).Error; err != nil {
			return fmt.Errorf("save generation execution: %w", err)
		}
		payload, err := json.Marshal(run)
		if err != nil {
			return fmt.Errorf("marshal generation event: %w", err)
		}
		if eventType == "" {
			eventType = "snapshot.updated"
		}
		eventRow := GenerationEventDB{
			ID:          "evt_" + uuid.NewString(),
			RunID:       run.ID,
			Sequence:    run.Sequence,
			Type:        eventType,
			PayloadJSON: string(payload),
			CreatedAt:   run.UpdatedAt,
		}
		if err := tx.Create(&eventRow).Error; err != nil {
			return fmt.Errorf("create generation event: %w", err)
		}
		event = generationEventFromDB(&eventRow)
		return nil
	})
	if err != nil {
		// The unique index is the final guard for concurrent duplicate requests.
		// If another transaction won, return its canonical execution instead of
		// surfacing a retryable duplicate-key failure to the agent.
		if existing, lookupErr := r.findGenerationExecutionByIdempotency(ctx, run); lookupErr == nil && existing != nil && existing.ID != run.ID {
			*run = *existing
			return nil, nil
		}
	}
	return event, err
}

func (r *GenerationRuntimeRepository) findGenerationExecutionByIdempotency(ctx context.Context, run *domain.GenerationExecution) (*domain.GenerationExecution, error) {
	userID, requestID := strings.TrimSpace(run.UserID), strings.TrimSpace(run.ClientRequestID)
	if userID == "" || requestID == "" || strings.TrimSpace(run.Kind) == "" {
		return nil, gorm.ErrRecordNotFound
	}
	var row GenerationExecutionDB
	err := r.db.WithContext(ctx).Where("idempotency_user_id = ? AND kind = ? AND client_request_id = ?", userID, run.Kind, requestID).Take(&row).Error
	if err != nil {
		return nil, err
	}
	return generationExecutionFromDB(&row)
}

func (r *GenerationRuntimeRepository) GetGenerationExecution(ctx context.Context, id string) (*domain.GenerationExecution, error) {
	var row GenerationExecutionDB
	if err := r.db.WithContext(ctx).Where("id = ?", strings.TrimSpace(id)).Take(&row).Error; err != nil {
		return nil, err
	}
	return generationExecutionFromDB(&row)
}

func (r *GenerationRuntimeRepository) FindLatestGenerationExecution(ctx context.Context, userID, kind, contentID string) (*domain.GenerationExecution, error) {
	userID, kind, contentID = strings.TrimSpace(userID), strings.TrimSpace(kind), strings.TrimSpace(contentID)
	if userID == "" || kind == "" || contentID == "" {
		return nil, gorm.ErrRecordNotFound
	}
	var row GenerationExecutionDB
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND kind = ? AND primary_content_id = ?", userID, kind, contentID).
		Order("created_at DESC").Take(&row).Error
	if err == nil {
		return generationExecutionFromDB(&row)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// Compatibility for executions written before primary_content_id existed.
	var legacy []GenerationExecutionDB
	if err := r.db.WithContext(ctx).Where("user_id = ? AND kind = ?", userID, kind).
		Order("created_at DESC").Limit(500).Find(&legacy).Error; err != nil {
		return nil, err
	}
	for i := range legacy {
		run, decodeErr := generationExecutionFromDB(&legacy[i])
		if decodeErr != nil {
			return nil, decodeErr
		}
		for _, value := range run.ContentIDs {
			if fmt.Sprint(value) == contentID {
				return run, nil
			}
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *GenerationRuntimeRepository) ListGenerationExecutions(ctx context.Context, kind string, limit int) ([]*domain.GenerationExecution, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	q := r.db.WithContext(ctx).Order("created_at DESC").Limit(limit)
	if kind = strings.TrimSpace(kind); kind != "" {
		q = q.Where("kind = ?", kind)
	}
	var rows []GenerationExecutionDB
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domain.GenerationExecution, 0, len(rows))
	for i := range rows {
		run, err := generationExecutionFromDB(&rows[i])
		if err != nil {
			return nil, err
		}
		out = append(out, run)
	}
	return out, nil
}

func (r *GenerationRuntimeRepository) ListGenerationEvents(ctx context.Context, runID string, afterSequence int64, limit int) ([]*domain.GenerationEvent, error) {
	if limit <= 0 || limit > 2000 {
		limit = 500
	}
	var rows []GenerationEventDB
	err := r.db.WithContext(ctx).Where("run_id = ? AND sequence > ?", strings.TrimSpace(runID), afterSequence).
		Order("sequence ASC").Limit(limit).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]*domain.GenerationEvent, 0, len(rows))
	for i := range rows {
		out = append(out, generationEventFromDB(&rows[i]))
	}
	return out, nil
}

func (r *GenerationRuntimeRepository) MarkGenerationEventPublished(ctx context.Context, eventID string, publishedAt time.Time) error {
	return r.db.WithContext(ctx).Model(&GenerationEventDB{}).Where("id = ?", eventID).Update("published_at", publishedAt).Error
}

func (r *GenerationRuntimeRepository) ListUnpublishedGenerationEvents(ctx context.Context, limit int) ([]*domain.GenerationEvent, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	var rows []GenerationEventDB
	if err := r.db.WithContext(ctx).Where("published_at IS NULL").Order("created_at ASC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domain.GenerationEvent, 0, len(rows))
	for i := range rows {
		out = append(out, generationEventFromDB(&rows[i]))
	}
	return out, nil
}

func (r *GenerationRuntimeRepository) SaveGenerationCheckpoint(ctx context.Context, checkpoint *domain.GenerationCheckpoint) error {
	if checkpoint == nil || strings.TrimSpace(checkpoint.ID) == "" {
		return fmt.Errorf("checkpoint id is required")
	}
	return r.db.WithContext(ctx).Save(&GenerationCheckpointDB{
		ID: checkpoint.ID, State: checkpoint.State, ExpiresAt: checkpoint.ExpiresAt,
	}).Error
}

func (r *GenerationRuntimeRepository) GetGenerationCheckpoint(ctx context.Context, id string) (*domain.GenerationCheckpoint, error) {
	var row GenerationCheckpointDB
	err := r.db.WithContext(ctx).Where("id = ? AND expires_at > ?", strings.TrimSpace(id), time.Now().UTC()).Take(&row).Error
	if err != nil {
		return nil, err
	}
	return &domain.GenerationCheckpoint{ID: row.ID, State: row.State, UpdatedAt: row.UpdatedAt, ExpiresAt: row.ExpiresAt}, nil
}

func generationExecutionToDB(run *domain.GenerationExecution) (*GenerationExecutionDB, error) {
	input, err := json.Marshal(run.Input)
	if err != nil {
		return nil, fmt.Errorf("marshal input: %w", err)
	}
	output, err := json.Marshal(run.Output)
	if err != nil {
		return nil, fmt.Errorf("marshal output: %w", err)
	}
	contentIDs, err := json.Marshal(run.ContentIDs)
	if err != nil {
		return nil, fmt.Errorf("marshal content ids: %w", err)
	}
	toolCalls, err := json.Marshal(run.ToolCalls)
	if err != nil {
		return nil, fmt.Errorf("marshal tool calls: %w", err)
	}
	var requestID, idempotencyUserID *string
	if strings.TrimSpace(run.ClientRequestID) != "" && strings.TrimSpace(run.UserID) != "" {
		rid, uid := strings.TrimSpace(run.ClientRequestID), strings.TrimSpace(run.UserID)
		requestID, idempotencyUserID = &rid, &uid
	}
	return &GenerationExecutionDB{
		ID: run.ID, UserID: run.UserID, Kind: run.Kind, Status: run.Status, Phase: run.Phase,
		Progress: run.Progress, AgentVersion: run.AgentVersion, UserIntent: run.UserIntent,
		WorkflowReleaseID: run.WorkflowReleaseID, WorkflowKey: run.WorkflowKey,
		WorkflowVersion: run.WorkflowVersion, WorkflowChecksum: run.WorkflowChecksum,
		PromptBundleJSON:    string(mustMarshalGenerationJSON(run.PromptBundle)),
		PromptSnapshotsJSON: string(mustMarshalGenerationJSON(run.PromptSnapshots)),
		InputJSON:           string(input), OutputJSON: string(output), ParentRunID: run.ParentRunID,
		BranchIndex: run.BranchIndex, Strategy: run.Strategy, ContentIDsJSON: string(contentIDs),
		PrimaryContentID: generationPrimaryContentID(run.ContentIDs),
		ToolCallsJSON:    string(toolCalls), Error: run.Error, ErrorCode: run.ErrorCode,
		TokensUsed: run.TokensUsed, ModelProvider: run.ModelProvider, ModelName: run.ModelName,
		CheckpointID: run.CheckpointID, ClientRequestID: requestID, IdempotencyUserID: idempotencyUserID,
		SourceTaskID: run.SourceTaskID, Sequence: run.Sequence, CreatedAt: run.CreatedAt,
		UpdatedAt: run.UpdatedAt, CompletedAt: run.CompletedAt,
	}, nil
}

func generationPrimaryContentID(contentIDs map[string]any) string {
	for _, key := range []string{"fragmentId", "storyboardId", "storyId", "branchId", "characterId"} {
		if value := strings.TrimSpace(fmt.Sprint(contentIDs[key])); value != "" && value != "<nil>" {
			return value
		}
	}
	return ""
}

func generationExecutionFromDB(row *GenerationExecutionDB) (*domain.GenerationExecution, error) {
	run := &domain.GenerationExecution{
		ID: row.ID, UserID: row.UserID, Kind: row.Kind, Status: row.Status, Phase: row.Phase,
		Progress: row.Progress, AgentVersion: row.AgentVersion, UserIntent: row.UserIntent,
		WorkflowReleaseID: row.WorkflowReleaseID, WorkflowKey: row.WorkflowKey,
		WorkflowVersion: row.WorkflowVersion, WorkflowChecksum: row.WorkflowChecksum,
		ParentRunID: row.ParentRunID, BranchIndex: row.BranchIndex, Strategy: row.Strategy,
		Error: row.Error, ErrorCode: row.ErrorCode, TokensUsed: row.TokensUsed,
		ModelProvider: row.ModelProvider, ModelName: row.ModelName, CheckpointID: row.CheckpointID,
		SourceTaskID: row.SourceTaskID, Sequence: row.Sequence, CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt, CompletedAt: row.CompletedAt,
	}
	if row.ClientRequestID != nil {
		run.ClientRequestID = *row.ClientRequestID
	}
	if err := decodeGenerationJSON(row.ID, "input", row.InputJSON, &run.Input); err != nil {
		return nil, err
	}
	if err := decodeGenerationJSON(row.ID, "output", row.OutputJSON, &run.Output); err != nil {
		return nil, err
	}
	if err := decodeGenerationJSON(row.ID, "contentIds", row.ContentIDsJSON, &run.ContentIDs); err != nil {
		return nil, err
	}
	if err := decodeGenerationJSON(row.ID, "toolCalls", row.ToolCallsJSON, &run.ToolCalls); err != nil {
		return nil, err
	}
	if err := decodeGenerationJSON(row.ID, "promptBundle", row.PromptBundleJSON, &run.PromptBundle); err != nil {
		return nil, err
	}
	if err := decodeGenerationJSON(row.ID, "promptSnapshots", row.PromptSnapshotsJSON, &run.PromptSnapshots); err != nil {
		return nil, err
	}
	return run, nil
}

func mustMarshalGenerationJSON(value any) []byte {
	b, _ := json.Marshal(value)
	return b
}

func decodeGenerationJSON(id, field, raw string, target any) error {
	if strings.TrimSpace(raw) == "" || raw == "null" {
		return nil
	}
	if err := json.Unmarshal([]byte(raw), target); err != nil {
		return fmt.Errorf("decode generation execution %s %s: %w", id, field, err)
	}
	return nil
}

func generationEventFromDB(row *GenerationEventDB) *domain.GenerationEvent {
	event := &domain.GenerationEvent{ID: row.ID, RunID: row.RunID, Sequence: row.Sequence, Type: row.Type, CreatedAt: row.CreatedAt}
	_ = json.Unmarshal([]byte(row.PayloadJSON), &event.Payload)
	return event
}

func isTerminalGenerationStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "succeeded", "failed", "cancelled", "canceled":
		return true
	default:
		return false
	}
}

var _ domain.GenerationRuntimeRepository = (*GenerationRuntimeRepository)(nil)
