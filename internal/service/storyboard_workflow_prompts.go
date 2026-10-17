package service

import (
	"fmt"
	"strings"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

const (
	storyboardWorkflowPromptNodeID = "generate_storyboard"
	storyboardBiblePromptSlot      = "bible_plan"
	storyboardScenePromptSlot      = "scene_plan"
	storyboardJSONRepairPromptSlot = "json_repair"
	storyboardBranchPromptNodeID   = "generate_storyboard_branch"
	storyboardBranchContentSlot    = "content"
	storyboardBranchSceneSlot      = "scene_plan"
	maxRenderedStoryboardPromptLen = 256 * 1024
)

type storyboardPromptResolution struct {
	SystemPrompt              string
	UserPrompt                string
	Model                     string
	Temperature               float32
	MaxTokens                 int32
	TemplateVersion           string
	Applied                   bool
	FallbackReason            string
	DeprecatedTemplateIgnored bool
}

func storyboardProtectedPrompt(storyboard *domain.Storyboard, system, user string, temperature float32, maxTokens int32) storyboardPromptResolution {
	var controls *domain.CreativeControls
	if storyboard != nil {
		controls = storyboard.CreativeControls
	}
	system, user = protectCreativePrompt(system, user, "", "", controls)
	return storyboardPromptResolution{SystemPrompt: system, UserPrompt: user, Model: "gemini-2.5-flash", Temperature: temperature, MaxTokens: maxTokens, TemplateVersion: storyboardPromptTemplateVersion}
}

func retainStoryboardTemplateAudit(prompt *domain.PromptTemplateVersion, fallback storyboardPromptResolution) storyboardPromptResolution {
	if prompt != nil {
		fallback.TemplateVersion = prompt.ID
		fallback.DeprecatedTemplateIgnored = true
		fallback.FallbackReason = "deprecated free-text template ignored; engineering prompt retained"
	}
	return fallback
}

func resolveStoryboardBranchPrompt(storyboard *domain.Storyboard, slot, coreSystem, coreUser string, data map[string]any, temperature float32, maxTokens int32) storyboardPromptResolution {
	result := storyboardProtectedPrompt(storyboard, coreSystem, coreUser, temperature, maxTokens)
	return retainStoryboardTemplateAudit(workflowPromptSnapshot(storyboard, storyboardBranchPromptNodeID, slot, slot == storyboardBranchContentSlot), result)
}

func shouldWarnStoryboardPromptFallback(reason string) bool {
	return strings.TrimSpace(reason) != "" && !strings.Contains(reason, "snapshot not found") && !strings.Contains(reason, "deprecated free-text")
}

func resolveStoryboardBiblePrompt(story *domain.Story, board *domain.Storyboard, snapshot storyboardGenerationContextSnapshot, alignment string, count int) storyboardPromptResolution {
	board = storyboardWithFrozenCreativeControls(board, snapshot)
	result := storyboardProtectedPrompt(board, buildStoryboardBiblePlanSystemPrompt(), buildStoryboardBiblePlanUserPrompt(story, board, snapshot, count), 0.32, 5000)
	if strings.TrimSpace(alignment) != "" {
		result.UserPrompt += "\nConfirmed user alignment snapshot:\n" + alignment
	}
	return retainStoryboardTemplateAudit(storyboardPromptSnapshotForSlot(board, storyboardBiblePromptSlot, true), result)
}

func resolveStoryboardScenePrompt(story *domain.Story, board *domain.Storyboard, snapshot storyboardGenerationContextSnapshot, plan *domain.StoryboardBiblePlan, alignment string, count int) storyboardPromptResolution {
	board = storyboardWithFrozenCreativeControls(board, snapshot)
	result := storyboardProtectedPrompt(board, buildStoryboardSceneWriterSystemPrompt(), buildStoryboardSceneWriterUserPrompt(story, board, snapshot, plan, count), 0.28, 6500)
	if strings.TrimSpace(alignment) != "" {
		result.UserPrompt += "\nConfirmed user alignment snapshot:\n" + alignment
	}
	return retainStoryboardTemplateAudit(storyboardPromptSnapshotForSlot(board, storyboardScenePromptSlot, false), result)
}

func resolveStoryboardJSONRepairPrompt(board *domain.Storyboard, broken, detail, step, operation string) storyboardPromptResolution {
	system := `You are a strict JSON repair tool. Output ONE JSON value only (a single JSON object as required). No markdown fences, no commentary before or after.
Rules:
- Preserve semantics when possible.
- If input is truncated, minimally close brackets/braces so the JSON parses.
- Never output prose outside JSON.`
	user := fmt.Sprintf("Repair into valid JSON. Failure detail:\n%s\n\nBroken model output:\n%s", detail, broken)
	result := storyboardProtectedPrompt(nil, system, user, 0.1, 8192)
	return retainStoryboardTemplateAudit(storyboardPromptSnapshotForSlot(board, storyboardJSONRepairPromptSlot, false), result)
}

func storyboardPromptSnapshotForSlot(storyboard *domain.Storyboard, slot string, allowLegacyDefault bool) *domain.PromptTemplateVersion {
	if prompt := workflowPromptSnapshot(storyboard, storyboardWorkflowPromptNodeID, slot, allowLegacyDefault); prompt != nil {
		return prompt
	}
	// The compatibility /fork endpoint executes the redesign stages underneath
	// a branch release; allow that immutable branch snapshot to drive the same
	// structured generation rather than falling back to hard-coded prompts.
	return workflowPromptSnapshot(storyboard, storyboardBranchPromptNodeID, slot, allowLegacyDefault)
}

func workflowPromptSnapshot(storyboard *domain.Storyboard, nodeID, slot string, allowDefault bool) *domain.PromptTemplateVersion {
	if storyboard == nil || len(storyboard.PromptSnapshots) == 0 {
		return nil
	}
	if prompt, ok := storyboard.PromptSnapshots[nodeID+":"+slot]; ok {
		return &prompt
	}
	if allowDefault {
		if prompt, ok := storyboard.PromptSnapshots[nodeID]; ok {
			return &prompt
		}
	}
	return nil
}

func storyboardWithFrozenCreativeControls(board *domain.Storyboard, snapshot storyboardGenerationContextSnapshot) *domain.Storyboard {
	if board == nil || snapshot.CreativeControls == nil {
		return board
	}
	frozen := *board
	frozen.CreativeControls = snapshot.CreativeControls
	return &frozen
}
