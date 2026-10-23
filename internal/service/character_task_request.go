package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

type characterExecutionReader interface {
	GetGenerationExecution(context.Context, string) (*domain.GenerationExecution, error)
}

func (s *Service) validateCharacterWorkflowRequest(ctx context.Context, userID string, req CharacterGenerationTaskRequest) error {
	if req.WorkflowRunID == "" {
		return nil
	}

	reader, ok := s.repo.(characterExecutionReader)
	if !ok {
		return fmt.Errorf("character workflow execution lookup unavailable")
	}
	run, err := reader.GetGenerationExecution(ctx, req.WorkflowRunID)
	if err != nil || run == nil || run.UserID != userID || run.WorkflowReleaseID != req.WorkflowReleaseID {
		return fmt.Errorf("%w: invalid character workflow execution", domain.ErrForbidden)
	}
	if run.Kind != "character" && !(run.Kind == "workflow" && req.WorkflowReleaseID != "" && run.Input["workflowSurface"] == "voyager.character") {
		return fmt.Errorf("%w: workflow execution is not a character creation", domain.ErrForbidden)
	}
	if storyID, ok := run.Input["storyId"].(string); ok && strings.TrimSpace(storyID) != "" && storyID != req.StoryID {
		return fmt.Errorf("%w: workflow story does not match character request", domain.ErrForbidden)
	}
	return nil
}
func characterTaskRequestIdentity(userID string, req CharacterGenerationTaskRequest) (string, string, error) {
	key := strings.TrimSpace(req.ClientRequestID)
	if key == "" && req.WorkflowRunID != "" {
		key = "workflow:" + req.WorkflowRunID
	}
	if len(key) > 256 {
		return "", "", fmt.Errorf("clientRequestId exceeds 256 bytes")
	}
	// Identity, all source fields, confirmed attributes, references and selected
	// release/controls are included. Only the transport retry key is excluded.
	req.ClientRequestID = ""
	normalizedControls := req.CreativeControls.Normalized()
	req.CreativeControls = &normalizedControls
	raw, err := json.Marshal(req)
	if err != nil {
		return "", "", err
	}
	intentHash := fmt.Sprintf("%x", sha256.Sum256(raw))
	if key == "" {
		if req.SourceType != domain.CharacterGenerationSourceManualForm && req.SourceType != domain.CharacterGenerationSourceManualPrompt {
			return "", "", nil
		}
		key = "manual-intent:" + intentHash
	}
	return fmt.Sprintf("%x", sha256.Sum256([]byte(userID+"\x00"+key))), intentHash, nil
}
func characterAttributesRecoveryError(task *domain.CharacterGenerationTask) error {
	stage := task.Stages["attributes"]
	if stage.Status != "processing" && !stage.OutcomeUnknown {
		return nil
	}
	stage.Status = "failed"
	stage.OutcomeUnknown = true
	stage.Error = "character_attributes_outcome_unknown: previous model/charge outcome requires reconciliation; automatic and task retries cannot resend"
	task.Stages["attributes"] = stage
	return fmt.Errorf("%s", stage.Error)
}

func completeConfirmedCharacterAttributes(attrs GeneratedCharacterAttributes) (string, bool) {
	raw, err := json.Marshal(attrs)
	if err != nil {
		return "", false
	}
	_, err = validateCharacterAttributesOutput(string(raw))
	return string(raw), err == nil
}
