package service

import (
	"encoding/json"
	"strings"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

func canonicalFragmentRequest(req domain.FragmentGenerationRequest) string {
	req.ClientMessageID, req.WorkflowRunID = "", ""
	req.BillingVersion, req.IncludeGenerationTrace = 0, false
	// Deprecated template bodies are ignored by execution and are not intent.
	req.WorkflowSystemPrompt, req.WorkflowUserPrompt, req.WorkflowPromptVersionID = "", "", ""
	req.WorkflowModelConfig, req.WorkflowOutputSchema = nil, nil
	req.UserInput = strings.TrimSpace(req.UserInput)
	req.TargetDraftFragmentID = strings.TrimSpace(req.TargetDraftFragmentID)
	req.ImageCount = normalizeFragmentGenerationImageCount(req.ImageCount)
	if req.ReplaceImageIndex > 0 {
		req.ImageCount = 1
	}
	if req.Style == "" {
		req.Style = "fantasy"
	}
	req.Visibility = domain.NormalizeFragmentVisibility(req.Visibility)
	req.ConsistencyLevel = normalizeFragmentConsistencyLevel(req.ConsistencyLevel)
	req.Language = normalizeGenerationLanguage(req.Language)
	controls := req.CreativeControls.Normalized()
	req.CreativeControls = &controls
	if len(req.ImageUrls) == 0 {
		req.ImageUrls = nil
	}
	if len(req.ReferenceSlots) == 0 {
		req.ReferenceSlots = nil
	}
	raw, _ := json.Marshal(req)
	return string(raw)
}

func validateFragmentRequestReplay(task *domain.FragmentGenerationTask, userID string, req domain.FragmentGenerationRequest) error {
	if task == nil || task.UserID != userID || canonicalFragmentRequest(task.Request) != canonicalFragmentRequest(req) {
		return domain.NewFragmentGenerationError(domain.FragmentGenerationErrorConflict, "clientMessageId was already used for a different fragment request", nil)
	}
	return nil
}
