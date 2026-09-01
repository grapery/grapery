package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

func fragmentPlansContainComicPages(scenes []domain.FragmentScenePlan) bool {
	for _, scene := range scenes {
		if scene.ComicPage != nil {
			return true
		}
	}
	return false
}

func fragmentHighSeverityComicPageIndexes(issues []domain.FragmentConsistencyIssue, imageURLs []string, sceneCount int) []int {
	byURL := make(map[string]int, len(imageURLs))
	for i, raw := range imageURLs {
		if value := strings.TrimSpace(raw); value != "" {
			byURL[value] = i
		}
	}
	seen := map[int]struct{}{}
	for _, issue := range issues {
		if !strings.EqualFold(strings.TrimSpace(issue.Severity), "high") {
			continue
		}
		index := issue.SceneIndex
		if mapped, ok := byURL[strings.TrimSpace(issue.ImageURL)]; ok {
			index = mapped
		}
		if index < 0 || index >= sceneCount {
			continue
		}
		seen[index] = struct{}{}
	}
	out := make([]int, 0, len(seen))
	for index := range seen {
		out = append(out, index)
	}
	sort.Ints(out)
	return out
}

func remapFragmentConsistencyIssuesForResult(issues []domain.FragmentConsistencyIssue, appendOffset, replaceImageIndex int) []domain.FragmentConsistencyIssue {
	out := append([]domain.FragmentConsistencyIssue(nil), issues...)
	for i := range out {
		if replaceImageIndex > 0 {
			out[i].SceneIndex = replaceImageIndex - 1
		} else if appendOffset > 0 {
			out[i].SceneIndex += appendOffset
		}
	}
	return out
}

func fragmentComicPageRepairDirective(index int, imageURL string, issues []domain.FragmentConsistencyIssue) string {
	var lines []string
	for _, issue := range issues {
		if !strings.EqualFold(strings.TrimSpace(issue.Severity), "high") {
			continue
		}
		issueURL := strings.TrimSpace(issue.ImageURL)
		if issue.SceneIndex != index && (issueURL == "" || issueURL != strings.TrimSpace(imageURL)) {
			continue
		}
		lines = append(lines, fmt.Sprintf("expected=%s; observed=%s; detail=%s",
			strings.TrimSpace(issue.Expected), strings.TrimSpace(issue.Observed), strings.TrimSpace(issue.Detail)))
	}
	if len(lines) == 0 {
		lines = append(lines, "the previous render failed the complete-page or identity audit")
	}
	return "\n\n[CORRECTIVE REGENERATION — MANDATORY] Redraw from the authoritative plan and reference assets. Fix every listed audit failure while preserving the story event. Do not copy the faulty composition. Preserve character identity, wardrobe, performance, spatial relations, and reserved overlay-text space. Audit failures:\n- " + strings.Join(lines, "\n- ")
}

func setFragmentGenerationSlotStatus(slots []domain.FragmentGenerationImageSlot, index int, status string) []domain.FragmentGenerationImageSlot {
	out := append([]domain.FragmentGenerationImageSlot(nil), slots...)
	for i := range out {
		if out[i].Index == index {
			out[i].Status = status
			out[i].ErrorMessage = ""
			break
		}
	}
	return out
}

