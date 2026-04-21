package service

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	genapi "github.com/grapestree/fgrapery/grapery/internal/genai"
)

// Limits for storyboard image pipeline (A/B/C): keep narrative; allow longer final prompts; optional text-to-image when narrative is rich.
const (
	storyboardImageT2INarrativeMinRunes = 120
	storyboardImageFinalMaxRunes        = 4000
	storyboardImageBeautyMaxRunes       = 600 // cap Gemini "beauty" expansion when preserving narrative block
)

// MergedStoryboardSceneDescriptionForImage returns plot scene text merged with the latest completed Step-2 detail when available.
func (s *Service) MergedStoryboardSceneDescriptionForImage(ctx context.Context, storyboardID, sceneID, plotDescription string) string {
	plotDescription = strings.TrimSpace(plotDescription)
	if s == nil || s.repo == nil {
		return plotDescription
	}
	storyboardID, sceneID = strings.TrimSpace(storyboardID), strings.TrimSpace(sceneID)
	if storyboardID == "" || sceneID == "" {
		return plotDescription
	}
	detailGen, err := s.repo.LatestCompletedSceneGenerationByScene(ctx, storyboardID, sceneID)
	if err != nil || detailGen == nil {
		return plotDescription
	}
	return strings.TrimSpace(mergeStoryboardSceneDescriptionForImage(plotDescription, detailGen.GeneratedDetail))
}

// mergeStoryboardSceneDescriptionForImage prefers Step-2 cinematic detail when present; otherwise uses the plot description.
func mergeStoryboardSceneDescriptionForImage(plotDescription, step2Detail string) string {
	plotDescription = strings.TrimSpace(plotDescription)
	step2Detail = strings.TrimSpace(step2Detail)
	if step2Detail == "" {
		return plotDescription
	}
	if plotDescription == "" {
		return step2Detail
	}
	if strings.Contains(plotDescription, step2Detail) {
		return plotDescription
	}
	return plotDescription + "\n\n[Scene detail / cinematography]\n" + step2Detail
}

func storyboardSceneNarrativeBlock(title, description string) string {
	title = strings.TrimSpace(title)
	description = strings.TrimSpace(description)
	var b strings.Builder
	b.WriteString("【分镜叙事 / Scene to illustrate — follow faithfully】\n")
	if title != "" {
		b.WriteString("Title: ")
		b.WriteString(title)
		b.WriteByte('\n')
	}
	if description != "" {
		b.WriteString("Description:\n")
		b.WriteString(description)
		b.WriteByte('\n')
	}
	out := strings.TrimSpace(b.String())
	if out == "【分镜叙事 / Scene to illustrate — follow faithfully】" {
		return ""
	}
	return out
}

func narrativeFingerprint(desc string) string {
	desc = strings.TrimSpace(desc)
	if desc == "" {
		return ""
	}
	r := []rune(desc)
	const maxFP = 80
	if len(r) > maxFP {
		return string(r[:maxFP])
	}
	return desc
}

func beautyAlreadyEmbedsNarrative(narrativeBlock, beauty string) bool {
	fp := narrativeFingerprint(narrativeBlock)
	if fp == "" {
		return false
	}
	return strings.Contains(beauty, fp)
}

// prependStoryboardImageNarrativeBlock puts the plot narrative before the model-expanded visual prompt.
func prependStoryboardImageNarrativeBlock(narrativeBlock, beauty string) string {
	narrativeBlock = strings.TrimSpace(narrativeBlock)
	beauty = strings.TrimSpace(beauty)
	if narrativeBlock == "" {
		return beauty
	}
	if beauty == "" {
		return narrativeBlock
	}
	if beautyAlreadyEmbedsNarrative(narrativeBlock, beauty) {
		return beauty
	}
	return narrativeBlock + "\n\n【画面与镜头 / Visual direction】\n" + beauty
}

func appendStoryboardImageToImageConstraints(prompt string, useReferenceImages bool) string {
	prompt = strings.TrimSpace(prompt)
	if !useReferenceImages || prompt == "" {
		return prompt
	}
	const c = "\n\n[Reference images: identity only] Use reference images ONLY to preserve the main cast's face, body type, hair, and costume identity. You MUST still redraw the full scene from the narrative above: correct pose, interaction, props, environment, and shot composition. Do NOT paste or lightly retouch the reference onto a new background; avoid a \"same pose, new backdrop\" result."
	return prompt + c
}

// truncateStoryboardImagePromptPreservingNarrative keeps the narrative block intact and trims the tail (visual direction) if needed.
func truncateStoryboardImagePromptPreservingNarrative(narrativeBlock, full string, maxTotalRunes int) string {
	full = strings.TrimSpace(full)
	if maxTotalRunes <= 0 || full == "" {
		return full
	}
	narrativeBlock = strings.TrimSpace(narrativeBlock)
	if narrativeBlock == "" {
		return truncateStringToMaxRunes(full, maxTotalRunes)
	}
	nr := []rune(narrativeBlock)
	if len(nr) >= maxTotalRunes {
		return string(nr[:maxTotalRunes])
	}
	sep := "\n\n【画面与镜头 / Visual direction】\n"
	idx := strings.Index(full, sep)
	if idx == -1 {
		return truncateStringToMaxRunes(full, maxTotalRunes)
	}
	prefix := full[:idx]
	beauty := strings.TrimSpace(full[idx+len(sep):])
	prefixRunes := utf8.RuneCountInString(prefix)
	if prefixRunes > maxTotalRunes {
		return truncateStringToMaxRunes(prefix, maxTotalRunes)
	}
	remaining := maxTotalRunes - prefixRunes - utf8.RuneCountInString(sep)
	if remaining <= 0 {
		return truncateStringToMaxRunes(prefix, maxTotalRunes)
	}
	beauty = truncateStringToMaxRunes(beauty, remaining)
	if beauty == "" {
		return strings.TrimSpace(prefix)
	}
	return strings.TrimSpace(prefix) + sep + beauty
}

// capGeminiImageBeautyOutput limits only the LLM-expanded portion before narrative merge (Step C: JSON fields stay short; final cap is separate).
func capGeminiImageBeautyOutput(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	return truncateStringToMaxRunes(s, storyboardImageBeautyMaxRunes)
}

// finalizeStoryboardImagePromptForAPI trims the full prompt while keeping the narrative block (already prepended by combineImagePrompt / fallbacks).
func finalizeStoryboardImagePromptForAPI(gen *domain.StoryboardImageGeneration) string {
	if gen == nil {
		return ""
	}
	nb := storyboardSceneNarrativeBlock(gen.SceneTitle, gen.SceneDescription)
	full := strings.TrimSpace(gen.GeneratedPrompt)
	return truncateStoryboardImagePromptPreservingNarrative(nb, full, storyboardImageFinalMaxRunes)
}

// selectStoryboardImageRefsAndOperation chooses text-to-image when narrative is long enough to avoid "portrait swap background" bias.
func selectStoryboardImageRefsAndOperation(gen *domain.StoryboardImageGeneration, narrativeRunes int) (refs []string, op genapi.OperationType, primaryURL string) {
	if gen == nil || len(gen.ReferenceImages) == 0 {
		return nil, genapi.OperationTextToImage, ""
	}
	refs = append([]string(nil), gen.ReferenceImages...)
	if gen.IsTransitionScene {
		return refs, genapi.OperationImageToImage, strings.TrimSpace(refs[0])
	}
	if narrativeRunes >= storyboardImageT2INarrativeMinRunes {
		return nil, genapi.OperationTextToImage, ""
	}
	return refs, genapi.OperationImageToImage, strings.TrimSpace(refs[0])
}
