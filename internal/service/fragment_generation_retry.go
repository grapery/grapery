package service

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

func fragmentRetryEligibility(task *domain.FragmentGenerationTask, lookup func(string) (*domain.AIGenerationRecord, error)) domain.FragmentGenerationRecovery {
	state := domain.FragmentGenerationRecovery{Reason: "not_failed"}
	if task == nil || task.Status != "failed" {
		return state
	}
	if fragmentSettlementRecoverable(task) {
		state.Reason = "settle_required"
		return state
	}
	if strings.HasPrefix(task.ErrorMessage, "billing_unsettled:") {
		state.Reason = "billing_reconciliation_required"
		state.RequiresReconciliation = true
		return state
	}
	if strings.TrimSpace(task.CheckpointJSON) == "" || fragmentCheckpointMissingForExecutedTask(task) {
		state.Reason = "checkpoint_missing"
		state.RequiresReconciliation = true
		return state
	}
	cp, err := newFragmentTaskCheckpoint(task.CheckpointJSON, nil)
	if err != nil {
		state.Reason = "checkpoint_invalid"
		state.RequiresReconciliation = true
		return state
	}
	for key, stage := range cp.Stages {
		if stage.State == "completed" {
			continue
		}
		if stage.State != "inflight" {
			state.Reason = "checkpoint_invalid"
			state.RequiresReconciliation = true
			return state
		}
		if strings.HasPrefix(key, "image_output:") {
			if lookup == nil {
				state.Reason = "image_status_unavailable"
				state.RequiresReconciliation = true
				return state
			}
			op := strings.TrimPrefix(key, "image_output:")
			record, err := lookup(uuid.NewSHA1(uuid.NameSpaceOID, []byte(task.UserID+":image:"+op)).String())
			if errors.Is(err, domain.ErrNotFound) {
				continue
			} // no persisted dispatch => no provider call
			if err != nil || record == nil || record.UserID != task.UserID || record.RelatedEntityID != task.ID {
				state.Reason = "image_status_unavailable"
				state.RequiresReconciliation = true
				return state
			}
			if record.Status != "failed" && !hasStoredImageOutput(record) {
				state.Reason = "image_outcome_unknown"
				state.RequiresReconciliation = true
				return state
			}
			continue
		}
		if key == "baseline" || key == "references" || key == "images" || key == "repair" || key == "prepared_commit" || strings.HasPrefix(key, "image_intent:") || strings.HasPrefix(key, "page_render:") {
			continue
		}
		state.Reason = "provider_outcome_unknown"
		state.RequiresReconciliation = true
		return state
	}
	state.CanRetry, state.Reason = true, "resume_saved_task"
	return state
}

func (s *FragmentGenerationService) RecoverySnapshot(ctx context.Context, task *domain.FragmentGenerationTask) domain.FragmentGenerationRecovery {
	return fragmentRetryEligibility(task, func(id string) (*domain.AIGenerationRecord, error) {
		if s.repo == nil {
			return nil, errors.New("image operation repository unavailable")
		}
		return s.repo.GetAIGenerationRecord(ctx, id)
	})
}

// RetryGeneration resumes the original operation identities and completed
// checkpoints. It never interprets a new append request as a failed-page retry.
func (s *FragmentGenerationService) RetryGeneration(ctx context.Context, taskID, userID string) (*domain.FragmentGenerationTask, string, error) {
	task, err := s.GetTask(ctx, taskID)
	if err != nil || task == nil {
		return nil, "", domain.NewFragmentGenerationError(domain.FragmentGenerationErrorNotFound, "task not found", err)
	}
	if task.UserID != userID {
		return nil, "", domain.NewFragmentGenerationError(domain.FragmentGenerationErrorForbidden, "task is not owned by this user", nil)
	}
	draftID := task.Request.TargetDraftFragmentID
	if task.Result != nil && task.Result.DraftFragmentID != "" {
		draftID = task.Result.DraftFragmentID
	}
	if draftID == "" {
		if draft, e := s.fragmentRepo.GetBySource(ctx, string(domain.FragmentSourceAIGeneration), task.ID); e == nil && draft != nil {
			draftID = draft.ID
		}
	}
	if draftID == "" {
		return nil, "", domain.NewFragmentGenerationError(domain.FragmentGenerationErrorConflict, "the original draft is unavailable", nil)
	}
	if !fragmentGenerationTaskActive(task.Status) {
		state := s.RecoverySnapshot(ctx, task)
		if !state.CanRetry {
			return nil, "", domain.NewFragmentGenerationError(domain.FragmentGenerationErrorConflict, "cannot retry this task: "+state.Reason, nil)
		}
	}
	if err = s.fragmentGenRepo.QueueRetry(ctx, taskID, draftID, userID, task.CheckpointJSON); err != nil {
		return nil, "", err
	}
	s.scheduleFragmentTask(taskID)
	updated, err := s.GetTask(ctx, taskID)
	return updated, draftID, err
}
