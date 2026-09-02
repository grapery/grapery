package service

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"text/template"

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
	SystemPrompt    string
	UserPrompt      string
	Model           string
	Temperature     float32
	MaxTokens       int32
	TemplateVersion string
	Applied         bool
	FallbackReason  string
}

func resolveStoryboardBranchPrompt(storyboard *domain.Storyboard, slot, legacySystem, legacyUser string, data map[string]any, defaultTemperature float32, defaultMaxTokens int32) storyboardPromptResolution {
	fallback := storyboardPromptResolution{
		SystemPrompt: legacySystem, UserPrompt: legacyUser, Model: "gemini-2.5-flash",
		Temperature: defaultTemperature, MaxTokens: defaultMaxTokens, TemplateVersion: storyboardPromptTemplateVersion,
	}
	prompt := workflowPromptSnapshot(storyboard, storyboardBranchPromptNodeID, slot, slot == storyboardBranchContentSlot)
	if prompt == nil {
		fallback.FallbackReason = "branch workflow prompt snapshot not found"
		return fallback
	}
	if prompt.Type != "chat" && prompt.Type != "text" {
		fallback.FallbackReason = "branch workflow prompt type is not text or chat"
		return fallback
	}
	if data == nil {
		data = make(map[string]any)
	}
	data["legacySystemPrompt"] = legacySystem
	data["legacyUserPrompt"] = legacyUser
	data["storyboardJSON"] = mustJSON(storyboard, "{}")
	return resolveStoryboardPromptTemplate(prompt, fallback, data, "branch_"+slot, "gemini-2.5-flash", defaultTemperature, defaultMaxTokens)
}

func shouldWarnStoryboardPromptFallback(reason string) bool {
	reason = strings.TrimSpace(reason)
	return reason != "" && !strings.Contains(reason, "snapshot not found")
}

func resolveStoryboardBiblePrompt(
	story *domain.Story,
	storyboard *domain.Storyboard,
	snapshot storyboardGenerationContextSnapshot,
	alignmentPrompt string,
	sceneCount int,
) storyboardPromptResolution {
	legacySystem := buildStoryboardBiblePlanSystemPrompt()
	legacyUser := buildStoryboardBiblePlanUserPrompt(story, storyboard, snapshot, sceneCount)
	fallback := storyboardPromptResolution{
		SystemPrompt: legacySystem, UserPrompt: legacyUser, Model: "gemini-2.5-flash",
		Temperature: 0.32, MaxTokens: 5000, TemplateVersion: storyboardPromptTemplateVersion,
	}
	prompt := storyboardPromptSnapshotForSlot(storyboard, storyboardBiblePromptSlot, true)
	if prompt == nil {
		fallback.FallbackReason = "workflow prompt snapshot not found"
		return fallback
	}
	if prompt.Type != "chat" && prompt.Type != "text" {
		fallback.FallbackReason = "workflow prompt type is not text or chat"
		return fallback
	}
	data := map[string]any{
		"legacySystemPrompt": legacySystem,
		"legacyUserPrompt":   legacyUser,
		"storyJSON":          mustJSON(story, "{}"),
		"storyboardJSON":     mustJSON(storyboard, "{}"),
		"contextJSON":        mustJSON(snapshot, "{}"),
		"alignmentPrompt":    alignmentPrompt,
		"sceneCount":         sceneCount,
	}
	systemPrompt := legacySystem
	if strings.TrimSpace(prompt.SystemTemplate) != "" {
		rendered, err := renderStoryboardPromptTemplate(prompt.Key+".system", prompt.SystemTemplate, data)
		if err != nil {
			fallback.FallbackReason = err.Error()
			return fallback
		}
		systemPrompt = rendered
	}
	userPrompt := legacyUser
	if strings.TrimSpace(prompt.UserTemplate) != "" {
		rendered, err := renderStoryboardPromptTemplate(prompt.Key+".user", prompt.UserTemplate, data)
		if err != nil {
			fallback.FallbackReason = err.Error()
			return fallback
		}
		userPrompt = rendered
	}
	if strings.TrimSpace(systemPrompt) == "" || strings.TrimSpace(userPrompt) == "" {
		fallback.FallbackReason = "workflow prompt rendered empty content"
		return fallback
	}
	model, temperature, maxTokens := storyboardPromptModelConfig(prompt.ModelConfig, "gemini-2.5-flash", 0.32, 5000)
	return storyboardPromptResolution{
		SystemPrompt: systemPrompt, UserPrompt: userPrompt, Model: model, Temperature: temperature,
		MaxTokens: maxTokens, TemplateVersion: prompt.ID, Applied: true,
	}
}

func resolveStoryboardScenePrompt(
	story *domain.Story,
	storyboard *domain.Storyboard,
	snapshot storyboardGenerationContextSnapshot,
	plan *domain.StoryboardBiblePlan,
	alignmentPrompt string,
	sceneCount int,
) storyboardPromptResolution {
	legacySystem := buildStoryboardSceneWriterSystemPrompt()
	legacyUser := buildStoryboardSceneWriterUserPrompt(story, storyboard, snapshot, plan, sceneCount)
	fallback := storyboardPromptResolution{
		SystemPrompt: legacySystem, UserPrompt: legacyUser, Model: "gemini-2.5-flash",
		Temperature: 0.28, MaxTokens: 6500, TemplateVersion: storyboardPromptTemplateVersion,
	}
	prompt := storyboardPromptSnapshotForSlot(storyboard, storyboardScenePromptSlot, false)
	if prompt == nil {
		fallback.FallbackReason = "scene_plan workflow prompt snapshot not found"
		return fallback
	}
	if prompt.Type != "chat" && prompt.Type != "text" {
		fallback.FallbackReason = "scene_plan workflow prompt type is not text or chat"
		return fallback
	}
	data := map[string]any{
		"legacySystemPrompt": legacySystem,
		"legacyUserPrompt":   legacyUser,
		"storyJSON":          mustJSON(story, "{}"),
		"storyboardJSON":     mustJSON(storyboard, "{}"),
		"contextJSON":        mustJSON(snapshot, "{}"),
		"biblePlanJSON":      mustJSON(plan, "{}"),
		"alignmentPrompt":    alignmentPrompt,
		"sceneCount":         sceneCount,
	}
	return resolveStoryboardPromptTemplate(prompt, fallback, data, "scene_plan", "gemini-2.5-flash", 0.28, 6500)
}

