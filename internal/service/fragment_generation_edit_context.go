package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

type fragmentDraftEditContext struct {
	DraftID     string
	Content     string
	ImageURLs   []string
	Trace       *domain.FragmentGenerationTrace
	Document    *domain.FragmentComicDocument
	TargetIndex int // 0 based
	AspectRatio string
	Style       string
}

func (s *FragmentGenerationService) loadFragmentDraftEditContext(ctx context.Context, userID string, req domain.FragmentGenerationRequest) (*fragmentDraftEditContext, error) {
	if req.ReplaceImageIndex <= 0 {
		return nil, nil
	}
	draft, err := s.fragmentRepo.GetByID(ctx, req.TargetDraftFragmentID)
	if err != nil {
		return nil, fmt.Errorf("load edit draft: %w", err)
	}
	if draft == nil || draft.UserID != userID || !draft.IsDraft {
		return nil, fmt.Errorf("edit draft is not writable")
	}
	return fragmentEditContextFromDraft(req, draft)
}

func fragmentEditContextFromDraft(req domain.FragmentGenerationRequest, draft *domain.Fragment) (*fragmentDraftEditContext, error) {
	urls := append([]string(nil), draft.MediaURLs...)
	if len(urls) == 0 {
		_ = json.Unmarshal([]byte(draft.ImageUrls), &urls)
	}
	index := req.ReplaceImageIndex - 1
	if index < 0 || index >= len(urls) || strings.TrimSpace(urls[index]) == "" {
		return nil, fmt.Errorf("edit page is unavailable")
	}
	trace := parseFragmentGenerationTraceMetadata(draft.GenerationMetadata)
	if trace == nil {
		trace = &domain.FragmentGenerationTrace{}
	}
	doc := fragmentComicDocumentFromExistingDraft(req, draft.ID, draft.Content, draft.AspectRatio, urls, trace)
	style := ""
	if draft.Style != nil {
		style = *draft.Style
	}
	return &fragmentDraftEditContext{DraftID: draft.ID, Content: draft.Content, ImageURLs: urls, Trace: trace, Document: doc, TargetIndex: index, AspectRatio: draft.AspectRatio, Style: style}, nil
}

func fragmentRequestWithEditContext(req domain.FragmentGenerationRequest, edit *fragmentDraftEditContext) domain.FragmentGenerationRequest {
	if edit == nil {
		return req
	}
	// Existing identity assets take priority over inspirational uploads. Include
	// the selected original and neighbors even for legacy bitmap-only drafts.
	refs := fragmentContinuationReferenceImageURLs(&fragmentDraftContinuationContext{PreviousTrace: edit.Trace}, 3)
	for _, index := range []int{edit.TargetIndex, edit.TargetIndex - 1, edit.TargetIndex + 1} {
		if index >= 0 && index < len(edit.ImageURLs) {
			refs = append(refs, edit.ImageURLs[index])
		}
	}
	req.ReferenceSlots = append([]domain.FragmentReferenceSlot{{Key: "edit_original_page", Label: "Selected original page (identity and causal authority)", Kind: "scene", Required: true, ImageURL: edit.ImageURLs[edit.TargetIndex]}}, req.ReferenceSlots...)
	req.ImageUrls = compactNonEmptyStrings(append(refs, req.ImageUrls...))
	req.AspectRatio = edit.AspectRatio
	if edit.Style != "" {
		req.Style = edit.Style
	}
	if edit.Document != nil {
		if edit.Document.CreativeContext.Style != "" {
			req.Style = edit.Document.CreativeContext.Style
		}
		req.Mood = edit.Document.CreativeContext.Mood
		if len(edit.Document.Pages) > 0 && edit.Document.Pages[0].Plan.Layout != nil {
			req.AspectRatio = edit.Document.Pages[0].Plan.Layout.PageAspectRatio
		}
	}
	req.UserInput = strings.TrimSpace(req.UserInput) + "\n\n" + formatFragmentEditContext(edit)
	return req
}

func formatFragmentEditContext(edit *fragmentDraftEditContext) string {
	if edit == nil {
		return ""
	}
	pages := make([]domain.FragmentComicPageDocument, 0, 3)
	if edit.Document != nil {
		for _, page := range edit.Document.Pages {
			if page.PageIndex >= edit.TargetIndex-1 && page.PageIndex <= edit.TargetIndex+1 {
				pages = append(pages, page)
			}
		}
	}
	payload := map[string]any{"operation": "replace_one_page", "targetPageIndex": edit.TargetIndex,
		"existingStory": edit.Content, "targetAndNeighborPages": pages,
		"constraints": []string{"Apply only the user's requested visual changes to the selected page.", "Preserve the selected page's story event, dialogue, causal role, character identities and core appearances unless the change explicitly targets that page detail.", "Neighbors are context only: never rewrite, render, reorder or replace them.", "Global visual bible and end-of-story state are locked. Do not introduce or rename entities or change global art style.", "Output exactly one page beat for the selected original page, with local index 0. Never restart or continue the story."}}
	if edit.Document != nil {
		payload["lockedVisualBible"] = edit.Document.VisualBible
		payload["lockedStoryState"] = edit.Document.StoryState
	}
	if payload["lockedVisualBible"] == nil {
		payload["lockedVisualBible"] = edit.Trace.VisualBible
	}
	raw, _ := json.Marshal(payload)
	return "AUTHORITATIVE PAGE EDIT CONTEXT (server-owned; takes precedence over template defaults)\n" + string(raw)
}

