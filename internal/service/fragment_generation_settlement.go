package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

func fragmentSettlementRecoverable(task *domain.FragmentGenerationTask) bool {
	if task == nil || task.Result == nil || task.Result.GenerationTrace == nil {
		return false
	}
	trace := task.Result.GenerationTrace
	if trace.BillingStatus != "pending" || trace.TextBillingTokens < 0 || task.Result.DraftFragmentID == "" {
		return false
	}
	if task.Status != "processing" && !(task.Status == "failed" && strings.HasPrefix(task.ErrorMessage, "billing_unsettled:")) {
		return false
	}
	return task.Result.ComicDocument != nil && len(task.Result.ComicDocument.Pages) > 0 && fragmentGenerationSlotsCompleted(task.Result.ImageSlots, task.Result.ExpectedImageCount)
}

func (s *FragmentGenerationService) finalizeFragmentTextSettlement(ctx context.Context, task *domain.FragmentGenerationTask, result *domain.FragmentGenerationResult) error {
	trace := result.GenerationTrace
	if trace == nil || trace.BillingStatus != "pending" || result.DraftFragmentID == "" {
		return fmt.Errorf("missing exact pending settlement source")
	}
	pendingMetadata := marshalFragmentGenerationMetadata(trace)
	if err := s.settleFragmentGenerationTextTokens(ctx, task.UserID, task.ID, trace.TextBillingTokens); err != nil {
		return err
	}
	trace.BillingStatus = "settled"
	if err := s.fragmentGenRepo.FinalizeFragmentSettlement(ctx, task.ID, result.DraftFragmentID, task.UserID, pendingMetadata, marshalFragmentGenerationMetadata(trace), result); err != nil {
		// Keep the recovery source pending in memory too; the ledger operation ID
		// makes the next settlement call an idempotent replay of the same debit.
		trace.BillingStatus = "pending"
		return err
	}
	return nil
}

// SettleGeneration retries only a persisted exact debit and atomic completion.
// It never calls a model or restarts generation. Owner/current-task checks also
// prevent an older finished task from overwriting a more recent draft edit.
func (s *FragmentGenerationService) SettleGeneration(ctx context.Context, taskID, userID string) (*domain.FragmentGenerationTask, error) {
	task, err := s.GetTask(ctx, taskID)
	if err != nil || task == nil {
		return nil, domain.NewFragmentGenerationError(domain.FragmentGenerationErrorNotFound, "task not found", err)
	}
	if task.UserID != userID {
		return nil, domain.NewFragmentGenerationError(domain.FragmentGenerationErrorForbidden, "task is not owned by this user", nil)
	}
	if task.Status == "completed" && task.Result != nil && task.Result.GenerationTrace != nil && task.Result.GenerationTrace.BillingStatus == "settled" {
		return task, nil
	}
	if !fragmentSettlementRecoverable(task) {
		return nil, domain.NewFragmentGenerationError(domain.FragmentGenerationErrorConflict, "this task has no complete pending settlement to recover", nil)
	}
	unlock := s.lockFragmentDraftGeneration(task.Result.DraftFragmentID)
	defer unlock()
	draft, err := s.fragmentRepo.GetByID(ctx, task.Result.DraftFragmentID)
	if err != nil || draft == nil || draft.UserID != userID || !draft.IsDraft || draft.GenerationTaskID != taskID {
		return nil, domain.NewFragmentGenerationError(domain.FragmentGenerationErrorConflict, "the draft changed; reopen its latest task", err)
	}
	if draft.GenerationMetadata != marshalFragmentGenerationMetadata(task.Result.GenerationTrace) {
		return nil, domain.NewFragmentGenerationError(domain.FragmentGenerationErrorConflict, "draft settlement source changed", nil)
	}
	if err = s.finalizeFragmentTextSettlement(ctx, task, task.Result); err != nil {
		return nil, err
	}
	return s.GetTask(ctx, taskID)
}

// Reference-asset and panel renders settle their own image ledger operations;
// only text planning and vision-audit usage belongs in the text debit.
func fragmentTextBillingTokens(extraction, pageBeats, pagePanels, audit int) int {
	total := 0
	for _, tokens := range []int{extraction, pageBeats, pagePanels, audit} {
		if tokens > 0 {
			total += tokens
		}
	}
	return total
}