func resolveStoryboardJSONRepairPrompt(storyboard *domain.Storyboard, broken, detail, step, operation string) storyboardPromptResolution {
	legacySystem := `You are a strict JSON repair tool. Output ONE JSON value only (a single JSON object as required). No markdown fences, no commentary before or after.
Rules:
- Preserve semantics when possible.
- If input is truncated, minimally close brackets/braces so the JSON parses.
- Never output prose outside JSON.`
	legacyUser := fmt.Sprintf("Repair into valid JSON. Failure detail:\n%s\n\nBroken model output:\n%s", detail, broken)
	fallback := storyboardPromptResolution{
		SystemPrompt: legacySystem, UserPrompt: legacyUser, Model: "gemini-2.5-flash",
		Temperature: 0.1, MaxTokens: 8192, TemplateVersion: storyboardPromptTemplateVersion,
	}
	prompt := storyboardPromptSnapshotForSlot(storyboard, storyboardJSONRepairPromptSlot, false)
	if prompt == nil {
		fallback.FallbackReason = "json_repair workflow prompt snapshot not found"
		return fallback
	}
	if prompt.Type != "chat" && prompt.Type != "text" {
		fallback.FallbackReason = "json_repair workflow prompt type is not text or chat"
		return fallback
	}
	data := map[string]any{
		"legacySystemPrompt": legacySystem,
		"legacyUserPrompt":   legacyUser,
		"brokenOutput":       broken,
		"failureDetail":      detail,
		"step":               step,
		"operation":          operation,
		"storyboardJSON":     mustJSON(storyboard, "{}"),
	}
	return resolveStoryboardPromptTemplate(prompt, fallback, data, "json_repair", "gemini-2.5-flash", 0.1, 8192)
}

func resolveStoryboardPromptTemplate(prompt *domain.PromptTemplateVersion, fallback storyboardPromptResolution, data map[string]any, label, defaultModel string, defaultTemperature float32, defaultMaxTokens int32) storyboardPromptResolution {
	systemPrompt := fallback.SystemPrompt
	if strings.TrimSpace(prompt.SystemTemplate) != "" {
		rendered, err := renderStoryboardPromptTemplate(prompt.Key+".system", prompt.SystemTemplate, data)
		if err != nil {
			fallback.FallbackReason = err.Error()
			return fallback
		}
		systemPrompt = rendered
	}
	userPrompt := fallback.UserPrompt
	if strings.TrimSpace(prompt.UserTemplate) != "" {
		rendered, err := renderStoryboardPromptTemplate(prompt.Key+".user", prompt.UserTemplate, data)
		if err != nil {
			fallback.FallbackReason = err.Error()
			return fallback
		}
		userPrompt = rendered
	}
	if strings.TrimSpace(systemPrompt) == "" || strings.TrimSpace(userPrompt) == "" {
		fallback.FallbackReason = label + " workflow prompt rendered empty content"
		return fallback
	}
	model, temperature, maxTokens := storyboardPromptModelConfig(prompt.ModelConfig, defaultModel, defaultTemperature, defaultMaxTokens)
	return storyboardPromptResolution{
		SystemPrompt: systemPrompt, UserPrompt: userPrompt, Model: model, Temperature: temperature,
		MaxTokens: maxTokens, TemplateVersion: prompt.ID, Applied: true,
	}
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

func renderStoryboardPromptTemplate(name, source string, data map[string]any) (string, error) {
	tmpl, err := template.New(name).Option("missingkey=error").Parse(source)
	if err != nil {
		return "", fmt.Errorf("parse workflow prompt %s: %w", name, err)
	}
	var out bytes.Buffer
	if err := tmpl.Execute(&out, data); err != nil {
		return "", fmt.Errorf("render workflow prompt %s: %w", name, err)
	}
	if out.Len() > maxRenderedStoryboardPromptLen {
		return "", fmt.Errorf("workflow prompt %s exceeds %d bytes", name, maxRenderedStoryboardPromptLen)
	}
	return strings.TrimSpace(out.String()), nil
}

func storyboardPromptModelConfig(config map[string]any, defaultModel string, defaultTemperature float32, defaultMaxTokens int32) (string, float32, int32) {
	model, temperature, maxTokens := defaultModel, defaultTemperature, defaultMaxTokens
	if value, ok := config["model"].(string); ok && strings.TrimSpace(value) != "" && len(value) <= 160 {
		model = strings.TrimSpace(value)
	}
	if value, ok := promptConfigFloat(config["temperature"]); ok && value >= 0 && value <= 2 {
		temperature = float32(value)
	}
	if value, ok := promptConfigInt(config["maxTokens"]); ok && value >= 256 && value <= 8192 {
		maxTokens = int32(value)
	}
	return model, temperature, maxTokens
}

func promptConfigFloat(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case string:
		parsed, err := strconv.ParseFloat(typed, 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func promptConfigInt(value any) (int, bool) {
	floatValue, ok := promptConfigFloat(value)
	if !ok || floatValue != float64(int(floatValue)) {
		return 0, false
	}
	return int(floatValue), true
}