func fragmentElementsForEdit(req domain.FragmentGenerationRequest, edit *fragmentDraftEditContext) *fragmentElementExtractionResult {
	result := &fragmentElementExtractionResult{Content: edit.Content, AspectRatio: req.AspectRatio}
	bible := edit.Trace.VisualBible
	if edit.Document != nil && edit.Document.VisualBible != nil {
		bible = edit.Document.VisualBible
	}
	// Do not mutate the persisted authority through downstream normalizers.
	if raw, err := json.Marshal(bible); err == nil {
		_ = json.Unmarshal(raw, &result.VisualBible)
	}
	result.VisualEvidence = append([]domain.FragmentVisualEvidence(nil), edit.Trace.VisualEvidence...)
	if result.VisualBible != nil {
		for _, entity := range result.VisualBible.Characters {
			result.Elements.Characters = append(result.Elements.Characters, entity.Name+": "+strings.Join(entity.ImmutableTraits, "; "))
		}
		for _, entity := range result.VisualBible.Props {
			result.Elements.Objects = append(result.Elements.Objects, entity.Name+": "+strings.Join(entity.ImmutableTraits, "; "))
		}
		for _, entity := range result.VisualBible.Locations {
			result.Elements.Scenes = append(result.Elements.Scenes, entity.Name+": "+strings.Join(entity.ImmutableTraits, "; "))
		}
	}
	ensureFragmentVisualBibleFallback(req, result)
	return result
}

func mergeFragmentReplacementTraceScenes(previous *domain.FragmentGenerationTrace, replacements []domain.FragmentScenePlan, oneBasedIndex int, originalImages []string) []domain.FragmentScenePlan {
	var scenes []domain.FragmentScenePlan
	if previous != nil {
		scenes = append(scenes, previous.Scenes...)
	}
	if len(replacements) == 0 {
		return scenes
	}
	replacement := replacements[0]
	replacement.Index = oneBasedIndex
	if oneBasedIndex > 0 && oneBasedIndex <= len(originalImages) {
		originalURL := strings.TrimSpace(originalImages[oneBasedIndex-1])
		for i := range scenes {
			if originalURL != "" && strings.TrimSpace(scenes[i].GeneratedImageURL) == originalURL {
				scenes[i] = replacement
				return scenes
			}
		}
		// Legacy traces mixed zero-based initial indices with one-based appended
		// indices. Array order is safe only when all original pages are represented.
		if len(scenes) == len(originalImages) {
			scenes[oneBasedIndex-1] = replacement
			return scenes
		}
	}
	return append(scenes, replacement)
}

func fragmentPrioritizedReferenceSlots(slots []domain.FragmentReferenceSlot) []domain.FragmentReferenceSlot {
	out := make([]domain.FragmentReferenceSlot, 0, len(slots))
	for _, slot := range slots {
		if slot.Required {
			out = append(out, slot)
		}
	}
	for _, kind := range []string{"character", "prop", "location"} {
		for _, slot := range slots {
			if !slot.Required && slot.Kind == kind {
				out = append(out, slot)
			}
		}
	}
	for _, slot := range slots {
		if !slot.Required && slot.Kind != "character" && slot.Kind != "prop" && slot.Kind != "location" {
			out = append(out, slot)
		}
	}
	return out
}

// Structured page edits keep the authored panel sequence, performance and text.
// Only artwork instructions change; legacy bitmaps use the constrained planner.
func fragmentExistingPageEditPlan(edit *fragmentDraftEditContext, changes string) ([]domain.FragmentScenePlan, bool) {
	if edit == nil || edit.Document == nil {
		return nil, false
	}
	for _, page := range edit.Document.Pages {
		if page.PageIndex != edit.TargetIndex || len(page.Plan.Panels) == 0 {
			continue
		}
		for _, panel := range page.Plan.Panels {
			if strings.TrimSpace(panel.ImagePrompt) == "" {
				return nil, false
			}
		}
		plan := page.Plan
		plan.Panels = cloneFragmentComicPanels(plan.Panels)
		directive, _ := json.Marshal(map[string]any{"requestedVisualChange": changes, "scope": "selected page artwork only; preserve each panel's event, text, entity identity, camera and layout; do not redraw a neighboring page"})
		keys := []string{}
		bindings := []domain.FragmentEntityBinding{}
		for i := range plan.Panels {
			plan.Panels[i].ImagePrompt += "\nPAGE-LOCAL VISUAL EDIT: " + string(directive)
			keys = append(keys, plan.Panels[i].ReferenceKeys...)
			bindings = append(bindings, plan.Panels[i].EntityBindings...)
		}
		plan.PlanningStatus = "reused_for_visual_edit"
		plan.PlanningError = ""
		return []domain.FragmentScenePlan{{Index: 0, SceneDesc: page.PageIntent, ImagePrompt: page.PageIntent, ReferenceKeys: normalizeFragmentKeyList(keys), EntityBindings: bindings, ComicPage: &plan}}, true
	}
	return nil, false
}

// Long drafts must retain the ending that the next page follows, rather than
// truncating away precisely the most recent causal context.
func fragmentContinuationNarrativeExcerpt(content string) string {
	runes := []rune(content)
	if len(runes) <= 2400 {
		return content
	}
	return string(runes[:400]) + "\n[中间正文略；以下为原稿末段]\n" + string(runes[len(runes)-2000:])
}

// Reusing a locked identity must not generate and bill a second asset that will
// be discarded by the authoritative previous-first asset merge afterwards.
func missingFragmentReferenceCandidates(candidates []fragmentReferenceAssetCandidate, existing []domain.FragmentReferenceAsset) []fragmentReferenceAssetCandidate {
	present := map[string]bool{}
	for _, asset := range existing {
		if strings.TrimSpace(asset.ImageURL) != "" {
			present[asset.Kind+"|"+asset.Key] = true
		}
	}
	out := make([]fragmentReferenceAssetCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		if !present[candidate.Kind+"|"+candidate.Key] {
			out = append(out, candidate)
		}
	}
	return out
}
