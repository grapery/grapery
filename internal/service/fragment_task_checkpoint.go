package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

type fragmentTaskCheckpoint struct {
	Version      int                                    `json:"version"`
	Stages       map[string]fragmentTaskCheckpointStage `json:"stages"`
	save         func(string) error
	persistErr   error
	operationErr error
}
type fragmentTaskCheckpointStage struct {
	State   string          `json:"state"`
	Payload json.RawMessage `json:"payload,omitempty"`
}
type fragmentTaskCheckpointKey struct{}

func fragmentCheckpointFromContext(ctx context.Context) *fragmentTaskCheckpoint {
	value, _ := ctx.Value(fragmentTaskCheckpointKey{}).(*fragmentTaskCheckpoint)
	return value
}
func newFragmentTaskCheckpoint(raw string, save func(string) error) (*fragmentTaskCheckpoint, error) {
	cp := &fragmentTaskCheckpoint{Version: 1, Stages: map[string]fragmentTaskCheckpointStage{}, save: save}
	if strings.TrimSpace(raw) != "" {
		if err := json.Unmarshal([]byte(raw), cp); err != nil {
			return nil, fmt.Errorf("invalid fragment checkpoint: %w", err)
		}
		if cp.Version != 1 || cp.Stages == nil {
			return nil, fmt.Errorf("unsupported fragment checkpoint")
		}
	}
	return cp, nil
}
func (cp *fragmentTaskCheckpoint) persist() error {
	raw, err := json.Marshal(cp)
	if err != nil {
		return err
	}
	err = cp.save(string(raw))
	if err != nil {
		cp.persistErr = err
	}
	return err
}

// Text/audit calls without a completed snapshot cannot be retried after a crash:
// the provider may have completed a paid request. Image stages may resume only
// through stable OperationID records, whose processing/unknown state fails closed.
func fragmentCheckpointStage[T any](ctx context.Context, key string, imageResumable bool, work func() (T, error)) (T, error) {
	var zero T
	if err := ctx.Err(); err != nil {
		return zero, err
	}
	cp := fragmentCheckpointFromContext(ctx)
	if cp == nil {
		return work()
	}
	if cp.persistErr != nil {
		return zero, cp.persistErr
	}
	if cp.operationErr != nil {
		return zero, cp.operationErr
	}
	if stage, exists := cp.Stages[key]; exists {
		if stage.State == "completed" {
			if err := json.Unmarshal(stage.Payload, &zero); err != nil {
				return zero, fmt.Errorf("checkpoint %s: %w", key, err)
			}
			return zero, nil
		}
		if stage.State != "inflight" {
			return zero, fmt.Errorf("invalid checkpoint stage %s", key)
		}
		if !imageResumable {
			return zero, fmt.Errorf("recovery_unknown_provider_outcome: %s; preserved completed stages; explicit retry required", key)
		}
	}
	cp.Stages[key] = fragmentTaskCheckpointStage{State: "inflight"}
	if err := cp.persist(); err != nil {
		return zero, err
	}
	output, err := work()
	if err != nil {
		return output, err
	}
	if err := ctx.Err(); err != nil {
		return zero, err
	}
	if cp.persistErr != nil {
		return zero, cp.persistErr
	}
	if cp.operationErr != nil {
		return zero, cp.operationErr
	}
	raw, err := json.Marshal(output)
	if err != nil {
		return zero, err
	}
	cp.Stages[key] = fragmentTaskCheckpointStage{State: "completed", Payload: raw}
	if err := cp.persist(); err != nil {
		return zero, err
	}
	return output, nil
}
func fragmentImageOperationID(taskID, scope, input string) string {
	sum := sha256.Sum256([]byte(scope + "\x00" + input))
	return "fragment:" + taskID + ":" + hex.EncodeToString(sum[:])
}

type fragmentCheckpointBaseline struct {
	Continuation *fragmentDraftContinuationContext
	Edit         *fragmentDraftEditContext
	Draft        *domain.Fragment
}
type fragmentCheckpointPagePlan struct {
	Plans  []domain.FragmentScenePlan
	Tokens int
}
type fragmentCheckpointImages struct {
	Result *domain.FragmentImageGenerationResult
	Scenes []domain.FragmentScenePlan
}
type fragmentCheckpointReferences struct {
	Assets []domain.FragmentReferenceAsset
	Tokens int
}
type fragmentCheckpointAudit struct {
	Issues   []domain.FragmentConsistencyIssue
	Tokens   int
	Provider string
	Count    int
	Skipped  string
}
type fragmentCheckpointRepair struct {
	Scenes []domain.FragmentScenePlan
	URLs   []string
	Tokens int
	Count  int
}
type fragmentCheckpointCommit struct {
	Draft  *domain.Fragment
	Result *domain.FragmentGenerationResult
	Create bool
}