// repairHighSeverityFragmentComicPages performs one corrective render for each
// page that failed a high-severity audit. It intentionally uses the clean
// identity assets rather than the faulty page as a new visual reference.
func (s *FragmentGenerationService) repairHighSeverityFragmentComicPages(
	ctx context.Context,
	userID, taskID, aspectRatio, language string,
	bible *domain.FragmentVisualBible,
	scenes []domain.FragmentScenePlan,
	referenceAssets []domain.FragmentReferenceAsset,
	userReferenceURLs []string,
	policy *domain.FragmentConsistencyPolicy,
	issues []domain.FragmentConsistencyIssue,
	imageURLs []string,
	partial *domain.FragmentGenerationResult,
	tokenBase int,
) ([]domain.FragmentScenePlan, []string, int, int) {
	indexes := fragmentHighSeverityComicPageIndexes(issues, imageURLs, len(scenes))
	if len(indexes) == 0 {
		return scenes, imageURLs, 0, 0
	}
	outScenes := append([]domain.FragmentScenePlan(nil), scenes...)
	outURLs := append([]string(nil), imageURLs...)
	totalTokens := 0
	repaired := 0
	for _, index := range indexes {
		if !s.fragmentGenerationTaskCanContinue(ctx, taskID) {
			break
		}
		scene := &outScenes[index]
		if scene.ComicPage != nil {
			repairScene := *scene
			// A corrective render must not reuse the assets that failed review.
			repairScene.PanelImageURLs = nil
			repairPlan := *scene.ComicPage
			repairPlan.Panels = append([]domain.FragmentComicPanelPlan(nil), scene.ComicPage.Panels...)
			directive := fragmentComicPageRepairDirective(index, outURLs[index], issues)
			for panelIndex := range repairPlan.Panels {
				repairPlan.Panels[panelIndex].ImagePrompt += directive
			}
			repairScene.ComicPage = &repairPlan
			var repairPolicy domain.FragmentConsistencyPolicy
			if policy != nil {
				repairPolicy = *policy
			}
			repairPolicy.SeriesSeed = fragmentStoryImageSeed(policy) + (index+1)*7919
			pageURL, panelURLs, tokens, err := s.generateFragmentComicPageFromPanels(
				ctx, userID, taskID, language, bible, &repairScene, referenceAssets, userReferenceURLs, &repairPolicy,
			)
			totalTokens += tokens
			if err == nil && strings.TrimSpace(pageURL) != "" {
				scene.Seed = repairPolicy.SeriesSeed
				scene.PanelImageURLs = append([]string(nil), panelURLs...)
				scene.GeneratedImageURL = strings.TrimSpace(pageURL)
				outURLs[index] = strings.TrimSpace(pageURL)
				repaired++
				if partial != nil {
					partial.ImageSlots = buildFragmentGenerationProgressSlots(taskID, partial, outScenes, outURLs, nil)
					partial.ImageProgress = fragmentGenerationProgressFromSlots(partial.ImageSlots)
					s.persistFragmentImagePartial(ctx, taskID, partial, outScenes, outURLs, tokenBase, totalTokens)
				}
				continue
			}
			s.logger.Warn("deterministic comic page repair failed; keeping previous page",
				zap.String("task_id", taskID), zap.Int("scene_index", index), zap.Error(err))
			continue
		}
		prompt := buildFragmentSceneImagePrompt(bible, *scene, language) + fragmentComicPageRepairDirective(index, outURLs[index], issues)
		seed := fragmentStoryImageSeed(policy) + (index+1)*7919
		options := cloneFragmentProviderOptions(policy)
		refs := mergeFragmentSceneReferenceAssets(userReferenceURLs, *scene, referenceAssets, fragmentMaxSceneReferenceImages)
		payload := map[string]interface{}{
			"prompt":      prompt,
			"aspectRatio": aspectRatio,
			"seed":        seed,
		}
		if len(options) > 0 {
			payload["options"] = options
		}
		if len(refs) > 0 {
			payload["referenceImages"] = refs
		}
		input, _ := json.Marshal(payload)
		if partial != nil {
			slotIndex := fragmentGenerationProgressSlotIndex(partial, len(outScenes), index)
			partial.ImageSlots = setFragmentGenerationSlotStatus(partial.ImageSlots, slotIndex, "generating")
			partial.ImageProgress = fragmentGenerationProgressFromSlots(partial.ImageSlots)
			s.persistFragmentImagePartial(ctx, taskID, partial, outScenes, outURLs, tokenBase, totalTokens)
		}
		aiReq := &domain.AITask{
			ID:                uuid.NewString(),
			UserID:            userID,
			Type:              domain.AITaskGenerateFragmentImages,
			Status:            domain.AITaskStatusProcessing,
			Input:             string(input),
			RelatedEntityID:   taskID,
			RelatedEntityType: "fragment_generation",
		}
		imageURL, tokens, err := s.generateFragmentSceneImageWithRetry(ctx, aiReq, scene.Index)
		totalTokens += tokens
		if err != nil || strings.TrimSpace(imageURL) == "" {
			s.logger.Warn("fragment comic page corrective regeneration failed",
				zap.String("task_id", taskID), zap.Int("scene_index", index), zap.Error(err))
			if partial != nil {
				slotIndex := fragmentGenerationProgressSlotIndex(partial, len(outScenes), index)
				partial.ImageSlots = setFragmentGenerationSlotStatus(partial.ImageSlots, slotIndex, "completed")
				partial.ImageProgress = fragmentGenerationProgressFromSlots(partial.ImageSlots)
			}
			continue
		}
		scene.Seed = seed
		scene.ProviderOptions = options
		scene.FinalImagePrompt = prompt
		scene.GeneratedImageURL = strings.TrimSpace(imageURL)
		outURLs[index] = strings.TrimSpace(imageURL)
		repaired++
		if partial != nil {
			partial.ImageSlots = buildFragmentGenerationProgressSlots(taskID, partial, outScenes, outURLs, nil)
			partial.ImageProgress = fragmentGenerationProgressFromSlots(partial.ImageSlots)
			s.persistFragmentImagePartial(ctx, taskID, partial, outScenes, outURLs, tokenBase, totalTokens)
		}
	}
	return outScenes, outURLs, totalTokens, repaired
}