func (s *FragmentGenerationService) commitFragmentCheckpoint(ctx context.Context, task *domain.FragmentGenerationTask, prepared *fragmentCheckpointCommit) error {
	if prepared == nil || prepared.Draft == nil || prepared.Result == nil {
		return fmt.Errorf("invalid prepared fragment commit")
	}
	if prepared.Create {
		if err := s.fragmentGenRepo.CreateDraftForTask(ctx, task.ID, prepared.Draft); err != nil {
			return err
		}
	} else {
		if err := s.fragmentGenRepo.UpdateDraftForTask(ctx, task.ID, prepared.Draft); err != nil {
			return err
		}
	}
	result := prepared.Result
	s.persistFragmentGenerationAssets(ctx, result.DraftFragmentID, task.ID, task.Request, result)
	s.recordFragmentGenerationOutputs(ctx, result.DraftFragmentID, task.UserID, task.ID, result)
	if err := s.fragmentGenRepo.UpdateResult(ctx, task.ID, result); err != nil {
		return err
	}
	if err := s.finalizeFragmentTextSettlement(ctx, task, result); err != nil {
		return fmt.Errorf("billing_unsettled: %w", err)
	}
	return nil
}

func fragmentCheckpointMissingForExecutedTask(task *domain.FragmentGenerationTask) bool {
	if task == nil || strings.TrimSpace(task.CheckpointJSON) != "" {
		return false
	}
	return task.Progress > 0 || strings.TrimSpace(task.CurrentStep) != "" || (task.Result != nil && (len(task.Result.ImageUrls) > 0 || len(task.Result.ScenePlan) > 0 || strings.TrimSpace(task.Result.Content) != ""))
}
func (s *FragmentGenerationService) ensureFragmentInitialRenderBalance(ctx context.Context, userID string, scenes []domain.FragmentScenePlan, assets []domain.FragmentReferenceAsset, stage string) error {
	if cp := fragmentCheckpointFromContext(ctx); cp != nil {
		if _, exists := cp.Stages[stage]; exists {
			return nil
		}
	}
	return s.ensureFragmentRenderBalance(ctx, userID, scenes, assets)
}

type fragmentCheckpointImageOutput struct {
	URL    string
	Tokens int
}

// Save the routed provider and exact image request before the provider call.
// Recovery reuses completed output or the original operation record; a pending
// operation without durable output is rejected by GenerateImage, never resent.
func (s *FragmentGenerationService) checkpointFragmentImage(ctx context.Context, request *domain.AITask, scope string, work func(*domain.AITask) (string, int, error)) (string, int, error) {
	if fragmentCheckpointFromContext(ctx) == nil {
		return work(request)
	}
	key := fragmentImageOperationID(request.RelatedEntityID, scope, request.Input)
	frozen, err := fragmentCheckpointStage(ctx, "image_intent:"+key, true, func() (domain.AITask, error) {
		copy := *request
		copy.ID = key
		copy.Provider = s.aiService.ResolveFragmentImageProvider(ctx, copy.UserID, copy.Provider)
		return copy, nil
	})
	if err != nil {
		return "", 0, err
	}
	output, err := fragmentCheckpointStage(ctx, "image_output:"+key, true, func() (fragmentCheckpointImageOutput, error) {
		url, tokens, err := work(&frozen)
		return fragmentCheckpointImageOutput{URL: url, Tokens: tokens}, err
	})
	var billingErr *domain.GenerationBillingError
	if errors.As(err, &billingErr) {
		fragmentCheckpointFromContext(ctx).operationErr = err
	}
	return output.URL, output.Tokens, err
}

// Fragment.MarshalJSON is a public response projection (imageUrls becomes an
// array while the underlying field is a string). Checkpoints store the actual
// struct contract so JSON round-trips preserve every draft field.
type fragmentCheckpointDraftWire domain.Fragment

func (baseline fragmentCheckpointBaseline) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Continuation *fragmentDraftContinuationContext
		Edit         *fragmentDraftEditContext
		Draft        *fragmentCheckpointDraftWire
	}{baseline.Continuation, baseline.Edit, (*fragmentCheckpointDraftWire)(baseline.Draft)})
}
func (commit fragmentCheckpointCommit) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Draft  *fragmentCheckpointDraftWire
		Result *domain.FragmentGenerationResult
		Create bool
	}{(*fragmentCheckpointDraftWire)(commit.Draft), commit.Result, commit.Create})
}
